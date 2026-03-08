package configs

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all runtime configuration for the application.
type Config struct {
	TelegramToken   string
	AllowedUserIDs  []int64
	Workspace       string
	WebListenAddr   string
	WebGatewayToken string
	// Email (optional)
	EmailAddress  string
	EmailPassword string
	SMTPHost      string
	SMTPPort      string
	IMAPHost      string
	IMAPPort      string
}

// Load resolves config with the following priority (highest → lowest):
//  1. Environment variables
//  2. .env file in cwd
//  3. ~/.axle/credentials.json  (persistent local store)
//  4. Interactive terminal prompt (when required fields are still missing)
//
// After any interactive prompt, values are saved to ~/.axle/credentials.json
// so subsequent runs start without prompts.
func Load() (*Config, error) {
	// --- Viper: env vars + optional .env file ---
	viper.SetConfigFile(".env")
	viper.SetConfigType("env")
	viper.AutomaticEnv()
	viper.SetDefault("WORKSPACE", ".")
	_ = viper.ReadInConfig() // .env is optional

	// --- Persistent store ---
	stored, err := loadStoredCreds()
	if err != nil {
		slog.Warn("無法讀取本地憑證，將重新設定", "error", err)
		stored = &storedCreds{}
	}

	// --- Merge (env/viper wins over stored) ---
	telegramToken := firstNonEmpty(viper.GetString("TELEGRAM_TOKEN"), stored.TelegramToken)
	allowedRaw := firstNonEmpty(viper.GetString("ALLOWED_USER_IDS"), stored.AllowedUserIDs)
	workspace := firstNonEmpty(viper.GetString("WORKSPACE"), stored.Workspace, ".")
	webListenAddr := firstNonEmpty(viper.GetString("WEB_LISTEN_ADDR"), stored.WebListenAddr, "127.0.0.1:8080")
	webGatewayToken := firstNonEmpty(viper.GetString("WEB_GATEWAY_TOKEN"), stored.WebGatewayToken)

	// --- Interactive prompt for missing required fields ---
	needsSave := false

	if telegramToken == "" {
		fmt.Println("\n🤖 Axle 首次設定，請提供必要的憑證：")
		telegramToken = promptRequired("Telegram Bot Token", "從 @BotFather 取得")
		stored.TelegramToken = telegramToken
		needsSave = true
	}

	if allowedRaw == "" {
		allowedRaw = promptRequired(
			"允許的 Telegram User ID",
			"可用逗號分隔多個 ID，從 @userinfobot 取得自己的 ID",
		)
		stored.AllowedUserIDs = allowedRaw
		needsSave = true
	}

	if webGatewayToken == "" {
		token, err := generateToken(24)
		if err != nil {
			return nil, fmt.Errorf("產生 Web Gateway Token 失敗: %w", err)
		}
		webGatewayToken = token
		stored.WebGatewayToken = token
		stored.WebListenAddr = webListenAddr
		needsSave = true
	}

	// --- Persist to ~/.axle/credentials.json ---
	if needsSave {
		stored.Workspace = workspace
		stored.WebListenAddr = webListenAddr
		stored.WebGatewayToken = webGatewayToken
		if err := saveStoredCreds(stored); err != nil {
			slog.Warn("⚠ 憑證儲存失敗，下次啟動需重新輸入", "error", err)
		} else {
			path, _ := CredsFilePath()
			slog.Info("✅ 憑證已儲存", "path", path)
		}
	}

	// --- Parse user IDs ---
	ids, err := parseUserIDs(allowedRaw)
	if err != nil {
		return nil, fmt.Errorf("解析 ALLOWED_USER_IDS 失敗: %w", err)
	}

	return &Config{
		TelegramToken:   telegramToken,
		AllowedUserIDs:  ids,
		Workspace:       workspace,
		WebListenAddr:   webListenAddr,
		WebGatewayToken: webGatewayToken,
		EmailAddress:    firstNonEmpty(viper.GetString("EMAIL_ADDRESS"), stored.EmailAddress),
		EmailPassword:   firstNonEmpty(viper.GetString("EMAIL_PASSWORD"), stored.EmailPassword),
		SMTPHost:        firstNonEmpty(viper.GetString("SMTP_HOST"), stored.SMTPHost, "smtp.gmail.com"),
		SMTPPort:        firstNonEmpty(viper.GetString("SMTP_PORT"), stored.SMTPPort, "587"),
		IMAPHost:        firstNonEmpty(viper.GetString("IMAP_HOST"), stored.IMAPHost, "imap.gmail.com"),
		IMAPPort:        firstNonEmpty(viper.GetString("IMAP_PORT"), stored.IMAPPort, "993"),
	}, nil
}

func generateToken(byteLen int) (string, error) {
	buf := make([]byte, byteLen)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func parseUserIDs(raw string) ([]int64, error) {
	if raw == "" {
		return nil, fmt.Errorf("User ID 不可為空（安全要求）")
	}
	parts := strings.Split(raw, ",")
	ids := make([]int64, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		var id int64
		if _, err := fmt.Sscanf(p, "%d", &id); err != nil {
			return nil, fmt.Errorf("無效的 User ID: %q", p)
		}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("User ID 不可為空（安全要求）")
	}
	return ids, nil
}
