package handler

import (
	"log/slog"

	tele "gopkg.in/telebot.v3"
)

// HandleStart handles the /start command. Shows task status + main menu.
func (h *Hub) HandleStart(c tele.Context) error {
	slog.Info("🎯 /start", "user_id", c.Sender().ID)

	running, taskName, elapsed := h.Tasks.Status()
	status := "待機中 🟢"
	if running {
		status = "執行中 🔴 — 任務：" + taskName + "（" + elapsed.String() + "）"
	}

	return c.Send(
		"🔧 *Axle 引擎已啟動*\n\n"+
			"當前模式：單兵作戰\n"+
			"任務狀態："+status+"\n\n"+
			"請選擇操作：",
		h.mm(c),
		tele.ModeMarkdown,
	)
}
