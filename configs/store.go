package configs

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const credsDirName = ".axle"
const credsFileName = "credentials.json"

// storedCreds is the schema persisted to ~/.axle/credentials.json.
type storedCreds struct {
	TelegramToken  string `json:"telegram_token,omitempty"`
	AllowedUserIDs string `json:"allowed_user_ids,omitempty"`
	Workspace      string `json:"workspace,omitempty"`
}

// CredsFilePath returns the absolute path to the credentials file.
func CredsFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("無法取得 Home 目錄: %w", err)
	}
	return filepath.Join(home, credsDirName, credsFileName), nil
}

// loadStoredCreds reads the credentials file. Returns empty struct if file
// does not exist (first run).
func loadStoredCreds() (*storedCreds, error) {
	path, err := CredsFilePath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &storedCreds{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("讀取憑證檔案失敗: %w", err)
	}
	var creds storedCreds
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("解析憑證檔案失敗（可能已損毀）: %w", err)
	}
	return &creds, nil
}

// saveStoredCreds writes credentials to ~/.axle/credentials.json with
// permissions 0600 (owner read/write only).
func saveStoredCreds(creds *storedCreds) error {
	path, err := CredsFilePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("建立憑證目錄失敗: %w", err)
	}
	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化憑證失敗: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("寫入憑證檔案失敗: %w", err)
	}
	return nil
}
