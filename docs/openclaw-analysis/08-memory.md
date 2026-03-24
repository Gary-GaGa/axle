# 08 — 記憶子系統

[← 總目錄](00-index.md) · [上一章](07-skills-plugins.md) · [下一章 →](09-security.md)

---

## 8.1 記憶的定位

Memory 在 OpenClaw 中是一個**特殊的 plugin slot**——同一時間只能有一個 memory backend 活躍。

### 為何需要記憶

- **Session 會 compact/reset**：對話歷史會被壓縮或清除
- **跨 session 知識**：Agent 需要記住使用者偏好、重要事實
- **長期學習**：助理應該越用越懂你

## 8.2 記憶架構

```
src/memory/
├── backend-config.test.ts      # Backend 設定測試
├── ...（30+ 個檔案）
```

```
extensions/
├── memory-core/                # 核心記憶 backend（檔案系統）
└── memory-lancedb/             # LanceDB 向量記憶 backend
```

### 記憶 Backend 比較

| Backend | 儲存方式 | 搜尋方式 | 適用場景 |
|---------|----------|----------|----------|
| **memory-core** | 本地檔案 | 關鍵字 / 時間排序 | 輕量、無依賴 |
| **memory-lancedb** | LanceDB 向量資料庫 | 語意向量搜尋 | 大量記憶 + 語意查詢 |

## 8.3 記憶操作

Agent 與記憶的互動：

| 操作 | 說明 |
|------|------|
| **自動存入** | Agent 對話中偵測到重要資訊時自動記住 |
| **手動存入** | Agent 被明確要求記住某事 |
| **上下文注入** | 每次推理前，從記憶中檢索相關片段加入 prompt |
| **搜尋** | Agent 可以主動搜尋記憶 |

### Prompt 中的記憶

```
[System prompt]
[AGENTS.md / SOUL.md / TOOLS.md]
[Skills]
[Memory: relevant entries]    ← 記憶注入點
[Session history]
[User message]
```

## 8.4 設計考量

### 8.4.1 Plugin Slot 模式

為何 memory 是 plugin slot 而非核心功能？

1. **實驗空間**：不同的記憶策略（檔案 vs 向量 vs 圖資料庫）各有優劣
2. **使用者選擇**：使用者可以選擇最適合的 backend
3. **收斂計劃**：長期計劃收斂到一個推薦方案

### 8.4.2 自動 vs 手動

- 自動存入降低使用者負擔
- 但需要「值得記住」的判斷邏輯，避免記住垃圾
- 手動存入是 fallback，確保重要事項不漏

### 8.4.3 隱私考量

- 記憶儲存在本地（~/.openclaw/）
- 不經過第三方伺服器
- 使用者可以隨時清除特定記憶

## 8.5 與 Axle 記憶系統的對比

| 面向 | OpenClaw | Axle |
|------|----------|------|
| 儲存 | 本地檔案 / LanceDB | JSON 檔案 (~/.axle/memory/) |
| 搜尋 | 關鍵字 / 向量語意搜尋 | 關鍵字匹配 + 時間排序 |
| 架構 | Plugin slot（可替換 backend） | Domain 層（固定 JSON backend） |
| Prompt 注入 | 自動檢索 + 注入 | 過濾後 fenced 注入 |
| 安全 | — | Untrusted fencing + rune-safe 截斷 |
| 並發 | — | Per-user 鎖 + 原子寫入 |

> 參考：[VISION.md — Plugins & Memory](https://github.com/openclaw/openclaw/blob/main/VISION.md)

---

[← 總目錄](00-index.md) · [上一章](07-skills-plugins.md) · [下一章：安全模型 →](09-security.md)
