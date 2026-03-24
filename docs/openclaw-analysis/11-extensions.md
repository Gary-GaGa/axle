# 11 — Extensions 與模型提供商

[← 總目錄](00-index.md) · [上一章](10-apps.md) · [下一章 →](12-infra-ops.md)

---

## 11.1 Extension 機制

Extension 是 OpenClaw 的**統一適配器框架**，涵蓋兩大類：

| 類別 | 說明 | 範例 |
|------|------|------|
| **Model Provider** | LLM API 適配 | openai, anthropic, google, ollama |
| **Channel Adapter** | 通訊平台適配 | telegram, discord, whatsapp, slack |
| **功能模組** | 獨立功能 | memory-core, memory-lancedb, talk-voice |

## 11.2 Model Provider Extensions

### 支援的提供商（70+）

| 提供商 | Extension | 說明 |
|--------|-----------|------|
| **OpenAI** | `extensions/openai/` | GPT 系列 + Codex |
| **Anthropic** | `extensions/anthropic/` | Claude 系列 |
| **Google** | `extensions/google/` | Gemini 系列 |
| **Mistral** | `extensions/mistral/` | Mistral 系列 |
| **xAI** | `extensions/xai/` | Grok 系列 |
| **Ollama** | `extensions/ollama/` | 本地模型 |
| **OpenRouter** | `extensions/openrouter/` | 多模型路由 |
| **Amazon Bedrock** | `extensions/amazon-bedrock/` | AWS 託管模型 |
| **Microsoft** | `extensions/microsoft/` | Azure OpenAI |
| **Together** | `extensions/together/` | Together AI |
| **Perplexity** | `extensions/perplexity/` | Perplexity API |
| **Venice** | `extensions/venice/` | Venice AI |
| **vLLM** | `extensions/vllm/` | 自建 vLLM server |
| **SGLang** | `extensions/sglang/` | SGLang runtime |
| **Hugging Face** | `extensions/huggingface/` | HF Inference |
| **NVIDIA** | `extensions/nvidia/` | NVIDIA NIM |
| **Copilot Proxy** | `extensions/copilot-proxy/` | GitHub Copilot |
| **GitHub Copilot** | `extensions/github-copilot/` | GitHub Copilot (直接) |
| 中國提供商 | minimax, moonshot, qwen-portal-auth, qianfan, volcengine, xiaomi, zai | — |

### Provider Extension 結構

```
extensions/<provider>/
├── package.json         # 元資料 + 依賴
├── src/
│   ├── index.ts         # 入口：註冊模型列表
│   ├── chat.ts          # Chat completion 實作
│   ├── models.ts        # 模型定義（能力、token 上限等）
│   └── auth.ts          # 認證邏輯（OAuth / API Key）
└── test/
    └── ...
```

### 設計考量

1. **統一介面**：所有 Provider 實作同一套 Chat Completion 介面
2. **模型元資料**：每個模型宣告自己的能力（vision、function calling、streaming 等）
3. **Auth 分離**：OAuth 和 API Key 是獨立的 auth 策略，可以混合使用
4. **Failover 友善**：統一介面使得 model failover 可以無縫切換

## 11.3 Channel Extensions

| 頻道 | Extension | SDK/Library |
|------|-----------|-------------|
| WhatsApp | `extensions/whatsapp/` | Baileys |
| Telegram | `extensions/telegram/` | grammY |
| Discord | `extensions/discord/` | discord.js |
| Slack | `extensions/slack/` | Bolt |
| Signal | `extensions/signal/` | signal-cli |
| Google Chat | `extensions/googlechat/` | Chat API |
| MS Teams | `extensions/msteams/` | Bot Framework |
| iMessage | `extensions/imessage/` | imsg (legacy) |
| BlueBubbles | `extensions/bluebubbles/` | BlueBubbles API |
| Matrix | `extensions/matrix/` | — |
| IRC | `extensions/irc/` | — |
| Feishu | `extensions/feishu/` | — |
| LINE | `extensions/line/` | LINE SDK |
| Nostr | `extensions/nostr/` | — |
| ... | ... | ... |

### 設計考量

1. **Extension = 邊界**：每個 Channel 有自己的依賴，不污染核心
2. **獨立測試**：`pnpm test:extension <name>` 可以單獨測試一個 extension
3. **Contract tests**：`pnpm test:contracts:channels` 確保所有 Channel 遵循共用介面

## 11.4 功能模組 Extensions

| Extension | 功能 |
|-----------|------|
| `memory-core` | 核心記憶 backend |
| `memory-lancedb` | 向量記憶 backend |
| `talk-voice` | 語音對話 |
| `voice-call` | 語音通話 |
| `elevenlabs` | ElevenLabs TTS |
| `diagnostics-otel` | OpenTelemetry 診斷 |
| `diffs` | 差異比較 |
| `firecrawl` | 網頁爬取 |
| `device-pair` | 裝置配對 |
| `llm-task` | LLM 任務管理 |
| `thread-ownership` | 討論串所有權 |

## 11.5 Extension 載入機制

```
Gateway 啟動
    │
    ├── 掃描 extensions/ 目錄
    ├── 讀取每個 extension 的 package.json
    ├── 根據 openclaw.json 設定決定啟用哪些
    ├── 載入啟用的 extension
    └── 註冊到對應的 registry（model / channel / feature）
```

### 設計目的

- **隨插即用**：加新 extension 只需要加一個目錄
- **選擇性載入**：只載入有設定的 extension，減少啟動時間
- **版本一致**：Monorepo 確保所有 extension 與核心版本一致

---

[← 總目錄](00-index.md) · [上一章](10-apps.md) · [下一章：基礎設施與維運 →](12-infra-ops.md)
