# Axle 完整教學指南

[← 返回首頁](../README.md) | [English](tutorial-en.md)

---

## 目錄

- [一、新手入門](#一新手入門)
- [二、基本操作](#二基本操作)
- [三、進階使用](#三進階使用)
- [四、自我升級](#四自我升級)

---

## 一、新手入門

### 1.1 取得 Telegram Bot Token

1. 在 Telegram 搜尋 **[@BotFather](https://t.me/BotFather)**
2. 傳送 `/newbot`，依指示填入名稱與 Username（Username 必須以 `bot` 結尾）
3. BotFather 回傳一串 Token（格式：`1234567890:ABCdef...`）— **妥善保管**

### 1.2 取得你的 Telegram User ID

1. 在 Telegram 搜尋 **[@userinfobot](https://t.me/userinfobot)**
2. 傳送任意訊息，Bot 會回傳你的數字 User ID

### 1.3 安裝 Copilot CLI

Axle 的 AI 功能依賴 GitHub Copilot CLI：

```bash
# macOS
brew install gh
gh auth login         # 登入 GitHub 帳號
brew install copilot  # 安裝 Copilot CLI

# 驗證
copilot --version
```

> 需要有效的 GitHub Copilot 訂閱（個人版或企業版）

### 1.4 安裝與啟動 Axle

```bash
# Clone 專案
git clone https://github.com/Gary-GaGa/axle.git
cd axle

# 編譯
go build -o axle ./cmd/axle

# 啟動
./axle
```

**首次設定流程：**

```
? 請輸入 Telegram Bot Token: 1234567890:ABCdef...
? 請輸入允許的 Telegram User ID（逗號分隔多個）: 98765432
✅ 設定已儲存至 ~/.axle/credentials.json
🚀 Axle v0.10.0 已啟動！
```

設定完成後，前往 Telegram 找到你的 Bot，傳送 `/start`。

### 1.5 第一次對話

傳送 `/start` 後，Bot 會回傳主選單：

```
⚔️ Axle 引擎已啟動 v0.10.0
模式：單兵作戰 | 模型：claude-sonnet-4.6

[📁 讀取代碼]  [✏️ 寫入檔案]
[🤖 Copilot]   [⚡ 執行指令]
[🔄 切換模型]  [📂 切換專案]
[📊 狀態]      [➕ 更多功能]
```

點擊「🤖 Copilot」，輸入「請問 Go 的 interface 是什麼？」，即可開始 AI 對話。

---

## 二、基本操作

### 2.1 讀取代碼

1. 點擊「📁 讀取代碼」
2. 輸入相對路徑（相對於當前 workspace），如 `main.go` 或 `internal/app/session.go`
3. Bot 回傳格式化的程式碼（最大 30 KB）

> 💡 使用「📂 目錄瀏覽」擴充功能可先查看目錄結構

### 2.2 執行 Shell 指令

1. 點擊「⚡ 執行指令」
2. 輸入指令，如 `go test ./...`
3. Bot 顯示危險等級與確認按鈕：
   - ✅ 安全指令 → 確認後執行
   - ⚠️ 警告指令 → 需二次確認
   - ⛔ 封鎖指令 → 直接拒絕（無法確認）
4. 點擊「✅ 確認執行」後，Bot 實時回傳輸出

### 2.3 切換 AI 模型

1. 點擊「🔄 切換模型」
2. 先選廠商（Anthropic / OpenAI / Google）
3. 再選具體模型（含費率資訊）

**推薦選擇：**

| 場景 | 推薦模型 |
|------|----------|
| 日常對話 | claude-sonnet-4.6（1x，均衡） |
| 複雜架構 | claude-opus-4.5（3x，強推理） |
| 快速問答 | claude-haiku-4.5（0.25x，速度快） |
| 省費率 | gpt-4.1（免費） |

### 2.4 擴充功能管理

點擊「➕ 更多功能」進入擴充功能清單：

```
✅ 📂 目錄瀏覽      ⬜ 🔎 搜尋代碼
⬜ 🔍 Web 搜尋      ⬜ 🌐 Web 擷取
✅ 🔀 Git 操作      ⬜ 🐙 GitHub
⬜ 📧 Email         ⬜ 📅 行事曆
...
```

點擊切換開關，返回主選單後啟用的功能按鈕會出現在主選單中。

### 2.5 切換工作目錄

1. 點擊「📂 切換專案」
2. 輸入絕對路徑，如 `/Users/gary/myproject`
3. 確認後所有操作都在新目錄下執行
4. 輸入 `reset` 恢復預設目錄

---

## 三、進階使用

### 3.1 Git 操作

啟用「🔀 Git 操作」後：

```
[📊 Status]  [📝 Diff]
[📋 Log]     [🚀 Commit & Push]
```

**Commit & Push 流程：**
1. 點擊「🚀 Commit & Push」
2. 輸入 commit message
3. 確認後自動執行 `git add -A && git commit -m "..." && git push`

### 3.2 GitHub 整合

啟用「🐙 GitHub」後，需確認 `gh auth status` 已登入：

```bash
gh auth login  # 若未登入
gh auth status # 確認登入狀態
```

功能包含：PR 列表、Issue 列表、CI 狀態、建立 PR。

### 3.3 排程任務

啟用「⏰ 排程任務」後：

1. 點擊「➕ 新增排程」
2. 輸入 Cron 表達式（如 `0 9 * * 1-5` = 週一到週五早上 9 點）
3. 輸入執行指令（如 `git status`）或 `@briefing`（自動每日簡報）

**常用 Cron 表達式：**

```
0 9 * * 1-5     每週一到週五早上 9 點
*/30 * * * *    每 30 分鐘
0 18 * * *      每天下午 6 點
0 0 * * 0       每週日午夜
```

排程持久化至 `~/.axle/schedules.json`，重啟後繼續執行。

### 3.4 子代理

啟用「👥 子代理」後：

1. 輸入自然語言描述任務（如「幫我分析這個 repo 的代碼品質」）
2. Axle 建立獨立子代理執行該任務
3. 子代理完成後回報結果
4. 點擊「📋 列出代理」可查看所有子代理狀態

### 3.5 插件系統

啟用「🧩 擴充技能」後：

1. 首次使用自動建立範例插件至 `~/.axle/plugins/`
2. 按照 YAML 格式新增自定義技能：

```yaml
# ~/.axle/plugins/myskill.yaml
name: my_tool
description: 我的自定義工具
type: shell  # 或 copilot
command: echo "Hello from plugin"
```

### 3.6 每日簡報

啟用「📢 每日簡報」後：

- 手動：點擊「📢 每日簡報」
- 自動：建立排程指令填 `@briefing`

簡報內容：系統狀態 + Git 狀態 + 今日行事曆 + 磁碟使用量。

### 3.7 RPG Dashboard

啟動後自動開啟瀏覽器至 `http://localhost:8080`：

- 每次執行技能都獲得 XP（Shell Strike +15 · Copilot Summon +25）
- 等級自動成長，觸發升級動畫
- 技能使用次數統計，可作為個人工作報告

---

## 四、自我升級

> ⚠️ 此功能會修改 Axle 原始碼並重新編譯，請在版本控制下使用

### 4.1 啟用自我升級

在「➕ 更多功能」中啟用「🔧 自我升級」。

### 4.2 描述需求

1. 點擊「🔧 自我升級」
2. 用自然語言描述想要的功能：
   ```
   例：加一個「天氣查詢」功能，輸入城市名，透過 wttr.in API 回傳天氣資訊
   ```
3. Axle 透過 Copilot CLI 分析原始碼，產出升級計畫

### 4.3 確認計畫

計畫顯示後，Bot 詢問確認：

```
📋 升級計畫：
1. 新增 internal/bot/skill/weather.go — FetchWeather(city) 函式
2. 新增 WeatherSkill 至擴充功能清單
3. 在 callback.go 中加入 HandleWeatherBtn

[✅ 確認執行]  [❌ 取消]
```

### 4.4 自動執行流程

確認後自動進行：

```
[1/5] 🔧 備份 binary → axle.bak
[2/5] 💻 修改代碼（Copilot CLI 執行）
[3/5] 📦 go build（編譯驗證）
[4/5] 🧪 go test（測試驗證）
[5/5] 🔀 git commit → 版本號 v0.10.0 → v0.10.1
⚡ 重啟中...
```

### 4.5 失敗回滾

若編譯或測試失敗：
- 自動從 `axle.bak` 回滾至舊版
- Bot 回報失敗原因
- 版本號不遞增

---

## 📌 快速參考

### 指令

| 指令 | 說明 |
|------|------|
| `/start` | 顯示主選單 |
| `/cancel` | 取消執行中的任務 |

### 儲存路徑

| 路徑 | 說明 |
|------|------|
| `~/.axle/credentials.json` | Token · User ID · Email 設定 |
| `~/.axle/rpg_state.json` | RPG 等級與 XP |
| `~/.axle/schedules.json` | 排程任務設定 |
| `~/.axle/plugins/` | 自定義插件 |
| `~/.axle/memory/` | AI 記憶體 |

### 限制值

| 項目 | 限制 |
|------|------|
| 同時任務數 | 1 |
| Shell 執行超時 | 60 秒 |
| Copilot 超時 | 5 分鐘 |
| 檔案讀取上限 | 30 KB |
| Shell 輸出上限 | 1 MB |

---

[← 返回首頁](../README.md) | [完整功能參考](features-zh.md) | [架構指南](architecture.md)
