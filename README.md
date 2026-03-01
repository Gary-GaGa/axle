# Axle

> 🔧 安全、本地的 Golang AI Agent — 透過 Telegram 控制你的開發工具。

Axle 是一個運行在本機的 Telegram Bot，讓你隨時隨地透過手機或電腦的 Telegram 客戶端操控開發環境中的工具：讀寫代碼、執行指令、與 GitHub Copilot CLI 對話、搜尋網頁、管理 Git 倉庫等。所有 AI 功能透過 GitHub Copilot CLI 驅動，無需額外 API Key。

## ✨ 功能總覽

### 預設功能（主選單常駐）

| 功能 | 說明 |
|------|------|
| 📁 讀取代碼 | 讀取 workspace 內的檔案並格式化顯示 |
| ✏️ 寫入檔案 | 在 workspace 內建立或覆寫檔案（含覆寫警告） |
| 🤖 Copilot 助手 | 與 GitHub Copilot CLI 持續對話，支援即時切換模型 |
| ⚡ 執行指令 | 在 workspace 中執行 Shell 指令（需確認，含危險偵測） |
| 🔄 切換模型 | 全局切換 Copilot 模型（兩步式：廠商 → 模型，含費率顯示） |
| 📂 切換專案 | 動態切換工作目錄，支援多專案同時開發 |
| 📊 系統狀態 | 查看任務狀態、選定模型、Workspace 路徑 |
| 🛑 取消任務 | 中斷正在執行的任務 |

### 擴充功能（可自行啟用，釘選至主選單）

透過「➕ 更多功能」開關切換，啟用後會動態加入主選單：

| 功能 | 說明 |
|------|------|
| 📂 目錄瀏覽 | 瀏覽 workspace 內的目錄結構 |
| 🔎 搜尋代碼 | 在 workspace 中以關鍵字搜尋檔案內容 |
| 🔍 Web 搜尋 | 透過 DuckDuckGo 搜尋（無需 API Key） |
| 🌐 Web 擷取 | 擷取任意 URL 的文字內容 |
| 🔀 Git 操作 | 查看 status / diff / log，一鍵 commit & push |
| 🐙 GitHub | PR / Issue 管理、CI 狀態、建立 PR（透過 `gh` CLI） |
| 📧 Email | 發送/讀取 Email（SMTP/IMAP） |
| 📅 行事曆 | 查看今日/明日/本週行程（macOS Calendar） |
| 📢 每日簡報 | 一鍵產生系統狀態 + Git + 行事曆 + 磁碟簡報 |
| 📄 PDF 處理 | 上傳 PDF → 文字提取 → AI 摘要 |
| 🖼 圖片分析 | 上傳圖片 → 尺寸/格式分析 → 儲存至 workspace |
| 🧩 擴充技能 | 載入自定義 YAML 技能插件 |
| 👥 子代理 | 委派子代理執行獨立任務，含清單管理 |
| ⏰ 排程任務 | Cron 排程自動執行 Shell 指令 |
| 🎮 RPG Dashboard | Web 即時儀表板，像素風格顯示 Agent 工作狀態（localhost:8080） |

## 🚀 快速開始

### 前置需求

- **Go 1.22+**
- **Telegram Bot Token**（從 [@BotFather](https://t.me/BotFather) 取得）
- **GitHub Copilot CLI**（`copilot` 指令需可在 PATH 中找到）
- **GitHub CLI**（選用，`gh` 指令，用於 GitHub 功能）
- macOS / Linux

### 安裝與啟動

```bash
# 1. Clone 並進入專案
git clone https://github.com/Gary-GaGa/axle.git
cd axle

# 2. 編譯
go build -o axle ./cmd/axle

# 3. 啟動（首次會互動式設定 Token）
./axle
```

首次執行時，Axle 會在終端機互動式詢問：

1. **Telegram Bot Token**（必填）
2. **允許的 Telegram User ID**（必填，安全白名單）

設定完成後憑證會儲存至 `~/.axle/credentials.json`（權限 `0600`），下次啟動自動載入。若憑證遺失會自動提示重新輸入。

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
│   │   ├── session.go         # 每用戶狀態機（Mode 路由 + 擴充功能偏好）
│   │   ├── plugin.go          # YAML 技能插件管理器
│   │   ├── scheduler.go       # Cron 排程引擎
│   │   └── rpg.go             # RPG 狀態引擎（XP/等級/成就/事件匯流排）
│   ├── web/                   # RPG Dashboard Web 服務
│   │   ├── server.go          # HTTP + WebSocket 伺服器
│   │   └── static/
│   │       └── index.html     # 像素風 RPG 儀表板（單頁應用）
│   └── bot/
│       ├── handler/           # Telegram Handler 層
│       │   ├── hub.go         # Hub 結構（共用依賴 + 動態選單產生器）
│       │   ├── menu.go        # 動態主選單 + 擴充功能切換 + 各子選單
│       │   ├── start.go       # /start 指令
│       │   ├── text.go        # 文字訊息路由器（含專案切換、子代理建立）
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
│           ├── web.go         # DuckDuckGo 搜尋 + URL 擷取
│           ├── listdir.go     # 目錄瀏覽（樹狀結構）
│           ├── search.go      # 代碼搜尋（grep 風格）
│           ├── git.go         # Git 操作（status / diff / log / commit+push）
│           ├── github.go      # GitHub API 操作（透過 gh CLI）
│           ├── email.go       # Email 發送（SMTP）+ 讀取（IMAP）
│           ├── calendar.go   # macOS 行事曆整合（icalBuddy / AppleScript）
│           ├── briefing.go   # 每日簡報產生器
│           ├── pdf.go         # PDF 文字提取
│           └── image.go       # 圖片分析（尺寸/格式/元資料）
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
- **動態選單**：主選單根據用戶啟用的擴充功能動態生成
- **自然語言路由**：閒置狀態下輸入文字自動進入 AI 對話，無需手動選擇
- **串流回應**：Copilot 回覆即時更新至同一則訊息（每 1.5 秒更新一次）

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

## 🎮 RPG Dashboard

啟動 Axle 後，自動在 `http://localhost:8080` 開啟像素風格的 RPG 儀表板：

- **角色面板**：顯示 Agent 等級、XP 進度條、稱號、裝備欄
- **任務卷軸**：即時顯示最近執行的任務與狀態
- **戰鬥日誌**：每次 Skill 執行都會記錄，含 XP 獎勵
- **技能統計**：各技能使用次數排行與視覺化
- **成就系統**：完成里程碑解鎖徽章

### 等級系統

| 等級 | 稱號 | XP |
|------|------|-----|
| 1-5 | 🟤 見習編碼師 | 0-100 |
| 6-10 | 🟢 程式碼遊俠 | 101-500 |
| 11-20 | 🔵 數據法師 | 501-2000 |
| 21-35 | 🟣 架構術士 | 2001-5000 |
| 36-50 | 🟡 傳奇工匠 | 5001-10000 |
| 50+ | 🔴 不朽引擎 | 10000+ |

XP 累計持久化至 `~/.axle/rpg_state.json`，重啟不歸零。透過 WebSocket 即時推送事件到瀏覽器。

## 📱 使用方式

### 基本流程

1. 在 Telegram 中對 Bot 發送 `/start`
2. Bot 顯示主選單（Inline Keyboard）
3. 點擊按鈕選擇功能（或透過「➕ 更多功能」啟用額外功能）
4. 依照提示輸入參數
5. 等待結果回傳

> 💡 **自然語言路由**：在主選單狀態下，直接輸入文字即自動進入 Copilot 對話模式（AI 會解讀你的需求並執行）。不需先按按鈕。

### 指令

| 指令 | 說明 |
|------|------|
| `/start` | 顯示主選單與任務狀態 |
| `/cancel` | 取消目前執行中的任務 |

### 擴充功能切換

1. 點擊主選單「➕ 更多功能」
2. 看到所有可選功能的開關清單（✅ 已啟用 / ⬜ 未啟用）
3. 點擊切換開關
4. 返回主選單後，已啟用功能的按鈕會出現在選單中

### Copilot 對話模式

1. 點擊「🤖 Copilot 助手」進入對話模式（或直接輸入文字自動進入）
2. 直接輸入問題或任務描述
3. **串流回應**：AI 回覆即時更新至同一則訊息，不用等全部完成
4. Bot 回覆後可繼續對話（持續會話）
5. 隨時可用「🔄 切換模型」更換 AI 模型
6. 按「⬅️ 返回主選單」離開對話模式

### 模型選擇（兩步式）

選擇模型時先選廠商（Anthropic / OpenAI / Google），再從該廠商的模型列表中選取。
支援 17 種 Copilot 模型，含費率資訊：

| 類別 | 模型 | 費率 |
|------|------|------|
| 免費 | gpt-5-mini, gpt-4.1 | 免費 |
| 低倍 | claude-haiku-4.5, gpt-5.1-codex-mini | 0.25x ~ 0.5x |
| 標準 | claude-sonnet-4.6, claude-sonnet-4.5, claude-sonnet-4, gpt-5.1, gpt-5.2, gemini-3-pro-preview | 1x |
| 中倍 | gpt-5.1-codex | 1.5x |
| 高倍 | claude-opus-4.6, claude-opus-4.5, gpt-5.3-codex, gpt-5.2-codex, gpt-5.1-codex-max | 3x |
| 旗艦 | claude-opus-4.6-fast | 33x |

### Git 操作

啟用「🔀 Git 操作」擴充後，可進行：
- **Status** — 查看工作區狀態
- **Diff** — 查看未暫存的變更
- **Diff (Staged)** — 查看已暫存的變更
- **Log** — 查看最近 10 筆提交記錄
- **Commit & Push** — 一鍵提交並推送（需確認）

### 排程任務

啟用「⏰ 排程任務」擴充後，可建立 Cron 排程：
- 支援標準 Cron 表達式（如 `*/5 * * * *`）
- 排程持久化至 `~/.axle/schedules.json`
- 可啟用/停用/刪除個別排程

### 擴充技能（插件）

啟用「🧩 擴充技能」擴充後，可載入 YAML 定義的自定義技能：
- 技能檔案放置於 `~/.axle/plugins/`
- 支援 shell 類型（執行指令）和 copilot 類型（AI 對話）
- 首次使用會自動建立範例插件

### GitHub 整合

啟用「🐙 GitHub」擴充後，可透過 `gh` CLI 進行：
- **PR 列表** — 查看 Open PR
- **Issue 列表** — 查看 Open Issues
- **CI 狀態** — 查看最近的 Workflow Runs
- **Repo 資訊** — 查看 Repository 詳情
- **建立 PR** — 互動式填寫 Title / Body 建立 Pull Request

### Email 整合

啟用「📧 Email」擴充後，可收發 Email：
- **發送** — 互動式填寫收件人 / 主旨 / 內文
- **讀取** — 查看信箱最近 5 封郵件標題

設定方式（環境變數或 `~/.axle/credentials.json`）：
```
EMAIL_ADDRESS=your@gmail.com
EMAIL_PASSWORD=your_app_password
SMTP_HOST=smtp.gmail.com   # 預設
SMTP_PORT=587               # 預設
IMAP_HOST=imap.gmail.com   # 預設
IMAP_PORT=993               # 預設
```
> Gmail 使用者需建立 [App Password](https://myaccount.google.com/apppasswords)

### PDF 文件處理

直接在 Telegram 上傳 PDF 檔案即可使用：
1. 上傳 PDF → 自動提取文字內容（最大 30KB）
2. 點擊「📝 AI 摘要」→ 透過 Copilot 產生繁中摘要

### 圖片分析

直接在 Telegram 傳送圖片即可：
- 自動分析尺寸、格式、檔案大小
- 圖片自動儲存至 workspace

### 行事曆

啟用「📅 行事曆」擴充後，可查看 macOS Calendar 事件：
- **今日行程** / **明日行程** / **本週行程**
- 優先使用 `icalBuddy`（`brew install ical-buddy`），自動降級至 AppleScript

### 每日簡報

啟用「📢 每日簡報」擴充後，一鍵產生綜合報告：
- 🖥 系統狀態（Go 版本、OS、Goroutines）
- 🔀 Git 狀態（當前 workspace 的 git status）
- 📅 今日行事曆
- 💾 磁碟使用量

> 💡 可搭配排程自動執行：建立排程時指令填 `@briefing`，即可定時自動發送簡報。

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

# 執行測試（含 race detector）
go test -race ./internal/...

# 直接運行
go run ./cmd/axle
```

## 🗺️ Roadmap

- [ ] 自我演進（Self-Evolution）— 透過對話自行修改代碼
- [ ] LINE 通訊管道支援
- [ ] Memory / Context 持久化
- [ ] 多用戶各自獨立任務槽位
- [ ] 圖片生成與分析

## License

MIT
