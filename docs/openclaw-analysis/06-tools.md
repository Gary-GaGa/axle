# 06 — 工具系統：Browser / Canvas / Nodes

[← 總目錄](00-index.md) · [上一章](05-agent-runtime.md) · [下一章 →](07-skills-plugins.md)

---

## 6.1 工具概覽

OpenClaw 提供一套 **first-class tools**，讓 Agent 不只是聊天，還能實際操作。

| 工具 | 說明 | 對應原始碼 |
|------|------|------------|
| **bash / process** | 在 host 或沙箱執行系統指令 | `src/process/` |
| **read / write / edit** | 讀寫工作區檔案 | — |
| **browser** | 控制 Chrome/Chromium 瀏覽器 | `src/browser/` |
| **canvas** | 推送視覺內容到 Canvas 工作區 | `src/canvas-host/` |
| **nodes** | 控制連接的裝置（相機、螢幕、通知等） | `src/node-host/` |
| **cron** | 定時任務 | `src/cron/` |
| **sessions_**** | Agent-to-Agent 通訊 | `src/sessions/` |
| **discord / slack** | 平台特定操作（設定角色、管理頻道等） | `extensions/discord/` etc. |

## 6.2 Browser 自動化

### 與 Axle 的關鍵差異

| 面向 | OpenClaw | Axle |
|------|----------|------|
| 瀏覽器 | 自管理的 Chrome/Chromium | Safari (AppleScript) |
| 控制方式 | CDP (Chrome DevTools Protocol) | AppleScript + osascript |
| Profile 隔離 | 專用 browser profile | 無（共享使用者 Safari） |
| 操作能力 | 完整 CDP：DOM 操作、JS 注入、網路攔截 | 簡易 DSL：open/wait/extract/screenshot |

### 架構

```
Agent ──▶ browser tool ──▶ src/browser/ ──▶ CDP (Chrome DevTools Protocol)
                                │
                                ├── 專用 Chrome instance
                                ├── Snapshots（DOM 快照）
                                ├── Actions（點擊、輸入、導航）
                                ├── Uploads（檔案上傳）
                                └── Profile management
```

### 設計考量

1. **隔離性**：使用專用的 Chrome profile，不影響使用者自己的瀏覽器
2. **可觀察性**：支援 DOM snapshot，Agent 可以「看到」頁面結構
3. **安全**：CDP 只在本地連接，不暴露到網路

## 6.3 Canvas / A2UI

Canvas 是一個 **Agent 驅動的視覺工作區**：

```
Agent ──▶ canvas.push(html/react) ──▶ Canvas 視窗 ──▶ 使用者看到視覺內容
Agent ──▶ canvas.eval(js) ──▶ Canvas 視窗 ──▶ 執行腳本
Agent ──▶ canvas.snapshot() ──▶ 擷取當前畫面 ──▶ Agent 分析
```

### A2UI (Agent-to-UI)

- Agent 可以推送完整的 HTML/React 應用到 Canvas
- Canvas 在 macOS app / iOS app / WebChat 中呈現
- 使用者可以與 Canvas 互動，Agent 可以觀察互動結果

### 設計目的

- **視覺化輸出**：不是所有東西都適合用文字表達
- **互動式體驗**：Agent 可以建立可互動的 UI
- **跨平台一致性**：同一個 Canvas 在不同裝置上都能呈現

## 6.4 Nodes（裝置控制）

Node 是 OpenClaw 對「連接裝置」的抽象：

### 裝置類型

| 裝置 | 能力 |
|------|------|
| **macOS (node mode)** | `system.run`、`system.notify`、canvas、camera |
| **iOS** | Voice Wake、Canvas、camera、screen recording |
| **Android** | 通知、位置、SMS、照片、通訊錄、日曆、動感偵測 |

### 通訊協議

```
Gateway ◄──── WebSocket ────▶ Node (裝置)
        node.list            → 列出可用能力
        node.describe        → 查詢能力詳情
        node.invoke          → 執行裝置操作
```

### 設計考量

1. **能力廣告**：Node 連線時會廣告自己支援的功能和權限狀態
2. **權限遵守**：macOS TCC 權限（螢幕錄製、相機等）在 node 端檢查
3. **遠端 + 本地混合**：
   - **Gateway 所在機器**：執行 bash/tool
   - **裝置 Node**：執行裝置特有操作（拍照、通知等）

## 6.5 Cron 與自動化

```json5
{
  cron: {
    jobs: [
      { id: "daily-briefing", schedule: "0 9 * * *", prompt: "Generate my daily briefing" },
      { id: "check-email",   schedule: "*/30 * * * *", prompt: "Check for urgent emails" },
    ]
  }
}
```

| 自動化類型 | 說明 |
|------------|------|
| **Cron jobs** | 定時觸發 Agent 執行任務 |
| **Wakeups** | 裝置端喚醒（Voice Wake） |
| **Webhooks** | 外部事件觸發（GitHub、Gmail 等） |
| **Gmail Pub/Sub** | Google Pub/Sub 即時推送新郵件 |

### 設計目的

- **主動式助理**：不只被動回應，也能主動執行排程任務
- **事件驅動**：外部事件可以觸發 Agent 行為
- **跨頻道通知**：排程結果可以推送到任何已連接的 Channel

## 6.6 Elevated Access

```
/elevated on   → 切換為高權限模式（bash 可執行任何指令）
/elevated off  → 恢復預設限制
```

| 面向 | 說明 |
|------|------|
| 適用範圍 | 僅限 `main` session |
| 持久化 | Gateway 記住 per-session 的 toggle 狀態 |
| 前提 | 需要在設定中啟用並加入 allowlist |

### 設計考量

- **預設安全**：Agent 預設不能執行危險操作
- **顯式切換**：需要使用者在對話中明確切換
- **Session 限定**：群組 session 不能 elevate

---

[← 總目錄](00-index.md) · [上一章](05-agent-runtime.md) · [下一章：Skills 與 Plugin 生態系 →](07-skills-plugins.md)
