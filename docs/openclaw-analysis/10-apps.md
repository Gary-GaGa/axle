# 10 — 原生 App：macOS / iOS / Android

[← 總目錄](00-index.md) · [上一章](09-security.md) · [下一章 →](11-extensions.md)

---

## 10.1 App 定位

> Gateway 獨立就能提供完整的助理體驗。所有 App 都是**可選的**附加功能。

| App | 平台 | 語言 | 功能定位 |
|-----|------|------|----------|
| **OpenClaw.app** | macOS | Swift | Menu bar 控制 + Voice Wake + Canvas |
| **iOS Node** | iOS | Swift | 語音 + Canvas + 相機 + 裝置配對 |
| **Android Node** | Android | Kotlin | 聊天 + 語音 + 裝置指令 |

## 10.2 macOS App

### 功能

| 功能 | 說明 |
|------|------|
| **Menu bar** | Gateway 狀態監控、快速操作 |
| **Voice Wake** | 自訂喚醒詞（如 "Hey Claw"） |
| **Push-to-Talk** | 按鍵即說語音輸入 |
| **WebChat** | 內嵌的 WebChat 介面 |
| **Canvas** | A2UI 視覺工作區顯示 |
| **Debug tools** | Gateway 日誌、事件檢視 |
| **Remote control** | 透過 SSH/Tailscale 連接遠端 Gateway |
| **Node mode** | 廣告 macOS 裝置能力（system.run, camera 等） |

### 設計考量

- **輕量**：Menu bar app，不是完整 GUI 應用
- **Non-blocking**：所有 AI 操作在 Gateway 進行，App 只是 UI 殼
- **Code signing**：需要簽名才能保持 macOS TCC 權限

## 10.3 iOS Node

### 功能

| 功能 | 說明 |
|------|------|
| **裝置配對** | Bonjour 自動發現 + 手動配對 |
| **Voice Wake** | 語音喚醒詞 |
| **Talk Mode** | 連續語音對話 |
| **Canvas** | 視覺工作區呈現 |
| **Camera** | 拍照/錄影給 Agent |
| **Screen recording** | 螢幕錄製 |

### 通訊

```
iOS App ◄──── WebSocket ────▶ Gateway
         node.list / node.invoke
```

## 10.4 Android Node

### 功能

| 功能 | 說明 |
|------|------|
| **Connect tab** | 掃碼/手動配對 Gateway |
| **Chat sessions** | 直接在 App 內對話 |
| **Voice tab** | 語音輸入/輸出 |
| **Canvas** | 視覺工作區 |
| **Camera / Screen** | 拍照、螢幕錄製 |
| **裝置指令** | 通知、位置、SMS、照片、通訊錄、日曆、動態偵測、App 更新 |

### Android 特有能力

| 指令類別 | 說明 |
|----------|------|
| **notifications** | 讀取/管理系統通知 |
| **location** | 取得裝置位置 |
| **sms** | 讀取/發送 SMS |
| **photos** | 存取相簿 |
| **contacts** | 讀取通訊錄 |
| **calendar** | 存取行事曆 |
| **motion** | 動態偵測 |

### 設計考量

- **Android 生態優勢**：Android 允許更深層的系統整合
- **權限分級**：每個能力需要獨立的 Android 權限
- **Gateway 為主**：AI 推理在 Gateway，Android 只負責裝置操作

## 10.5 Voice 系統

### Voice Wake（喚醒詞）

```
[裝置 MIC] ──▶ 本地語音辨識 ──▶ 匹配喚醒詞 ──▶ 開始聆聽
```

- macOS / iOS 支援
- 本地處理，不上傳音訊
- 自訂喚醒詞

### Talk Mode（連續語音）

```
[使用者說話] ──▶ 語音轉文字 ──▶ Agent 處理 ──▶ 文字轉語音 ──▶ [播放回應]
```

- TTS 支援：ElevenLabs（高品質）+ 系統 TTS（免費）
- STT：Whisper（OpenAI）或 Sherpa-ONNX（本地）
- Android：連續語音模式

### 設計考量

1. **本地優先**：喚醒詞辨識在裝置端，不浪費 API 額度
2. **降級方案**：ElevenLabs 不可用時 fallback 到系統 TTS
3. **隱私**：喚醒詞偵測不上傳音訊

## 10.6 Shared 模組

```
apps/shared/
```

跨平台 App 共用的邏輯：
- WebSocket 連線管理
- 事件序列化/反序列化
- Canvas 渲染共用邏輯
- 配對流程

---

[← 總目錄](00-index.md) · [上一章](09-security.md) · [下一章：Extensions 與模型提供商 →](11-extensions.md)
