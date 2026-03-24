# 04 — 多頻道訊息整合

[← 總目錄](00-index.md) · [上一章](03-gateway.md) · [下一章 →](05-agent-runtime.md)

---

## 4.1 頻道概覽

OpenClaw 支援 22+ 個通訊頻道，是目前開源 AI 助理中覆蓋最廣的：

| 分類 | 頻道 | 實作方式 |
|------|------|----------|
| **即時通訊** | WhatsApp | Baileys (非官方) |
| | Telegram | grammY |
| | Signal | signal-cli |
| | LINE | LINE Messaging API |
| | Zalo / Zalo Personal | — |
| **企業協作** | Slack | Bolt SDK |
| | Discord | discord.js |
| | Google Chat | Chat API |
| | Microsoft Teams | Bot Framework |
| | Feishu (飛書) | — |
| | Mattermost | — |
| | Matrix | — |
| **Apple 生態** | iMessage (legacy) | imsg (macOS-only) |
| | BlueBubbles | BlueBubbles API (推薦) |
| **其他** | IRC | — |
| | Nostr | — |
| | Twitch | — |
| | Tlon (Urbit) | — |
| | Synology Chat | — |
| | Nextcloud Talk | — |
| **自建** | WebChat | Gateway 內建 |

## 4.2 Channel Adapter 架構

```
外部 IM Platform
      │
      ▼
┌─────────────────┐
│ Channel Adapter  │  ← extensions/<channel>/
│  (Extension)     │
└────────┬────────┘
         │ 標準化事件
         ▼
┌─────────────────┐
│   src/channels/  │  ← 共用路由邏輯
│   src/routing/   │
└────────┬────────┘
         │
         ▼
      Gateway
```

### 每個 Channel Adapter 負責

1. **認證**：管理各平台的 token / session / 裝置連結
2. **訊息轉換**：將平台特有格式轉為標準化 message 物件
3. **媒體處理**：圖片、音訊、影片的上傳/下載/轉碼
4. **分段發送**：長訊息按平台限制自動分段（chunking）
5. **回應格式**：Markdown → 平台原生格式（按鈕、表情等）

## 4.3 DM 配對安全

### 問題

連接到真實 IM 平台代表任何人都可以發 DM 給 bot。
不加限制的話，陌生人可以白嫖你的 API 額度。

### 解法：Pairing 機制

```
預設 DM 政策 = "pairing"

1. 陌生人發訊息 → bot 回傳一個短配對碼
2. 使用者在終端機執行 openclaw pairing approve <channel> <code>
3. 通過後加入本地 allowlist
```

| 政策 | 說明 |
|------|------|
| `pairing` | 預設。未授權用戶只收到配對碼 |
| `open` | 所有 DM 都處理（需搭配 `allowFrom: ["*"]`） |

### 設計目的

- **防止未授權使用**：不是好友的人無法白嫖 LLM 額度
- **顯式授權**：需要在終端機操作，確保是裝置擁有者同意
- **分頻道控制**：每個 Channel 可以有不同的 DM 政策

## 4.4 群組路由

### 群組特殊邏輯

| 機制 | 說明 |
|------|------|
| **Mention gating** | 群組中需要 @bot 才會觸發回應 |
| **Reply tags** | 追蹤回覆鏈，維持對話上下文 |
| **Session 隔離** | 每個群組有獨立 session（不共享對話歷史） |
| **Activation mode** | `mention` 或 `always`（用 `/activation` 切換） |
| **Per-channel chunking** | 依據各平台訊息長度限制自動分段 |

### 設計考量

- **防止干擾**：群組中不應該對每則訊息都回應
- **上下文清晰**：群組 session 與 DM session 分開，避免資訊洩漏
- **可控性**：群組管理員可以隨時切換 activation 模式

## 4.5 媒體管道 (Media Pipeline)

```
外部平台 ──▶ 下載 ──▶ 檔案類型判斷 ──▶ 轉碼/壓縮 ──▶ Agent 處理
                                              │
                                              ├── 圖片 → 視覺理解
                                              ├── 音訊 → 語音轉文字
                                              └── 影片 → 擷取關鍵幀
```

| 媒體類型 | 處理 | 相關模組 |
|----------|------|----------|
| 圖片 | 直接傳給支援 vision 的模型 | `src/media/` + `src/image-generation/` |
| 音訊 | Whisper 轉文字 → 文字處理 | `src/media/` + `skills/openai-whisper/` |
| 影片 | 擷取幀 + 音軌 | `src/media/` + `skills/video-frames/` |

### 設計目的

- **統一管道**：不論從哪個 Channel 來的媒體，都走同一套處理流程
- **安全限制**：檔案大小上限 + 暫存檔生命週期管理
- **降級處理**：模型不支援 vision 時，圖片會被忽略而非報錯

---

[← 總目錄](00-index.md) · [上一章](03-gateway.md) · [下一章：Agent 執行時與 Session 模型 →](05-agent-runtime.md)
