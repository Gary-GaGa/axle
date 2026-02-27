package middleware

import (
	"log/slog"

	tele "gopkg.in/telebot.v3"
)

// AuthMiddleware returns a telebot middleware that silently rejects
// messages from users not in the whitelist. Rejected users receive
// no response (stealth mode) and a warning is logged.
func AuthMiddleware(allowedIDs []int64) tele.MiddlewareFunc {
	whitelist := make(map[int64]struct{}, len(allowedIDs))
	for _, id := range allowedIDs {
		whitelist[id] = struct{}{}
	}

	return func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			sender := c.Sender()
			// Log every incoming update for observability
			slog.Info("📩 收到更新",
				"user_id", sender.ID,
				"username", sender.Username,
				"data", c.Data(),
			)

			if _, ok := whitelist[sender.ID]; !ok {
				slog.Warn("🚫 未授權存取 — 已忽略",
					"user_id", sender.ID,
					"username", sender.Username,
				)
				// Stealth mode: no response to unauthorized users
				return nil
			}

			slog.Info("✅ 已授權，轉發處理",
				"user_id", sender.ID,
				"username", sender.Username,
			)
			return next(c)
		}
	}
}
