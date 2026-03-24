package handler

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/garyellow/axle/internal/app"
	"github.com/garyellow/axle/internal/bot/skill"
	tele "gopkg.in/telebot.v3"
)

// ── Menu skill buttons ────────────────────────────────────────────────────────

// HandleReadCodeBtn sets session to await file path input.
func (h *Hub) HandleReadCodeBtn(c tele.Context) error {
	userID := c.Sender().ID
	slog.Info("🔘 選單: 讀取代碼", "user_id", userID)
	_ = c.Respond()
	h.Sessions.Update(userID, func(s *app.UserSession) { s.Mode = app.ModeAwaitReadPath })
	return c.Send("📁 *讀取代碼*\n\n請輸入檔案路徑（相對於 workspace）\n範例：`cmd/axle/main.go`", tele.ModeMarkdown)
}

// HandleExecBtn sets session to await shell command input.
func (h *Hub) HandleExecBtn(c tele.Context) error {
	userID := c.Sender().ID
	slog.Info("🔘 選單: 執行指令", "user_id", userID)
	_ = c.Respond()
	h.Sessions.Update(userID, func(s *app.UserSession) { s.Mode = app.ModeAwaitExecCmd })
	return c.Send("⚡ *執行指令*\n\n請輸入 Shell 指令（下一步將請求確認）\n範例：`ls -la`", tele.ModeMarkdown)
}

// HandleCopilotBtn enters Copilot conversation mode using the current model.
// If no model is selected yet, shows model selection first.
func (h *Hub) HandleCopilotBtn(c tele.Context) error {
	userID := c.Sender().ID
	slog.Info("🔘 選單: Copilot 助手", "user_id", userID)
	_ = c.Respond()

	sess := h.Sessions.GetCopy(userID)
	model := sess.SelectedModel
	if model == "" {
		model = skill.DefaultModel
	}

	h.Sessions.Update(userID, func(s *app.UserSession) {
		s.Mode = app.ModeAwaitCopilotPrompt
		if s.SelectedModel == "" {
			s.SelectedModel = skill.DefaultModel
		}
	})

	return c.Send(
		fmt.Sprintf("🤖 *Copilot 對話模式*\n\n當前模型：`%s`\n請輸入你的問題或任務描述：\n\n_（可隨時切換模型或返回主選單）_", model),
		CopilotSessionMenu,
		tele.ModeMarkdown,
	)
}

// HandleStatus shows current task status and workspace info.
func (h *Hub) HandleStatus(c tele.Context) error {
	slog.Info("🔘 選單: 系統狀態", "user_id", c.Sender().ID)
	_ = c.Respond()

	sess := h.Sessions.GetCopy(c.Sender().ID)
	model := sess.SelectedModel
	if model == "" {
		model = skill.DefaultModel + "（預設）"
	}

	running, taskName, elapsed := h.Tasks.Status()
	taskStatus := "待機中 🟢"
	if running {
		taskStatus = fmt.Sprintf("執行中 🔴\n  任務：%s\n  耗時：%.0f 秒", taskName, elapsed.Seconds())
	}

	ws := h.workspaceFor(c.Sender().ID)
	wsLabel := ws
	if sess.ActiveWorkspace != "" {
		wsLabel += "（自訂）"
	} else {
		wsLabel += "（預設）"
	}

	// Sub-agent count
	agentCount := 0
	if h.SubAgents != nil {
		agentCount = h.SubAgents.RunningCount(c.Sender().ID)
	}

	// Memory count
	memCount := 0
	if h.Memory != nil {
		memCount = h.Memory.Count(c.Sender().ID)
	}

	// Plugin count
	pluginCount := 0
	if h.Plugins != nil {
		pluginCount = h.Plugins.Count()
	}

	// Schedule count
	schedCount := 0
	if h.Scheduler != nil {
		schedCount = len(h.Scheduler.List())
	}

	// Workflow count
	workflowCount := 0
	if h.Workflows != nil {
		workflowCount = h.Workflows.RunningCount(c.Sender().ID)
	}

	return c.Send(fmt.Sprintf(
		"📊 *系統狀態*\n\n"+
			"• 版本：`v%s`\n"+
			"• 任務狀態：%s\n"+
			"• 選定模型：`%s`\n"+
			"• Workspace：`%s`\n"+
			"• 對話記憶：%d 筆\n"+
			"• 子代理：%d 個執行中\n"+
			"• 工作流：%d 個執行中\n"+
			"• 擴充技能：%d 個\n"+
			"• 排程任務：%d 個",
		app.Version, taskStatus, model, wsLabel, memCount, agentCount, workflowCount, pluginCount, schedCount,
	), h.mm(c), tele.ModeMarkdown)
}

// HandleCancelTask cancels the currently running task.
func (h *Hub) HandleCancelTask(c tele.Context) error {
	slog.Info("🔘 選單: 取消任務", "user_id", c.Sender().ID)
	_ = c.Respond()

	if cancelled := h.Tasks.Cancel(); !cancelled {
		return c.Send("ℹ️ 目前沒有正在執行的任務", h.mm(c))
	}
	// Also reset session mode
	h.Sessions.Reset(c.Sender().ID)
	return c.Send("🛑 取消信號已發送，任務將停止...", h.mm(c))
}

// HandleCancel is the /cancel command (same as cancel button).
func (h *Hub) HandleCancel(c tele.Context) error {
	slog.Info("🎯 /cancel", "user_id", c.Sender().ID)
	if cancelled := h.Tasks.Cancel(); !cancelled {
		h.Sessions.Reset(c.Sender().ID)
		return c.Send("ℹ️ 目前沒有正在執行的任務", h.mm(c))
	}
	h.Sessions.Reset(c.Sender().ID)
	return c.Send("🛑 取消信號已發送，任務將停止...", h.mm(c))
}

// ── Exec confirm/cancel buttons ───────────────────────────────────────────────

// HandleExecConfirm executes the pending command after user confirmation.
func (h *Hub) HandleExecConfirm(c tele.Context) error {
	userID := c.Sender().ID
	slog.Info("✅ 確認執行指令", "user_id", userID)
	_ = c.Respond()

	var pendingCmd string
	h.Sessions.Update(userID, func(s *app.UserSession) {
		pendingCmd = s.PendingCmd
		s.Mode = app.ModeIdle
		s.PendingCmd = ""
	})

	if pendingCmd == "" {
		return h.sendMenu(c, "⚠️ 沒有待執行的指令")
	}
	return h.RunExecTask(c, pendingCmd)
}

// HandleExecDangerConfirm executes a dangerous command after explicit secondary confirmation.
func (h *Hub) HandleExecDangerConfirm(c tele.Context) error {
	userID := c.Sender().ID
	slog.Warn("⚠️ 用戶確認執行危險指令", "user_id", userID)
	_ = c.Respond()

	var pendingCmd string
	h.Sessions.Update(userID, func(s *app.UserSession) {
		pendingCmd = s.PendingCmd
		s.Mode = app.ModeIdle
		s.PendingCmd = ""
	})

	if pendingCmd == "" {
		return h.sendMenu(c, "⚠️ 沒有待執行的指令")
	}
	return h.RunExecTask(c, pendingCmd)
}

// HandleExecCancelBtn cancels the pending exec confirmation.
func (h *Hub) HandleExecCancelBtn(c tele.Context) error {
	slog.Info("❌ 取消執行確認", "user_id", c.Sender().ID)
	_ = c.Respond()
	h.Sessions.Reset(c.Sender().ID)
	return h.sendMenu(c, "❌ 指令已取消")
}

// ── Model selection (two-step: vendor → model) ───────────────────────────────

// HandleVendorSelect handles a vendor choice, showing models for that vendor.
// Payload: c.Args() = [source, vendor]
func (h *Hub) HandleVendorSelect(c tele.Context) error {
	userID := c.Sender().ID
	args := c.Args()
	if len(args) < 2 {
		_ = c.Respond()
		return h.sendMenu(c, "⚠️ 未收到廠商名稱")
	}

	source := args[0]
	vendor := args[1]

	slog.Info("🔘 選擇廠商", "vendor", vendor, "source", source, "user_id", userID)
	_ = c.Respond()

	label := skill.ProviderLabel[vendor]
	if label == "" {
		label = vendor
	}

	return c.Send(
		fmt.Sprintf("🔄 *%s 模型列表*\n\n請選擇模型：", label),
		BuildModelMenu(source, vendor),
		tele.ModeMarkdown,
	)
}

// HandleModelSelect handles a model choice from the model selection keyboard.
// Payload: c.Args() = [source, modelID]
func (h *Hub) HandleModelSelect(c tele.Context) error {
	userID := c.Sender().ID
	args := c.Args()
	if len(args) < 2 {
		_ = c.Respond()
		return h.sendMenu(c, "⚠️ 未收到模型名稱")
	}

	source := args[0]
	model := args[1]

	slog.Info("🔘 選擇模型", "model", model, "source", source, "user_id", userID)
	_ = c.Respond()

	// Update RPG equipment (weapon = model)
	if h.RPG != nil {
		snap := h.RPG.Snapshot()
		snap.Equipment.Weapon = model
		h.RPG.UpdateEquipment(snap.Equipment)
	}

	switch source {
	case "copilot":
		// In-session model switch: stay in Copilot prompt mode
		h.Sessions.Update(userID, func(s *app.UserSession) {
			s.SelectedModel = model
			s.Mode = app.ModeAwaitCopilotPrompt
		})
		return c.Send(
			fmt.Sprintf("✅ 模型已切換：`%s`\n\n請繼續輸入問題：", model),
			CopilotSessionMenu,
			tele.ModeMarkdown,
		)
	default:
		// From main menu: just set the model and return to main menu
		h.Sessions.Update(userID, func(s *app.UserSession) {
			s.SelectedModel = model
			s.Mode = app.ModeIdle
		})
		return c.Send(
			fmt.Sprintf("✅ 全局模型已設定：`%s`\n\n後續所有 Copilot 操作都會使用此模型。", model),
			h.mm(c),
			tele.ModeMarkdown,
		)
	}
}

// HandleBackToMain returns the user to the main menu.
func (h *Hub) HandleBackToMain(c tele.Context) error {
	slog.Info("⬅️ 返回主選單", "user_id", c.Sender().ID)
	_ = c.Respond()
	h.Sessions.Reset(c.Sender().ID)
	return h.sendMenu(c, "🔧 主選單")
}

// HandleBackToVendor returns the user to the vendor selection menu.
// Payload: c.Args() = [source]
func (h *Hub) HandleBackToVendor(c tele.Context) error {
	slog.Info("⬅️ 返回廠商列表", "user_id", c.Sender().ID)
	_ = c.Respond()

	source := "main"
	if args := c.Args(); len(args) > 0 {
		source = args[0]
	}

	sess := h.Sessions.GetCopy(c.Sender().ID)
	current := sess.SelectedModel
	if current == "" {
		current = skill.DefaultModel + "（預設）"
	}

	return c.Send(
		fmt.Sprintf("🔄 *選擇廠商*\n\n當前模型：`%s`\n請選擇 AI 廠商：", current),
		BuildVendorMenu(source),
		tele.ModeMarkdown,
	)
}

// ── Write file button ─────────────────────────────────────────────────────────

// HandleWriteFileBtn sets session to await file path input for writing.
func (h *Hub) HandleWriteFileBtn(c tele.Context) error {
	userID := c.Sender().ID
	slog.Info("🔘 選單: 寫入檔案", "user_id", userID)
	_ = c.Respond()
	h.Sessions.Update(userID, func(s *app.UserSession) { s.Mode = app.ModeAwaitWritePath })
	return c.Send("✏️ *寫入檔案*\n\n請輸入檔案路徑（相對於 workspace）\n範例：`notes/todo.md`", tele.ModeMarkdown)
}

// ── Web buttons ───────────────────────────────────────────────────────────────

// HandleWebSearchBtn sets session to await search query input.
func (h *Hub) HandleWebSearchBtn(c tele.Context) error {
	userID := c.Sender().ID
	slog.Info("🔘 選單: Web 搜尋", "user_id", userID)
	_ = c.Respond()
	h.Sessions.Update(userID, func(s *app.UserSession) { s.Mode = app.ModeAwaitWebSearch })
	return c.Send("🔍 *Web 搜尋*\n\n請輸入搜尋關鍵字：", tele.ModeMarkdown)
}

// HandleWebFetchBtn sets session to await URL input.
func (h *Hub) HandleWebFetchBtn(c tele.Context) error {
	userID := c.Sender().ID
	slog.Info("🔘 選單: Web 擷取", "user_id", userID)
	_ = c.Respond()
	h.Sessions.Update(userID, func(s *app.UserSession) { s.Mode = app.ModeAwaitWebURL })
	return c.Send("🌐 *Web 擷取*\n\n請輸入網址（URL）\n範例：`https://example.com`", tele.ModeMarkdown)
}

// HandleMemoryBtn shows the memory/history submenu.
func (h *Hub) HandleMemoryBtn(c tele.Context) error {
	userID := c.Sender().ID
	slog.Info("🔘 選單: 記憶 / 歷史", "user_id", userID)
	_ = c.Respond()

	if h.Memory == nil {
		return h.sendMenu(c, "⚠️ 記憶系統未初始化")
	}

	return c.Send(
		fmt.Sprintf("🧠 *記憶 / 歷史*\n\n目前已儲存：%d 筆\n\n請選擇操作：", h.Memory.Count(userID)),
		MemoryMenu,
		tele.ModeMarkdown,
	)
}

// HandleMemorySearch starts the memory search flow.
func (h *Hub) HandleMemorySearch(c tele.Context) error {
	userID := c.Sender().ID
	slog.Info("🧠 搜尋歷史", "user_id", userID)
	_ = c.Respond()
	h.Sessions.Update(userID, func(s *app.UserSession) { s.Mode = app.ModeAwaitMemorySearch })
	return c.Send("🔎 *搜尋歷史*\n\n請輸入關鍵字：\n範例：`Taipei weather`", tele.ModeMarkdown)
}

// HandleMemoryRecent shows the most recent memory entries.
func (h *Hub) HandleMemoryRecent(c tele.Context) error {
	userID := c.Sender().ID
	slog.Info("🧠 最近對話", "user_id", userID)
	_ = c.Respond()

	if h.Memory == nil {
		return c.Send("⚠️ 記憶系統未初始化", MemoryMenu)
	}

	entries := h.Memory.Recent(userID, 10)
	if len(entries) == 0 {
		return c.Send("🕘 目前沒有任何記憶紀錄", MemoryMenu)
	}

	var sb strings.Builder
	sb.WriteString("🕘 *最近對話 / 工具歷史*\n\n")
	for _, entry := range entries {
		ts := entry.Timestamp.Format("01-02 15:04")
		sb.WriteString(fmt.Sprintf("• `%s` [%s/%s] %s\n", ts, entry.Source, entry.Kind, truncateForView(entry.Content, 160)))
		sb.WriteString("\n")
	}
	return c.Send(sb.String(), MemoryMenu, tele.ModeMarkdown)
}

// HandleMemoryClear clears all memory for the current user.
func (h *Hub) HandleMemoryClear(c tele.Context) error {
	userID := c.Sender().ID
	slog.Warn("🧹 清除記憶", "user_id", userID)
	_ = c.Respond()

	if h.Memory == nil {
		return c.Send("⚠️ 記憶系統未初始化", MemoryMenu)
	}
	if err := h.Memory.Clear(userID); err != nil {
		return c.Send("❌ 清除失敗："+err.Error(), MemoryMenu)
	}
	return c.Send("🧹 記憶已清除", MemoryMenu)
}

// HandleBrowserBtn shows the browser submenu.
func (h *Hub) HandleBrowserBtn(c tele.Context) error {
	slog.Info("🔘 選單: Browser", "user_id", c.Sender().ID)
	_ = c.Respond()
	return c.Send("🌐 *Browser 自動化*\n\n支援 `open / wait / extract / screenshot` 的安全腳本。", BrowserMenu, tele.ModeMarkdown)
}

// HandleBrowserRun starts the browser script input flow.
func (h *Hub) HandleBrowserRun(c tele.Context) error {
	userID := c.Sender().ID
	slog.Info("🌐 Browser 執行腳本", "user_id", userID)
	_ = c.Respond()
	h.Sessions.Update(userID, func(s *app.UserSession) { s.Mode = app.ModeAwaitBrowserScript })
	return c.Send(
		"▶️ *Browser 腳本*\n\n請貼上腳本：\n\n```text\nopen https://1.1.1.1\nwait 2s\nextract body\nscreenshot .axle/browser/example.png\n```",
		tele.ModeMarkdown,
	)
}

// HandleBrowserExamples shows example browser scripts.
func (h *Hub) HandleBrowserExamples(c tele.Context) error {
	slog.Info("🌐 Browser 範例", "user_id", c.Sender().ID)
	_ = c.Respond()
	return c.Send(
		"📘 *Browser 腳本範例*\n\n"+
			"```text\n"+
			"open https://1.1.1.1\n"+
			"wait 2s\n"+
			"extract body\n"+
			"screenshot .axle/browser/example-home.png\n"+
			"```\n\n"+
			"• `open`：開啟頁面\n"+
			"• `wait`：等待頁面渲染\n"+
			"• `extract`：擷取 body 或 CSS selector 文字\n"+
			"• `screenshot`：將視窗畫面存進 workspace",
		BrowserMenu,
		tele.ModeMarkdown,
	)
}

// HandleGatewayBtn shows local web gateway information.
func (h *Hub) HandleGatewayBtn(c tele.Context) error {
	slog.Info("🌉 Web Gateway", "user_id", c.Sender().ID)
	_ = c.Respond()

	webURL := "http://127.0.0.1" + h.WebListenAddr
	if strings.HasPrefix(h.WebListenAddr, "127.0.0.1") || strings.HasPrefix(h.WebListenAddr, "localhost") {
		webURL = "http://" + h.WebListenAddr
	}

	return c.Send(
		fmt.Sprintf("🌉 *Web Gateway*\n\n"+
			"網址：`%s/chat`\n"+
			"Token：`%s`\n\n"+
			"可用功能：\n"+
			"• Web Chat 第二通道\n"+
			"• 搜尋歷史 / 最近記憶\n"+
			"• Browser 腳本執行\n"+
			"• 工作流建立 / 查詢 / 取消",
			webURL, h.WebGatewayToken,
		),
		h.mm(c),
		tele.ModeMarkdown,
	)
}

// ── Workspace switch ──────────────────────────────────────────────────────────

// HandleSwitchProjectBtn sets session to await a workspace path.
func (h *Hub) HandleSwitchProjectBtn(c tele.Context) error {
	userID := c.Sender().ID
	slog.Info("🔘 選單: 切換專案", "user_id", userID)
	_ = c.Respond()

	sess := h.Sessions.GetCopy(userID)
	current := h.workspaceFor(userID)
	extra := ""
	if sess.ActiveWorkspace != "" {
		extra = fmt.Sprintf("\n（預設：`%s`）", h.Workspace)
	}

	h.Sessions.Update(userID, func(s *app.UserSession) { s.Mode = app.ModeAwaitProjectPath })
	return c.Send(
		fmt.Sprintf("📂 *切換工作目錄*\n\n當前目錄：`%s`%s\n\n請輸入絕對路徑：\n範例：`/Users/gary/myproject`\n\n_（輸入 `reset` 恢復為預設目錄）_", current, extra),
		tele.ModeMarkdown,
	)
}

// ── Model switch from main menu ───────────────────────────────────────────────

// HandleSwitchModelBtn shows vendor selection from the main menu context.
func (h *Hub) HandleSwitchModelBtn(c tele.Context) error {
	userID := c.Sender().ID
	slog.Info("🔘 選單: 切換模型", "user_id", userID)
	_ = c.Respond()

	sess := h.Sessions.GetCopy(userID)
	current := sess.SelectedModel
	if current == "" {
		current = skill.DefaultModel + "（預設）"
	}

	return c.Send(
		fmt.Sprintf("🔄 *切換全局模型*\n\n當前模型：`%s`\n請選擇 AI 廠商：", current),
		BuildVendorMenu("main"),
		tele.ModeMarkdown,
	)
}

// ── Copilot session controls ──────────────────────────────────────────────────

// HandleCopilotSwitchModel shows vendor selection within Copilot conversation.
func (h *Hub) HandleCopilotSwitchModel(c tele.Context) error {
	userID := c.Sender().ID
	slog.Info("🔘 Copilot: 切換模型", "user_id", userID)
	_ = c.Respond()

	sess := h.Sessions.GetCopy(userID)
	current := sess.SelectedModel
	if current == "" {
		current = skill.DefaultModel
	}

	return c.Send(
		fmt.Sprintf("🔄 *切換 Copilot 模型*\n\n當前模型：`%s`\n請選擇 AI 廠商：", current),
		BuildVendorMenu("copilot"),
		tele.ModeMarkdown,
	)
}

// HandleCopilotExit exits the Copilot conversation mode and returns to main menu.
func (h *Hub) HandleCopilotExit(c tele.Context) error {
	slog.Info("⬅️ 離開 Copilot 對話", "user_id", c.Sender().ID)
	_ = c.Respond()
	h.Sessions.Update(c.Sender().ID, func(s *app.UserSession) {
		s.Mode = app.ModeIdle
	})
	return h.sendMenu(c, "🔧 已返回主選單")
}

// ── Directory listing button ──────────────────────────────────────────────────

// HandleListDirBtn sets session to await directory path input.
func (h *Hub) HandleListDirBtn(c tele.Context) error {
	userID := c.Sender().ID
	slog.Info("🔘 選單: 目錄瀏覽", "user_id", userID)
	_ = c.Respond()
	h.Sessions.Update(userID, func(s *app.UserSession) { s.Mode = app.ModeAwaitListPath })
	return c.Send("📂 *目錄瀏覽*\n\n請輸入相對路徑（留空或輸入 `.` 為根目錄）\n範例：`cmd/axle`", tele.ModeMarkdown)
}

// ── Code search button ────────────────────────────────────────────────────────

// HandleSearchBtn sets session to await search query input.
func (h *Hub) HandleSearchBtn(c tele.Context) error {
	userID := c.Sender().ID
	slog.Info("🔘 選單: 搜尋代碼", "user_id", userID)
	_ = c.Respond()
	h.Sessions.Update(userID, func(s *app.UserSession) { s.Mode = app.ModeAwaitSearchQuery })
	return c.Send("🔎 *搜尋代碼*\n\n請輸入搜尋關鍵字（不區分大小寫）\n範例：`func main`", tele.ModeMarkdown)
}

// ── Git operations ────────────────────────────────────────────────────────────

// HandleGitBtn shows the Git operations submenu.
func (h *Hub) HandleGitBtn(c tele.Context) error {
	slog.Info("🔘 選單: Git 操作", "user_id", c.Sender().ID)
	_ = c.Respond()
	return c.Send("🔀 *Git 操作*\n\n請選擇操作：", GitMenu, tele.ModeMarkdown)
}

// HandleGitStatus runs git status.
func (h *Hub) HandleGitStatus(c tele.Context) error {
	userID := c.Sender().ID
	slog.Info("🔀 Git Status", "user_id", userID)
	_ = c.Respond()

	result, err := skill.GitStatus(context.Background(), h.workspaceFor(userID))
	if err != nil {
		h.emitRPG("git_status", "error", false)
		return c.Send("❌ "+err.Error(), GitMenu)
	}
	h.emitRPG("git_status", h.workspaceFor(userID), true)
	chunks := skill.SplitMessage("📊 *Git Status*\n\n```\n" + result + "\n```")
	for i, chunk := range chunks {
		if i == len(chunks)-1 {
			return c.Send(chunk, GitMenu, tele.ModeMarkdown)
		}
		c.Send(chunk, tele.ModeMarkdown)
	}
	return nil
}

// HandleGitDiff runs git diff (unstaged).
func (h *Hub) HandleGitDiff(c tele.Context) error {
	userID := c.Sender().ID
	slog.Info("🔀 Git Diff", "user_id", userID)
	_ = c.Respond()

	result, err := skill.GitDiff(context.Background(), h.workspaceFor(userID), false)
	if err != nil {
		h.emitRPG("git_diff", "unstaged", false)
		return c.Send("❌ "+err.Error(), GitMenu)
	}
	h.emitRPG("git_diff", "unstaged", true)
	chunks := skill.SplitMessage("📝 *Git Diff (Unstaged)*\n\n```diff\n" + result + "\n```")
	for i, chunk := range chunks {
		if i == len(chunks)-1 {
			return c.Send(chunk, GitMenu, tele.ModeMarkdown)
		}
		c.Send(chunk, tele.ModeMarkdown)
	}
	return nil
}

// HandleGitDiffStaged runs git diff --cached (staged).
func (h *Hub) HandleGitDiffStaged(c tele.Context) error {
	userID := c.Sender().ID
	slog.Info("🔀 Git Diff Staged", "user_id", userID)
	_ = c.Respond()

	result, err := skill.GitDiff(context.Background(), h.workspaceFor(userID), true)
	if err != nil {
		h.emitRPG("git_diff", "staged", false)
		return c.Send("❌ "+err.Error(), GitMenu)
	}
	h.emitRPG("git_diff", "staged", true)
	chunks := skill.SplitMessage("📦 *Git Diff (Staged)*\n\n```diff\n" + result + "\n```")
	for i, chunk := range chunks {
		if i == len(chunks)-1 {
			return c.Send(chunk, GitMenu, tele.ModeMarkdown)
		}
		c.Send(chunk, tele.ModeMarkdown)
	}
	return nil
}

// HandleGitLog runs git log --oneline.
func (h *Hub) HandleGitLog(c tele.Context) error {
	userID := c.Sender().ID
	slog.Info("🔀 Git Log", "user_id", userID)
	_ = c.Respond()

	result, err := skill.GitLog(context.Background(), h.workspaceFor(userID), 15)
	if err != nil {
		h.emitRPG("git_log", "error", false)
		return c.Send("❌ "+err.Error(), GitMenu)
	}
	h.emitRPG("git_log", h.workspaceFor(userID), true)
	chunks := skill.SplitMessage("📜 *Git Log (最近 15 筆)*\n\n```\n" + result + "\n```")
	for i, chunk := range chunks {
		if i == len(chunks)-1 {
			return c.Send(chunk, GitMenu, tele.ModeMarkdown)
		}
		c.Send(chunk, tele.ModeMarkdown)
	}
	return nil
}

// HandleGitCommitPush enters commit message input mode.
func (h *Hub) HandleGitCommitPush(c tele.Context) error {
	userID := c.Sender().ID
	slog.Info("🔀 Git Commit+Push", "user_id", userID)
	_ = c.Respond()
	h.Sessions.Update(userID, func(s *app.UserSession) { s.Mode = app.ModeAwaitGitCommitMsg })
	return c.Send("🚀 *Git Commit + Push*\n\n請輸入 commit 訊息：", tele.ModeMarkdown)
}

// HandleGitCommitConfirm executes git add -A + commit + push.
func (h *Hub) HandleGitCommitConfirm(c tele.Context) error {
	userID := c.Sender().ID
	slog.Warn("🚀 確認 Git Commit+Push", "user_id", userID)
	_ = c.Respond()

	var pendingMsg string
	h.Sessions.Update(userID, func(s *app.UserSession) {
		pendingMsg = s.PendingCmd
		s.Mode = app.ModeIdle
		s.PendingCmd = ""
	})

	if pendingMsg == "" {
		return h.sendMenu(c, "⚠️ 沒有待提交的訊息")
	}

	return h.RunExecTask(c, fmt.Sprintf("git add -A && git commit -m '%s' && git push",
		strings.ReplaceAll(pendingMsg, "'", "'\\''")))
}

// HandleGitCommitCancel cancels the git commit.
func (h *Hub) HandleGitCommitCancel(c tele.Context) error {
	slog.Info("❌ 取消 Git Commit", "user_id", c.Sender().ID)
	_ = c.Respond()
	h.Sessions.Reset(c.Sender().ID)
	return c.Send("❌ Git 操作已取消", GitMenu)
}

// ── Sub-agent buttons ─────────────────────────────────────────────────────────

// HandleSubAgentsBtn shows the sub-agent submenu.
func (h *Hub) HandleSubAgentsBtn(c tele.Context) error {
	slog.Info("🔘 選單: 子代理", "user_id", c.Sender().ID)
	_ = c.Respond()

	count := 0
	if h.SubAgents != nil {
		count = h.SubAgents.RunningCount(c.Sender().ID)
	}

	return c.Send(fmt.Sprintf("👥 *子代理系統*\n\n執行中的代理：%d\n\n請選擇操作：", count),
		SubAgentMenu, tele.ModeMarkdown)
}

// HandleSubAgentCreate starts the sub-agent creation flow.
func (h *Hub) HandleSubAgentCreate(c tele.Context) error {
	userID := c.Sender().ID
	slog.Info("👥 建立子代理", "user_id", userID)
	_ = c.Respond()
	h.Sessions.Update(userID, func(s *app.UserSession) { s.Mode = app.ModeAwaitSubAgentName })
	return c.Send("👥 *建立子代理*\n\n請輸入代理名稱：\n範例：`前端審查員`", tele.ModeMarkdown)
}

// HandleSubAgentList lists all sub-agents for the user.
func (h *Hub) HandleSubAgentList(c tele.Context) error {
	userID := c.Sender().ID
	slog.Info("👥 查看子代理清單", "user_id", userID)
	_ = c.Respond()

	if h.SubAgents == nil {
		return c.Send("⚠️ 子代理系統未初始化", SubAgentMenu)
	}

	agents := h.SubAgents.List(userID)
	if len(agents) == 0 {
		return c.Send("📋 目前沒有任何子代理", SubAgentMenu)
	}

	var sb strings.Builder
	sb.WriteString("📋 *子代理清單*\n\n")
	for _, a := range agents {
		elapsed := ""
		if a.Status == app.SubAgentRunning {
			elapsed = fmt.Sprintf(" (%.0fs)", time.Since(a.CreatedAt).Seconds())
		}
		sb.WriteString(fmt.Sprintf("• `%s` — %s %s%s\n  任務：%s\n\n",
			a.ID, a.Name, a.Status.String(), elapsed, a.Task))
	}

	// Build dynamic cancel buttons for running agents
	m := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, a := range agents {
		if a.Status == app.SubAgentRunning {
			rows = append(rows, m.Row(
				m.Data(fmt.Sprintf("🛑 取消 %s", a.Name), "subagent_cancel", a.ID),
			))
		}
	}
	rows = append(rows, m.Row(m.Data("⬅️ 返回主選單", "back_main")))
	m.Inline(rows...)

	return c.Send(sb.String(), m, tele.ModeMarkdown)
}

// HandleSubAgentCancel cancels a specific sub-agent.
func (h *Hub) HandleSubAgentCancel(c tele.Context) error {
	_ = c.Respond()
	args := c.Args()
	if len(args) < 1 {
		return c.Send("⚠️ 未指定代理 ID", SubAgentMenu)
	}

	agentID := args[0]
	slog.Info("🛑 取消子代理", "id", agentID, "user_id", c.Sender().ID)

	if h.SubAgents == nil || !h.SubAgents.Cancel(agentID) {
		return c.Send("ℹ️ 該代理不在執行中或不存在", SubAgentMenu)
	}
	return c.Send(fmt.Sprintf("🛑 子代理 `%s` 已取消", agentID), h.mm(c), tele.ModeMarkdown)
}

// ── Workflow buttons ───────────────────────────────────────────────────────────

// HandleWorkflowsBtn shows the workflow submenu.
func (h *Hub) HandleWorkflowsBtn(c tele.Context) error {
	slog.Info("🔘 選單: 工作流", "user_id", c.Sender().ID)
	_ = c.Respond()

	count := 0
	if h.Workflows != nil {
		count = h.Workflows.RunningCount(c.Sender().ID)
	}

	return c.Send(
		fmt.Sprintf("🧭 *背景工作流*\n\n執行中的工作流：%d\n\n請選擇操作：", count),
		WorkflowMenu,
		tele.ModeMarkdown,
	)
}

// HandleWorkflowCreate starts the workflow request flow.
func (h *Hub) HandleWorkflowCreate(c tele.Context) error {
	userID := c.Sender().ID
	slog.Info("🧭 建立工作流", "user_id", userID)
	_ = c.Respond()
	h.Sessions.Update(userID, func(s *app.UserSession) { s.Mode = app.ModeAwaitWorkflowRequest })
	return c.Send(
		"🧭 *建立背景工作流*\n\n請描述想交給 Axle 背景執行的工作：\n\n_例如：分析這個 repo 的安全風險，若需要可查官方文件，再整理成摘要。_",
		tele.ModeMarkdown,
	)
}

// HandleWorkflowList lists workflows for the current user.
func (h *Hub) HandleWorkflowList(c tele.Context) error {
	userID := c.Sender().ID
	slog.Info("🧭 查看工作流", "user_id", userID)
	_ = c.Respond()

	if h.Workflows == nil {
		return c.Send("⚠️ 工作流系統未初始化", WorkflowMenu)
	}

	workflows := h.Workflows.List(userID)
	if len(workflows) == 0 {
		return c.Send("📋 目前沒有任何工作流", WorkflowMenu)
	}

	var sb strings.Builder
	sb.WriteString("📋 *工作流清單*\n\n")
	m := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, wf := range workflows {
		sb.WriteString(fmt.Sprintf("• `%s` — %s\n", wf.ID, wf.Status.Label()))
		sb.WriteString(fmt.Sprintf("  需求：%s\n", truncateForView(wf.Request, 120)))
		if len(wf.Steps) > 0 {
			sb.WriteString(fmt.Sprintf("  步驟：%d\n", len(wf.Steps)))
		}
		if wf.ResultSummary != "" {
			sb.WriteString(fmt.Sprintf("  摘要：%s\n", truncateForView(wf.ResultSummary, 140)))
		}
		if wf.Error != "" {
			sb.WriteString(fmt.Sprintf("  錯誤：%s\n", truncateForView(wf.Error, 140)))
		}
		sb.WriteString("\n")

		if wf.Status == app.WorkflowPlanning || wf.Status == app.WorkflowRunning {
			rows = append(rows, m.Row(
				m.Data(fmt.Sprintf("🛑 取消 %s", wf.ID), "workflow_cancel", wf.ID),
			))
		}
	}
	rows = append(rows, m.Row(m.Data("⬅️ 返回主選單", "back_main")))
	m.Inline(rows...)

	return c.Send(sb.String(), m, tele.ModeMarkdown)
}

// HandleWorkflowCancel cancels a running workflow.
func (h *Hub) HandleWorkflowCancel(c tele.Context) error {
	_ = c.Respond()
	args := c.Args()
	if len(args) < 1 || h.Workflows == nil {
		return c.Send("⚠️ 工作流參數錯誤", WorkflowMenu)
	}

	workflowID := args[0]
	wf, ok := h.Workflows.Get(workflowID)
	if !ok || wf.UserID != c.Sender().ID {
		return c.Send("ℹ️ 找不到此工作流", WorkflowMenu)
	}
	slog.Info("🛑 取消工作流", "id", workflowID, "user_id", c.Sender().ID)
	if !h.Workflows.Cancel(workflowID) {
		return c.Send("ℹ️ 該工作流不在執行中或不存在", WorkflowMenu)
	}
	return c.Send(fmt.Sprintf("🛑 工作流 `%s` 已取消", workflowID), h.mm(c), tele.ModeMarkdown)
}

// ── Plugin buttons ────────────────────────────────────────────────────────────

// HandlePluginsBtn shows the plugin list.
func (h *Hub) HandlePluginsBtn(c tele.Context) error {
	slog.Info("🔘 選單: 擴充技能", "user_id", c.Sender().ID)
	_ = c.Respond()

	if h.Plugins == nil {
		return h.sendMenu(c, "⚠️ 插件系統未初始化")
	}

	plugins := h.Plugins.List()
	if len(plugins) == 0 {
		return c.Send("🧩 *擴充技能*\n\n目前沒有可用的插件\n\n將 YAML 插件放入 `~/.axle/plugins/` 目錄即可載入",
			h.mm(c), tele.ModeMarkdown)
	}

	m := &tele.ReplyMarkup{}
	var rows []tele.Row
	for i, p := range plugins {
		label := fmt.Sprintf("🧩 %s", p.Name)
		if p.Description != "" {
			label = fmt.Sprintf("🧩 %s — %s", p.Name, p.Description)
		}
		// Truncate label for Telegram button (max ~64 bytes for data)
		if len(label) > 40 {
			label = label[:40] + "..."
		}
		rows = append(rows, m.Row(
			m.Data(label, "plugin_exec", fmt.Sprintf("%d", i)),
		))
	}
	rows = append(rows, m.Row(m.Data("🔄 重新載入", "plugin_reload")))
	rows = append(rows, m.Row(m.Data("⬅️ 返回主選單", "back_main")))
	m.Inline(rows...)

	return c.Send(fmt.Sprintf("🧩 *擴充技能*（共 %d 個）\n\n請選擇要執行的插件：", len(plugins)),
		m, tele.ModeMarkdown)
}

// HandlePluginExec executes a specific plugin.
func (h *Hub) HandlePluginExec(c tele.Context) error {
	_ = c.Respond()
	args := c.Args()
	if len(args) < 1 || h.Plugins == nil {
		return h.sendMenu(c, "⚠️ 插件參數錯誤")
	}

	idx := 0
	fmt.Sscanf(args[0], "%d", &idx)
	plugin, ok := h.Plugins.Get(idx)
	if !ok {
		return h.sendMenu(c, "⚠️ 插件不存在")
	}

	slog.Info("🧩 執行插件", "name", plugin.Name, "user_id", c.Sender().ID)

	// Safety check for plugin commands
	level, reasons := skill.CheckCommandSafety(plugin.Command)
	if level == skill.DangerBlocked {
		return h.sendMenu(c, fmt.Sprintf("⛔ 插件指令被封鎖：%s", strings.Join(reasons, ", ")))
	}

	if plugin.Confirm || level == skill.DangerWarning {
		h.Sessions.Update(c.Sender().ID, func(s *app.UserSession) {
			s.Mode = app.ModeAwaitExecConfirm
			s.PendingCmd = plugin.Command
		})
		warning := ""
		if len(reasons) > 0 {
			warning = fmt.Sprintf("\n⚠️ 風險：%s", strings.Join(reasons, ", "))
		}
		return c.Send(
			fmt.Sprintf("🧩 *插件：%s*\n\n```bash\n%s\n```%s\n\n確認執行？", plugin.Name, plugin.Command, warning),
			ExecMenu,
			tele.ModeMarkdown,
		)
	}

	// Direct execution
	ws := h.Workspace
	if plugin.UseWorkspace {
		ws = h.workspaceFor(c.Sender().ID)
	}

	return h.RunExecTask(c, fmt.Sprintf("cd %s && %s", ws, plugin.Command))
}

// HandlePluginReload reloads plugins from disk.
func (h *Hub) HandlePluginReload(c tele.Context) error {
	slog.Info("🔄 重新載入插件", "user_id", c.Sender().ID)
	_ = c.Respond()

	if h.Plugins == nil {
		return h.sendMenu(c, "⚠️ 插件系統未初始化")
	}

	if err := h.Plugins.Reload(); err != nil {
		return h.sendMenu(c, "❌ 載入失敗："+err.Error())
	}
	return h.sendMenu(c, fmt.Sprintf("✅ 插件已重新載入（共 %d 個）", h.Plugins.Count()))
}

// ── Scheduler buttons ─────────────────────────────────────────────────────────

// HandleSchedulerBtn shows the scheduler submenu.
func (h *Hub) HandleSchedulerBtn(c tele.Context) error {
	slog.Info("🔘 選單: 排程任務", "user_id", c.Sender().ID)
	_ = c.Respond()
	return c.Send("⏰ *排程任務*\n\n請選擇操作：", SchedulerMenu, tele.ModeMarkdown)
}

// HandleSchedCreate starts the schedule creation flow.
func (h *Hub) HandleSchedCreate(c tele.Context) error {
	slog.Info("⏰ 建立排程", "user_id", c.Sender().ID)
	_ = c.Respond()
	h.Sessions.Update(c.Sender().ID, func(s *app.UserSession) { s.Mode = app.ModeAwaitSchedName })
	return c.Send("⏰ *建立排程*\n\n請輸入排程名稱：\n範例：`健康檢查`", tele.ModeMarkdown)
}

// HandleSchedList shows all schedules.
func (h *Hub) HandleSchedList(c tele.Context) error {
	slog.Info("⏰ 查看排程", "user_id", c.Sender().ID)
	_ = c.Respond()

	if h.Scheduler == nil {
		return c.Send("⚠️ 排程系統未初始化", SchedulerMenu)
	}

	schedules := h.Scheduler.List()
	if len(schedules) == 0 {
		return c.Send("📋 目前沒有排程任務", SchedulerMenu)
	}

	var sb strings.Builder
	sb.WriteString("📋 *排程清單*\n\n")
	for _, s := range schedules {
		status := "✅ 啟用"
		if !s.Enabled {
			status = "⏸ 停用"
		}
		sb.WriteString(fmt.Sprintf("• `%s` — %s %s\n  指令：`%s`\n  間隔：每 %d 分鐘\n\n",
			s.ID, s.Name, status, s.Command, s.Interval))
	}

	// Build dynamic buttons
	m := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, s := range schedules {
		toggleLabel := "⏸ 停用"
		if !s.Enabled {
			toggleLabel = "▶️ 啟用"
		}
		rows = append(rows, m.Row(
			m.Data(fmt.Sprintf("%s %s", toggleLabel, s.Name), "sched_toggle", s.ID),
			m.Data(fmt.Sprintf("🗑 刪除 %s", s.Name), "sched_delete", s.ID),
		))
	}
	rows = append(rows, m.Row(m.Data("⬅️ 返回主選單", "back_main")))
	m.Inline(rows...)

	return c.Send(sb.String(), m, tele.ModeMarkdown)
}

// HandleSchedDelete deletes a schedule.
func (h *Hub) HandleSchedDelete(c tele.Context) error {
	_ = c.Respond()
	args := c.Args()
	if len(args) < 1 || h.Scheduler == nil {
		return h.sendMenu(c, "⚠️ 排程參數錯誤")
	}

	schedID := args[0]
	slog.Info("🗑 刪除排程", "id", schedID, "user_id", c.Sender().ID)

	if !h.Scheduler.Delete(schedID) {
		return c.Send("ℹ️ 排程不存在", SchedulerMenu)
	}
	return h.sendMenu(c, fmt.Sprintf("✅ 排程 `%s` 已刪除", schedID))
}

// HandleSchedToggle toggles a schedule on/off.
func (h *Hub) HandleSchedToggle(c tele.Context) error {
	_ = c.Respond()
	args := c.Args()
	if len(args) < 1 || h.Scheduler == nil {
		return h.sendMenu(c, "⚠️ 排程參數錯誤")
	}

	schedID := args[0]
	slog.Info("⏰ 切換排程", "id", schedID, "user_id", c.Sender().ID)

	enabled, ok := h.Scheduler.Toggle(schedID)
	if !ok {
		return c.Send("ℹ️ 排程不存在", SchedulerMenu)
	}

	status := "✅ 已啟用"
	if !enabled {
		status = "⏸ 已停用"
	}
	return h.sendMenu(c, fmt.Sprintf("⏰ 排程 `%s` %s", schedID, status))
}

// ── Extras toggle menu ────────────────────────────────────────────────────────

// HandleExtrasBtn shows the extras toggle menu.
func (h *Hub) HandleExtrasBtn(c tele.Context) error {
	slog.Info("🔘 選單: 更多功能", "user_id", c.Sender().ID)
	_ = c.Respond()
	return h.showExtrasMenu(c)
}

// HandleToggleExtra toggles a specific extra feature on/off in the main menu.
func (h *Hub) HandleToggleExtra(c tele.Context) error {
	_ = c.Respond()
	args := c.Args()
	if len(args) < 1 {
		return h.sendMenu(c, "⚠️ 參數錯誤")
	}

	featureID := args[0]
	userID := c.Sender().ID
	slog.Info("🔘 切換功能", "feature", featureID, "user_id", userID)

	h.Sessions.Update(userID, func(s *app.UserSession) {
		if s.EnabledExtras == nil {
			s.EnabledExtras = make(map[string]bool)
		}
		s.EnabledExtras[featureID] = !s.EnabledExtras[featureID]
		if !s.EnabledExtras[featureID] {
			delete(s.EnabledExtras, featureID)
		}
	})

	return h.showExtrasMenu(c)
}

// showExtrasMenu builds and sends the extras toggle menu.
func (h *Hub) showExtrasMenu(c tele.Context) error {
	sess := h.Sessions.GetCopy(c.Sender().ID)
	extras := sess.EnabledExtras

	m := &tele.ReplyMarkup{}
	var rows []tele.Row

	for _, ef := range ExtraFeatures {
		icon := "◻️"
		if extras[ef.ID] {
			icon = "✅"
		}
		rows = append(rows, m.Row(
			m.Data(fmt.Sprintf("%s %s", icon, ef.Label), "toggle_extra", ef.ID),
		))
	}
	rows = append(rows, m.Row(m.Data("⬅️ 返回主選單", "back_main")))
	m.Inline(rows...)

	enabledCount := len(extras)
	return c.Send(
		fmt.Sprintf("⚙️ *更多功能*\n\n點擊可將功能加入/移除主選單\n已啟用：%d 項\n\n💡 上傳 PDF 或圖片可直接處理", enabledCount),
		m,
		tele.ModeMarkdown,
	)
}

// ── GitHub handlers ───────────────────────────────────────────────────────────

// HandleGitHubBtn shows GitHub sub-menu.
func (h *Hub) HandleGitHubBtn(c tele.Context) error {
	slog.Info("🔘 選單: GitHub", "user_id", c.Sender().ID)
	_ = c.Respond()
	if err := skill.GHCheckInstalled(); err != nil {
		return h.sendMenu(c, "❌ "+err.Error())
	}
	return c.Send("🐙 *GitHub 操作*\n\n請選擇操作：", GitHubMenu, tele.ModeMarkdown)
}

// HandleGHPRList lists open pull requests.
func (h *Hub) HandleGHPRList(c tele.Context) error {
	slog.Info("🔘 GitHub: PR 列表", "user_id", c.Sender().ID)
	_ = c.Respond()
	c.Send("📋 載入 PR 列表中...")

	ctx, cancel := context.WithTimeout(context.Background(), skill.GHTimeout())
	defer cancel()

	out, err := skill.GHPRList(ctx, h.workspaceFor(c.Sender().ID))
	if err != nil {
		h.emitRPG("github", "PR list", false)
		return c.Send("❌ "+err.Error(), GitHubMenu)
	}
	h.emitRPG("github", "PR list", true)
	return c.Send(fmt.Sprintf("📋 *Pull Requests*\n\n```\n%s\n```", out), GitHubMenu, tele.ModeMarkdown)
}

// HandleGHIssueList lists open issues.
func (h *Hub) HandleGHIssueList(c tele.Context) error {
	slog.Info("🔘 GitHub: Issue 列表", "user_id", c.Sender().ID)
	_ = c.Respond()
	c.Send("📋 載入 Issue 列表中...")

	ctx, cancel := context.WithTimeout(context.Background(), skill.GHTimeout())
	defer cancel()

	out, err := skill.GHIssueList(ctx, h.workspaceFor(c.Sender().ID))
	if err != nil {
		h.emitRPG("github", "Issue list", false)
		return c.Send("❌ "+err.Error(), GitHubMenu)
	}
	h.emitRPG("github", "Issue list", true)
	return c.Send(fmt.Sprintf("📋 *Issues*\n\n```\n%s\n```", out), GitHubMenu, tele.ModeMarkdown)
}

// HandleGHCIStatus shows CI/CD status.
func (h *Hub) HandleGHCIStatus(c tele.Context) error {
	slog.Info("🔘 GitHub: CI 狀態", "user_id", c.Sender().ID)
	_ = c.Respond()
	c.Send("🔄 載入 CI 狀態中...")

	ctx, cancel := context.WithTimeout(context.Background(), skill.GHTimeout())
	defer cancel()

	out, err := skill.GHCIStatus(ctx, h.workspaceFor(c.Sender().ID))
	if err != nil {
		h.emitRPG("github", "CI status", false)
		return c.Send("❌ "+err.Error(), GitHubMenu)
	}
	h.emitRPG("github", "CI status", true)
	return c.Send(fmt.Sprintf("🔄 *CI/CD 狀態*\n\n```\n%s\n```", out), GitHubMenu, tele.ModeMarkdown)
}

// HandleGHRepoView shows repository info.
func (h *Hub) HandleGHRepoView(c tele.Context) error {
	slog.Info("🔘 GitHub: Repo 資訊", "user_id", c.Sender().ID)
	_ = c.Respond()

	ctx, cancel := context.WithTimeout(context.Background(), skill.GHTimeout())
	defer cancel()

	out, err := skill.GHRepoView(ctx, h.workspaceFor(c.Sender().ID))
	if err != nil {
		h.emitRPG("github", "Repo view", false)
		return c.Send("❌ "+err.Error(), GitHubMenu)
	}
	h.emitRPG("github", "Repo view", true)
	chunks := skill.SplitMessage(fmt.Sprintf("📦 *Repository*\n\n```\n%s\n```", out))
	for i, chunk := range chunks {
		if i == len(chunks)-1 {
			return c.Send(chunk, GitHubMenu, tele.ModeMarkdown)
		}
		c.Send(chunk, tele.ModeMarkdown)
	}
	return nil
}

// HandleGHPRCreate starts PR creation flow.
func (h *Hub) HandleGHPRCreate(c tele.Context) error {
	slog.Info("🔘 GitHub: 建立 PR", "user_id", c.Sender().ID)
	_ = c.Respond()
	h.Sessions.Update(c.Sender().ID, func(s *app.UserSession) { s.Mode = app.ModeAwaitGHPRTitle })
	return c.Send("➕ *建立 Pull Request*\n\n請輸入 PR 標題：", tele.ModeMarkdown)
}

// ── Email handlers ────────────────────────────────────────────────────────────

// HandleEmailBtn shows email sub-menu.
func (h *Hub) HandleEmailBtn(c tele.Context) error {
	slog.Info("🔘 選單: Email", "user_id", c.Sender().ID)
	_ = c.Respond()
	if h.EmailConfig == nil || !h.EmailConfig.IsConfigured() {
		return h.sendMenu(c, "❌ Email 尚未設定\n\n請在環境變數或 `~/.axle/credentials.json` 中設定：\n• `EMAIL_ADDRESS`\n• `EMAIL_PASSWORD`\n• `SMTP_HOST` (預設 smtp.gmail.com)\n\n_Gmail 需使用 App Password_")
	}
	return c.Send("📧 *Email*\n\n請選擇操作：", EmailMenu, tele.ModeMarkdown)
}

// HandleEmailSend starts email compose flow.
func (h *Hub) HandleEmailSend(c tele.Context) error {
	slog.Info("🔘 Email: 發送", "user_id", c.Sender().ID)
	_ = c.Respond()
	h.Sessions.Update(c.Sender().ID, func(s *app.UserSession) { s.Mode = app.ModeAwaitEmailTo })
	return c.Send("📤 *發送 Email*\n\n請輸入收件人地址：", tele.ModeMarkdown)
}

// HandleEmailRead reads recent emails from inbox.
func (h *Hub) HandleEmailRead(c tele.Context) error {
	slog.Info("🔘 Email: 讀取", "user_id", c.Sender().ID)
	_ = c.Respond()
	c.Send("📥 讀取信箱中...")

	summaries, err := skill.ReadEmails(*h.EmailConfig, 5)
	if err != nil {
		h.emitRPG("email_read", "error", false)
		return c.Send("❌ 讀取失敗："+err.Error(), EmailMenu)
	}
	h.emitRPG("email_read", fmt.Sprintf("%d 封", len(summaries)), true)
	if len(summaries) == 0 {
		return c.Send("📥 信箱為空", EmailMenu)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📥 *最近 %d 封郵件*\n\n", len(summaries)))
	for i, s := range summaries {
		sb.WriteString(fmt.Sprintf("%d. *%s*\n   👤 %s\n   📅 %s\n\n", i+1, s.Subject, s.From, s.Date))
	}
	return c.Send(sb.String(), EmailMenu, tele.ModeMarkdown)
}

// ── PDF/Document handler ──────────────────────────────────────────────────────

// HandleDocument processes uploaded documents (PDF, etc.)
func (h *Hub) HandleDocument(c tele.Context) error {
	doc := c.Message().Document
	if doc == nil {
		return nil
	}
	userID := c.Sender().ID
	slog.Info("📄 收到文件", "filename", doc.FileName, "mime", doc.MIME, "size", doc.FileSize, "user_id", userID)

	// Only handle PDF
	if doc.MIME != "application/pdf" {
		return h.sendMenu(c, fmt.Sprintf("📄 收到檔案：`%s`\n\n目前僅支援 PDF 文件處理", doc.FileName))
	}

	c.Send("📄 正在處理 PDF...")

	// Download file
	reader, err := h.Bot.File(&doc.File)
	if err != nil {
		return h.sendMenu(c, "❌ 下載檔案失敗："+err.Error())
	}
	defer reader.Close()

	// Save to temp file
	tmpFile, err := os.CreateTemp("", "axle-pdf-*.pdf")
	if err != nil {
		return h.sendMenu(c, "❌ 建立暫存檔失敗："+err.Error())
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := io.Copy(tmpFile, reader); err != nil {
		tmpFile.Close()
		return h.sendMenu(c, "❌ 儲存檔案失敗："+err.Error())
	}
	tmpFile.Close()

	// Extract text
	text, err := skill.ExtractPDFText(tmpPath)
	if err != nil {
		h.emitRPG("pdf", doc.FileName, false)
		return h.sendMenu(c, "❌ "+err.Error())
	}
	h.emitRPG("pdf", doc.FileName, true)

	// Store PDF text for potential summarization
	h.Sessions.Update(userID, func(s *app.UserSession) {
		s.PendingCmd = text
	})

	chunks := skill.SplitMessage(text)
	for i, chunk := range chunks {
		if i == len(chunks)-1 {
			return c.Send(chunk, PDFMenu)
		}
		c.Send(chunk)
		time.Sleep(chunkSendDelay)
	}
	return nil
}

// HandlePDFSummarize sends extracted PDF text to Copilot for summarization.
func (h *Hub) HandlePDFSummarize(c tele.Context) error {
	userID := c.Sender().ID
	slog.Info("🔘 PDF: AI 摘要", "user_id", userID)
	_ = c.Respond()

	sess := h.Sessions.GetCopy(userID)
	if sess.PendingCmd == "" {
		return h.sendMenu(c, "⚠️ 無 PDF 文字可供摘要，請先上傳 PDF")
	}

	model := sess.SelectedModel
	if model == "" {
		model = skill.DefaultModel
	}

	// Truncate for context window
	pdfText := sess.PendingCmd
	if len(pdfText) > 6000 {
		pdfText = pdfText[:6000]
	}

	prompt := fmt.Sprintf("請用繁體中文摘要以下 PDF 文件內容，列出重點：\n\n%s", pdfText)
	h.Sessions.Update(userID, func(s *app.UserSession) { s.PendingCmd = "" })

	return h.RunCopilotTask(c, prompt, model)
}

// ── Photo/Image handler ───────────────────────────────────────────────────────

// HandlePhoto processes uploaded photos.
func (h *Hub) HandlePhoto(c tele.Context) error {
	photo := c.Message().Photo
	if photo == nil {
		return nil
	}
	userID := c.Sender().ID
	slog.Info("🖼 收到圖片", "file_id", photo.FileID, "user_id", userID)

	c.Send("🖼 正在分析圖片...")

	// Download file
	reader, err := h.Bot.File(&photo.File)
	if err != nil {
		return h.sendMenu(c, "❌ 下載圖片失敗："+err.Error())
	}
	defer reader.Close()

	// Save to temp file
	tmpFile, err := os.CreateTemp("", "axle-img-*")
	if err != nil {
		return h.sendMenu(c, "❌ 建立暫存檔失敗："+err.Error())
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := io.Copy(tmpFile, reader); err != nil {
		tmpFile.Close()
		return h.sendMenu(c, "❌ 儲存圖片失敗："+err.Error())
	}
	tmpFile.Close()

	// Analyze image metadata
	info, err := skill.AnalyzeImage(tmpPath)
	if err != nil {
		h.emitRPG("image", "analysis", false)
		return h.sendMenu(c, "⚠️ 圖片解析失敗："+err.Error()+"\n\n圖片已儲存但無法分析格式")
	}
	h.emitRPG("image", info.Format, true)

	// Save to workspace if user wants
	ws := h.workspaceFor(userID)
	savedName := fmt.Sprintf("image_%d.%s", time.Now().Unix(), info.Format)
	savedPath := filepath.Join(ws, savedName)
	if data, err := os.ReadFile(tmpPath); err == nil {
		if err := os.WriteFile(savedPath, data, 0644); err == nil {
			slog.Info("🖼 圖片已儲存", "path", savedPath)
		}
	}

	msg := info.String() + fmt.Sprintf("\n\n💾 已儲存至：`%s`", savedName)
	return c.Send(msg, h.mm(c), tele.ModeMarkdown)
}

// ── Calendar handlers ─────────────────────────────────────────────────────────

func (h *Hub) HandleCalendarBtn(c tele.Context) error {
	_ = c.Respond()
	return c.Send("📅 *行事曆*\n\n請選擇查看範圍：", CalendarMenu, tele.ModeMarkdown)
}

func (h *Hub) HandleCalToday(c tele.Context) error {
	_ = c.Respond()
	return h.execCalendar(c, "today")
}

func (h *Hub) HandleCalTomorrow(c tele.Context) error {
	_ = c.Respond()
	return h.execCalendar(c, "tomorrow")
}

func (h *Hub) HandleCalWeek(c tele.Context) error {
	_ = c.Respond()
	return h.execCalendar(c, "week")
}

func (h *Hub) execCalendar(c tele.Context, period string) error {
	userID := c.Sender().ID
	slog.Info("📅 行事曆查詢", "period", period, "user_id", userID)
	c.Send("📅 查詢中...")

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	var result string
	var err error
	switch period {
	case "today":
		result, err = skill.CalendarToday(ctx)
	case "tomorrow":
		result, err = skill.CalendarTomorrow(ctx)
	case "week":
		result, err = skill.CalendarWeek(ctx)
	}

	if err != nil {
		h.emitRPG("calendar", period, false)
		return h.sendMenu(c, "❌ "+err.Error())
	}
	h.emitRPG("calendar", period, true)
	return c.Send(result, h.mm(c), tele.ModeMarkdown)
}

// ── Briefing handler ──────────────────────────────────────────────────────────

func (h *Hub) HandleBriefingBtn(c tele.Context) error {
	_ = c.Respond()
	userID := c.Sender().ID
	slog.Info("📢 每日簡報", "user_id", userID)
	c.Send("📢 產生簡報中...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result := skill.GenerateBriefing(ctx, h.workspaceFor(userID))
	h.emitRPG("briefing", "daily", true)
	chunks := skill.SplitMessage(result)
	for i, chunk := range chunks {
		if i == len(chunks)-1 {
			return c.Send(chunk, h.mm(c), tele.ModeMarkdown)
		}
		c.Send(chunk, tele.ModeMarkdown)
	}
	return nil
}

// ── Self-Upgrade handlers ─────────────────────────────────────────────────────

// HandleSelfUpgradeBtn starts the self-upgrade flow.
func (h *Hub) HandleSelfUpgradeBtn(c tele.Context) error {
	slog.Info("🔧 選單: 自我升級", "user_id", c.Sender().ID)
	_ = c.Respond()
	h.Sessions.Update(c.Sender().ID, func(s *app.UserSession) {
		s.Mode = app.ModeAwaitUpgradeRequest
	})
	return c.Send(
		"🔧 *自我升級模式*\n\n"+
			"請描述你想要新增或修改的功能：\n\n"+
			"_例如：「加一個天氣查詢功能」_",
		tele.ModeMarkdown,
	)
}

// HandleUpgradeConfirm executes the upgrade plan.
func (h *Hub) HandleUpgradeConfirm(c tele.Context) error {
	userID := c.Sender().ID
	slog.Warn("🔧 確認自我升級", "user_id", userID)
	_ = c.Respond()

	var request, plan string
	h.Sessions.Update(userID, func(s *app.UserSession) {
		request = s.PendingUpgradeReq
		plan = s.PendingUpgradePlan
		s.Mode = app.ModeIdle
	})

	if request == "" || plan == "" {
		return h.sendMenu(c, "⚠️ 升級請求遺失，請重新操作")
	}

	sess := h.Sessions.GetCopy(userID)
	model := sess.SelectedModel
	if model == "" {
		model = skill.DefaultModel
	}

	srcDir := h.SourceDir
	chat := c.Chat()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("💥 自我升級 panic", "recover", r)
				h.Bot.Send(chat, "💥 升級過程異常中止", h.mmFor(userID))
			}
		}()

		// Step 1: Backup binary
		h.Bot.Send(chat, "📦 *Step 1/6* — 備份 binary...", tele.ModeMarkdown)
		if err := skill.UpgradeBackupBinary(srcDir); err != nil {
			h.Bot.Send(chat, "❌ 備份失敗："+err.Error(), h.mmFor(userID))
			return
		}

		// Step 2: Apply changes via Copilot
		h.Bot.Send(chat, "⚙️ *Step 2/6* — Copilot 正在修改程式碼...", tele.ModeMarkdown)
		summary, err := skill.UpgradeApply(context.Background(), srcDir, model, request, plan)
		if err != nil {
			h.Bot.Send(chat, "❌ 開發失敗："+err.Error(), h.mmFor(userID))
			return
		}
		h.emitRPG("self_upgrade", "apply", true)

		// Step 3: Bump version
		h.Bot.Send(chat, "🏷 *Step 3/6* — 更新版本號...", tele.ModeMarkdown)
		newVer, err := skill.BumpVersion(srcDir)
		if err != nil {
			h.Bot.Send(chat, "⚠️ 版本號更新失敗："+err.Error(), h.mmFor(userID))
			newVer = app.Version // fallback
		}

		// Step 4: Build + Vet
		h.Bot.Send(chat, "🔨 *Step 4/6* — 編譯 + 檢查...", tele.ModeMarkdown)
		if err := skill.UpgradeBuild(context.Background(), srcDir); err != nil {
			h.Bot.Send(chat, fmt.Sprintf("❌ 編譯失敗，自動回滾...\n\n```\n%s\n```", err.Error()), tele.ModeMarkdown)
			skill.UpgradeRollbackBinary(srcDir)
			h.emitRPG("self_upgrade", "build failed", false)
			h.Bot.Send(chat, "↩️ 已回滾至舊版本", h.mmFor(userID))
			return
		}

		// Step 5: Test
		h.Bot.Send(chat, "🧪 *Step 5/6* — 執行測試...", tele.ModeMarkdown)
		testOut, err := skill.UpgradeTest(context.Background(), srcDir)
		if err != nil {
			h.Bot.Send(chat, fmt.Sprintf("❌ 測試失敗，自動回滾...\n\n```\n%s\n```", testOut), tele.ModeMarkdown)
			skill.UpgradeRollbackBinary(srcDir)
			h.emitRPG("self_upgrade", "test failed", false)
			h.Bot.Send(chat, "↩️ 已回滾至舊版本", h.mmFor(userID))
			return
		}

		// Step 6: Git commit
		h.Bot.Send(chat, "📝 *Step 6/6* — Git commit...", tele.ModeMarkdown)
		commitSummary := request
		if len(commitSummary) > 50 {
			commitSummary = commitSummary[:50]
		}
		if err := skill.UpgradeCommit(context.Background(), srcDir, newVer, commitSummary); err != nil {
			slog.Warn("升級 commit 失敗", "error", err)
			h.Bot.Send(chat, "⚠️ Git commit 失敗（升級仍生效）："+err.Error())
		}

		// Report summary
		msg := fmt.Sprintf("✅ *升級完成！*\n\n"+
			"📋 版本：`v%s`\n"+
			"📝 變更摘要：\n%s\n\n"+
			"🔄 即將重啟 Axle...",
			newVer, summary)
		chunks := skill.SplitMessage(msg)
		for _, chunk := range chunks {
			h.Bot.Send(chat, chunk, tele.ModeMarkdown)
			time.Sleep(300 * time.Millisecond)
		}

		h.emitRPG("self_upgrade", "v"+newVer, true)

		// Signal restart
		time.Sleep(2 * time.Second)
		if h.RestartCh != nil {
			close(h.RestartCh)
		}
	}()

	return c.Send("🚀 升級流程啟動...\n\n_請勿中斷，過程約需 2-5 分鐘_", tele.ModeMarkdown)
}

// HandleUpgradeCancel cancels the self-upgrade.
func (h *Hub) HandleUpgradeCancel(c tele.Context) error {
	slog.Info("❌ 取消自我升級", "user_id", c.Sender().ID)
	_ = c.Respond()
	h.Sessions.Reset(c.Sender().ID)
	return h.sendMenu(c, "❌ 自我升級已取消")
}

func truncateForView(s string, max int) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\n", " "))
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
