package handler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime/debug"
	"time"

	"github.com/garyellow/axle/internal/app"
	"github.com/garyellow/axle/internal/bot/skill"
	tele "gopkg.in/telebot.v3"
)

const chunkSendDelay = 300 * time.Millisecond // rate-limit between Telegram messages

// Hub holds shared dependencies for all bot handlers.
// All fields are safe for concurrent use.
type Hub struct {
	Tasks     *app.TaskManager
	Sessions  *app.SessionManager
	Bot       *tele.Bot
	Workspace string
	Memory    *app.MemoryStore
	SubAgents *app.SubAgentManager
	Plugins   *app.PluginManager
	Scheduler *app.ScheduleManager
	AllowedUserIDs []int64
	EmailConfig    *skill.EmailConfig
}

// NewHub creates a Hub wired with the provided dependencies.
func NewHub(tasks *app.TaskManager, sessions *app.SessionManager, bot *tele.Bot, workspace string) *Hub {
	return &Hub{
		Tasks:     tasks,
		Sessions:  sessions,
		Bot:       bot,
		Workspace: workspace,
	}
}

// workspaceFor returns the user's active workspace, falling back to the global default.
func (h *Hub) workspaceFor(userID int64) string {
	sess := h.Sessions.GetCopy(userID)
	if sess.ActiveWorkspace != "" {
		return sess.ActiveWorkspace
	}
	return h.Workspace
}

// ── Internal helpers ──────────────────────────────────────────────────────────

// mm returns the dynamic main menu for the current user.
func (h *Hub) mm(c tele.Context) *tele.ReplyMarkup {
	return BuildMainMenu(h.Sessions.GetCopy(c.Sender().ID).EnabledExtras)
}

// mmFor returns the dynamic main menu for a specific user ID.
func (h *Hub) mmFor(userID int64) *tele.ReplyMarkup {
	return BuildMainMenu(h.Sessions.GetCopy(userID).EnabledExtras)
}

// sendMenu sends a message with the user's dynamic main menu attached.
func (h *Hub) sendMenu(c tele.Context, text string) error {
	return c.Send(text, h.mm(c))
}

// sendChunks sends multiple message chunks, attaching the user's main menu to the last one.
// Includes a small delay between sends to avoid Telegram rate limits.
func (h *Hub) sendChunks(chat tele.Recipient, chunks []string, userID int64) {
	menu := h.mmFor(userID)
	for i, chunk := range chunks {
		if i > 0 {
			time.Sleep(chunkSendDelay)
		}
		if i == len(chunks)-1 {
			h.Bot.Send(chat, chunk, menu)
		} else {
			h.Bot.Send(chat, chunk)
		}
	}
}

// sendCopilotChunks sends chunks with CopilotSessionMenu on the last one.
func (h *Hub) sendCopilotChunks(chat tele.Recipient, chunks []string) {
	for i, chunk := range chunks {
		if i > 0 {
			time.Sleep(chunkSendDelay)
		}
		if i == len(chunks)-1 {
			h.Bot.Send(chat, chunk, CopilotSessionMenu)
		} else {
			h.Bot.Send(chat, chunk)
		}
	}
}

// progressReporter sends a "still running" update every 15 s until stopCh is closed.
func (h *Hub) progressReporter(chat tele.Recipient, taskName string, start time.Time, stopCh <-chan struct{}) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-stopCh:
			return
		case t := <-ticker.C:
			elapsed := t.Sub(start)
			h.Bot.Send(chat, fmt.Sprintf("⏳ 任務「%s」執行中... 已耗時 %ds", taskName, int(elapsed.Seconds())))
		}
	}
}

// tryStartTask tries to acquire the task slot, sending an error to the user if busy.
// Returns (ctx, done, true) on success; (nil, nil, false) if a task is already running.
// Caller MUST call done() (via defer) when the task finishes.
func (h *Hub) tryStartTask(c tele.Context, name string) (ctx context.Context, done func(), ok bool) {
	ctx, ok = h.Tasks.TryStart(name)
	if !ok {
		running, taskName, elapsed := h.Tasks.Status()
		_ = running
		c.Send(fmt.Sprintf(
			"⚠️ 目前已有任務「%s」執行中（已耗時 %.0f 秒）\n\n請等待完成，或按下 🛑 取消任務",
			taskName, elapsed.Seconds(),
		), h.mm(c))
		return nil, nil, false
	}
	return ctx, h.Tasks.Done, true
}

// ── Exec task ─────────────────────────────────────────────────────────────────

// RunExecTask starts a shell execution in a background goroutine.
func (h *Hub) RunExecTask(c tele.Context, command string) error {
	userID := c.Sender().ID
	chat := c.Chat()

	ctx, done, ok := h.tryStartTask(c, fmt.Sprintf("Shell[%d]", userID))
	if !ok {
		return nil
	}

	slog.Info("⚡ 啟動 Shell 任務", "cmd", command, "user_id", userID)
	c.Send(fmt.Sprintf("⚡ 執行中：\n```bash\n%s\n```", command), tele.ModeMarkdown)

	go func() {
		defer done()
		defer func() {
			if r := recover(); r != nil {
				slog.Error("💥 Shell 任務 panic", "recover", r, "stack", string(debug.Stack()))
				h.Bot.Send(chat, "💥 任務異常中止，請查看 Log", h.mmFor(userID))
			}
		}()

		stopProg := make(chan struct{})
		defer close(stopProg)
		go h.progressReporter(chat, "Shell", time.Now(), stopProg)

		out, err := skill.ExecShell(ctx, h.workspaceFor(userID), command)

		switch {
		case errors.Is(err, context.Canceled):
			slog.Info("🛑 Shell 任務已取消", "user_id", userID)
			h.Bot.Send(chat, "🛑 指令執行已取消", h.mmFor(userID))
		case err != nil:
			slog.Error("❌ Shell 任務失敗", "error", err)
			h.Bot.Send(chat, "❌ "+err.Error(), h.mmFor(userID))
		default:
			slog.Info("✅ Shell 任務完成", "user_id", userID)
			h.sendChunks(chat, skill.SplitMessage("```\n"+out+"\n```"), userID)
		}
	}()
	return nil
}

// ── Copilot task ──────────────────────────────────────────────────────────────

// RunCopilotTask starts a Copilot CLI task in a background goroutine.
func (h *Hub) RunCopilotTask(c tele.Context, prompt, model string) error {
	userID := c.Sender().ID
	chat := c.Chat()

	if model == "" {
		model = skill.DefaultModel
	}

	// Save user prompt to memory
	if h.Memory != nil {
		_ = h.Memory.Add(userID, "user", prompt, model)
	}

	// Build context from memory
	contextPrefix := ""
	if h.Memory != nil {
		contextPrefix = h.Memory.BuildContext(userID, 10)
	}

	fullPrompt := contextPrefix + prompt
	truncWarning := ""
	if len(fullPrompt) > skill.MaxPromptChars {
		truncWarning = fmt.Sprintf("\n⚠️ 提示詞過長，已截斷至 %d 字元", skill.MaxPromptChars)
	}

	ctx, done, ok := h.tryStartTask(c, fmt.Sprintf("Copilot[%d]", userID))
	if !ok {
		return nil
	}

	slog.Info("🤖 啟動 Copilot 任務", "model", model, "user_id", userID)
	c.Send(fmt.Sprintf("🤖 Copilot 任務已啟動（模型：`%s`）%s\n⏳ 執行中，請稍候...", model, truncWarning), tele.ModeMarkdown)

	go func() {
		defer done()
		defer func() {
			if r := recover(); r != nil {
				slog.Error("💥 Copilot 任務 panic", "recover", r, "stack", string(debug.Stack()))
				h.Bot.Send(chat, "💥 任務異常中止，請查看 Log", h.mmFor(userID))
			}
		}()

		stopProg := make(chan struct{})
		defer close(stopProg)
		go h.progressReporter(chat, "Copilot", time.Now(), stopProg)

		chunks, err := skill.RunCopilot(ctx, h.workspaceFor(userID), model, fullPrompt)

		switch {
		case errors.Is(err, context.Canceled):
			slog.Info("🛑 Copilot 任務已取消", "user_id", userID)
			h.Bot.Send(chat, "🛑 Copilot 任務已取消", h.mmFor(userID))
		case err != nil:
			slog.Error("❌ Copilot 任務失敗", "error", err)
			h.Bot.Send(chat, "❌ "+err.Error(), CopilotSessionMenu)
		default:
			slog.Info("✅ Copilot 任務完成", "chunks", len(chunks), "user_id", userID)
			// Save assistant response to memory
			if h.Memory != nil {
				fullResp := ""
				for _, ch := range chunks {
					fullResp += ch
				}
				_ = h.Memory.Add(userID, "assistant", fullResp, model)
			}
			h.sendCopilotChunks(chat, chunks)
		}
	}()
	return nil
}
