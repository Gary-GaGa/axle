# Axle Clean Architecture + DDD 重構方向

## 目的

本文件定義 Axle 從目前的「Telegram Handler + skill + app manager」混合式結構，逐步重構為符合 **Clean Architecture** 與 **Domain-Driven Design (DDD)** 概念的方向。

目標不是一次性推翻現有功能，而是：

- 保留既有功能與使用方式
- 以低風險、可回退的方式分批遷移
- 讓核心規則脫離 Telegram / Web / CLI / OS I/O 細節
- 建立可測試、可擴充、可替換 adapter 的結構

---

## 現況摘要

目前 Axle 主要由下列幾個區塊組成：

- `internal/bot/handler/`：Telegram 介面、互動流程、狀態切換、任務啟動
- `internal/bot/skill/`：檔案、Shell、Copilot、GitHub、Browser 等能力實作
- `internal/app/`：Session、TaskManager、Memory、Workflow、Scheduler、SubAgent、RPG、Plugin 等狀態與服務
- `internal/web/`：本地 Web Gateway 與 RPG Dashboard

這個結構對功能擴張很有效率，但也有幾個問題：

- `handler` 直接協調大量業務規則與外部操作
- `app` 同時承擔 domain 規則、application orchestration、stateful manager
- `skill` 混合了 domain rule、usecase、infrastructure adapter
- 核心概念（session、task slot、安全分級、workspace 邊界）沒有明確的 domain 邊界
- 很多邏輯只能從 Telegram / Web 路徑間接驗證

---

## Axle 的領域劃分（Bounded Contexts）

以下是目前最合理的初始 bounded contexts：

### 1. Interaction

負責對話互動狀態與使用者工作上下文：

- Session mode
- pending input / pending command / pending compose state
- selected model
- active workspace
- enabled extras

### 2. Execution

負責「能不能執行」與「目前是否可執行」的規則：

- 單任務槽位（single task slot）
- Shell 指令安全分級
- 執行前確認策略

### 3. Workspace

負責目前 workspace 範圍內的檔案與代碼探索：

- path sandbox / path normalization
- read code
- list directory
- search code
- write file（已在 follow-up slice 納入）

### 4. Memory

負責長期記憶與檢索：

- memory entry
- search ranking
- recent context
- RAG recall context

### 5. Workflow

負責背景工作流的生命週期與步驟依賴：

- workflow aggregate
- workflow step
- capacity rule
- step execution orchestration

### 6. Automation

負責長時任務與週期性任務：

- sub-agent
- schedule / periodic execution

### 7. Plugin

負責使用者自定義技能與插件清單：

- plugin definition
- plugin loading / validation

### 8. Gamification

負責 RPG/XP 事件與狀態：

- XP
- level tier
- event log
- achievement snapshot

---

## 目標分層

依照 Clean Architecture，新的核心結構採用：

```text
internal/
  domain/
    <context>/
      entity / aggregate / value object / repository interface

  usecase/
    port/
      in/
    dto/
    <context>/
      service.go

  interface/
    in/
      telegram/
      web/
    out/
      persistence/
      tooling/

  infrastructure/
    files/
    logger/
    config/
```

### 依賴方向

- `domain` 只能依賴 Go standard library
- `usecase` 只能依賴 `domain`
- `interface` 依賴 `usecase` / `domain`
- `infrastructure` 提供共用技術細節，不放業務規則

---

## 具體存放方式

### Domain 層

```text
internal/domain/
  interaction/
  execution/
  workspace/
  memory/
  workflow/
  automation/
  plugin/
  gamification/
```

### Usecase 層

```text
internal/usecase/
  port/in/
  dto/
  interaction/
  execution/
  workspace/
  memory/
  workflow/
  automation/
  plugin/
  gamification/
```

### Interface Adapter 層

```text
internal/interface/
  in/
    telegram/
    web/
  out/
    persistence/
      json/
    tooling/
      shell/
      copilot/
      browser/
```

### Infrastructure 層

```text
internal/infrastructure/
  files/
  logger/
  config/
```

---

## 遷移原則

### 原則 1：先抽核心，再改入口

第一階段不直接大拆 `internal/bot/handler/text.go`、`callback.go`、`internal/web/gateway.go`。  
先把核心規則抽到 `domain/usecase`，再讓舊 handler 轉成薄包裝。

### 原則 2：保留相容 API

現有 `internal/app` 與 `internal/bot/skill` 中被大量使用的型別或函式，初期保留原檔案與原函式名稱，改成 thin wrapper / facade，降低風險。

### 原則 3：每次只搬一小塊

避免一次遷移整個 workflow / gateway / self-upgrade。  
優先搬移：

1. Interaction
2. Execution
3. Workspace（read-only slice）
4. Workspace（writable slice）
5. Memory
6. Workflow（foundation slice）

### 原則 4：所有新核心邏輯都要有 unit test

至少補上：

- domain rule tests
- usecase service tests
- compatibility wrapper smoke tests（必要時）

### 原則 5：不碰無關既有變更

這次重構不主動處理目前 worktree 中既有的：

- `cmd/foodsafety/main.go`
- `go.mod`
- `go.sum`

除非與本次重構直接相關，否則不修改、不回滾。

---

## 第一批遷移範圍

### Slice A：Interaction Foundation

搬移內容：

- `internal/app/session.go`

目標：

- 建立 `internal/domain/interaction`
- 將 session mode、user session、session manager 放入 domain
- `internal/app/session.go` 改為 compatibility facade

### Slice B：Execution Foundation

搬移內容：

- `internal/app/taskmanager.go`
- `internal/bot/skill/safety.go`

目標：

- 建立 `internal/domain/execution`
- 將 task slot 與 command safety rule 放入 domain
- 原本 `app/` 與 `skill/` 路徑保留 wrapper

### Slice C：Workspace Read-only

搬移內容：

- `internal/bot/skill/readcode.go`
- `internal/bot/skill/listdir.go`
- `internal/bot/skill/search.go`

目標：

- 建立 `internal/domain/workspace`
- 建立 `internal/usecase/workspace`
- 建立 `internal/interface/out/persistence/localfs`
- 舊的 `skill` 函式保留 wrapper，讓 handler 幾乎不用改

### Slice D：Workspace Writable

搬移內容：

- `internal/bot/skill/writefile.go`
- writable path existence check
- rooted local filesystem write adapter

目標：

- 將 writable workspace flow 併入 `internal/domain/workspace`
- 讓 `internal/usecase/workspace` 同時支援 exists/write
- 保留 `skill.WriteFile` / `skill.FileExists` 相容 wrapper
- 延續 rooted file handling、symlink/hard-link/FIFO 防護

### Slice E：Memory

搬移內容：

- `internal/app/memory.go`
- per-user memory cache
- JSON persistence under `~/.axle/memory`
- lexical search / recent context / RAG context rules

目標：

- 建立 `internal/domain/memory`
- 建立 `internal/usecase/memory`
- 建立 `internal/interface/out/persistence/json`
- 保留 `app.MemoryStore` / `app.MemoryEntry` / `app.MemorySearchHit` 相容 facade
- 讓 Telegram / Web / Workflow 等既有呼叫端不用同步大改

### Slice F：Workflow Foundation

搬移內容：

- workflow status / step status / capacity / dependency rules
- workflow planner prompt / fallback / JSON plan parsing
- single-step execution orchestration

目標：

- 建立 `internal/domain/workflow`
- 建立 `internal/usecase/workflow`
- 保留 `app.WorkflowManager` 為相容 runtime shell
- 先不大拆 goroutine / cancellation / persistence / notice adapter
- 讓 `internal/app/workflow.go` 開始委派給新的 domain/usecase 核心

---

## 暫緩項目

以下項目不作為第一批遷移主體：

- `internal/bot/handler/text.go`
- `internal/bot/handler/callback.go`
- `internal/web/gateway.go`
- `internal/bot/skill/selfupgrade.go`

原因不是它們不重要，而是：

- 相依面大
- 風險高
- 一次搬動太容易引發功能退化

等第一批新架構穩定後，再進行第二階段。

---

## 第二階段預計方向

當第一批完成後，下一步會是：

1. 將 `Workflow` 的 persistence / runtime shell / output port 再往內收斂
2. 將 `handler/text.go` 改為真正的 input adapter
3. 將 `internal/web` 與 `internal/bot/handler` 各自轉為 `interface/in` adapter
4. 逐步拆出 Automation / Plugin / Gamification context

---

## 完成判準

本次第一階段完成時，應滿足：

- repo 內已有明確的 `domain/usecase/interface/infrastructure` 目錄
- Interaction / Execution / Workspace(read-only + writable) / Memory / Workflow foundation 已遷入新結構
- 舊 API 仍可用
- 相關 unit tests 存在且通過
- 既有 Telegram / Web 主要行為不被破壞

---

## 補充說明

這份重構不是為了「看起來像 DDD」，而是為了讓 Axle 的核心概念能夠：

- 脫離 Telegram 與 Web 入口獨立演化
- 更容易在未來加入 CLI / MCP / 多通道 adapter
- 降低大型 handler 與 manager 的耦合
- 讓每個 bounded context 的責任更清楚

第一階段的重點是 **建立正確的骨架與遷移模式**，而不是一次把所有程式碼都搬完。
