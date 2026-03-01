package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/garyellow/axle/configs"
	"github.com/garyellow/axle/internal/app"
	"github.com/garyellow/axle/internal/bot/handler"
	"github.com/garyellow/axle/internal/bot/skill"
	"github.com/garyellow/axle/internal/web"

	tele "gopkg.in/telebot.v3"
	mw "github.com/garyellow/axle/internal/bot/middleware"
)

func main() {
	// --- Logger ---
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	slog.Info("🚀 Axle 啟動中...")

	// --- Config ---
	cfg, err := configs.Load()
	if err != nil {
		slog.Error("設定載入失敗", "error", err)
		os.Exit(1)
	}
	slog.Info("✅ 設定載入完成",
		"allowed_users", cfg.AllowedUserIDs,
		"workspace", cfg.Workspace,
	)

	// --- Axle home directory ---
	home, _ := os.UserHomeDir()
	axleDir := filepath.Join(home, ".axle")

	// --- Telegram Bot ---
	bot, err := tele.NewBot(tele.Settings{
		Token:  cfg.TelegramToken,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	})
	if err != nil {
		slog.Error("Telegram Bot 初始化失敗", "error", err)
		os.Exit(1)
	}

	// --- Shared services ---
	tasks := &app.TaskManager{}
	sessions := app.NewSessionManager()
	hub := handler.NewHub(tasks, sessions, bot, cfg.Workspace)
	hub.AllowedUserIDs = cfg.AllowedUserIDs

	// --- Email config ---
	hub.EmailConfig = &skill.EmailConfig{
		SMTPHost: cfg.SMTPHost,
		SMTPPort: cfg.SMTPPort,
		IMAPHost: cfg.IMAPHost,
		IMAPPort: cfg.IMAPPort,
		Address:  cfg.EmailAddress,
		Password: cfg.EmailPassword,
	}

	// --- Memory store ---
	memory, err := app.NewMemoryStore(axleDir)
	if err != nil {
		slog.Warn("記憶系統初始化失敗", "error", err)
	} else {
		hub.Memory = memory
		for _, uid := range cfg.AllowedUserIDs {
			_ = memory.Load(uid)
		}
		slog.Info("✅ 記憶系統已載入")
	}

	// --- Sub-agent manager ---
	hub.SubAgents = app.NewSubAgentManager()

	// --- Plugin manager ---
	plugins, err := app.NewPluginManager(axleDir)
	if err != nil {
		slog.Warn("插件系統初始化失敗", "error", err)
	} else {
		hub.Plugins = plugins
	}

	// --- Scheduler ---
	scheduler, err := app.NewScheduleManager(axleDir)
	if err != nil {
		slog.Warn("排程系統初始化失敗", "error", err)
	} else {
		hub.Scheduler = scheduler
		// Set up execution callback: run command and notify users
		scheduler.SetExecFunc(func(schedID, command string) {
			slog.Info("⏰ 排程觸發", "id", schedID, "cmd", command)

			// Internal commands (prefixed with @)
			if command == "@briefing" {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				briefText := skill.GenerateBriefing(ctx, cfg.Workspace)
				for _, uid := range cfg.AllowedUserIDs {
					chat := tele.ChatID(uid)
					chunks := skill.SplitMessage(briefText)
					for _, chunk := range chunks {
						bot.Send(chat, chunk, tele.ModeMarkdown)
					}
				}
				return
			}

			for _, uid := range cfg.AllowedUserIDs {
				chat := tele.ChatID(uid)
				bot.Send(chat, fmt.Sprintf("⏰ 排程任務執行中：`%s`", command), tele.ModeMarkdown)
			}
			// Run using ExecShell directly (not via task slot to avoid blocking)
			out, execErr := func() (string, error) {
				ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
				defer cancel()
				return skill.ExecShell(ctx, cfg.Workspace, command)
			}()
			for _, uid := range cfg.AllowedUserIDs {
				chat := tele.ChatID(uid)
				if execErr != nil {
					bot.Send(chat, fmt.Sprintf("❌ 排程 `%s` 失敗：%s", schedID, execErr.Error()), tele.ModeMarkdown)
				} else {
					chunks := skill.SplitMessage(fmt.Sprintf("✅ 排程 `%s` 完成：\n```\n%s\n```", schedID, out))
					for _, chunk := range chunks {
						bot.Send(chat, chunk, tele.ModeMarkdown)
					}
				}
			}
		})
		scheduler.StartAll()
		slog.Info("✅ 排程系統已啟動")
	}

	// --- RPG Dashboard ---
	rpg := app.NewRPGManager(axleDir)
	hub.RPG = rpg

	webSrv := web.NewServer(":8080", rpg)
	webSrv.Start()

	// --- Middleware: Auth Whitelist ---
	bot.Use(mw.AuthMiddleware(cfg.AllowedUserIDs))

	// --- Commands ---
	bot.Handle("/start", hub.HandleStart)
	bot.Handle("/cancel", hub.HandleCancel)

	// --- Text fallback (route by session mode) ---
	bot.Handle(tele.OnText, hub.HandleText)

	// --- Main menu callbacks ---
	bot.Handle(&handler.BtnReadCode, hub.HandleReadCodeBtn)
	bot.Handle(&handler.BtnWriteFile, hub.HandleWriteFileBtn)
	bot.Handle(&handler.BtnListDir, hub.HandleListDirBtn)
	bot.Handle(&handler.BtnSearch, hub.HandleSearchBtn)
	bot.Handle(&handler.BtnExec, hub.HandleExecBtn)
	bot.Handle(&handler.BtnCopilot, hub.HandleCopilotBtn)
	bot.Handle(&handler.BtnWebSearch, hub.HandleWebSearchBtn)
	bot.Handle(&handler.BtnWebFetch, hub.HandleWebFetchBtn)
	bot.Handle(&handler.BtnGit, hub.HandleGitBtn)
	bot.Handle(&handler.BtnPlugins, hub.HandlePluginsBtn)
	bot.Handle(&handler.BtnSubAgents, hub.HandleSubAgentsBtn)
	bot.Handle(&handler.BtnScheduler, hub.HandleSchedulerBtn)
	bot.Handle(&handler.BtnSwitchModel, hub.HandleSwitchModelBtn)
	bot.Handle(&handler.BtnSwitchProject, hub.HandleSwitchProjectBtn)
	bot.Handle(&handler.BtnStatus, hub.HandleStatus)
	bot.Handle(&handler.BtnCancel, hub.HandleCancelTask)

	// --- Exec confirm/cancel ---
	bot.Handle(&handler.BtnExecConfirm, hub.HandleExecConfirm)
	bot.Handle(&handler.BtnExecDangerConfirm, hub.HandleExecDangerConfirm)
	bot.Handle(&handler.BtnExecCancel, hub.HandleExecCancelBtn)

	// --- Model selection (vendor → model two-step flow) ---
	bot.Handle(&handler.BtnSelectVendor, hub.HandleVendorSelect)
	bot.Handle(&handler.BtnSelectModel, hub.HandleModelSelect)
	bot.Handle(&handler.BtnBackToMain, hub.HandleBackToMain)
	bot.Handle(&handler.BtnBackToVendor, hub.HandleBackToVendor)

	// --- Copilot session controls ---
	bot.Handle(&handler.BtnCopilotSwitchModel, hub.HandleCopilotSwitchModel)
	bot.Handle(&handler.BtnCopilotExit, hub.HandleCopilotExit)

	// --- Git operations ---
	bot.Handle(&handler.BtnGitStatus, hub.HandleGitStatus)
	bot.Handle(&handler.BtnGitDiff, hub.HandleGitDiff)
	bot.Handle(&handler.BtnGitDiffStaged, hub.HandleGitDiffStaged)
	bot.Handle(&handler.BtnGitLog, hub.HandleGitLog)
	bot.Handle(&handler.BtnGitCommitPush, hub.HandleGitCommitPush)
	bot.Handle(&handler.BtnGitCommitConfirm, hub.HandleGitCommitConfirm)
	bot.Handle(&handler.BtnGitCommitCancel, hub.HandleGitCommitCancel)

	// --- Sub-agents ---
	bot.Handle(&handler.BtnSubAgentCreate, hub.HandleSubAgentCreate)
	bot.Handle(&handler.BtnSubAgentList, hub.HandleSubAgentList)
	bot.Handle(&handler.BtnSubAgentCancel, hub.HandleSubAgentCancel)

	// --- Plugins ---
	bot.Handle(&handler.BtnPluginExec, hub.HandlePluginExec)
	bot.Handle(&handler.BtnPluginReload, hub.HandlePluginReload)

	// --- Scheduler ---
	bot.Handle(&handler.BtnSchedCreate, hub.HandleSchedCreate)
	bot.Handle(&handler.BtnSchedList, hub.HandleSchedList)
	bot.Handle(&handler.BtnSchedDelete, hub.HandleSchedDelete)
	bot.Handle(&handler.BtnSchedToggle, hub.HandleSchedToggle)

	// --- Extras toggle ---
	bot.Handle(&handler.BtnExtras, hub.HandleExtrasBtn)
	bot.Handle(&handler.BtnToggleExtra, hub.HandleToggleExtra)

	// --- GitHub operations ---
	bot.Handle(&handler.BtnGitHub, hub.HandleGitHubBtn)
	bot.Handle(&handler.BtnGHPRList, hub.HandleGHPRList)
	bot.Handle(&handler.BtnGHIssueList, hub.HandleGHIssueList)
	bot.Handle(&handler.BtnGHCIStatus, hub.HandleGHCIStatus)
	bot.Handle(&handler.BtnGHRepoView, hub.HandleGHRepoView)
	bot.Handle(&handler.BtnGHPRCreate, hub.HandleGHPRCreate)

	// --- Email ---
	bot.Handle(&handler.BtnEmail, hub.HandleEmailBtn)
	bot.Handle(&handler.BtnEmailSend, hub.HandleEmailSend)
	bot.Handle(&handler.BtnEmailRead, hub.HandleEmailRead)

	// --- Calendar ---
	bot.Handle(&handler.BtnCalendar, hub.HandleCalendarBtn)
	bot.Handle(&handler.BtnCalToday, hub.HandleCalToday)
	bot.Handle(&handler.BtnCalTomorrow, hub.HandleCalTomorrow)
	bot.Handle(&handler.BtnCalWeek, hub.HandleCalWeek)

	// --- Briefing ---
	bot.Handle(&handler.BtnBriefing, hub.HandleBriefingBtn)

	// --- Document & Photo upload ---
	bot.Handle(tele.OnDocument, hub.HandleDocument)
	bot.Handle(tele.OnPhoto, hub.HandlePhoto)

	// --- PDF actions ---
	bot.Handle(&handler.BtnPDFSummarize, hub.HandlePDFSummarize)

	// --- Graceful Shutdown ---
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		sig := <-sigCh
		slog.Info("收到終止信號，正在關閉...", "signal", sig)
		tasks.Cancel() // cancel any running task

		// Stop scheduler
		if scheduler != nil {
			scheduler.StopAll()
		}

		// Stop web dashboard
		webSrv.Shutdown()

		// Notify all whitelisted users before shutdown (with timeout)
		notifyDone := make(chan struct{})
		go func() {
			defer close(notifyDone)
			for _, uid := range cfg.AllowedUserIDs {
				chat := tele.ChatID(uid)
				if _, err := bot.Send(chat, "⚠️ Axle 服務正在關閉，Bot 即將離線。\n\n_收到終止信號："+sig.String()+"_", tele.ModeMarkdown); err != nil {
					slog.Warn("關閉通知發送失敗", "user_id", uid, "error", err)
				}
			}
		}()

		// Wait for notifications with 10s hard deadline
		select {
		case <-notifyDone:
			slog.Info("關閉通知已發送")
		case <-time.After(10 * time.Second):
			slog.Warn("關閉通知超時，強制停止")
		}

		bot.Stop()
	}()

	slog.Info("🤖 Axle Bot 已上線，開始接收訊息...")

	// Notify all whitelisted users that bot is online
	for _, uid := range cfg.AllowedUserIDs {
		chat := tele.ChatID(uid)
		if _, err := bot.Send(chat, "🟢 Axle 引擎已啟動，等待指令...", handler.MainMenu); err != nil {
			slog.Warn("啟動通知發送失敗", "user_id", uid, "error", err)
		}
	}

	bot.Start()
}
