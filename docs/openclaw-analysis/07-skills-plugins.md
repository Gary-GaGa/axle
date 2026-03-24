# 07 — Skills 與 Plugin 生態系

[← 總目錄](00-index.md) · [上一章](06-tools.md) · [下一章 →](08-memory.md)

---

## 7.1 Skills vs Plugins

OpenClaw 有兩種擴展機制，定位不同：

| 面向 | Skills | Plugins |
|------|--------|---------|
| **本質** | Prompt 片段（Markdown） | 程式碼模組（TypeScript） |
| **載入方式** | 注入到 system prompt | 註冊為 tool / hook |
| **建立難度** | 低（只需寫 Markdown） | 高（需要寫 TypeScript） |
| **能力範圍** | 引導 Agent 行為 | 新增工具、修改流程 |
| **分發** | ClawHub / workspace | npm package / 本地 |
| **核心收錄** | 門檻高，優先 ClawHub | 門檻極高 |

## 7.2 Skills 系統

### Skill 結構

```
skills/<skill-name>/
└── SKILL.md          # Skill 描述 + 使用指引
```

SKILL.md 範例：

```markdown
---
name: weather
description: Get current weather for any location
tools: ["bash"]
---

# Weather Skill

When the user asks about weather, use `curl` to query wttr.in:

\`\`\`bash
curl -s "wttr.in/<location>?format=3"
\`\`\`
```

### Skill 分類

| 類別 | 說明 | 範例 |
|------|------|------|
| **Bundled** | 隨 OpenClaw 安裝的內建 skill | weather, github, coding-agent |
| **Managed** | 透過 ClawHub 安裝的社群 skill | — |
| **Workspace** | 使用者在 workspace 內自訂的 skill | — |

### 52 個內建 Skills（精選）

| Skill | 功能 |
|-------|------|
| `github` | GitHub 操作（PR、issue、code review） |
| `coding-agent` | 程式碼開發輔助 |
| `obsidian` | Obsidian 筆記整合 |
| `notion` | Notion 整合 |
| `spotify-player` | Spotify 控制 |
| `canvas` | Canvas 視覺工作區操作 |
| `clawhub` | 搜尋/安裝 ClawHub skills |
| `skill-creator` | 協助建立新 skill |
| `nano-pdf` | PDF 讀取/處理 |
| `weather` | 天氣查詢 |
| `tmux` | Tmux session 管理 |
| `trello` | Trello 看板操作 |

### 設計考量

1. **低門檻**：任何人都可以用 Markdown 寫 skill，不需要寫程式
2. **Prompt 注入**：Skill 的內容直接成為 system prompt 的一部分
3. **工具限定**：Skill 可以宣告它需要哪些工具（`tools: ["bash", "browser"]`）
4. **ClawHub 優先**：新 skill 應該先發布到 ClawHub，不進核心

## 7.3 ClawHub

ClawHub 是 OpenClaw 的 **Skill 社群 registry**：

- 網址：[clawhub.com](https://clawhub.com)
- Agent 可以自動搜尋和安裝 skill
- 內建 `clawhub` skill 提供搜尋/安裝功能
- awesome-openclaw-skills（39K+ stars）收錄了 5,400+ 個社群 skill

### 設計目的

- **生態系**：鼓勵社群貢獻，降低進入核心的壓力
- **自動發現**：Agent 可以自動找到需要的能力
- **品質控制**：核心只收錄少數高品質 skill

## 7.4 Plugin 系統

### Plugin 載入

```
src/plugins/
├── build-smoke-entry.ts    # 建置驗證
├── ...（40+ 個檔案）
```

Plugin 透過 npm 包分發，在 Gateway 啟動時載入。

### Plugin API

- **Tool registration**：註冊新的 Agent 工具
- **Hook registration**：掛接 lifecycle event（message received、response sent 等）
- **Memory slot**：Memory 是特殊的 plugin slot（只能有一個 active）

### 設計考量

1. **核心精簡**：選擇性功能應該是 plugin，不是核心
2. **獨立維護**：Plugin 在自己的 repo 維護，不增加核心負擔
3. **明確邊界**：Plugin SDK 定義了 plugin 能做什麼和不能做什麼

## 7.5 MCP 整合

OpenClaw 不直接內建 MCP runtime，而是透過 **mcporter** 橋接：

```
Agent ──▶ mcporter bridge ──▶ MCP Server
```

> 參考：[github.com/steipete/mcporter](https://github.com/steipete/mcporter)

### 為何不直接整合

- **解耦**：MCP 規範變動快，橋接模式避免核心被拖著改
- **靈活**：隨時加減 MCP server，不需重啟 Gateway
- **穩定**：MCP churn 不影響核心穩定性

---

[← 總目錄](00-index.md) · [上一章](06-tools.md) · [下一章：記憶子系統 →](08-memory.md)
