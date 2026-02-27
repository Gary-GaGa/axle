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
	BtnListDir     = MainMenu.Data("📂 目錄瀏覽", "skill_listdir")
	BtnSearch      = MainMenu.Data("🔎 搜尋代碼", "skill_search")
	BtnExec        = MainMenu.Data("⚡ 執行指令", "skill_exec")
	BtnCopilot     = MainMenu.Data("🤖 Copilot 助手", "skill_copilot")
	BtnWebSearch   = MainMenu.Data("🔍 Web 搜尋", "skill_websearch")
	BtnWebFetch    = MainMenu.Data("🌐 Web 擷取", "skill_webfetch")
	BtnGit         = MainMenu.Data("🔀 Git 操作", "skill_git")
	BtnPlugins     = MainMenu.Data("🧩 擴充技能", "skill_plugins")
	BtnSubAgents   = MainMenu.Data("👥 子代理", "skill_subagents")
	BtnScheduler   = MainMenu.Data("⏰ 排程任務", "skill_scheduler")
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

// ── Git submenu ──────────────────────────────────────────────────────────────

var GitMenu = &tele.ReplyMarkup{}

var (
	BtnGitStatus     = GitMenu.Data("📊 Status", "git_status")
	BtnGitDiff       = GitMenu.Data("📝 Diff", "git_diff")
	BtnGitDiffStaged = GitMenu.Data("📦 Diff (Staged)", "git_diff_staged")
	BtnGitLog        = GitMenu.Data("📜 Log", "git_log")
	BtnGitCommitPush = GitMenu.Data("🚀 Commit + Push", "git_commit_push")
	BtnGitBack       = GitMenu.Data("⬅️ 返回主選單", "back_main")
)

// Git commit confirmation menu
var GitCommitMenu = &tele.ReplyMarkup{}

var (
	BtnGitCommitConfirm = GitCommitMenu.Data("✅ 確認 Commit+Push", "git_commit_confirm")
	BtnGitCommitCancel  = GitCommitMenu.Data("❌ 取消", "git_commit_cancel")
)

// ── Sub-agent submenu ─────────────────────────────────────────────────────────

var SubAgentMenu = &tele.ReplyMarkup{}

var (
	BtnSubAgentCreate = SubAgentMenu.Data("➕ 建立子代理", "subagent_create")
	BtnSubAgentList   = SubAgentMenu.Data("📋 查看清單", "subagent_list")
	BtnSubAgentBack   = SubAgentMenu.Data("⬅️ 返回主選單", "back_main")
)

// BtnSubAgentCancel is the base button for cancelling a specific sub-agent.
var BtnSubAgentCancel = tele.Btn{Unique: "subagent_cancel"}

// ── Plugin submenu ────────────────────────────────────────────────────────────

// BtnPluginExec is the base button for executing a specific plugin.
var BtnPluginExec = tele.Btn{Unique: "plugin_exec"}
var BtnPluginReload = tele.Btn{Unique: "plugin_reload"}

// ── Scheduler submenu ─────────────────────────────────────────────────────────

var SchedulerMenu = &tele.ReplyMarkup{}

var (
	BtnSchedCreate = SchedulerMenu.Data("➕ 建立排程", "sched_create")
	BtnSchedList   = SchedulerMenu.Data("📋 查看排程", "sched_list")
	BtnSchedBack   = SchedulerMenu.Data("⬅️ 返回主選單", "back_main")
)

// BtnSchedDelete is the base button for deleting a specific schedule.
var BtnSchedDelete = tele.Btn{Unique: "sched_delete"}
// BtnSchedToggle is the base button for toggling a specific schedule.
var BtnSchedToggle = tele.Btn{Unique: "sched_toggle"}

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
		MainMenu.Row(BtnReadCode, BtnWriteFile, BtnListDir),
		MainMenu.Row(BtnSearch, BtnExec, BtnCopilot),
		MainMenu.Row(BtnWebSearch, BtnWebFetch, BtnGit),
		MainMenu.Row(BtnPlugins, BtnSubAgents, BtnScheduler),
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
	GitMenu.Inline(
		GitMenu.Row(BtnGitStatus, BtnGitDiff),
		GitMenu.Row(BtnGitDiffStaged, BtnGitLog),
		GitMenu.Row(BtnGitCommitPush),
		GitMenu.Row(BtnGitBack),
	)
	GitCommitMenu.Inline(
		GitCommitMenu.Row(BtnGitCommitConfirm, BtnGitCommitCancel),
	)
	SubAgentMenu.Inline(
		SubAgentMenu.Row(BtnSubAgentCreate, BtnSubAgentList),
		SubAgentMenu.Row(BtnSubAgentBack),
	)
	SchedulerMenu.Inline(
		SchedulerMenu.Row(BtnSchedCreate, BtnSchedList),
		SchedulerMenu.Row(BtnSchedBack),
	)
}

