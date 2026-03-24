# 01 — 專案概覽與設計哲學

[← 總目錄](00-index.md) · [下一章 →](02-architecture.md)

---

## 1.1 OpenClaw 是什麼

OpenClaw 是一個**開源的個人 AI 助理平台**，核心口號：

> *Your own personal AI assistant. Any OS. Any Platform. The lobster way. 🦞*

它讓你在自己的裝置上運行 AI 助理，透過你已經在用的通訊軟體（WhatsApp、Telegram、Slack、Discord、Signal、iMessage 等 22+ 個頻道）與之互動。

### 關鍵特色

| 特色 | 說明 |
|------|------|
| **Local-first** | Gateway 跑在你自己的裝置上，資料不經第三方伺服器 |
| **多頻道統一收件匣** | 22+ 個 IM 頻道 + WebChat + 原生 App |
| **多模型支援** | OpenAI、Anthropic、Google、Mistral、Ollama 等 70+ 模型提供商 |
| **工具導向** | Browser 自動化、Canvas 視覺工作區、裝置 Nodes（相機、螢幕錄製等） |
| **Skills 生態** | 52 個內建 skill + ClawHub 社群 registry |
| **語音互動** | Voice Wake（喚醒詞）+ Talk Mode（連續語音對話） |
| **跨裝置** | macOS menu bar app + iOS node + Android node |

## 1.2 起源與演進

```
Warelay → Clawdbot → Moltbot → OpenClaw
```

由 **Peter Steinberger**（前 PSPDFKit 創辦人）發起，最初是學習 AI 的個人實驗專案。
後來快速發展成社群驅動的大型開源專案。

## 1.3 設計哲學

### 1.3.1 「助理優先」而非「聊天機器人」

OpenClaw 的定位不是一般的 chatbot，而是能**真正執行任務**的助理。
它可以瀏覽網頁、操作裝置、讀寫檔案、執行指令。

### 1.3.2 「Gateway 是控制平面」

Gateway 是整個系統的心臟，但它本身不是產品——**助理**才是產品。
Gateway 只負責協調：Session、Channel、Tool、Event。

> 參考：[VISION.md](https://github.com/openclaw/openclaw/blob/main/VISION.md)
> *"The goal: a personal assistant that is easy to use, supports a wide range of platforms, and respects privacy and security."*

### 1.3.3 「安全預設，能力不減」

安全設計是一種「有意識的取捨」——預設要夠安全，但不犧牲真正的工作能力。
高風險路徑需要使用者明確 opt-in。

### 1.3.4 「核心精簡，延伸外移」

- 核心保持精簡（Gateway + Agent + Session）
- 額外功能透過 **Skills**（prompt-based）和 **Plugins**（code-based）擴展
- 新功能優先發布到 ClawHub，不進核心
- MCP 整合透過 `mcporter` 橋接，不直接嵌入核心

### 1.3.5 「TypeScript 是刻意選擇」

> *"TypeScript was chosen to keep OpenClaw hackable by default. It is widely known, fast to iterate in, and easy to read, modify, and extend."*

作為一個「編排系統」（orchestration system），TypeScript 的生態和可讀性是主要考量。

## 1.4 專案規模

| 指標 | 數據 |
|------|------|
| GitHub Stars | 323K+ |
| 通訊頻道支援 | 22+ |
| 模型提供商 | 70+ extensions |
| 內建 Skills | 52 |
| 原生 App 平台 | macOS + iOS + Android |
| 主要語言 | TypeScript |
| 建置工具 | pnpm monorepo + tsdown |
| 測試框架 | Vitest |
| License | MIT |

## 1.5 與 Axle 的定位對比

| 維度 | OpenClaw | Axle |
|------|----------|------|
| 語言 | TypeScript | Go |
| 架構 | Gateway 控制平面 + 多 Channel Adapter | 單一 Telegram Bot + Web UI |
| 頻道 | 22+ IM + 原生 App | Telegram + Web |
| 工具 | Browser (Chrome/CDP) + Canvas + Nodes | Browser (Safari/AppleScript) + 工作區讀寫 |
| 部署 | npm CLI / Docker / Nix | 單一 Go binary |
| 規模 | 大型社群專案 | 個人專案 |

---

[← 總目錄](00-index.md) · [下一章：整體架構與分層 →](02-architecture.md)
