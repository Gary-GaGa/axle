# 05 — Agent 執行時與 Session 模型

[← 總目錄](00-index.md) · [上一章](04-channels.md) · [下一章 →](06-tools.md)

---

## 5.1 Pi Agent Runtime

OpenClaw 的 Agent 引擎稱為 **Pi**，它以 RPC 模式運行在 Gateway 內部。

### 核心流程

```
1. 使用者送出訊息
2. Gateway 路由到對應 Session
3. 建構 prompt（system + memory + context + tools + user message）
4. 呼叫 Model Provider API
5. 模型回應（可能包含 tool_use）
6. 如果有 tool_use → 執行工具 → 把結果放回 prompt → 再次呼叫模型
7. 最終文字回應 → 透過 Channel Adapter 發回
```

### 關鍵特性

| 特性 | 說明 |
|------|------|
| **Tool streaming** | 工具呼叫結果即時串流回 Client |
| **Block streaming** | 回應分 block 串流，每個 block 有明確邊界 |
| **Multi-turn tool loop** | 模型可以連續呼叫多個工具，直到得到最終答案 |
| **Thinking levels** | 支援 `off`/`minimal`/`low`/`medium`/`high`/`xhigh` 思考深度 |

## 5.2 Session 模型

Session 是 OpenClaw 管理對話上下文的核心概念。

### Session 類型

| 類型 | 建立方式 | 說明 |
|------|----------|------|
| **main** | DM 聊天 | 使用者的主要直接對話 session |
| **group** | 群組聊天 | 每個群組一個獨立 session |
| **spawned** | Agent 自行建立 | 由 Agent 透過 `sessions_send` 建立的子 session |

### Session 生命週期

```
建立 ──▶ 活躍 ──▶ Compaction ──▶ 活躍 ──▶ ... ──▶ Reset/Prune
                    │
                    └── 壓縮對話歷史，保留摘要
```

### Compaction（壓縮）

| 機制 | 說明 |
|------|------|
| 手動 | 使用者發送 `/compact` |
| 自動 | Token 數接近模型上限時自動觸發 |
| 過程 | 保留最近的對話 + 生成歷史摘要 |

### 設計考量

1. **Session 隔離**：不同 Channel 的同一個人會有不同的 session（但可以設定共享）
2. **成本控制**：Compaction 避免 prompt 無限成長
3. **上下文品質**：摘要保留關鍵資訊，而非簡單截斷

## 5.3 Multi-Agent 路由

OpenClaw 支援將不同的 Channel/帳號路由到不同的 Agent：

```json5
{
  agents: {
    default: { model: "anthropic/claude-opus-4-6", workspace: "~/.openclaw/workspace" },
    coding:  { model: "openai/codex-mini", workspace: "~/code" },
  },
  routing: {
    "telegram:+886...": "coding",
    "discord:*":        "default",
  }
}
```

### 設計目的

- **角色分離**：不同用途用不同的 Agent 設定
- **工作區隔離**：每個 Agent 有獨立的 workspace
- **模型選擇**：不同任務用最適合的模型
- **Session 隔離**：Agent 之間不共享對話歷史

## 5.4 Agent-to-Agent 通訊

Agent 可以透過 Session Tools 互相溝通：

| 工具 | 說明 |
|------|------|
| `sessions_list` | 發現活躍的 sessions（agents） |
| `sessions_history` | 取得其他 session 的對話記錄 |
| `sessions_send` | 發訊息給另一個 session，可選回覆 |

### 機制

- **Reply-back**：A 發訊息給 B，B 處理完可以自動回覆 A
- **REPLY_SKIP** / **ANNOUNCE_SKIP**：控制是否產生回覆/通知
- **防止無限迴圈**：有最大 ping-pong 深度限制

### 設計目的

- **協作**：不同專長的 Agent 可以分工合作
- **不跳平台**：Agent 之間的溝通不需要切換 IM 頻道

## 5.5 Workspace 與 Prompt 注入

### Workspace 結構

```
~/.openclaw/workspace/
├── AGENTS.md          # Agent 行為指令
├── SOUL.md            # 人格設定
├── TOOLS.md           # 自訂工具說明
└── skills/
    └── <skill>/
        └── SKILL.md   # Skill prompt
```

### Prompt 組裝順序

1. **System prompt**（內建核心指令）
2. **AGENTS.md**（使用者自訂行為）
3. **SOUL.md**（人格）
4. **TOOLS.md**（自訂工具描述）
5. **Skills**（活躍 skill 的 SKILL.md）
6. **Memory context**（記憶檢索結果）
7. **Session history**（對話歷史）
8. **User message**（當前訊息）

### 設計考量

- **使用者可控**：使用者可以完全自訂 Agent 的行為
- **分層覆蓋**：每一層都可以被更外層覆蓋
- **Skill 熱插拔**：新增/移除 skill 不需要重啟

## 5.6 Model Failover

```json5
{
  agent: {
    model: "anthropic/claude-opus-4-6",
    fallbacks: ["openai/gpt-5.2", "google/gemini-3-ultra"],
  }
}
```

| 機制 | 說明 |
|------|------|
| **自動切換** | 主模型 API 失敗時自動嘗試下一個 |
| **Auth 輪替** | 支援 OAuth + API Key 輪替 |
| **Retry 策略** | 指數退避 + 最大重試次數 |
| **Token 保護** | 模型切換時重新計算 token 預算 |

### 設計目的

- **高可用**：單一模型提供商故障不影響助理可用性
- **成本優化**：可以用便宜模型當 fallback
- **使用者透明**：切換在背景發生，使用者不需介入

> 參考：[docs.openclaw.ai/concepts/model-failover](https://docs.openclaw.ai/concepts/model-failover)

---

[← 總目錄](00-index.md) · [上一章](04-channels.md) · [下一章：工具系統 →](06-tools.md)
