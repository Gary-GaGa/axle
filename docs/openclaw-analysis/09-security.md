# 09 — 安全模型與沙箱

[← 總目錄](00-index.md) · [上一章](08-memory.md) · [下一章 →](10-apps.md)

---

## 9.1 安全哲學

> *"Security in OpenClaw is a deliberate tradeoff: strong defaults without killing capability."*

OpenClaw 的安全設計核心原則：
- **預設安全**：未經設定的功能預設關閉
- **能力不減**：受信任的使用者可以完全解鎖
- **顯式 opt-in**：高風險操作需要使用者明確授權

## 9.2 威脅模型

### 主要威脅來源

| 威脅 | 說明 | 防護 |
|------|------|------|
| **未授權 DM** | 陌生人透過 IM 發訊息 | Pairing 機制 |
| **Prompt injection** | 外部內容注入惡意指令 | Input provenance 標記 |
| **工具濫用** | Agent 執行危險系統指令 | 沙箱 + elevated toggle |
| **API 濫用** | 未授權使用 LLM 額度 | DM allowlist + pairing |
| **資料外洩** | Agent 洩漏敏感資訊 | Session 隔離 + sandbox |

## 9.3 沙箱機制

### 預設行為

| Session 類型 | 執行環境 | 說明 |
|--------------|----------|------|
| **main** | Host（直接執行） | 個人助理模式，完全信任 |
| **non-main** | Docker 沙箱 | 群組/Channel session |

### 設定

```json5
{
  agents: {
    defaults: {
      sandbox: {
        mode: "non-main",   // "off" | "non-main" | "all"
      }
    }
  }
}
```

### 沙箱工具白名單

```
允許：bash, process, read, write, edit, sessions_list, sessions_history, sessions_send, sessions_spawn
禁止：browser, canvas, nodes, cron, discord, gateway
```

### Docker 沙箱架構

```
Gateway ──▶ Docker container (per-session)
            ├── 隔離的檔案系統
            ├── 限制的網路存取
            ├── bash 執行在容器內
            └── Agent 工具在容器內運行
```

### 設計考量

1. **main session 信任**：個人助理場景，main session 就是使用者自己，應該有完全權限
2. **非 main session 不信任**：群組中的訊息來自他人，可能包含攻擊
3. **per-session 隔離**：每個 session 一個容器，互不影響
4. **效能取捨**：Docker 啟動有延遲，但安全值得

## 9.4 Input Provenance

`src/sessions/input-provenance.ts` 負責追蹤輸入來源：

| 來源 | 信任等級 | 說明 |
|------|----------|------|
| **owner** | 高 | 裝置擁有者直接輸入 |
| **paired** | 中 | 已通過 pairing 驗證的用戶 |
| **group** | 低 | 群組中的成員 |
| **unknown** | 無 | 未驗證的來源 |

### 設計目的

- **分級信任**：不同來源的輸入有不同的權限
- **Prompt injection 防護**：低信任來源的內容會被標記為 untrusted
- **審計追蹤**：知道每一段輸入的來源

## 9.5 DM 安全政策

### 層級

```
1. Pairing（預設）─── 需要配對碼
     ↓ opt-in
2. Open ─────────── 接受所有 DM（需要明確設定 allowFrom: ["*"]）
```

### 分頻道控制

```json5
{
  channels: {
    telegram: { dmPolicy: "pairing" },
    discord:  { dmPolicy: "open", allowFrom: ["user123"] },
    slack:    { dmPolicy: "pairing" },
  }
}
```

### Doctor 檢查

`openclaw doctor` 會掃描：
- 是否有頻道設定為 open DM 但沒有 allowlist
- 是否有沙箱模式關閉但有群組連線
- 是否有過期的憑證

## 9.6 Secrets 管理

```
src/secrets/
```

- 憑證存在 `~/.openclaw/credentials/`
- 不進入版本控制
- Gateway 啟動時載入
- 支援 detect-secrets 掃描

## 9.7 macOS 權限模型

macOS App 以 node mode 運行時，權限透過 TCC 管理：

| 權限 | 用途 | 缺失時行為 |
|------|------|------------|
| Screen Recording | `system.run` + 螢幕擷取 | `PERMISSION_MISSING` |
| Camera | 拍照 | `PERMISSION_MISSING` |
| Notifications | 推送通知 | 失敗 |
| Accessibility | 進階自動化 | 受限 |

### 設計考量

- **遵循 OS 權限**：不繞過 macOS 的安全機制
- **明確失敗**：權限不足時回傳清楚的錯誤碼
- **需要簽名**：App 需要 code signing 才能持久保持權限

---

[← 總目錄](00-index.md) · [上一章](08-memory.md) · [下一章：原生 App →](10-apps.md)
