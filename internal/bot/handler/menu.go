package handler

import (
	"fmt"

	"github.com/garyellow/axle/internal/bot/skill"
	tele "gopkg.in/telebot.v3"
)

// ── Button unique keys (used by bot.Handle) ──────────────────────────────────

var MainMenu = &tele.ReplyMarkup{} // default menu (no extras), built in init()

var (
	BtnReadCode      = MainMenu.Data("📁 讀取代碼", "skill_read")
	BtnWriteFile     = MainMenu.Data("✏️ 寫入檔案", "skill_write")
	BtnListDir       = MainMenu.Data("📂 目錄瀏覽", "skill_listdir")
	BtnSearch        = MainMenu.Data("🔎 搜尋代碼", "skill_search")
	BtnExec          = MainMenu.Data("⚡ 執行指令", "skill_exec")
	BtnCopilot       = MainMenu.Data("🤖 Copilot 助手", "skill_copilot")
	BtnWebSearch     = MainMenu.Data("🔍 Web 搜尋", "skill_websearch")
	BtnWebFetch      = MainMenu.Data("🌐 Web 擷取", "skill_webfetch")
	BtnMemory        = MainMenu.Data("🧠 記憶 / 歷史", "skill_memory")
	BtnBrowser       = MainMenu.Data("🌐 Browser", "skill_browser")
	BtnGit           = MainMenu.Data("🔀 Git 操作", "skill_git")
	BtnPlugins       = MainMenu.Data("🧩 擴充技能", "skill_plugins")
	BtnSubAgents     = MainMenu.Data("👥 子代理", "skill_subagents")
	BtnWorkflows     = MainMenu.Data("🧭 工作流", "skill_workflows")
	BtnScheduler     = MainMenu.Data("⏰ 排程任務", "skill_scheduler")
	BtnGateway       = MainMenu.Data("🌉 Web Gateway", "skill_gateway")
	BtnGitHub        = MainMenu.Data("🐙 GitHub", "skill_github")
	BtnEmail         = MainMenu.Data("📧 Email", "skill_email")
	BtnCalendar      = MainMenu.Data("📅 行事曆", "skill_calendar")
	BtnBriefing      = MainMenu.Data("📢 每日簡報", "skill_briefing")
	BtnSwitchModel   = MainMenu.Data("🔄 切換模型", "switch_model")
	BtnSwitchProject = MainMenu.Data("📂 切換專案", "switch_project")
	BtnStatus        = MainMenu.Data("📊 系統狀態", "status")
	BtnCancel        = MainMenu.Data("🛑 取消任務", "cancel_task")
)

// ── Extra features (togglable in main menu) ──────────────────────────────────

// ExtraFeature describes an optional feature that can be pinned to the main menu.
type ExtraFeature struct {
	ID     string
	Label  string
	Unique string
}

// ExtraFeatures defines all optional features that can be toggled.
var ExtraFeatures = []ExtraFeature{
	{"listdir", "📂 目錄瀏覽", "skill_listdir"},
	{"search", "🔎 搜尋代碼", "skill_search"},
	{"websearch", "🔍 Web 搜尋", "skill_websearch"},
	{"webfetch", "🌐 Web 擷取", "skill_webfetch"},
	{"memory", "🧠 記憶 / 歷史", "skill_memory"},
	{"browser", "🌐 Browser", "skill_browser"},
	{"git", "🔀 Git 操作", "skill_git"},
	{"github", "🐙 GitHub", "skill_github"},
	{"email", "📧 Email", "skill_email"},
	{"calendar", "📅 行事曆", "skill_calendar"},
	{"briefing", "📢 每日簡報", "skill_briefing"},
	{"plugins", "🧩 擴充技能", "skill_plugins"},
	{"subagents", "👥 子代理", "skill_subagents"},
	{"workflows", "🧭 工作流", "skill_workflows"},
	{"scheduler", "⏰ 排程任務", "skill_scheduler"},
	{"gateway", "🌉 Web Gateway", "skill_gateway"},
	{"selfupgrade", "🔧 自我升級", "skill_selfupgrade"},
}

// BtnExtras opens the extras toggle menu.
var BtnExtras = tele.Btn{Unique: "extras_menu"}

// BtnToggleExtra toggles a single extra feature. Payload = feature ID.
var BtnToggleExtra = tele.Btn{Unique: "toggle_extra"}

// BuildMainMenu constructs the main menu dynamically based on enabled extras.
func BuildMainMenu(extras map[string]bool) *tele.ReplyMarkup {
	m := &tele.ReplyMarkup{}
	rows := []tele.Row{
		m.Row(
			m.Data("📁 讀取代碼", "skill_read"),
			m.Data("✏️ 寫入檔案", "skill_write"),
			m.Data("🤖 Copilot 助手", "skill_copilot"),
		),
		m.Row(
			m.Data("⚡ 執行指令", "skill_exec"),
			m.Data("🔄 切換模型", "switch_model"),
			m.Data("📂 切換專案", "switch_project"),
		),
	}

	// Add enabled extras in rows of 3
	var extraBtns []tele.Btn
	for _, ef := range ExtraFeatures {
		if extras[ef.ID] {
			extraBtns = append(extraBtns, m.Data(ef.Label, ef.Unique))
		}
	}
	for i := 0; i < len(extraBtns); i += 3 {
		end := i + 3
		if end > len(extraBtns) {
			end = len(extraBtns)
		}
		rows = append(rows, m.Row(extraBtns[i:end]...))
	}

	rows = append(rows,
		m.Row(
			m.Data("➕ 更多功能", "extras_menu"),
			m.Data("📊 系統狀態", "status"),
			m.Data("🛑 取消任務", "cancel_task"),
		),
	)

	m.Inline(rows...)
	return m
}

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

var BtnSelectVendor = tele.Btn{Unique: "select_vendor"}
var BtnSelectModel = tele.Btn{Unique: "select_model"}
var BtnBackToMain = tele.Btn{Unique: "back_main"}
var BtnBackToVendor = tele.Btn{Unique: "back_vendor"}

// ── Copilot session menu ─────────────────────────────────────────────────────

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

var GitCommitMenu = &tele.ReplyMarkup{}

var (
	BtnGitCommitConfirm = GitCommitMenu.Data("✅ 確認 Commit+Push", "git_commit_confirm")
	BtnGitCommitCancel  = GitCommitMenu.Data("❌ 取消", "git_commit_cancel")
)

// ── Memory submenu ────────────────────────────────────────────────────────────

var MemoryMenu = &tele.ReplyMarkup{}

var (
	BtnMemorySearch = MemoryMenu.Data("🔎 搜尋歷史", "memory_search")
	BtnMemoryRecent = MemoryMenu.Data("🕘 最近對話", "memory_recent")
	BtnMemoryClear  = MemoryMenu.Data("🧹 清除記憶", "memory_clear")
	BtnMemoryBack   = MemoryMenu.Data("⬅️ 返回主選單", "back_main")
)

// ── Browser submenu ───────────────────────────────────────────────────────────

var BrowserMenu = &tele.ReplyMarkup{}

var (
	BtnBrowserRun      = BrowserMenu.Data("▶️ 執行腳本", "browser_run")
	BtnBrowserExamples = BrowserMenu.Data("📘 腳本範例", "browser_examples")
	BtnBrowserBack     = BrowserMenu.Data("⬅️ 返回主選單", "back_main")
)

// ── Sub-agent submenu ─────────────────────────────────────────────────────────

var SubAgentMenu = &tele.ReplyMarkup{}

var (
	BtnSubAgentCreate = SubAgentMenu.Data("➕ 建立子代理", "subagent_create")
	BtnSubAgentList   = SubAgentMenu.Data("📋 查看清單", "subagent_list")
	BtnSubAgentBack   = SubAgentMenu.Data("⬅️ 返回主選單", "back_main")
)

var BtnSubAgentCancel = tele.Btn{Unique: "subagent_cancel"}

// ── Workflow submenu ──────────────────────────────────────────────────────────

var WorkflowMenu = &tele.ReplyMarkup{}

var (
	BtnWorkflowCreate = WorkflowMenu.Data("➕ 建立工作流", "workflow_create")
	BtnWorkflowList   = WorkflowMenu.Data("📋 查看工作流", "workflow_list")
	BtnWorkflowBack   = WorkflowMenu.Data("⬅️ 返回主選單", "back_main")
)

var BtnWorkflowCancel = tele.Btn{Unique: "workflow_cancel"}

// ── Plugin submenu ────────────────────────────────────────────────────────────

var BtnPluginExec = tele.Btn{Unique: "plugin_exec"}
var BtnPluginReload = tele.Btn{Unique: "plugin_reload"}

// ── Scheduler submenu ─────────────────────────────────────────────────────────

var SchedulerMenu = &tele.ReplyMarkup{}

var (
	BtnSchedCreate = SchedulerMenu.Data("➕ 建立排程", "sched_create")
	BtnSchedList   = SchedulerMenu.Data("📋 查看排程", "sched_list")
	BtnSchedBack   = SchedulerMenu.Data("⬅️ 返回主選單", "back_main")
)

var BtnSchedDelete = tele.Btn{Unique: "sched_delete"}
var BtnSchedToggle = tele.Btn{Unique: "sched_toggle"}

// ── GitHub submenu ────────────────────────────────────────────────────────────

var GitHubMenu = &tele.ReplyMarkup{}

var (
	BtnGHPRList    = GitHubMenu.Data("📋 PR 列表", "gh_pr_list")
	BtnGHIssueList = GitHubMenu.Data("📋 Issue 列表", "gh_issue_list")
	BtnGHCIStatus  = GitHubMenu.Data("🔄 CI 狀態", "gh_ci_status")
	BtnGHPRCreate  = GitHubMenu.Data("➕ 建立 PR", "gh_pr_create")
	BtnGHRepoView  = GitHubMenu.Data("📦 Repo 資訊", "gh_repo_view")
	BtnGHBack      = GitHubMenu.Data("⬅️ 返回主選單", "back_main")
)

// ── Email submenu ─────────────────────────────────────────────────────────────

var EmailMenu = &tele.ReplyMarkup{}

var (
	BtnEmailSend = EmailMenu.Data("📤 發送 Email", "email_send")
	BtnEmailRead = EmailMenu.Data("📥 讀取信箱", "email_read")
	BtnEmailBack = EmailMenu.Data("⬅️ 返回主選單", "back_main")
)

// ── Calendar submenu ──────────────────────────────────────────────────────────

var CalendarMenu = &tele.ReplyMarkup{}

var (
	BtnCalToday    = CalendarMenu.Data("📅 今日行程", "cal_today")
	BtnCalTomorrow = CalendarMenu.Data("📅 明日行程", "cal_tomorrow")
	BtnCalWeek     = CalendarMenu.Data("📅 本週行程", "cal_week")
	BtnCalBack     = CalendarMenu.Data("⬅️ 返回主選單", "back_main")
)

// ── PDF action menu (after upload) ────────────────────────────────────────────

var PDFMenu = &tele.ReplyMarkup{}

var (
	BtnPDFSummarize = PDFMenu.Data("📝 AI 摘要", "pdf_summarize")
	BtnPDFBack      = PDFMenu.Data("⬅️ 返回主選單", "back_main")
)

// ── Self-Upgrade menu ─────────────────────────────────────────────────────────

var UpgradeMenu = &tele.ReplyMarkup{}

var (
	BtnUpgradeConfirm = UpgradeMenu.Data("✅ 確認升級", "upgrade_confirm")
	BtnUpgradeCancel  = UpgradeMenu.Data("❌ 取消", "upgrade_cancel")
)

var BtnSelfUpgrade = tele.Btn{Unique: "skill_selfupgrade"}

// ── Menu builders ─────────────────────────────────────────────────────────────

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
	def := BuildMainMenu(nil)
	MainMenu.InlineKeyboard = def.InlineKeyboard

	ExecMenu.Inline(ExecMenu.Row(BtnExecConfirm, BtnExecCancel))
	ExecDangerMenu.Inline(ExecDangerMenu.Row(BtnExecDangerConfirm, BtnExecDangerCancel))
	CopilotSessionMenu.Inline(CopilotSessionMenu.Row(BtnCopilotSwitchModel, BtnCopilotExit))
	GitMenu.Inline(
		GitMenu.Row(BtnGitStatus, BtnGitDiff),
		GitMenu.Row(BtnGitDiffStaged, BtnGitLog),
		GitMenu.Row(BtnGitCommitPush),
		GitMenu.Row(BtnGitBack),
	)
	GitCommitMenu.Inline(GitCommitMenu.Row(BtnGitCommitConfirm, BtnGitCommitCancel))
	MemoryMenu.Inline(
		MemoryMenu.Row(BtnMemorySearch, BtnMemoryRecent),
		MemoryMenu.Row(BtnMemoryClear),
		MemoryMenu.Row(BtnMemoryBack),
	)
	BrowserMenu.Inline(
		BrowserMenu.Row(BtnBrowserRun, BtnBrowserExamples),
		BrowserMenu.Row(BtnBrowserBack),
	)
	SubAgentMenu.Inline(
		SubAgentMenu.Row(BtnSubAgentCreate, BtnSubAgentList),
		SubAgentMenu.Row(BtnSubAgentBack),
	)
	WorkflowMenu.Inline(
		WorkflowMenu.Row(BtnWorkflowCreate, BtnWorkflowList),
		WorkflowMenu.Row(BtnWorkflowBack),
	)
	SchedulerMenu.Inline(
		SchedulerMenu.Row(BtnSchedCreate, BtnSchedList),
		SchedulerMenu.Row(BtnSchedBack),
	)
	GitHubMenu.Inline(
		GitHubMenu.Row(BtnGHPRList, BtnGHIssueList),
		GitHubMenu.Row(BtnGHCIStatus, BtnGHRepoView),
		GitHubMenu.Row(BtnGHPRCreate),
		GitHubMenu.Row(BtnGHBack),
	)
	EmailMenu.Inline(
		EmailMenu.Row(BtnEmailSend, BtnEmailRead),
		EmailMenu.Row(BtnEmailBack),
	)
	CalendarMenu.Inline(
		CalendarMenu.Row(BtnCalToday, BtnCalTomorrow),
		CalendarMenu.Row(BtnCalWeek),
		CalendarMenu.Row(BtnCalBack),
	)
	PDFMenu.Inline(
		PDFMenu.Row(BtnPDFSummarize, BtnPDFBack),
	)
}
