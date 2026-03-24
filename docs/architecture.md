# Axle — 架構與開發指南 / Architecture & Dev Guide

> 正在進行 Clean Architecture + DDD 重構，請搭配閱讀 [`refactor-clean-ddd-direction.md`](refactor-clean-ddd-direction.md) 了解新的分層方向與遷移策略。

[← 返回首頁 / Back to Home](../README.md)

---

## 目錄 / Contents

- [目錄結構](#目錄結構)
- [DDD/Clean 重構方向](refactor-clean-ddd-direction.md)
- [設計原則](#設計原則)
- [安全機制](#安全機制)
- [並發模型](#並發模型)
- [開發指南](#開發指南)

---

## 目錄結構

```
axle/
├── cmd/axle/
│   └── main.go              # 進入點、Handler 註冊、Graceful Shutdown
├── configs/
│   ├── config.go            # Viper 多層設定載入
│   ├── store.go             # ~/.axle/credentials.json 持久化
│   └── prompt.go            # 終端互動式輸入
├── internal/
│   ├── app/
│   │   ├── taskmanager.go   # 單任務槽位 + Context 取消
│   │   ├── session.go       # 每用戶狀態機（Mode 路由）
│   │   ├── plugin.go        # YAML 插件管理器
│   │   ├── scheduler.go     # Cron 排程引擎
│   │   ├── rpg.go           # RPG 狀態引擎（XP/等級/事件匯流排）
│   │   └── version.go       # 版本號常數
│   ├── web/
│   │   ├── server.go        # HTTP + WebSocket 伺服器
│   │   └── static/
│   │       └── index.html   # 像素風 RPG 儀表板
│   └── bot/
│       ├── handler/
│       │   ├── hub.go       # Hub 結構（共用依賴 + 動態選單）
│       │   ├── menu.go      # 動態主選單 + 子選單定義
│       │   ├── start.go     # /start 指令
│       │   ├── text.go      # 文字訊息路由器
│       │   └── callback.go  # 所有按鈕回呼處理器
│       ├── middleware/
│       │   └── auth.go      # 白名單認證（隱形模式）
│       └── skill/           # 技能模組（純函式，可單獨測試）
│           ├── models.go    # 模型清單 + 費率 + 廠商分類
│           ├── readcode.go  # 沙箱檔案讀取
│           ├── writefile.go # 沙箱檔案寫入
│           ├── exec.go      # Shell 執行
│           ├── copilot.go   # Copilot CLI 呼叫器
│           ├── safety.go    # 三級危險偵測
│           ├── selfupgrade.go # 自我升級管線
│           └── ...          # web, git, github, email, calendar, ...
├── docs/
│   ├── tutorial-zh.md
│   ├── tutorial-en.md
│   ├── features-zh.md
│   └── architecture.md     ← 你在這裡
├── .env.example
└── go.mod
```

---

## 設計原則

### Clean Architecture 分層

```
cmd/axle  →  internal/bot/handler  →  internal/bot/skill
              internal/app (state)
              internal/web (dashboard)
```

- `cmd/` — 僅做組裝與啟動，不含業務邏輯
- `internal/bot/handler/` — Telegram 框架依賴層，處理訊息路由
- `internal/bot/skill/` — 純業務函式，不依賴 Telegram 框架，可單獨測試
- `internal/app/` — 應用狀態（TaskManager、SessionManager、RPGState）

### 可測試性

所有 `skill/` 函式只接受基本型別或標準函式庫型別，無 Telegram 依賴：

```go
// 正確：skill 層可直接測試
func FetchWeather(city string) (string, error) { ... }

// Handler 層包裝
func (h *Hub) HandleWeatherBtn(c tele.Context) error {
    result, err := skill.FetchWeather(query)
    c.Send(result)
}
```

### 動態選單

主選單根據 `Session.EnabledExtras` 動態生成，每個按鈕都對應一個 `ExtraFeature` 定義，開關統一管理：

```go
type ExtraFeature struct {
    ID      string
    Label   string
    Handler func(*Hub) tele.HandlerFunc
}
```

---

## 安全機制

### 白名單認證（隱形模式）

所有 Telegram 更新先通過 `AuthMiddleware`：
- 只有 `ALLOWED_USER_IDS` 中的用戶可操作
- 未授權者**完全無回應**（不暴露 Bot 存在）
- 每次請求記錄 Log（含 User ID 與 Username）

```go
func AuthMiddleware(allowedIDs map[int64]bool) tele.MiddlewareFunc {
    return func(next tele.HandlerFunc) tele.HandlerFunc {
        return func(c tele.Context) error {
            if !allowedIDs[c.Sender().ID] {
                log.Warn("unauthorized", "id", c.Sender().ID)
                return nil  // 隱形：不回應
            }
            return next(c)
        }
    }
}
```

### 沙箱檔案系統

所有檔案操作通過 `resolveAndValidate()`：

```go
func resolveAndValidate(workspace, relPath string) (string, error) {
    cleaned := filepath.Clean("/" + relPath)        // 移除 ..
    full := filepath.Join(workspace, cleaned)
    if !strings.HasPrefix(full, workspace+"/") {    // 防止逃逸
        return "", errors.New("path escape blocked")
    }
    return full, nil
}
```

### 三級危險偵測

`safety.go` 對 Shell 指令進行分級：

| 等級 | 處理方式 | 範例 |
|------|----------|------|
| ⛔ 封鎖 | 直接拒絕 | `rm -rf /`, `mkfs`, Fork bomb |
| ⚠️ 警告 | 二次確認（紅色） | `rm`, `sudo`, `git push -f` |
| ✅ 安全 | 一般確認 | `ls`, `cat`, `go build` |

### Graceful Shutdown

```go
select {
case sig := <-sigCh:         // SIGINT / SIGTERM
    // 正常關閉流程
case <-hub.RestartCh:        // 自我升級觸發
    // syscall.Exec 熱重啟
}
```

---

## 並發模型

### 任務槽位（TaskManager）

每個用戶同時只有一個任務槽位，防止並發衝突：

```go
type TaskManager struct {
    mu     sync.Mutex
    cancel context.CancelFunc
}

func (tm *TaskManager) Start(fn func(ctx context.Context)) bool {
    tm.mu.Lock()
    defer tm.mu.Unlock()
    if tm.cancel != nil {
        return false  // 已有任務進行中
    }
    ctx, cancel := context.WithCancel(context.Background())
    tm.cancel = cancel
    go func() {
        defer tm.Done()
        fn(ctx)
    }()
    return true
}
```

### Panic Recovery

所有任務 goroutine 都有 panic recovery：

```go
defer func() {
    if r := recover(); r != nil {
        log.Error("task panic", "error", r, "stack", debug.Stack())
        bot.Send(user, "❌ 任務發生嚴重錯誤，已自動恢復")
    }
}()
```

---

## 開發指南

### 新增技能

1. 在 `internal/bot/skill/` 建立新檔案，實作純函式
2. 在 `internal/bot/handler/menu.go` 的 `ExtraFeatures` 中新增定義
3. 在 `internal/bot/handler/callback.go` 實作對應的 Handler
4. 在 `internal/app/rpg.go` 的 `skillDefs` 中新增 XP 定義（選用）

### 新增測試

```bash
# 執行測試
go test -race ./internal/...

# 覆蓋率報告
go test -coverprofile=cover.out ./internal/app/ ./internal/bot/skill/
go tool cover -html=cover.out
```

### 本地開發流程

```bash
# 1. 編譯
go build -race -o axle ./cmd/axle

# 2. 靜態分析
go vet ./...

# 3. 執行（Dev 模式，顯示詳細 Log）
LOG_LEVEL=debug ./axle

# 4. 測試 RPG Dashboard
open http://localhost:8080
```

### Email 設定

在 `~/.axle/credentials.json` 中新增：

```json
{
  "email_address": "your@gmail.com",
  "email_password": "your_app_password",
  "smtp_host": "smtp.gmail.com",
  "smtp_port": "587",
  "imap_host": "imap.gmail.com",
  "imap_port": "993"
}
```

> Gmail 用戶需建立 [App Password](https://myaccount.google.com/apppasswords)

---

[← 返回首頁](../README.md) | [完整教學](tutorial-zh.md) | [功能參考](features-zh.md)
