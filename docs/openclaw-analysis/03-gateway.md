# 03 — Gateway 控制平面

[← 總目錄](00-index.md) · [上一章](02-architecture.md) · [下一章 →](04-channels.md)

---

## 3.1 Gateway 的角色

Gateway 是 OpenClaw 的**唯一控制平面**，所有 Client 都透過它交互：

```
Channel Adapters ─┐
CLI ──────────────┤
macOS App ────────┼──▶ Gateway (ws://127.0.0.1:18789)
iOS/Android Node ─┤
WebChat ──────────┤
Control UI ───────┘
```

### 職責

| 職責 | 說明 |
|------|------|
| **Session 管理** | 建立、維護、compaction、prune sessions |
| **Channel 路由** | 將外部 IM 訊息路由到正確的 Agent session |
| **Agent 調度** | 透過 RPC 呼叫 Pi Agent Runtime |
| **WebSocket API** | 提供標準化的 WS 事件協議 |
| **Config 管理** | 讀取/驗證/熱重載 `openclaw.json` |
| **Cron 排程** | 定時任務和喚醒 |
| **Webhook 接收** | Gmail Pub/Sub、自訂 webhook |
| **Web 服務** | 提供 Control UI 和 WebChat |
| **Tailscale 整合** | 自動設定 Serve/Funnel（遠端存取） |

## 3.2 為何採用單一 Gateway

### 設計考量

1. **簡化部署**：一個 process 就是整個控制平面，不需要 orchestrator。
2. **統一事件模型**：所有 Client 看到同一份事件流，不需要跨服務同步。
3. **Local-first 友善**：跑在個人裝置上，不需要複雜的基礎設施。
4. **低延遲**：同一 process 內的 routing，比跨服務呼叫快得多。

### 取捨

- **不適合水平擴展**：單一 Gateway 意味著無法分散到多台機器。但對個人助理場景，這不是問題。
- **重啟會中斷所有連線**：Gateway 重啟時所有 Channel 會短暫斷線。

## 3.3 WebSocket 協議

Gateway 暴露 `ws://127.0.0.1:18789`，支援以下主要方法：

| 方法 | 方向 | 說明 |
|------|------|------|
| `agent.invoke` | Client → GW | 觸發 Agent 推理 |
| `agent.event.*` | GW → Client | Agent 回應事件（text、tool_use、streaming） |
| `sessions.patch` | Client → GW | 修改 session 參數（model、thinkingLevel 等） |
| `sessions.list` | Client → GW | 列出 active sessions |
| `node.list` / `node.invoke` | Both | Node 能力查詢與呼叫 |
| `config.get` / `config.set` | Client → GW | 組態讀寫 |

### 設計目的

- **統一傳輸層**：不論 Client 是 CLI、App 還是 WebChat，都用同一套 WS 協議。
- **即時事件推送**：Streaming、typing indicator、presence 都透過 WS 事件推送。
- **雙向通訊**：Node 可以既接收指令，也主動回報裝置狀態。

## 3.4 組態系統

### 主設定檔

`~/.openclaw/openclaw.json`（JSON5 格式）

```json5
{
  agent: {
    model: "anthropic/claude-opus-4-6",
  },
  channels: {
    telegram: { botToken: "..." },
    whatsapp: { allowFrom: ["+886..."] },
  },
  gateway: {
    port: 18789,
    bind: "loopback",
    tailscale: { mode: "off" },
  },
}
```

### 設計考量

- **JSON5**：允許註解和尾隨逗號，比純 JSON 適合手動編輯。
- **環境變數優先**：`TELEGRAM_BOT_TOKEN` 等環境變數會覆蓋設定檔。
- **熱重載**：Gateway 監聽設定檔變更，部分參數可以不重啟生效。
- **Doctor 驗證**：`openclaw doctor` 會檢查設定、權限、安全風險。

## 3.5 Tailscale 整合

Gateway 預設綁定到 loopback（127.0.0.1），不暴露公網。
遠端存取透過 Tailscale 提供：

| 模式 | 說明 | 安全要求 |
|------|------|----------|
| `off` | 不使用 Tailscale（預設） | — |
| `serve` | Tailnet-only HTTPS | 自動使用 Tailscale 身份標頭 |
| `funnel` | 公開 HTTPS | 強制要求 password auth |

### 為何選擇 Tailscale

- **零信任網路**：不需要打開 port，不需要 DDNS。
- **內建 TLS**：Tailscale 自動處理 HTTPS 證書。
- **安全分級**：`serve` = tailnet-only，`funnel` = 公開但強制密碼。

> 參考：[docs.openclaw.ai/gateway/tailscale](https://docs.openclaw.ai/gateway/tailscale)

## 3.6 Daemon 模式

`openclaw onboard --install-daemon` 會安裝系統 daemon：
- **macOS**：launchd user service
- **Linux**：systemd user service

### 設計目的

- **Always-on**：個人助理需要一直在線，daemon 確保 Gateway 開機自動啟動。
- **背景運行**：不佔用終端機。
- **自動恢復**：crash 後自動重啟。

---

[← 總目錄](00-index.md) · [上一章](02-architecture.md) · [下一章：多頻道訊息整合 →](04-channels.md)
