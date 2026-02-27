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
		case app.ModeAwaitReadPath,
			app.ModeAwaitWebSearch, app.ModeAwaitWebURL,
			app.ModeAwaitProjectPath:
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
		}
	})

	model := snap.SelectedModel
	if model == "" {
		model = skill.DefaultModel
	}

	switch snap.Mode {
	case app.ModeIdle:
		return h.sendMenu(c, "💡 請使用下方選單操作，或輸入 /start 顯示主選單")

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

	default:
		return h.sendMenu(c, "💡 請選擇操作")
	}
}

// execReadCode performs a synchronous file read (fast, no task lock needed).
func (h *Hub) execReadCode(c tele.Context, relPath string) error {
	slog.Info("📁 讀取代碼", "path", relPath, "user_id", c.Sender().ID)

	result, err := skill.ReadCode(context.Background(), h.workspaceFor(c.Sender().ID), relPath)
	if err != nil {
		return h.sendMenu(c, "❌ "+err.Error())
	}

	chunks := skill.SplitMessage(result)
	for i, chunk := range chunks {
		if i == len(chunks)-1 {
			return c.Send(chunk, MainMenu, tele.ModeMarkdown)
		}
		c.Send(chunk, tele.ModeMarkdown)
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
			MainMenu,
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
		return h.sendMenu(c, "❌ 搜尋失敗："+err.Error())
	}
	if len(results) == 0 {
		return h.sendMenu(c, "🔍 未找到相關結果")
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🔍 *搜尋結果*：`%s`\n\n", query))
	for i, r := range results {
		sb.WriteString(fmt.Sprintf("%d. [%s](%s)\n", i+1, r.Title, r.URL))
		if r.Desc != "" {
			sb.WriteString(fmt.Sprintf("   %s\n", r.Desc))
		}
		sb.WriteString("\n")
	}
	return c.Send(sb.String(), MainMenu, tele.ModeMarkdown)
}

// execWebFetch fetches a URL and displays extracted text content.
func (h *Hub) execWebFetch(c tele.Context, rawURL string) error {
	slog.Info("🌐 Web 擷取", "url", rawURL, "user_id", c.Sender().ID)
	c.Send("🌐 擷取中...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	text, err := skill.WebFetch(ctx, rawURL)
	if err != nil {
		return h.sendMenu(c, "❌ 擷取失敗："+err.Error())
	}

	header := fmt.Sprintf("🌐 *擷取結果*：`%s`\n\n", rawURL)
	chunks := skill.SplitMessage(header + text)
	for i, chunk := range chunks {
		if i == len(chunks)-1 {
			return c.Send(chunk, MainMenu)
		}
		c.Send(chunk)
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
