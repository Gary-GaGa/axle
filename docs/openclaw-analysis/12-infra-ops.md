# 12 — 基礎設施、部署與維運

[← 總目錄](00-index.md) · [上一章](11-extensions.md)

---

## 12.1 安裝方式

| 方式 | 指令 | 適用場景 |
|------|------|----------|
| **npm（推薦）** | `npm install -g openclaw@latest` | 個人裝置 |
| **pnpm** | `pnpm add -g openclaw@latest` | 偏好 pnpm 的開發者 |
| **Docker** | `docker compose up` | 伺服器部署 |
| **Nix** | `nix run github:openclaw/nix-openclaw` | 聲明式設定 |
| **From source** | `git clone` + `pnpm build` | 開發/貢獻 |

### Onboarding

```bash
openclaw onboard --install-daemon
```

互動式引導：
1. 設定 Gateway port 和 bind
2. 選擇和設定模型提供商
3. 連接通訊頻道
4. 安裝 daemon（launchd/systemd）

## 12.2 Docker 部署

### Dockerfile 分層

| Dockerfile | 用途 |
|------------|------|
| `Dockerfile` | 主 Gateway image |
| `Dockerfile.sandbox` | Per-session 沙箱 image |
| `Dockerfile.sandbox-browser` | 含 Chrome 的沙箱 image |
| `Dockerfile.sandbox-common` | 沙箱共用 base image |

### docker-compose.yml

```yaml
services:
  openclaw:
    build: .
    ports:
      - "18789:18789"
    volumes:
      - ~/.openclaw:/root/.openclaw
    environment:
      - TELEGRAM_BOT_TOKEN=...
```

### 設計考量

- **分層 image**：沙箱 image 獨立，避免主 image 過大
- **Volume mapping**：設定和憑證透過 volume 掛載
- **環境變數**：敏感資訊透過環境變數傳入

## 12.3 發行頻道

| 頻道 | 說明 | npm dist-tag |
|------|------|-------------|
| **stable** | 正式版（`vYYYY.M.D`） | `latest` |
| **beta** | 預覽版（`vYYYY.M.D-beta.N`） | `beta` |
| **dev** | 開發版（main HEAD） | `dev` |

### 切換頻道

```bash
openclaw update --channel stable|beta|dev
```

### 版本命名

```
v2026.3.19          # stable
v2026.3.19-beta.3   # beta
v2026.3.19-1        # patch release
```

## 12.4 Doctor 診斷

```bash
openclaw doctor
```

Doctor 會檢查：

| 檢查項目 | 說明 |
|----------|------|
| **Config 驗證** | 設定格式和必要欄位 |
| **安全掃描** | DM 政策、沙箱狀態 |
| **Channel 連線** | 各頻道的連線狀態 |
| **權限檢查** | macOS TCC 權限 |
| **版本相容** | Node.js 版本、依賴版本 |
| **Migration** | 需要的設定遷移 |

### 設計目的

- **自助診斷**：使用者不需要去 Discord 問問題
- **Migration 自動化**：版本升級後自動提示需要的變更
- **安全警示**：主動提醒不安全的設定

## 12.5 Control UI

Gateway 內建提供 Web UI：

| 功能 | 說明 |
|------|------|
| **Dashboard** | Gateway 狀態總覽 |
| **Sessions** | 活躍 session 管理 |
| **Config** | 設定瀏覽/編輯 |
| **Logs** | 即時日誌 |
| **WebChat** | 內嵌 WebChat 介面 |

### 技術選型

- **Lit**（Web Components）：輕量、原生 DOM
- **Vite**：開發/建置工具
- **Legacy decorators**：Lit 搭配 legacy decorator 模式

### 設計考量

- **Gateway 直接服務**：不需要額外的 web server
- **即時更新**：透過 WebSocket 即時顯示狀態變更
- **輕量**：Lit 比 React/Vue 更小，適合嵌入式 UI

## 12.6 Logging 與可觀察性

```
src/logging/
src/logger.ts
```

| 機制 | 說明 |
|------|------|
| **結構化日誌** | JSON 格式 |
| **日誌等級** | debug / info / warn / error |
| **OpenTelemetry** | 可選的 diagnostics-otel extension |
| **Usage tracking** | Token 用量追蹤 |

## 12.7 CI/CD

| 工具 | 用途 |
|------|------|
| **GitHub Actions** | CI 主力 |
| **Blacksmith** | 贊助商提供的 CI 基礎設施 |
| **Vitest** | 單元/整合/E2E 測試 |
| **oxlint** | Linting |
| **Prettier** | 格式化 |
| **detect-secrets** | Secret 掃描 |
| **ShellCheck** | Shell script 品質 |
| **knip** | Dead code 檢測 |
| **jscpd** | 重複程式碼檢測 |
| **markdownlint** | Markdown 格式化 |
| **pre-commit** | Git hook |

### 測試分層

```bash
pnpm test                     # 全部測試
pnpm test:unit                # 單元測試
pnpm test:extensions          # Extension 測試
pnpm test:channels            # Channel 測試
pnpm test:contracts           # Contract 測試
pnpm test:extension <name>    # 單一 extension
pnpm test:e2e                 # E2E 測試
```

## 12.8 Remote Gateway（Linux 部署）

Gateway 可以跑在遠端 Linux 伺服器，Client 透過以下方式連接：

```
[macOS App / CLI] ──── Tailscale / SSH tunnel ────▶ [Linux Gateway]
                                                         │
                                                         ├── Channel 連線
                                                         ├── Bash 執行
                                                         └── Agent 推理

[iOS / Android Node] ──── WebSocket ────▶ [Linux Gateway]
                                                │
                                                └── node.invoke 回傳裝置操作
```

### 設計考量

- **Gateway = 重計算**：跑在有算力的機器上
- **Node = 裝置操作**：只在需要裝置功能時才呼叫
- **Always-on**：Linux 伺服器比筆電更適合 24/7 運行
- **Fly.io 支援**：內建 `fly.toml` + `render.yaml`

## 12.9 Nix 支援

```bash
nix run github:openclaw/nix-openclaw
```

- **聲明式**：Nix flake 定義完整環境
- **可重現**：確保每次建置結果一致
- **NixOS module**：可以直接加入 NixOS 設定

---

[← 總目錄](00-index.md) · [上一章](11-extensions.md)
