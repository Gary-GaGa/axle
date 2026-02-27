package handler

import (
	"fmt"

	"github.com/garyellow/axle/internal/bot/skill"
	tele "gopkg.in/telebot.v3"
)

// ── Main interaction menu ─────────────────────────────────────────────────────

var MainMenu = &tele.ReplyMarkup{}

var (
	BtnReadCode    = MainMenu.Data("📁 讀取代碼", "skill_read")
	BtnWriteFile   = MainMenu.Data("✏️ 寫入檔案", "skill_write")
	BtnExec        = MainMenu.Data("⚡ 執行指令", "skill_exec")
	BtnCopilot     = MainMenu.Data("🤖 Copilot 助手", "skill_copilot")
	BtnWebSearch   = MainMenu.Data("🔍 Web 搜尋", "skill_websearch")
	BtnWebFetch    = MainMenu.Data("🌐 Web 擷取", "skill_webfetch")
	BtnSwitchModel   = MainMenu.Data("🔄 切換模型", "switch_model")
	BtnSwitchProject = MainMenu.Data("📂 切換專案", "switch_project")
	BtnStatus        = MainMenu.Data("📊 系統狀態", "status")
	BtnCancel        = MainMenu.Data("🛑 取消任務", "cancel_task")
)

// ── Exec confirmation menu ───────────────────────────────────────────────────

var ExecMenu = &tele.ReplyMarkup{}

var (
	BtnExecConfirm = ExecMenu.Data("✅ 確認執行", "exec_confirm")
	BtnExecCancel  = ExecMenu.Data("❌ 取消", "exec_cancel")
)

// ExecDangerMenu is shown for dangerous commands with an extra warning.
var ExecDangerMenu = &tele.ReplyMarkup{}

var (
	BtnExecDangerConfirm = ExecDangerMenu.Data("⚠️ 我確認要執行", "exec_danger_confirm")
	BtnExecDangerCancel  = ExecDangerMenu.Data("❌ 取消", "exec_cancel")
)

// ── Model selection: two-step (vendor → model) ──────────────────────────────

// BtnSelectVendor is the base btn used as the handler key for vendor choices.
var BtnSelectVendor = tele.Btn{Unique: "select_vendor"}

// BtnSelectModel is the base btn used as the handler key for all model choices.
// Individual model buttons share this Unique; payload carries the model name.
var BtnSelectModel = tele.Btn{Unique: "select_model"}

// BtnBackToMain returns user to the main menu from model selection.
var BtnBackToMain = tele.Btn{Unique: "back_main"}

// BtnBackToVendor returns user to the vendor list from model selection.
var BtnBackToVendor = tele.Btn{Unique: "back_vendor"}

// ── Copilot session menu (shown during active conversation) ──────────────────

var CopilotSessionMenu = &tele.ReplyMarkup{}

var (
	BtnCopilotSwitchModel = CopilotSessionMenu.Data("🔄 切換模型", "copilot_switch_model")
	BtnCopilotExit        = CopilotSessionMenu.Data("⬅️ 返回主選單", "copilot_exit")
)

// BuildVendorMenu constructs the inline keyboard of model providers.
// source indicates what triggered the model selection: "main" or "copilot".
func BuildVendorMenu(source string) *tele.ReplyMarkup {
	m := &tele.ReplyMarkup{}
	var rows []tele.Row

	for _, provider := range skill.ProviderOrder {
		label := skill.ProviderLabel[provider]
		count := len(skill.ModelsByProvider(provider))
		rows = append(rows, m.Row(
			m.Data(fmt.Sprintf("%s (%d)", label, count), "select_vendor", source, provider),
		))
	}
	rows = append(rows, m.Row(m.Data("⬅️ 返回主選單", "back_main")))
	m.Inline(rows...)
	return m
}

// BuildModelMenu constructs the inline keyboard of models for a specific vendor.
// source indicates what triggered the model selection: "main" or "copilot".
func BuildModelMenu(source, vendor string) *tele.ReplyMarkup {
	m := &tele.ReplyMarkup{}
	models := skill.ModelsByProvider(vendor)
	rows := make([]tele.Row, 0, len(models)/2+2)

	for i := 0; i < len(models); i += 2 {
		if i+1 < len(models) {
			rows = append(rows, m.Row(
				m.Data(models[i].ModelLabel(), "select_model", source, models[i].ID),
				m.Data(models[i+1].ModelLabel(), "select_model", source, models[i+1].ID),
			))
		} else {
			rows = append(rows, m.Row(
				m.Data(models[i].ModelLabel(), "select_model", source, models[i].ID),
			))
		}
	}
	rows = append(rows, m.Row(m.Data("⬅️ 返回廠商列表", "back_vendor", source)))
	m.Inline(rows...)
	return m
}

func init() {
	MainMenu.Inline(
		MainMenu.Row(BtnReadCode, BtnWriteFile),
		MainMenu.Row(BtnExec, BtnCopilot),
		MainMenu.Row(BtnWebSearch, BtnWebFetch),
		MainMenu.Row(BtnSwitchModel, BtnSwitchProject),
		MainMenu.Row(BtnStatus, BtnCancel),
	)
	ExecMenu.Inline(
		ExecMenu.Row(BtnExecConfirm, BtnExecCancel),
	)
	ExecDangerMenu.Inline(
		ExecDangerMenu.Row(BtnExecDangerConfirm, BtnExecDangerCancel),
	)
	CopilotSessionMenu.Inline(
		CopilotSessionMenu.Row(BtnCopilotSwitchModel, BtnCopilotExit),
	)
}

