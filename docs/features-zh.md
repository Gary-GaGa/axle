# Axle 完整功能參考

[← 返回首頁](../README.md) | [English](../README.en.md)

---

## 目錄

- [預設功能](#預設功能)
- [擴充功能](#擴充功能)
- [特色功能](#特色功能)
- [模型清單](#模型清單)
- [防護限制](#防護限制)

---

## 預設功能

開箱即用，無需額外設定。

| 功能 | 說明 |
|------|------|
| 📁 **讀取代碼** | 讀取 workspace 內的檔案並格式化顯示（最大 30 KB） |
| ✏️ **寫入檔案** | 在 workspace 內建立或覆寫檔案（含覆寫警告） |
| 🤖 **Copilot 助手** | 與 GitHub Copilot CLI 持續對話，支援串流即時回應 |
| ⚡ **執行指令** | 在 workspace 中執行 Shell 指令（需確認，含危險偵測） |
| 🔄 **切換模型** | 兩步式選擇：廠商 → 模型（含費率顯示） |
| 📂 **切換專案** | 動態切換工作目錄，多專案同時開發 |
| 📊 **系統狀態** | 查看任務狀態、選定模型、Workspace 路徑、版本號 |
| 🛑 **取消任務** | 中斷正在執行的任務 |

---

## 擴充功能

透過「➕ 更多功能」開關切換，啟用後動態加入主選單。

### 檔案與代碼

| 功能 | 說明 |
|------|------|
| 📂 **目錄瀏覽** | 瀏覽 workspace 內的目錄樹狀結構 |
| 🔎 **搜尋代碼** | 在 workspace 中以關鍵字搜尋檔案內容（ripgrep 風格） |

### Web

| 功能 | 說明 | 前置需求 |
|------|------|----------|
| 🔍 **Web 搜尋** | 透過 DuckDuckGo 搜尋（無需 API Key） | 無 |
| 🌐 **Web 擷取** | 擷取任意 URL 的文字內容（最大 100 KB） | 無 |
| 🌉 **Web Gateway** | 本地第二通道：Web Chat / 記憶搜尋 / Browser / 工作流 | 啟動 Axle 後使用 Token 登入 |

### 記憶與知識

| 功能 | 說明 |
|------|------|
| 🧠 **記憶 / 歷史** | 長期持久化聊天、工具輸出、工作流摘要；支援搜尋歷史 |
| 🧩 **RAG Recall** | 每次 Copilot 對話會自動補入近期上下文與相關長期記憶 |

### Git & GitHub

| 功能 | 說明 | 前置需求 |
|------|------|----------|
| 🔀 **Git 操作** | Status / Diff / Log / Commit & Push | `git` |
| 🐙 **GitHub** | PR · Issue · CI 狀態 · 建立 PR | `gh auth login` |

### 通訊

| 功能 | 說明 | 前置需求 |
|------|------|----------|
| 📧 **Email** | 發送（SMTP）/ 讀取（IMAP）| `EMAIL_*` 設定 |
| 📅 **行事曆** | 今日/明日/本週行程（macOS Calendar） | `ical-buddy` |
| 📢 **每日簡報** | 系統狀態 + Git + 行事曆 + 磁碟 | 無 |

### 媒體

| 功能 | 說明 |
|------|------|
| 📄 **PDF 處理** | 上傳 PDF → 文字提取 → AI 摘要 |
| 🖼 **圖片分析** | 上傳圖片 → 尺寸/格式分析 → 儲存至 workspace |

### 自動化

| 功能 | 說明 |
|------|------|
| ⏰ **排程任務** | Cron 排程自動執行 Shell 指令，持久化 |
| 👥 **子代理** | 委派子代理執行獨立任務，含清單管理 |
| 🧭 **背景工作流** | 自動規劃 2-4 步驟，背景執行並可列出 / 取消 / 追蹤結果 |
| 🧩 **擴充技能** | 載入 YAML 定義的自定義插件技能 |
| 🌐 **Browser 自動化** | 透過 Safari 腳本執行 `open / wait / extract / screenshot` |

### 視覺化

| 功能 | 說明 |
|------|------|
| 🎮 **RPG Dashboard** | 像素風 Web 儀表板，即時顯示工作狀態（localhost:8080） |

---

## 特色功能

### 🔧 自我升級

> 需啟用「🔧 自我升級」擴充功能

描述需求 → Copilot CLI 分析並規劃 → 人工確認 → 自動修改代碼 → 編譯 → 測試 → git commit → 重啟

安全機制：人工確認計畫 / 失敗自動回滾 / 版本號自動遞增

### 🧠 長期記憶 / 可搜尋歷史

Axle 會持久化保存：
- Telegram / Web Chat 對話
- Browser 擷取摘要
- Shell / 子代理 / 工作流結果

並提供：
- `🔎 搜尋歷史`
- `🕘 最近對話`
- 自動 RAG Recall（將相關記憶補入 Copilot Prompt）

### 🌉 Web Gateway

除了 Telegram 之外，Axle 也提供本地 Web Gateway：
- `GET /chat`：操作介面
- `/api/chat/send`：Web Chat
- `/api/memory/*`：記憶搜尋 / 最近記憶 / 清除
- `/api/browser/run`：Browser 腳本
- `/api/workflows*`：建立 / 列表 / 取消工作流

採用 **Bearer Token** 驗證，Token 會自動生成並保存在 `~/.axle/credentials.json`。  
Gateway 預設只監聽 `127.0.0.1:8080`，且 API 僅接受 `Authorization: Bearer <token>`。

### 🌐 Browser 自動化

目前採 **安全讀取型 MVP**，支援：

```text
open https://<public-ip>
wait 2s
extract body
screenshot page.png
```

說明：
- `open`：開啟頁面
- `wait`：等待 JS 頁面渲染
- `extract`：擷取 `body` 或 CSS selector 文字
- `screenshot`：將畫面存進本次 `.axle/browser/run-*` artifact 目錄

安全限制：
- Browser automation **預設停用**
- 如需自行承擔風險啟用，請設定環境變數 `AXLE_ALLOW_UNSAFE_BROWSER=1`
- 即使啟用，Safari automation 仍不是完整的網路沙箱；請只對受信任的公開目標使用
- 僅允許公開 `http/https` **public IP** 目標
- 所有 screenshot / result 輸出仍限制在目前 workspace 的 `.axle/browser/run-*`

### 🧭 背景工作流

工作流可將複雜需求拆成 2-4 個步驟，於背景執行並持久化。  
每個步驟目前可使用：
- `copilot`
- `browser`

保護措施：
- 同一使用者最多同時執行 3 個背景工作流
- 系統整體最多同時保留 8 個執行中的背景工作流

### 🗣️ 自然語言路由

在主選單狀態下，直接輸入任何文字即自動進入 AI 對話模式，無需先點選「🤖 Copilot」按鈕。

### ⚡ 串流回應

Copilot 對話回覆即時更新至同一則訊息（每 1.5 秒更新一次），不用等全部生成完畢才看到。

### 📤 直接上傳

在任何狀態下，直接傳送 PDF 或圖片給 Bot，即觸發對應的處理流程。

---

## 模型清單

共 17 種 Copilot 模型，透過「🔄 切換模型」選擇：

| 費率 | 廠商 | 模型 |
|------|------|------|
| 🆓 免費 | OpenAI | gpt-5-mini, gpt-4.1 |
| 0.25x | Anthropic | claude-haiku-4.5 |
| 0.5x | OpenAI | gpt-5.1-codex-mini |
| 1x | Anthropic | claude-sonnet-4.6, claude-sonnet-4.5, claude-sonnet-4 |
| 1x | OpenAI | gpt-5.1, gpt-5.2 |
| 1x | Google | gemini-3-pro-preview |
| 1.5x | OpenAI | gpt-5.1-codex |
| 3x | Anthropic | claude-opus-4.6, claude-opus-4.5 |
| 3x | OpenAI | gpt-5.3-codex, gpt-5.2-codex, gpt-5.1-codex-max |
| 33x | Anthropic | claude-opus-4.6-fast |

---

## 防護限制

| 項目 | 限制值 | 說明 |
|------|--------|------|
| 同時任務數 | 1 | 防止資源競爭 |
| Shell 執行超時 | 60 秒 | 防止指令永遠掛住 |
| Copilot 超時 | 5 分鐘 | 防止 CLI 無回應 |
| Prompt 長度上限 | 8,000 字元 | 防止 Context 溢出 |
| Telegram 訊息分段 | 4,000 字元/段 | 超長回應自動分段 |
| 檔案讀取上限 | 30 KB | 避免訊息過大 |
| 檔案寫入上限 | 1 MB | 防止磁碟濫用 |
| Shell 輸出上限 | 1 MB | 防止記憶體耗盡 |
| Web 擷取上限 | 100 KB | 限制網頁下載量 |
| 分段發送間隔 | 300 ms | 避免 Telegram API 限流 |

---

[← 返回首頁](../README.md) | [完整教學](tutorial-zh.md) | [架構指南](architecture.md)
