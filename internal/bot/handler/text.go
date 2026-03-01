package handler

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/garyellow/axle/internal/app"
	"github.com/garyellow/axle/internal/bot/skill"
	tele "gopkg.in/telebot.v3"
)

// HandleText routes incoming text messages based on the user's current session mode.
func (h *Hub) HandleText(c tele.Context) error {
	userID := c.Sender().ID
	text := c.Text()

	slog.Info("💬 文字訊息", "user_id", userID, "text", text)

	// Atomically read current mode and transition state
	var snap app.UserSession
	h.Sessions.Update(userID, func(s *app.UserSession) {
		snap = *s // capture state before transition
		switch s.Mode {
		case app.ModeIdle:
			s.Mode = app.ModeAwaitCopilotPrompt // NLP routing: enter copilot session
		case app.ModeAwaitReadPath,
			app.ModeAwaitWebSearch, app.ModeAwaitWebURL,
			app.ModeAwaitProjectPath,
			app.ModeAwaitListPath, app.ModeAwaitSearchQuery:
			s.Mode = app.ModeIdle // consume: next message starts fresh
		case app.ModeAwaitCopilotPrompt:
			// Stay in Copilot session — mode remains ModeAwaitCopilotPrompt
		case app.ModeAwaitExecCmd:
			s.Mode = app.ModeAwaitExecConfirm
			s.PendingCmd = text
		case app.ModeAwaitWritePath:
			s.Mode = app.ModeAwaitWriteContent
			s.PendingPath = text
		case app.ModeAwaitWriteContent:
			s.Mode = app.ModeIdle
		case app.ModeAwaitGitCommitMsg:
			s.Mode = app.ModeIdle
			s.PendingCmd = text
		case app.ModeAwaitSubAgentName:
			s.Mode = app.ModeAwaitSubAgentTask
			s.PendingAgent = text
		case app.ModeAwaitSubAgentTask:
			s.Mode = app.ModeIdle
		case app.ModeAwaitSchedName:
			s.Mode = app.ModeAwaitSchedInterval
			s.PendingSchedName = text
		case app.ModeAwaitSchedInterval:
			s.Mode = app.ModeAwaitSchedCommand
			s.PendingSchedCmd = "" // interval stored in PendingCmd temporarily
			s.PendingCmd = text
		case app.ModeAwaitSchedCommand:
			s.Mode = app.ModeIdle
		case app.ModeAwaitEmailTo:
			s.Mode = app.ModeAwaitEmailSubject
			s.PendingEmailTo = text
		case app.ModeAwaitEmailSubject:
			s.Mode = app.ModeAwaitEmailBody
			s.PendingEmailSubj = text
		case app.ModeAwaitEmailBody:
			s.Mode = app.ModeIdle
		case app.ModeAwaitGHPRTitle:
			s.Mode = app.ModeAwaitGHPRBody
			s.PendingPRTitle = text
		case app.ModeAwaitGHPRBody:
			s.Mode = app.ModeIdle
		}
	})

	model := snap.SelectedModel
	if model == "" {
		model = skill.DefaultModel
	}

	switch snap.Mode {
	case app.ModeIdle:
		// NLP routing: auto-forward to Copilot session
		slog.Info("🧠 NLP 路由 → Copilot", "user_id", userID)
		return h.RunCopilotTask(c, text, model)

	case app.ModeAwaitReadPath:
		return h.execReadCode(c, text)

	case app.ModeAwaitExecCmd:
		return h.showExecConfirm(c, text)

	case app.ModeAwaitExecConfirm:
		return c.Send("⏳ 請先使用上方按鈕確認或取消指令", ExecMenu)

	case app.ModeAwaitCopilotPrompt:
		return h.RunCopilotTask(c, text, model)

	case app.ModeAwaitWritePath:
		return h.handleWritePathInput(c, text)

	case app.ModeAwaitWriteContent:
		return h.handleWriteContentInput(c, snap.PendingPath, text)

	case app.ModeAwaitWebSearch:
		return h.execWebSearch(c, text)

	case app.ModeAwaitWebURL:
		return h.execWebFetch(c, text)

	case app.ModeAwaitProjectPath:
		return h.handleProjectPathInput(c, text)

	case app.ModeAwaitListPath:
		return h.execListDir(c, text)

	case app.ModeAwaitSearchQuery:
		return h.execSearchCode(c, text)

	case app.ModeAwaitGitCommitMsg:
		return h.showGitCommitConfirm(c, text)

	case app.ModeAwaitSubAgentName:
		return c.Send("📝 請輸入子代理的任務描述：")

	case app.ModeAwaitSubAgentTask:
		return h.execCreateSubAgent(c, snap.PendingAgent, text)

	case app.ModeAwaitSchedName:
		return c.Send("⏱ 請輸入執行間隔（分鐘）：")

	case app.ModeAwaitSchedInterval:
		return c.Send("⚡ 請輸入要執行的 Shell 指令：")

	case app.ModeAwaitSchedCommand:
		return h.execCreateSchedule(c, snap.PendingSchedName, snap.PendingCmd, text)

	case app.ModeAwaitEmailTo:
		return c.Send("📝 請輸入郵件主旨：")

	case app.ModeAwaitEmailSubject:
		return c.Send("✉️ 請輸入郵件內文：")

	case app.ModeAwaitEmailBody:
		return h.execSendEmail(c, snap.PendingEmailTo, snap.PendingEmailSubj, text)

	case app.ModeAwaitGHPRTitle:
		return c.Send("📝 請輸入 PR 描述（body）：")

	case app.ModeAwaitGHPRBody:
		return h.execCreatePR(c, snap.PendingPRTitle, text)

	default:
		return h.sendMenu(c, "💡 請選擇操作")
	}
}

// execReadCode performs a synchronous file read (fast, no task lock needed).
func (h *Hub) execReadCode(c tele.Context, relPath string) error {
	slog.Info("📁 讀取代碼", "path", relPath, "user_id", c.Sender().ID)

	result, err := skill.ReadCode(context.Background(), h.workspaceFor(c.Sender().ID), relPath)
	if err != nil {
		h.emitRPG("read_code", relPath, false)
		return h.sendMenu(c, "❌ "+err.Error())
	}
	h.emitRPG("read_code", relPath, true)

	chunks := skill.SplitMessage(result)
	for i, chunk := range chunks {
		if i == len(chunks)-1 {
			return c.Send(chunk, h.mm(c), tele.ModeMarkdown)
		}
		if _, err := c.Bot().Send(c.Chat(), chunk, tele.ModeMarkdown); err != nil {
			slog.Warn("⚠️ 訊息發送失敗", "chunk", i, "error", err)
		}
	}
	return nil
}

// showExecConfirm shows the command preview with confirm/cancel buttons.
// Dangerous commands get an extra warning and a different confirmation button.
func (h *Hub) showExecConfirm(c tele.Context, cmd string) error {
	level, reasons := skill.CheckCommandSafety(cmd)

	switch level {
	case skill.DangerBlocked:
		slog.Warn("⛔ 指令被封鎖", "cmd", cmd, "reasons", reasons, "user_id", c.Sender().ID)
		h.Sessions.Reset(c.Sender().ID)
		return c.Send(
			fmt.Sprintf("⛔ *指令被封鎖*\n\n偵測到極度危險操作：\n• %s\n\n此指令已被拒絕執行。", strings.Join(reasons, "\n• ")),
			h.mm(c),
			tele.ModeMarkdown,
		)

	case skill.DangerWarning:
		slog.Warn("⚠️ 危險指令需二次確認", "cmd", cmd, "reasons", reasons, "user_id", c.Sender().ID)
		return c.Send(
			fmt.Sprintf(
				"🔴 *危險指令偵測*\n\n"+
					"```bash\n%s\n```\n\n"+
					"⚠️ 偵測到以下風險：\n• %s\n\n"+
					"*你確定要執行嗎？此操作可能無法復原。*",
				cmd, strings.Join(reasons, "\n• "),
			),
			ExecDangerMenu,
			tele.ModeMarkdown,
		)

	default:
		slog.Info("⚡ 等待確認", "cmd", cmd, "user_id", c.Sender().ID)
		return c.Send(
			fmt.Sprintf("⚡ *準備執行指令*\n\n```bash\n%s\n```\n\n確認執行？", cmd),
			ExecMenu,
			tele.ModeMarkdown,
		)
	}
}

// ── Write file flow ───────────────────────────────────────────────────────────

// handleWritePathInput validates path and prompts for content.
func (h *Hub) handleWritePathInput(c tele.Context, relPath string) error {
	slog.Info("✏️ 寫入路徑", "path", relPath, "user_id", c.Sender().ID)

	exists, err := skill.FileExists(h.workspaceFor(c.Sender().ID), relPath)
	if err != nil {
		h.Sessions.Reset(c.Sender().ID)
		return h.sendMenu(c, "❌ "+err.Error())
	}

	warning := ""
	if exists {
		warning = "\n⚠️ *此檔案已存在，將會被覆蓋！*"
	}

	return c.Send(
		fmt.Sprintf("✏️ 準備寫入：`%s`%s\n\n請輸入檔案內容：", relPath, warning),
		tele.ModeMarkdown,
	)
}

// handleWriteContentInput performs the actual file write.
func (h *Hub) handleWriteContentInput(c tele.Context, relPath, content string) error {
	slog.Info("✏️ 寫入檔案", "path", relPath, "size", len(content), "user_id", c.Sender().ID)

	if err := skill.WriteFile(h.workspaceFor(c.Sender().ID), relPath, content); err != nil {
		return h.sendMenu(c, "❌ "+err.Error())
	}

	return h.sendMenu(c, fmt.Sprintf("✅ 檔案已寫入：`%s`（%d bytes）", relPath, len(content)))
}

// ── Web search/fetch ──────────────────────────────────────────────────────────

// execWebSearch performs a DuckDuckGo search and displays results.
func (h *Hub) execWebSearch(c tele.Context, query string) error {
	slog.Info("🔍 Web 搜尋", "query", query, "user_id", c.Sender().ID)
	c.Send("🔍 搜尋中...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	results, err := skill.WebSearch(ctx, query)
	if err != nil {
		h.emitRPG("web_search", query, false)
		return h.sendMenu(c, "❌ 搜尋失敗："+err.Error())
	}
	if len(results) == 0 {
		h.emitRPG("web_search", query, true)
		return h.sendMenu(c, "🔍 未找到相關結果")
	}
	h.emitRPG("web_search", query, true)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🔍 *搜尋結果*：`%s`\n\n", query))
	for i, r := range results {
		sb.WriteString(fmt.Sprintf("%d. [%s](%s)\n", i+1, r.Title, r.URL))
		if r.Desc != "" {
			sb.WriteString(fmt.Sprintf("   %s\n", r.Desc))
		}
		sb.WriteString("\n")
	}
	return c.Send(sb.String(), h.mm(c), tele.ModeMarkdown)
}

// execWebFetch fetches a URL and displays extracted text content.
func (h *Hub) execWebFetch(c tele.Context, rawURL string) error {
	slog.Info("🌐 Web 擷取", "url", rawURL, "user_id", c.Sender().ID)
	c.Send("🌐 擷取中...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	text, err := skill.WebFetch(ctx, rawURL)
	if err != nil {
		h.emitRPG("web_fetch", rawURL, false)
		return h.sendMenu(c, "❌ 擷取失敗："+err.Error())
	}
	h.emitRPG("web_fetch", rawURL, true)

	header := fmt.Sprintf("🌐 *擷取結果*：`%s`\n\n", rawURL)
	chunks := skill.SplitMessage(header + text)
	for i, chunk := range chunks {
		if i == len(chunks)-1 {
			return c.Send(chunk, h.mm(c))
		}
		if _, err := c.Bot().Send(c.Chat(), chunk); err != nil {
			slog.Warn("⚠️ 訊息發送失敗", "chunk", i, "error", err)
		}
	}
	return nil
}

// ── Workspace switch flow ─────────────────────────────────────────────────────

// handleProjectPathInput validates and sets the user's active workspace.
func (h *Hub) handleProjectPathInput(c tele.Context, input string) error {
	userID := c.Sender().ID
	slog.Info("📂 切換專案路徑", "input", input, "user_id", userID)

	// "reset" returns to default workspace
	if strings.EqualFold(strings.TrimSpace(input), "reset") {
		h.Sessions.Update(userID, func(s *app.UserSession) { s.ActiveWorkspace = "" })
		return h.sendMenu(c, fmt.Sprintf("📂 已恢復為預設工作目錄：`%s`", h.Workspace))
	}

	absPath := strings.TrimSpace(input)

	// Must be absolute path
	if !strings.HasPrefix(absPath, "/") {
		return h.sendMenu(c, "❌ 請輸入絕對路徑（以 `/` 開頭）")
	}

	// Validate directory exists
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return h.sendMenu(c, "❌ 路徑不存在：`"+absPath+"`")
		}
		return h.sendMenu(c, "❌ 無法存取路徑："+err.Error())
	}
	if !info.IsDir() {
		return h.sendMenu(c, "❌ 路徑不是目錄：`"+absPath+"`")
	}

	h.Sessions.Update(userID, func(s *app.UserSession) { s.ActiveWorkspace = absPath })
	slog.Info("📂 工作目錄已切換", "workspace", absPath, "user_id", userID)
	return h.sendMenu(c, fmt.Sprintf("✅ 工作目錄已切換至：`%s`", absPath))
}

// ── Directory listing ─────────────────────────────────────────────────────────

// execListDir lists directory contents in tree format.
func (h *Hub) execListDir(c tele.Context, relPath string) error {
	slog.Info("📂 目錄瀏覽", "path", relPath, "user_id", c.Sender().ID)

	result, err := skill.ListDir(context.Background(), h.workspaceFor(c.Sender().ID), relPath, 3)
	if err != nil {
		h.emitRPG("list_dir", relPath, false)
		return h.sendMenu(c, "❌ "+err.Error())
	}
	h.emitRPG("list_dir", relPath, true)

	chunks := skill.SplitMessage("```\n" + result + "\n```")
	for i, chunk := range chunks {
		if i == len(chunks)-1 {
			return c.Send(chunk, h.mm(c), tele.ModeMarkdown)
		}
		if _, err := c.Bot().Send(c.Chat(), chunk, tele.ModeMarkdown); err != nil {
			slog.Warn("⚠️ 訊息發送失敗", "chunk", i, "error", err)
		}
	}
	return nil
}

// ── Code search ───────────────────────────────────────────────────────────────

// execSearchCode performs recursive code search.
func (h *Hub) execSearchCode(c tele.Context, pattern string) error {
	slog.Info("🔎 搜尋代碼", "pattern", pattern, "user_id", c.Sender().ID)
	c.Send("🔎 搜尋中...")

	results, err := skill.SearchCode(context.Background(), h.workspaceFor(c.Sender().ID), pattern)
	if err != nil {
		h.emitRPG("search_code", pattern, false)
		return h.sendMenu(c, "❌ 搜尋失敗："+err.Error())
	}
	h.emitRPG("search_code", pattern, true)

	text := skill.FormatSearchResults(pattern, results)
	chunks := skill.SplitMessage(text)
	for i, chunk := range chunks {
		if i == len(chunks)-1 {
			return c.Send(chunk, h.mm(c), tele.ModeMarkdown)
		}
		if _, err := c.Bot().Send(c.Chat(), chunk, tele.ModeMarkdown); err != nil {
			slog.Warn("⚠️ 訊息發送失敗", "chunk", i, "error", err)
		}
	}
	return nil
}

// ── Git operations ────────────────────────────────────────────────────────────

// showGitCommitConfirm shows commit message confirmation.
func (h *Hub) showGitCommitConfirm(c tele.Context, msg string) error {
	slog.Info("🔀 Git commit 確認", "msg", msg, "user_id", c.Sender().ID)
	return c.Send(
		fmt.Sprintf("🚀 *Git Commit + Push*\n\n提交訊息：\n```\n%s\n```\n\n⚠️ 此操作將 `git add -A` + `commit` + `push`\n確認執行？", msg),
		GitCommitMenu,
		tele.ModeMarkdown,
	)
}

// ── Sub-agent creation ────────────────────────────────────────────────────────

// execCreateSubAgent creates and runs a sub-agent.
func (h *Hub) execCreateSubAgent(c tele.Context, name, task string) error {
	userID := c.Sender().ID
	slog.Info("👥 建立子代理", "name", name, "task", task, "user_id", userID)

	if h.SubAgents == nil {
		return h.sendMenu(c, "⚠️ 子代理系統未初始化")
	}

	sess := h.Sessions.GetCopy(userID)
	model := sess.SelectedModel
	if model == "" {
		model = skill.DefaultModel
	}

	ws := h.workspaceFor(userID)
	agent, ctx := h.SubAgents.Create(userID, name, task, model, ws)
	chat := c.Chat()

	c.Send(fmt.Sprintf("🤖 子代理 `%s` 已建立\n任務：%s\n模型：`%s`\n⏳ 執行中...", agent.ID, task, model), tele.ModeMarkdown)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("💥 子代理 panic", "id", agent.ID, "recover", r)
				h.SubAgents.Fail(agent.ID, fmt.Sprintf("panic: %v", r))
				h.Bot.Send(chat, fmt.Sprintf("💥 子代理 `%s` 異常中止", agent.ID), h.mmFor(userID))
			}
		}()

		chunks, err := skill.RunCopilot(ctx, ws, model, task)
		if err != nil {
			h.SubAgents.Fail(agent.ID, err.Error())
			h.Bot.Send(chat, fmt.Sprintf("❌ 子代理 `%s` 失敗：%s", agent.ID, err.Error()), h.mmFor(userID))
			return
		}

		result := ""
		for _, ch := range chunks {
			result += ch
		}
		h.SubAgents.Complete(agent.ID, result)

		msg := fmt.Sprintf("✅ 子代理 `%s` 完成\n\n%s", agent.ID, result)
		msgChunks := skill.SplitMessage(msg)
		h.sendChunks(chat, msgChunks, userID)
	}()

	return nil
}

// ── Schedule creation ─────────────────────────────────────────────────────────

// execCreateSchedule creates a new scheduled task.
func (h *Hub) execCreateSchedule(c tele.Context, name, intervalStr, command string) error {
	slog.Info("⏰ 建立排程", "name", name, "interval", intervalStr, "cmd", command, "user_id", c.Sender().ID)

	if h.Scheduler == nil {
		return h.sendMenu(c, "⚠️ 排程系統未初始化")
	}

	interval := 0
	fmt.Sscanf(intervalStr, "%d", &interval)
	if interval <= 0 {
		return h.sendMenu(c, "❌ 間隔必須為正整數（分鐘）")
	}

	sched, err := h.Scheduler.Add(name, command, interval)
	if err != nil {
		return h.sendMenu(c, "❌ "+err.Error())
	}

	return h.sendMenu(c, fmt.Sprintf("✅ 排程已建立\n\n• ID：`%s`\n• 名稱：%s\n• 指令：`%s`\n• 間隔：每 %d 分鐘", sched.ID, sched.Name, sched.Command, sched.Interval))
}

// ── Email send ────────────────────────────────────────────────────────────────

func (h *Hub) execSendEmail(c tele.Context, to, subject, body string) error {
	slog.Info("📧 發送 Email", "to", to, "subject", subject, "user_id", c.Sender().ID)

	if h.EmailConfig == nil || !h.EmailConfig.IsConfigured() {
		return h.sendMenu(c, "❌ Email 未設定")
	}

	c.Send("📤 發送中...")
	if err := skill.SendEmail(*h.EmailConfig, to, subject, body); err != nil {
		return h.sendMenu(c, "❌ 發送失敗："+err.Error())
	}
	return h.sendMenu(c, fmt.Sprintf("✅ Email 已發送\n\n📬 收件人：`%s`\n📝 主旨：%s", to, subject))
}

// ── GitHub PR create ──────────────────────────────────────────────────────────

func (h *Hub) execCreatePR(c tele.Context, title, body string) error {
	slog.Info("🐙 建立 PR", "title", title, "user_id", c.Sender().ID)
	c.Send("➕ 建立 PR 中...")

	ctx, cancel := context.WithTimeout(context.Background(), skill.GHTimeout())
	defer cancel()

	out, err := skill.GHPRCreate(ctx, h.workspaceFor(c.Sender().ID), title, body)
	if err != nil {
		return h.sendMenu(c, "❌ "+err.Error())
	}
	return h.sendMenu(c, fmt.Sprintf("✅ PR 已建立\n\n%s", out))
}
