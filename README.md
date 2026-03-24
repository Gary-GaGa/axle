# Axle

<div align="center">

**🌐 語言切換**  
[繁體中文](README.md) | [English](README.en.md)

[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)](https://go.dev)
[![License](https://img.shields.io/badge/License-MIT-green)](LICENSE)
[![Platform](https://img.shields.io/badge/Platform-macOS%20%7C%20Linux-lightgrey)]()

</div>

> 🔧 安全、本地的 Golang AI Agent — 透過 Telegram 控制你的開發工具，無需額外 API Key。

---

## 📖 文件導覽

| 文件 | 說明 |
|------|------|
| 🚀 [快速安裝指南 → 新手入門](docs/tutorial-zh.md#一新手入門) | 5 分鐘完成安裝與第一次對話 |
| 📚 [完整教學（新手到進階）](docs/tutorial-zh.md) | Telegram 操作 · Git · 排程 · 自我升級 |
| 🔧 [完整功能參考](docs/features-zh.md) | 所有功能的詳細說明 |
| 🏗️ [架構與開發指南](docs/architecture.md) | 架構設計、安全機制、貢獻指南 |

---

## ✨ 功能亮點

<table>
<tr>
<td width="50%">

**🔌 預設功能（開箱即用）**
- 📁 讀取 / ✏️ 寫入代碼
- 🤖 Copilot 助手（串流回應）
- ⚡ Shell 指令執行（危險偵測）
- 🔄 切換 AI 模型（17 種）
- 📊 系統狀態 · 🛑 取消任務

</td>
<td width="50%">

**🧩 擴充功能（按需啟用）**
- 🔀 Git · 🐙 GitHub 整合
- 📧 Email · 📅 行事曆
- 🔍 Web 搜尋 · 🌐 URL 擷取
- 🧠 記憶 / 歷史 · 🌐 Browser
- 👥 子代理 · 🧭 背景工作流
- 🌉 Web Gateway · 🎮 RPG Dashboard

</td>
</tr>
<tr>
<td colspan="2">

**✨ 特色功能**
- 🔧 **自我升級**：描述需求 → AI 規劃 → 自動修改代碼 → 編譯測試 → 重啟
- 🧠 **長期記憶 + RAG**：可搜尋歷史，並自動將相關記憶補進 Copilot Prompt
- 🌉 **雙通道互動**：除了 Telegram，還可用本地 Web Gateway（預設 `127.0.0.1:8080`）操作聊天 / Browser / 工作流
- 🌐 **Browser 安全預設**：Browser automation 目前預設停用；需自行設定 `AXLE_ALLOW_UNSAFE_BROWSER=1` 才會啟用。即使啟用，它仍不是完整的網路沙箱，僅建議對受信任的公開目標使用。
- 🧭 **背景工作流**：先規劃 2-4 個步驟，再於背景執行與追蹤
- 🎮 **RPG Dashboard**：瀏覽器即時監控，像素風格，含等級與成就系統
- 🗣️ **自然語言路由**：直接輸入文字自動進入 AI 對話

</td>
</tr>
</table>

---

## 🚀 快速開始

### 前置需求

| 工具 | 用途 | 是否必要 |
|------|------|:--------:|
| Go 1.22+ | 編譯運行 | ✅ 必要 |
| Telegram Bot Token | Bot 介面（[@BotFather](https://t.me/BotFather)） | ✅ 必要 |
| GitHub Copilot CLI (`copilot`) | AI 核心功能 | ✅ 必要 |
| GitHub CLI (`gh`) | GitHub 整合功能 | 選用 |
| icalBuddy | macOS 行事曆 | 選用 |

### 三步安裝

```bash
# 1. Clone
git clone https://github.com/Gary-GaGa/axle.git && cd axle

# 2. 編譯
go build -o axle ./cmd/axle

# 3. 啟動（首次自動引導設定）
./axle
```

首次啟動時，Axle 會互動式詢問 **Telegram Bot Token** 和 **你的 Telegram User ID**，設定後自動儲存至 `~/.axle/credentials.json`，下次啟動免設定。  
本地 Web Gateway 的 Bearer Token 也會在首次啟動時自動產生並一併保存。

> 💡 進階設定（多用戶、Email、GitHub）請見 [完整教學](docs/tutorial-zh.md)

---

## 🔒 安全設計

- **白名單隱形模式**：未授權用戶發訊息完全無回應
- **沙箱檔案系統**：所有操作嚴格限制在 workspace，禁止 `../` 逃逸
- **本地 Web Gateway**：預設只監聽 `127.0.0.1:8080`，所有 Gateway API 需 Bearer Token
- **三級危險偵測**：⛔ 封鎖 → ⚠️ 二次確認 → ✅ 一般確認
- **Human-in-the-Loop**：Shell 指令執行前必須人工按鈕確認

> 詳細說明見 [架構文件 — 安全機制](docs/architecture.md#安全機制)

---

## 🎮 RPG Dashboard

啟動後自動開啟 `http://127.0.0.1:8080`，即時顯示 Agent 工作狀態（角色面板 / 任務卷軸 / 戰鬥日誌 / 技能統計）。等級從 🟤 見習編碼師 成長到 🔴 不朽引擎，XP 持久化重啟不歸零。

---

## 📝 開發指令

```bash
go build -race -o axle ./cmd/axle  # 編譯
go vet ./...                        # 靜態分析
go test -race ./internal/...        # 測試
```

---

## License

MIT
