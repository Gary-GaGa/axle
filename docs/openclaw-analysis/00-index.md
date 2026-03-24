# 🦞 OpenClaw 架構深度分析 — 總目錄

> **版本基準**：openclaw/openclaw `main` branch（2026-03-19 snapshot）
> **License**：MIT
> **Stars**：323K+ · **語言**：TypeScript（核心）+ Swift/Kotlin（原生 App）

---

## 文件導覽

本分析從「入門概覽」逐步推進到「底層設計」，共分為 12 個章節。
每個章節獨立成檔，可按需閱讀。

| #  | 文件 | 主題 | 難度 |
|----|------|------|------|
| 01 | [01-overview.md](01-overview.md) | 專案概覽與設計哲學 | ⭐ |
| 02 | [02-architecture.md](02-architecture.md) | 整體架構與分層 | ⭐⭐ |
| 03 | [03-gateway.md](03-gateway.md) | Gateway 控制平面 | ⭐⭐ |
| 04 | [04-channels.md](04-channels.md) | 多頻道訊息整合 | ⭐⭐ |
| 05 | [05-agent-runtime.md](05-agent-runtime.md) | Agent 執行時與 Session 模型 | ⭐⭐⭐ |
| 06 | [06-tools.md](06-tools.md) | 工具系統：Browser / Canvas / Nodes | ⭐⭐⭐ |
| 07 | [07-skills-plugins.md](07-skills-plugins.md) | Skills 與 Plugin 生態系 | ⭐⭐ |
| 08 | [08-memory.md](08-memory.md) | 記憶子系統 | ⭐⭐⭐ |
| 09 | [09-security.md](09-security.md) | 安全模型與沙箱 | ⭐⭐⭐ |
| 10 | [10-apps.md](10-apps.md) | 原生 App：macOS / iOS / Android | ⭐⭐ |
| 11 | [11-extensions.md](11-extensions.md) | Extensions 與模型提供商 | ⭐⭐⭐ |
| 12 | [12-infra-ops.md](12-infra-ops.md) | 基礎設施、部署與維運 | ⭐⭐ |

---

## 閱讀建議

- **快速了解**：01 → 02 → 03
- **開發者導向**：01 → 02 → 05 → 06 → 07
- **安全 / 營運**：09 → 12 → 03
- **與 Axle 對照**：01 → 04 → 05 → 08（已有 Axle 對應功能的領域）

---

## 參考來源

- GitHub：[openclaw/openclaw](https://github.com/openclaw/openclaw)
- 官方文件：[docs.openclaw.ai](https://docs.openclaw.ai)
- DeepWiki：[deepwiki.com/openclaw/openclaw](https://deepwiki.com/openclaw/openclaw)
- Vision：[VISION.md](https://github.com/openclaw/openclaw/blob/main/VISION.md)
- Contributing：[CONTRIBUTING.md](https://github.com/openclaw/openclaw/blob/main/CONTRIBUTING.md)
