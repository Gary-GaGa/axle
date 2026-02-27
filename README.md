# Axle

> 🔧 安全、本地的 Golang AI Agent — 透過 Telegram 控制你的開發工具。

Axle 是一個運行在本機的 Telegram Bot，讓你隨時隨地透過手機或電腦的 Telegram 客戶端操控開發環境中的工具：讀寫代碼、執行指令、與 GitHub Copilot CLI 對話、搜尋網頁等。

## ✨ 功能總覽

| 功能 | 說明 |
|------|------|
| 📁 讀取代碼 | 讀取 workspace 內的檔案並格式化顯示 |
| ✏️ 寫入檔案 | 在 workspace 內建立或覆寫檔案（含覆寫警告） |
| ⚡ 執行指令 | 在 workspace 中執行 Shell 指令（需確認，含危險偵測） |
| 🤖 Copilot 助手 | 與 GitHub Copilot CLI 持續對話，支援即時切換模型 |
| 🔍 Web 搜尋 | 透過 DuckDuckGo 搜尋（無需 API Key） |
| 🌐 Web 擷取 | 擷取任意 URL 的文字內容 |
| 🔄 切換模型 | 全局切換 Copilot 模型（兩步式：廠商 → 模型，含費率顯示） |
| 📂 切換專案 | 動態切換工作目錄，支援多專案同時開發 |
| 📊 系統狀態 | 查看任務狀態、選定模型、Workspace 路徑 |
| 🛑 取消任務 | 中斷正在執行的任務 |

## 🚀 快速開始

### 前置需求

- **Go 1.22+**
- **Telegram Bot Token**（從 [@BotFather](https://t.me/BotFather) 取得）
- **GitHub Copilot CLI**（`copilot` 指令需可在 PATH 中找到）
- macOS / Linux

### 安裝與啟動

```bash
# 1. Clone 並進入專案
git clone https://github.com/garyellow/axle.git
cd axle

# 2. 編譯
go build -o axle ./cmd/axle

# 3. 啟動（首次會互動式設定 Token）
./axle
```

首次執行時，Axle 會在終端機互動式詢問：

1. **Telegram Bot Token**（必填）
2. **允許的 Telegram User ID**（必填，安全白名單）
3. **OpenAI API Key**（選填）

設定完成後憑證會儲存至 `~/.axle/credentials.json`（權限 `0600`），下次啟動自動載入。

### 環境變數（進階）

你也可以透過 `.env` 檔案或環境變數來設定：

```bash
cp .env.example .env
vim .env  # 填入 TELEGRAM_TOKEN, ALLOWED_USER_IDS 等
```

設定優先順序（高→低）：

1. 環境變數
2. `.env` 檔案
3. `~/.axle/credentials.json`
4. 互動式終端提示

## 🏗️ 架構設計

```
axle/
├── cmd/axle/                  # 程式進入點
│   └── main.go                # 啟動流程、Handler 註冊、Graceful Shutdown
├── configs/                   # 設定模組
│   ├── config.go              # Viper 多層設定載入
│   ├── store.go               # ~/.axle/credentials.json 持久化
│   └── prompt.go              # 終端互動式輸入
├── internal/
│   ├── app/                   # 應用核心
│   │   ├── taskmanager.go     # 單任務槽位 + Context 取消
│   │   └── session.go         # 每用戶狀態機（Mode 路由）
│   └── bot/
│       ├── handler/           # Telegram Handler 層
│       │   ├── hub.go         # Hub 結構（共用依賴 + 任務啟動器 + workspace helper）
│       │   ├── menu.go        # 所有 InlineKeyboard 定義（含兩步式模型選擇）
│       │   ├── start.go       # /start 指令
│       │   ├── text.go        # 文字訊息路由器（含專案切換流程）
│       │   └── callback.go    # 按鈕回呼處理器
│       ├── middleware/
│       │   └── auth.go        # 白名單 Auth 中間件（隱形模式）
│       └── skill/             # 技能模組（可獨立測試）
│           ├── models.go      # Copilot 模型清單 + 費率 + 廠商分類
│           ├── readcode.go    # 沙箱檔案讀取
│           ├── writefile.go   # 沙箱檔案寫入 + resolveAndValidate 共用函式
│           ├── exec.go        # Shell 執行（限時 + 輸出上限 1MB）
│           ├── copilot.go     # Copilot CLI 呼叫器（5 分鐘超時）
│           ├── safety.go      # 危險指令偵測（3 級制）
│           └── web.go         # DuckDuckGo 搜尋 + URL 擷取
├── docs/                      # 文件
├── scripts/                   # 輔助腳本
├── .env.example               # 環境變數範本
├── .gitignore
├── go.mod
└── go.sum
```

### 設計原則

- **Clean Architecture**：`cmd` → `internal/app` → `internal/bot` 分層明確
- **單一職責**：每個 `skill` 是獨立的純函式，不依賴 Telegram 框架
- **併發安全**：所有共享狀態（TaskManager、SessionManager）使用 `sync.Mutex` / `sync.RWMutex`
- **可測試性**：Skill 層僅依賴標準函式庫，可直接單元測試

## 🔒 安全機制

### 1. 白名單認證（Stealth Mode）

所有 Telegram 更新都先經過 `AuthMiddleware`：
- 只有 `ALLOWED_USER_IDS` 中的用戶可以操作
- 未授權者**不會收到任何回應**（隱形模式），避免暴露 Bot 存在
- 每次請求都記錄 Log（含 User ID 與 Username）

### 2. 沙箱檔案系統

所有檔案操作（讀取、寫入、存在檢查）都通過 `resolveAndValidate()`：
- `filepath.Clean("/"+relPath)` 正規化路徑，移除 `..` 等元素
- `filepath.Join(absWorkspace, cleaned)` + prefix 檢查確保路徑在 workspace 內
- 任何路徑逃逸嘗試都被拒絕並記錄

### 3. 危險指令偵測（三級制）

| 等級 | 處理方式 | 範例 |
|------|----------|------|
| ⛔ 封鎖 | 直接拒絕，不可執行 | `rm -rf /`, `mkfs`, `dd of=/dev/`, Fork bomb |
| ⚠️ 警告 | 需二次確認（紅色警告） | `rm`, `sudo`, `git push -f`, `chmod`, `DROP TABLE` |
| ✅ 安全 | 一般確認即可 | `ls`, `cat`, `go build` 等 |

### 4. Human-in-the-Loop

所有 Shell 指令執行前都需用戶按鈕確認。危險指令需額外的二次確認。

### 5. 憑證安全

- `~/.axle/credentials.json` 權限為 `0600`（僅擁有者可讀寫）
- 目錄權限 `0700`
- 敏感資訊不在 Log 中輸出

## 📂 動態工作目錄

Axle 支援在運行時動態切換工作目錄（Workspace），讓你在不重啟服務的情況下操作不同專案：

1. 點擊主選單「📂 切換專案」
2. 輸入目標目錄的**絕對路徑**（如 `/Users/gary/myproject`）
3. 系統驗證路徑存在且為目錄後生效
4. 輸入 `reset` 可恢復為預設工作目錄

切換後，所有操作（讀取、寫入、Shell 執行、Copilot CLI）都會在新目錄下執行。每位用戶的工作目錄獨立設定，互不影響。

## 📱 使用方式

### 基本流程

1. 在 Telegram 中對 Bot 發送 `/start`
2. Bot 顯示主選單（Inline Keyboard）
3. 點擊按鈕選擇功能
4. 依照提示輸入參數
5. 等待結果回傳

### 指令

| 指令 | 說明 |
|------|------|
| `/start` | 顯示主選單與任務狀態 |
| `/cancel` | 取消目前執行中的任務 |

### Copilot 對話模式

1. 點擊「🤖 Copilot 助手」進入對話模式
2. 直接輸入問題或任務描述
3. Bot 回覆後可繼續對話（持續會話）
4. 隨時可用「🔄 切換模型」更換 AI 模型
5. 按「⬅️ 返回主選單」離開對話模式

### 模型選擇（兩步式）

選擇模型時先選廠商（Anthropic / OpenAI / Google），再從該廠商的模型列表中選取。
支援 17 種 Copilot 模型，含費率資訊：

| 類別 | 模型 | 費率 |
|------|------|------|
| 免費 | gpt-5-mini, gpt-4.1 | 免費 |
| 低倍 | claude-haiku-4.5 | 0.25x |
| 標準 | claude-sonnet-4.6, claude-sonnet-4.5, claude-sonnet-4, gpt-5.1, gpt-5.2, gemini-3-pro-preview | 1x |
| 中倍 | gpt-5.1-codex | 1.5x |
| 高倍 | claude-opus-4.6, claude-opus-4.5, gpt-5.3-codex, gpt-5.2-codex, gpt-5.1-codex-max | 3x |
| 旗艦 | claude-opus-4.6-fast | 33x |

## ⚙️ 防護與限制

| 項目 | 限制值 | 說明 |
|------|--------|------|
| 同時任務數 | 1 | 防止資源競爭，保證穩定 |
| Shell 執行超時 | 60 秒 | 防止指令永遠掛住 |
| Copilot 超時 | 5 分鐘 | 防止 CLI 程序無回應 |
| Prompt 長度上限 | 8,000 字元 | 防止 Context 溢出 |
| Telegram 訊息分段 | 4,000 字元/段 | 超長回應自動分段發送 |
| 檔案讀取上限 | 30 KB | 避免 Telegram 訊息過大 |
| 檔案寫入上限 | 1 MB | 防止磁碟濫用 |
| Shell 輸出上限 | 1 MB | 防止記憶體耗盡（OOM） |
| Web 擷取上限 | 100 KB | 限制網頁下載量 |
| 分段發送間隔 | 300 ms | 避免觸發 Telegram API 限流 |

## 🔧 Graceful Shutdown

當服務收到終止信號（`SIGINT` / `SIGTERM`）時：

1. 取消所有正在執行的任務
2. 向所有白名單用戶發送離線通知
3. 停止 Telegram Bot

啟動時也會向所有白名單用戶發送上線通知。

## 🛡️ 錯誤處理

- **Task goroutine panic**：自動 `recover()`，記錄完整 stack trace，通知用戶
- **Copilot CLI 掛起**：5 分鐘後自動超時終止
- **Shell 輸出過大**：截斷至 1 MB，附加截斷提示
- **網路異常**：Web 操作有 30 秒超時，失敗時顯示錯誤訊息
- **憑證遺失**：下次啟動時自動偵測並重新提示輸入

## 📝 開發

```bash
# 編譯（含 race detector）
go build -race -o axle ./cmd/axle

# 靜態分析
go vet ./...

# 直接運行
go run ./cmd/axle
```

## 🗺️ Roadmap

- [ ] 子代理委派系統（Sub-Agent Swarm）
- [ ] 自我演進（Self-Evolution）— 透過對話自行修改代碼
- [ ] LINE 通訊管道支援
- [ ] 單元測試覆蓋率提升
- [ ] 多用戶各自獨立任務槽位

## License

MIT
