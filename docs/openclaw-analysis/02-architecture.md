# 02 — 整體架構與分層

[← 總目錄](00-index.md) · [上一章](01-overview.md) · [下一章 →](03-gateway.md)

---

## 2.1 架構鳥瞰圖

```
 ┌──────────────────────────────────────────────────────────────────┐
 │                       使用者 (User)                              │
 │   WhatsApp · Telegram · Slack · Discord · Signal · iMessage     │
 │   WebChat · macOS App · iOS Node · Android Node · CLI           │
 └──────────────┬───────────────────────────────────────────────────┘
                │
                ▼
 ┌──────────────────────────────────────────────────────────────────┐
 │                    Gateway (控制平面)                             │
 │   ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────────────┐  │
 │   │ Channels │ │ Sessions │ │  Router  │ │   WebSocket API  │  │
 │   └──────────┘ └──────────┘ └──────────┘ └──────────────────┘  │
 │   ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────────────┐  │
 │   │  Config  │ │   Cron   │ │  Hooks   │ │  Control UI/Web  │  │
 │   └──────────┘ └──────────┘ └──────────┘ └──────────────────┘  │
 └──────────────┬───────────────────────────────────────────────────┘
                │
                ▼
 ┌──────────────────────────────────────────────────────────────────┐
 │                    Pi Agent Runtime (RPC)                        │
 │   ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────────────┐  │
 │   │  Tools   │ │  Memory  │ │ Plugins  │ │     Skills       │  │
 │   └──────────┘ └──────────┘ └──────────┘ └──────────────────┘  │
 │   ┌──────────┐ ┌──────────┐ ┌──────────┐                       │
 │   │ Browser  │ │  Canvas  │ │  Nodes   │                       │
 │   └──────────┘ └──────────┘ └──────────┘                       │
 └──────────────┬───────────────────────────────────────────────────┘
                │
                ▼
 ┌──────────────────────────────────────────────────────────────────┐
 │              Model Providers (Extensions)                        │
 │   OpenAI · Anthropic · Google · Mistral · Ollama · 70+          │
 └──────────────────────────────────────────────────────────────────┘
```

## 2.2 分層設計

OpenClaw 採用的是**中心化 Gateway + 扇出式 Adapter** 的架構，而非傳統的微服務。

### 層次說明

| 層 | 職責 | 對應原始碼 |
|----|------|------------|
| **Channel Adapters** | 接收/發送各平台訊息 | `src/channels/` + `extensions/<channel>/` |
| **Gateway** | 控制平面：路由、Session 管理、WebSocket API | `src/gateway/` |
| **Agent Runtime (Pi)** | AI 推理 + 工具呼叫 + Streaming | `src/agents/` |
| **Tools** | 瀏覽器、Canvas、Node 操作、指令執行 | `src/browser/` · `src/canvas-host/` · `src/process/` |
| **Skills / Plugins** | 能力擴展（prompt / code） | `skills/` · `src/plugins/` · `extensions/` |
| **Model Providers** | LLM API 適配器 | `extensions/<provider>/` |
| **Companion Apps** | 原生桌面/行動 App | `apps/macos/` · `apps/ios/` · `apps/android/` |
| **UI** | Control UI + WebChat | `ui/` |

### 設計考量

1. **Gateway 作為單一入口**：所有 Client（Channel、CLI、App、WebChat）都透過 WebSocket 連到同一個 Gateway，統一了事件模型。
2. **Agent 與 Gateway 分離**：Agent Runtime 以 RPC 方式被 Gateway 呼叫，這使得 Agent 可以獨立迭代、獨立測試。
3. **Extension 機制統一 Channel 與 Provider**：無論是「Telegram 頻道適配」還是「OpenAI 模型適配」，都是同一套 Extension 載入機制。

## 2.3 Monorepo 結構

```
openclaw/
├── src/                    # 核心原始碼（Gateway + Agent + Tools）
│   ├── gateway/            # Gateway 控制平面
│   ├── agents/             # Pi Agent Runtime
│   ├── channels/           # Channel 共用邏輯
│   ├── sessions/           # Session 模型
│   ├── routing/            # 訊息路由
│   ├── browser/            # Browser 自動化
│   ├── canvas-host/        # Canvas/A2UI host
│   ├── memory/             # 記憶子系統
│   ├── security/           # 安全防護
│   ├── plugins/            # Plugin SDK + 載入器
│   ├── providers/          # Model provider 抽象
│   ├── config/             # 組態管理
│   ├── cli/                # CLI 指令
│   ├── cron/               # 定時任務
│   ├── hooks/              # Lifecycle hooks
│   ├── media/              # 媒體處理 pipeline
│   ├── tts/                # Text-to-Speech
│   ├── web-search/         # Web 搜尋整合
│   └── ...                 # 其他子模組
├── extensions/             # 模型提供商 + Channel Adapter（70+）
├── skills/                 # 內建 Skills（52）
├── packages/               # 獨立 npm packages
│   ├── clawdbot/           # 歷史相容 package
│   └── moltbot/            # 歷史相容 package
├── apps/                   # 原生 App
│   ├── macos/              # macOS menu bar app（Swift）
│   ├── ios/                # iOS node（Swift）
│   ├── android/            # Android node（Kotlin）
│   └── shared/             # App 共用邏輯
├── ui/                     # Control UI（Lit + Vite）
├── test/                   # E2E / integration tests
├── docs/                   # 文件
└── scripts/                # 開發/CI 腳本
```

### 設計目的

- **Monorepo**：用 pnpm workspace 管理所有子專案，確保版本一致性。
- **`extensions/` 與 `skills/` 分離**：Extensions 是程式碼層面的適配器（TypeScript）；Skills 是 prompt 層面的能力描述（Markdown）。
- **`packages/`**：保留歷史名稱（Clawdbot、Moltbot）的向後相容性。

## 2.4 關鍵設計模式

### 2.4.1 Event-Driven Architecture

Gateway 基於 WebSocket 的事件驅動模型：
- Client 連線後訂閱事件流
- Channel Adapter 將外部訊息轉換為標準化事件
- Agent 處理事件並產生回應事件

### 2.4.2 Adapter Pattern

每個 Channel 和 Model Provider 都是一個 Adapter：
- 統一介面：接收標準化的 message / event
- 內部轉換：適配各平台的 SDK/API

### 2.4.3 Plugin Slot Pattern

Memory 採用「只能有一個 active plugin」的設計：
- 預設提供多種 memory backend
- 使用者選擇一個作為當前使用
- 未來計劃收斂到一個推薦方案

> 參考：[VISION.md — Plugins & Memory](https://github.com/openclaw/openclaw/blob/main/VISION.md)

---

[← 總目錄](00-index.md) · [上一章](01-overview.md) · [下一章：Gateway 控制平面 →](03-gateway.md)
