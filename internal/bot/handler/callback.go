package handler

import (
	"fmt"
	"log/slog"

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

	return c.Send(fmt.Sprintf(
		"📊 *系統狀態*\n\n"+
			"• 任務狀態：%s\n"+
			"• 選定模型：`%s`\n"+
			"• Workspace：`%s`",
		taskStatus, model, wsLabel,
	), MainMenu, tele.ModeMarkdown)
}

// HandleCancelTask cancels the currently running task.
func (h *Hub) HandleCancelTask(c tele.Context) error {
	slog.Info("🔘 選單: 取消任務", "user_id", c.Sender().ID)
	_ = c.Respond()

	if cancelled := h.Tasks.Cancel(); !cancelled {
		return c.Send("ℹ️ 目前沒有正在執行的任務", MainMenu)
	}
	// Also reset session mode
	h.Sessions.Reset(c.Sender().ID)
	return c.Send("🛑 取消信號已發送，任務將停止...", MainMenu)
}

// HandleCancel is the /cancel command (same as cancel button).
func (h *Hub) HandleCancel(c tele.Context) error {
	slog.Info("🎯 /cancel", "user_id", c.Sender().ID)
	if cancelled := h.Tasks.Cancel(); !cancelled {
		h.Sessions.Reset(c.Sender().ID)
		return c.Send("ℹ️ 目前沒有正在執行的任務", MainMenu)
	}
	h.Sessions.Reset(c.Sender().ID)
	return c.Send("🛑 取消信號已發送，任務將停止...", MainMenu)
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
			MainMenu,
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
