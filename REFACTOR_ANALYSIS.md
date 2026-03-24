# AXLE Clean Architecture + DDD Refactor Analysis

**Generated:** 2024  
**Repository:** /Users/gary/Documents/project/gary/axle  
**Codebase Size:** ~8,600 LOC (internal/)  
**Current Architecture:** Hybrid layered (no domain boundaries)

---

## QUICK SUMMARY

The Axle codebase has **7 natural bounded contexts** (Execution, Memory, Workflow, Agent, Gamification, Plugin, Code Browsing) currently merged into `internal/app/` with no clear separation. The presentation layer is monolithic: `callback.go` (1,664 LOC) + `text.go` (727 LOC) route all Telegram interactions through a god object (`handler.Hub`). 

**Refactor Goal:** Extract domains into `internal/domain/*`, introduce ports/interfaces, split handlers by domain, and create a testable application service layer with command/query bus.

**Timeline:** 20 weeks (5 phases), 1-2 developers, ~400-600 LOC/week, all tests preserved throughout.

---

## 1. CURRENT ARCHITECTURE

### Directory Structure
```
internal/
├── app/             [Mixed state + business logic - 18 files]
│   ├── taskmanager.go      (Task execution - 70 LOC logic)
│   ├── memory.go           (User history - 440 LOC)
│   ├── session.go          (Session state - 119 LOC)
│   ├── workflow.go         (Background tasks - 870 LOC) ⭐ LARGEST
│   ├── scheduler.go        (Cron scheduling - 205 LOC)
│   ├── subagent.go         (Delegated tasks - 193 LOC)
│   ├── rpg.go              (Gamification - 363 LOC)
│   ├── plugin.go           (User plugins - 146 LOC)
│   ├── version.go          (Constant - 1 LOC)
│   ├── identity.go         (User ID - 22 LOC)
│   └── fileutil.go         (Utility - 30 LOC)
│
├── bot/
│   ├── handler/            [Telegram callbacks - 3,158 LOC]
│   │   ├── callback.go     (76 handlers - 1,664 LOC) ⭐ MONOLITH
│   │   ├── text.go         (State machine - 727 LOC) ⭐ MONOLITH
│   │   ├── hub.go          (Coordinator - 358 LOC)
│   │   ├── menu.go         (UI buttons - 381 LOC)
│   │   └── start.go        (Init - 28 LOC)
│   │
│   ├── skill/              [Utilities - 3,904 LOC]
│   │   ├── exec.go         (Shell execution - 81 LOC)
│   │   ├── copilot.go      (AI chat - 163 LOC)
│   │   ├── web.go          (Fetch/search - 210 LOC)
│   │   ├── email.go        (SMTP/IMAP - 227 LOC)
│   │   ├── browser.go      (Automation - 484 LOC)
│   │   ├── git.go          (Git ops - 69 LOC)
│   │   ├── github.go       (GitHub API - 76 LOC)
│   │   ├── search.go       (Code search - 141 LOC)
│   │   ├── listdir.go      (File listing - 118 LOC)
│   │   ├── readcode.go     (File reading - 43 LOC)
│   │   ├── pdf.go          (PDF ops - 45 LOC)
│   │   ├── image.go        (Image ops - 74 LOC)
│   │   ├── calendar.go     (Calendar - 100 LOC)
│   │   ├── briefing.go     (News digest - 56 LOC)
│   │   ├── selfupgrade.go  (Self-upgrade - 209 LOC)
│   │   ├── models.go       (Model catalog - 112 LOC)
│   │   ├── safety.go       (Command safety - 64 LOC)
│   │   └── [test files]
│   │
│   └── middleware/
│       └── auth.go         (User whitelist)
│
├── web/                    [HTTP adapter - 650 LOC]
│   ├── server.go           (HTTP server - 267 LOC)
│   ├── gateway.go          (WebSocket API - 380 LOC)
│   └── static/             (HTML/CSS/JS)
│
└── pkg/                    [Unused - empty]
```

### Key Observations

**Problems:**
1. **No Domain Boundaries** — Everything in `app/` is equally important; no aggregates or subdomains
2. **Hub God Object** — `handler.Hub` holds 12+ dependencies; coordinates all domain calls; 358 LOC
3. **Monolithic Handlers** — callback.go (1,664) + text.go (727) = 38% of handler code in 2 files
4. **Mixed Concerns in app/** — State managers (MemoryStore, ScheduleManager) + business logic mixed
5. **No Interface/Port Pattern** — Direct struct composition; hard to test; hard to swap implementations
6. **Skill Functions Not Domain Objects** — ExecShell, WebFetch, SendEmail are utilities, not entities
7. **State Machine Without Abstraction** — text.go switch/case on Mode enum; no pattern

**Strengths:**
- ✅ Clear separation: bot/ (presentation) vs app/ (logic)
- ✅ Good test coverage (many `_test.go` files)
- ✅ Stateless skills; easy to test utility functions
- ✅ Minimal external dependencies (Telebot, Viper, gorilla/websocket)

---

## 2. DOMAIN GROUPINGS

### The 7 Bounded Contexts

#### **EXECUTION CONTEXT** (Execute shell commands safely)
- **Files:** `app/taskmanager.go` (~70 LOC logic), `skill/exec.go` (81 LOC), `skill/safety.go` (64 LOC)
- **Aggregate Root:** `TaskManager` (currently in app/)
- **Key Entities:** PendingCmd, ExecutionResult, SafetyLevel
- **Business Rules:**
  - One task at a time per user (TaskManager.TryStart)
  - Dangerous commands require secondary confirmation
  - Output capped at 1 MB
  - Timeout: 60 seconds
- **Current Handler Touch Points:** callback.go (~164), text.go (~177)
- **Invariants:** No concurrent execution per user

#### **MEMORY/HISTORY CONTEXT** (Store & retrieve user conversation history)
- **Files:** `app/memory.go` (440 LOC), `app/session.go` (119 LOC)
- **Aggregate Root:** `MemoryStore` + `UserSession`
- **Key Entities:** MemoryEntry, MemorySearchHit, UserSession
- **Business Rules:**
  - Per-user memory isolation
  - Search ranking by relevance + recency
  - Automatic content truncation (4,000 chars)
  - Session state machine (Mode enum: 36 variants)
  - Persistent JSON file per user
- **Current Handler Touch Points:** text.go (~336 for search), callback.go (~252 for memory buttons)
- **Invariants:** Memory entries immutable once added (append-only log)

#### **WORKFLOW CONTEXT** (Multi-step background task orchestration)
- **Files:** `app/workflow.go` (870 LOC) ⭐ **LARGEST SINGLE FILE**
- **Aggregate Root:** `WorkflowManager` + `Workflow`
- **Key Entities:** Workflow, WorkflowStep, WorkflowStatus, WorkflowNotice
- **Business Rules:**
  - Max 3 active workflows per user, 8 total
  - Steps can have dependencies (wait for step X before running Y)
  - Two step types: "copilot" (AI) or "browser" (automation)
  - Persistent JSON file (`~/.axle/workflows.json`)
  - Status lifecycle: Planning → Running → Completed/Failed/Cancelled
  - Results stored; summary generated via Copilot
- **Current Handler Touch Points:** callback.go (~261 for workflow buttons), text.go (~590)
- **Invariants:** Step execution order respects dependencies

#### **AGENT CONTEXT** (Delegated background tasks + periodic scheduling)
- **Files:** `app/subagent.go` (193 LOC), `app/scheduler.go` (205 LOC)
- **Aggregate Roots:** `SubAgentManager`, `ScheduleManager`
- **Key Entities:** SubAgent, Schedule
- **Business Rules:**
  - SubAgent: async task delegation to Copilot (streaming)
  - Scheduler: cron-like periodic execution
  - Scheduling interval: 1+ minutes
  - Commands executed in workspace; output sent to all users
  - Persistent schedules JSON file
- **Current Handler Touch Points:** callback.go (~247 for sub-agents, ~273 for scheduler)
- **Invariants:** Schedules run exactly once per interval; no concurrent executions

#### **GAMIFICATION CONTEXT** (XP, levels, achievements)
- **Files:** `app/rpg.go` (363 LOC)
- **Aggregate Root:** `RPGManager`
- **Key Entities:** RPGEvent, RPGProfile, LevelInfo, RPGSkillDef
- **Business Rules:**
  - 23 skills with XP rewards
  - 6 level tiers with cumulative XP thresholds
  - XP awards on skill success only
  - Web dashboard displays stats + sprites
  - Persistent JSON profile per user
- **Current Handler Touch Points:** callback.go (~60 for status display)
- **Invariants:** XP is monotonic (never decreases)

#### **PLUGIN CONTEXT** (User-defined YAML skills)
- **Files:** `app/plugin.go` (146 LOC)
- **Aggregate Root:** `PluginManager`
- **Key Entities:** Plugin (YAML config)
- **Business Rules:**
  - Plugins are YAML files in `~/.axle/plugins/`
  - Execute arbitrary shell commands
  - Optional confirmation gate
  - Can run in workspace context
- **Current Handler Touch Points:** callback.go (~267 for plugin execution)
- **Invariants:** None (simple YAML loader + executor)

#### **CODE BROWSING CONTEXT** (File operations: read, list, search)
- **Files:** `skill/readcode.go` (43 LOC), `skill/listdir.go` (118 LOC), `skill/search.go` (141 LOC)
- **Aggregate Root:** None (utility functions)
- **Key Entities:** FileInfo, SearchResult
- **Business Rules:**
  - Relative paths resolved in workspace
  - Binary files detected and skipped
  - Search spans entire workspace with pattern matching
  - Response chunked for Telegram limits (4,000 chars)
- **Current Handler Touch Points:** text.go (~177 for read, ~455 for list, ~480 for search)
- **Invariants:** Read-only operations; no mutations

---

### Cross-Cutting Infrastructure

| Layer | Files | Purpose | LOC |
|-------|-------|---------|-----|
| **Utilities/Skills** | `skill/exec.go`, `web.go`, `email.go`, `git.go`, `github.go`, `browser.go`, `pdf.go`, `image.go`, `calendar.go`, `briefing.go`, `selfupgrade.go`, `safety.go` | Stateless operations + integrations | ~3,800 |
| **Models & Catalog** | `skill/models.go` | Copilot model definitions; UI formatting | 112 |
| **Presentation** | `handler/*.go` | Telegram button callbacks; state routing | 3,158 |
| **Web API** | `web/*.go` | HTTP server + WebSocket gateway | 650 |
| **Configuration** | `configs/*.go` | Viper + local credential store | ~150 |
| **Authentication** | `middleware/auth.go` | User whitelist validation | ~20 |

---

## 3. CLASSIFICATION: DOMAIN vs INFRASTRUCTURE

### BOUNDED CONTEXTS (→ become `internal/domain/*`)
```
Execution       ✓ Business logic: safety validation, timing, concurrency
Memory          ✓ Business logic: ranking, persistence, lifecycle
Workflow        ✓ Business logic: orchestration, dependencies, status
Agent           ✓ Business logic: delegation, scheduling
Gamification    ✓ Business logic: XP calculation, leveling
Plugin          ✓ Business logic: loading, execution, confirmation
CodeBrowsing    ✓ Business logic: file ops, search, formatting
```

### INFRASTRUCTURE & ADAPTERS (→ become `internal/adapters/*`)
```
Handler/Hub         ✓ Adapter: translates Telegram → commands
Skill functions     ✓ Adapter: wraps external tools (shell, web, email, etc.)
Web Gateway         ✓ Adapter: HTTP API adapter
Configuration       ✓ Infrastructure: loads env + store
Authentication      ✓ Infrastructure: policy enforcement
```

---

## 4. INCREMENTAL MIGRATION ROADMAP (20 Weeks)

### **PHASE 1: FOUNDATION (Weeks 1-2) — Risk: 🟩 LOW**

**Objective:** Establish folder structure and create abstraction layer.

**Tasks:**

1. Create folder structure:
   ```
   internal/
   ├── domain/
   │   ├── execution/
   │   ├── memory/
   │   ├── workflow/
   │   ├── agent/
   │   ├── gamification/
   │   ├── plugin/
   │   ├── codebrowsing/
   │   └── shared/           (value objects, events, errors)
   ├── ports/                (interface abstractions)
   ├── application/
   │   ├── command/
   │   ├── query/
   │   └── bus.go
   ├── adapters/
   │   ├── handlers/         (Telegram)
   │   ├── skills/           (utilities)
   │   └── storage/          (persistence)
   ├── repositories/         (persistence contracts)
   ├── dtos/                 (response objects)
   └── [existing: app/, bot/, web/, configs/]
   ```

2. Create port interfaces:
   - `ports/exec_runner.go` — interface for ExecShell
   - `ports/skill_provider.go` — interfaces for web, email, git, github, browser, pdf, image
   - `ports/memory_repository.go` — interface for memory persistence
   - `ports/config_loader.go` — interface for configuration
   - `ports/skill_catalog.go` — interface for model definitions

3. Move simple files (no behavior change):
   - `app/session.go` → `domain/shared/session.go`
   - `app/version.go` → `domain/shared/version.go`
   - `app/identity.go` → `domain/shared/identity.go`
   - `bot/skill/models.go` → `domain/shared/ai_models.go`

4. Create domain error types:
   - `domain/shared/errors.go` — DomainError, ExecutionError, MemoryError, etc.

5. Create event skeleton:
   - `domain/shared/events.go` — Event interface; event types (populated in Phase 3+)

**Deliverables:**
- ✅ Folder structure created
- ✅ Port interfaces defined
- ✅ Simple files moved
- ✅ All tests still pass

**Effort:** 2-3 days  
**Risk:** None (no behavior change)

---

### **PHASE 2: DOMAIN EXTRACTION (Weeks 3-5) — Risk: 🟨 MEDIUM**

**Objective:** Move business logic from `app/` to `domain/`, keeping `app/` as service wrappers.

**Execution Domain (Week 3a):**
```
Move: app/taskmanager.go → domain/execution/executor.go
  + Rename TaskManager → Executor
  + Extract method: IsDangerous() → domain/execution/safety_validator.go
  
Keep: app/execution_service.go (wrapper)
  + Calls domain/execution/Executor
  + Manages context, logging, metrics
  
Files affected:
  - handler/callback.go (~164 line: RunExecTask)
  - handler/text.go (~177 line: execReadCode)
  - handler/text.go (~201 line: showExecConfirm)
```

**Memory Domain (Week 3b):**
```
Move: app/memory.go → domain/memory/memory_store.go
  + Extract searchRank() → domain/memory/search_ranker.go
  + Extract normalizeEntry() → domain/memory/entry_normalizer.go
  + Create domain/memory/memory_repository.go (interface for I/O)
  
Keep: app/memory_service.go (wrapper)
  
Files affected:
  - handler/text.go (~336 line: execMemorySearch)
  - handler/callback.go (~252 lines: memory buttons)
  - web/gateway.go (memory API)
```

**RPG Domain (Week 3c):**
```
Move: app/rpg.go → domain/gamification/rpg_manager.go
  + Extract calculateXP() → domain/gamification/xp_calculator.go
  + Extract levelFor() → domain/gamification/level_service.go
  
Keep: app/gamification_service.go (wrapper)
  
Files affected:
  - handler/callback.go (~60 lines: status display)
  - web/server.go (RPG dashboard)
```

**Scheduler Domain (Week 4a):**
```
Move: app/scheduler.go → domain/agent/scheduler.go
  + Keep timer loop; extract startLocked() → domain/agent/schedule_executor.go
  
Keep: app/scheduler_service.go (wrapper)
  
Files affected:
  - handler/callback.go (~273 lines: scheduler buttons)
  - cmd/axle/main.go (scheduler setup)
```

**SubAgent Domain (Week 4b):**
```
Move: app/subagent.go → domain/agent/sub_agent_manager.go
  + Extract context management → domain/agent/agent_lifecycle.go
  
Keep: app/agent_service.go (wrapper)
  
Files affected:
  - handler/callback.go (~247 lines: sub-agent buttons)
```

**Workflow Domain (Week 4-5, 2 weeks):**
```
Move: app/workflow.go → domain/workflow/ (split 3 files):
  1. domain/workflow/workflow.go (Workflow + WorkflowStep entities)
  2. domain/workflow/orchestrator.go (orchestration logic + step execution)
  3. domain/workflow/executor.go (step type handlers)
  
Extract:
  - defaultWorkflowCopilotRunner() → domain/workflow/copilot_step_handler.go
  - defaultWorkflowBrowserRunner() → domain/workflow/browser_step_handler.go
  - planWorkflow() (Copilot call) → keep in orchestrator
  
Keep: app/workflow_service.go (wrapper over WorkflowManager)
  
Files affected:
  - handler/callback.go (~261 lines: workflow buttons)
  - handler/text.go (~590 lines: execCreateWorkflow)
  - web/gateway.go (workflow API)
```

**Deliverables:**
- ✅ All domains extracted to `domain/*/`
- ✅ All `app/` files are now wrappers (ExecutionService, MemoryService, etc.)
- ✅ Handler callbacks still work (unchanged)
- ✅ All tests pass

**Effort:** 2 weeks  
**Risk:** Medium (behavior preserved via wrappers, but more moving parts)

---

### **PHASE 3: APPLICATION SERVICES (Weeks 6-7) — Risk: 🟨 MEDIUM**

**Objective:** Create command/query bus; make logic testable without Telegram.

**Command Layer:**
```
Create: application/command/*.go (one per handler flow)
  - execute_command.go (Execute shell command)
  - read_code_command.go (Read file)
  - write_file_command.go (Write file)
  - search_code_command.go (Search files)
  - list_dir_command.go (List directory)
  - web_search_command.go (Web search)
  - web_fetch_command.go (Fetch URL)
  - search_memory_command.go (Memory search)
  - create_workflow_command.go (Create workflow)
  - create_sub_agent_command.go (Create sub-agent)
  - create_schedule_command.go (Create schedule)
  - ... (15-20 total)
  
Each command:
  type XxxCommand struct {
    UserID int64
    [command params]
  }
  
  type XxxHandler struct {
    [domain services + repos]
  }
  
  func (h *XxxHandler) Handle(cmd XxxCommand) (string, error) {
    // call domains, update repos
  }
```

**Query Layer:**
```
Create: application/query/*.go (one per read operation)
  - get_workflow_status_query.go
  - get_memory_query.go
  - get_status_query.go
  - list_schedules_query.go
  - ... (5-10 total)
  
Each query:
  type XxxQuery struct {
    UserID int64
    [query params]
  }
  
  type XxxQueryHandler struct {
    [repos]
  }
  
  func (h *XxxQueryHandler) Handle(q XxxQuery) (interface{}, error) {
    // read from repos, return DTO
  }
```

**Command Bus:**
```
Create: application/bus.go
  type CommandBus interface {
    Dispatch(ctx context.Context, cmd interface{}) (interface{}, error)
  }
  
  type InProcessBus struct {
    handlers map[string]interface{} // cmd type → handler
  }
  
  func (b *InProcessBus) Dispatch(ctx context.Context, cmd interface{}) (interface{}, error) {
    // find handler, call Handle()
  }
```

**Update Handlers:**
```
Before:
  func (h *Hub) HandleExecBtn(c tele.Context) error {
    return h.RunExecTask(c, cmd)
  }

After:
  func (h *Hub) HandleExecBtn(c tele.Context) error {
    result, err := h.commandBus.Dispatch(context.Background(), &ExecuteCommand{
      UserID: c.Sender().ID,
      Cmd: c.Text(),
    })
    if err != nil {
      return c.Send(fmt.Sprintf("❌ Error: %v", err))
    }
    return c.Send(result.(string))
  }
```

**Deliverables:**
- ✅ 15-20 command handlers created
- ✅ 5-10 query handlers created
- ✅ Command bus wired
- ✅ Handlers dispatch to bus (still work)
- ✅ Logic now testable: `bus.Dispatch(cmd)` without Telegram
- ✅ All tests pass

**Effort:** 1 week  
**Risk:** Medium (more layers, but pattern is simple)

---

### **PHASE 4: HANDLER REFACTOR (Weeks 8-10) — Risk: 🔴 HIGH**

**Objective:** Split monolithic handlers; abstract skill functions.

**Split callback.go (1,664 LOC → 5 files):**
```
handler/
├── execution_handler.go    (exec, readcode, writefile, list, search)
├── memory_handler.go       (memory search/recent/clear)
├── workflow_handler.go     (workflow create/list/status)
├── agent_handler.go        (sub-agent, scheduler)
├── gamification_handler.go (RPG, status)
├── copilot_handler.go      (copilot chat, model selection)
├── git_handler.go          (git operations)
├── github_handler.go       (github integration)
├── web_handler.go          (web search/fetch)
├── browser_handler.go      (browser automation)
├── email_handler.go        (email send/read)
├── calendar_handler.go     (calendar ops)
├── plugin_handler.go       (plugin execution)
└── [base.go for shared helpers]

Each file: <200 LOC, focused on one domain
```

**Split text.go (727 LOC → state machine + sub-dispatchers):**
```
handler/
├── state_machine.go        (Mode enum router; ~100 LOC)
├── mode_dispatcher.go      (dispatch by Mode; ~50 LOC)
└── [individual mode handlers if needed]
```

**Create Skill Adapters (wrap all utility functions):**
```
adapters/skills/
├── exec_adapter.go         (wraps skill.ExecShell)
├── web_adapter.go          (wraps skill.WebFetch, WebSearch)
├── email_adapter.go        (wraps skill.SendEmail, ReadEmail)
├── git_adapter.go          (wraps skill.GitStatus, GitDiff, etc.)
├── github_adapter.go       (wraps skill.GitHubAPI)
├── browser_adapter.go      (wraps skill.BrowserRun)
├── pdf_adapter.go          (wraps skill.SummarizePDF)
├── image_adapter.go        (wraps skill.AnalyzeImage)
├── calendar_adapter.go     (wraps skill.GetCalendar)
├── briefing_adapter.go     (wraps skill.GenerateBriefing)
└── selfupgrade_adapter.go  (wraps skill.SelfUpgrade)

Each adapter implements a port:
  // skill/web_adapter.go
  type WebAdapter struct {}
  
  func (a *WebAdapter) Fetch(ctx context.Context, url string) (string, error) {
    return skill.WebFetch(ctx, url)
  }
  
  func (a *WebAdapter) Search(ctx context.Context, query string) (string, error) {
    return skill.WebSearch(ctx, query)
  }
```

**Create Response DTOs:**
```
dtos/
├── execution_response.go   (ExecResult{Output, Error})
├── memory_response.go      (MemoryHit[]{Entry, Score, Snippet})
├── workflow_response.go    (WorkflowStatus{ID, Status, Steps[]{...}})
├── file_response.go        (FileContent{Path, Content, Size})
└── ...

Benefits:
  - Decouple domain models from HTTP/Telegram responses
  - Can change domain without breaking adapters
  - Easy to add new field to response
```

**Deliverables:**
- ✅ callback.go split into 12 domain-specific handlers (max 200 LOC each)
- ✅ text.go split into state machine + sub-handlers
- ✅ All skill functions wrapped in adapters
- ✅ Response DTOs created
- ✅ Handlers are thin routing (dispatch command + format response)
- ✅ All tests pass

**Effort:** 3 weeks (largest phase)  
**Risk:** High (touching most handler code; needs integration tests)

---

### **PHASE 5: REPOSITORIES & TESTING (Weeks 11-20) — Risk: 🟩 LOW**

**Weeks 11-12: Repositories**

```
repositories/
├── memory_repository.go           (MemoryRepository interface)
├── workflow_repository.go         (WorkflowRepository interface)
├── schedule_repository.go         (ScheduleRepository interface)
├── plugin_repository.go           (PluginRepository interface)
└── rpg_repository.go              (RPGRepository interface)

adapters/storage/
├── json_memory_repository.go      (file-based memory)
├── json_workflow_repository.go    (file-based workflow)
├── json_schedule_repository.go    (file-based schedule)
├── json_plugin_repository.go      (YAML file-based plugin)
├── json_rpg_repository.go         (file-based RPG profile)

Benefits:
  - Can swap JSON → PostgreSQL/MongoDB later
  - Domains don't know about storage
  - Easy to add in-memory repository for testing
```

**Weeks 13-20: Testing & Documentation**

```
Test Suite:
  ✓ domain/*/test_helpers.go — Mocks for all ports
  ✓ domain/*/*_test.go — Unit tests (business logic)
  ✓ application/command/*_test.go — Command handler tests
  ✓ application/query/*_test.go — Query handler tests
  ✓ adapters/handlers/*_test.go — Handler callback tests
  ✓ adapters/skills/*_test.go — Skill adapter tests
  ✓ internal/integration/*_test.go — End-to-end flows
  
Documentation:
  ✓ docs/ARCHITECTURE.md — Context map + layer responsibilities
  ✓ docs/ADRs/ — Architecture Decision Records
  ✓ README updates — New folder structure
  ✓ CONTRIBUTING.md — How to add new skill/domain
```

**Deliverables:**
- ✅ Repository pattern for all persistence
- ✅ Comprehensive test suite (unit + integration)
- ✅ Architecture documentation
- ✅ ADRs for key decisions
- ✅ All tests pass
- ✅ Full refactor complete

**Effort:** 2 weeks core + testing throughout  
**Risk:** Low (mostly mechanical)

---

## 5. FILES TO MOVE/WRAP FIRST

### **PRIORITY 1: LOW-RISK, HIGH-VALUE** (Days 1-3)
```
Session State Management (no business logic):
  session.go       (119 LOC) → domain/shared/session.go
  version.go       (1 LOC)   → domain/shared/version.go
  identity.go      (22 LOC)  → domain/shared/identity.go
  models.go        (112 LOC) → domain/shared/ai_models.go

Action: Move files, update imports, run tests
Risk: None
Test: All existing tests pass (import changes only)
Effort: 1 day

Task Execution (clear business logic):
  taskmanager.go   (~70 LOC logic) → domain/execution/executor.go
  exec.go          (81 LOC)        → domain/execution/exec_runner.go
  safety.go        (64 LOC)        → domain/execution/safety_validator.go

Action: Move + extract safety checks
Risk: Low (behavior preserved)
Test: Existing tests cover all paths
Effort: 1 day
```

### **PRIORITY 2: MEDIUM-RISK, HIGH-VALUE** (Weeks 2-3)
```
Memory System (search ranking extracted):
  memory.go        (440 LOC) → domain/memory/memory_store.go
                               + domain/memory/search_ranker.go
                               + domain/memory/memory_repository.go

Action: Extract search ranking logic, create interface
Risk: Medium (more moving parts)
Test: Unit tests for ranking, integration tests for end-to-end
Effort: 3 days

Gamification (XP calculation extracted):
  rpg.go           (363 LOC) → domain/gamification/rpg_manager.go
                               + domain/gamification/xp_calculator.go
                               + domain/gamification/level_service.go

Action: Extract XP and level logic
Risk: Low (pure math)
Test: Unit tests for XP and levels
Effort: 2 days

Scheduling (timer loop isolated):
  scheduler.go     (205 LOC) → domain/agent/scheduler.go
                               + domain/agent/schedule_executor.go

Action: Extract timer execution
Risk: Low (straightforward cron loop)
Test: Integration tests for scheduling
Effort: 2 days

Sub-Agents (context management isolated):
  subagent.go      (193 LOC) → domain/agent/sub_agent_manager.go
                               + domain/agent/agent_lifecycle.go

Action: Extract lifecycle management
Risk: Low (straightforward context management)
Test: Integration tests for delegation
Effort: 2 days
```

### **PRIORITY 3: HIGH-RISK, HIGHEST-VALUE** (Weeks 3-5 and 8-10)

**Workflow (LARGEST, split into 3 files):**
```
workflow.go (870 LOC) → domain/workflow/{workflow.go, orchestrator.go, executor.go}

  1. workflow.go          (Workflow + WorkflowStep entities, 200 LOC)
  2. orchestrator.go      (orchestration logic, 300 LOC)
  3. executor.go          (step execution + handlers, 200 LOC)
  
  Extract:
    - Copilot step handler → domain/workflow/copilot_step_handler.go
    - Browser step handler → domain/workflow/browser_step_handler.go
    - Dependency logic → domain/workflow/dependency_resolver.go
    - Status update logic → domain/workflow/status_machine.go

Action: Split 870 LOC into manageable domain entities + services
Risk: High (largest file, most complex state)
Test: Heavy unit + integration testing
Effort: 2 weeks
```

**Handlers (MONOLITH, split into 12 files):**
```
callback.go (1,664 LOC) → handler/{execution,memory,workflow,agent,gamification,copilot,git,github,web,browser,email,calendar,plugin}_handler.go

  Target: Each file < 200 LOC, focused on one domain
  
  handler/execution_handler.go (~200 LOC):
    - HandleExecBtn
    - HandleExecConfirm
    - HandleExecDangerConfirm
    - HandleExecCancelBtn

  handler/memory_handler.go (~150 LOC):
    - HandleMemoryBtn
    - HandleMemorySearch
    - HandleMemoryRecent
    - HandleMemoryClear

  [similar for workflow, agent, etc.]

Action: Organize by domain; make each handler thin (dispatch + format)
Risk: High (large refactor, many touch points)
Test: Handler tests for each button flow
Effort: 2 weeks

text.go (727 LOC) → handler/{state_machine.go, mode_dispatcher.go}

  handler/state_machine.go (~100 LOC):
    - HandleText (main router by Mode)
    - Mode enum handlers

  handler/mode_dispatcher.go (~50 LOC):
    - dispatch(mode, text) → call appropriate handler

Action: Simplify state machine; extract mode handling
Risk: Medium (state machine is complex)
Test: State machine tests for all Mode transitions
Effort: 1 week
```

**Skill Adapters (~3,900 LOC utilities → 12 adapter files):**
```
adapters/skills/{exec,web,email,git,github,browser,pdf,image,calendar,briefing,selfupgrade}_adapter.go

Each adapter wraps 1-2 skill functions behind an interface.

Example:
  // adapters/skills/web_adapter.go
  type WebAdapter struct{}
  
  func (a *WebAdapter) Fetch(ctx context.Context, url string) (string, error) {
    return skill.WebFetch(ctx, url)
  }
  
  func (a *WebAdapter) Search(ctx context.Context, query string) (string, error) {
    return skill.WebSearch(ctx, query)
  }

Action: Create adapter files; inject into command handlers
Risk: Medium (multiple files to wire)
Test: Adapter tests mock external calls
Effort: 1 week
```

---

## 6. IMPLEMENTATION PATTERN: Execution Domain Example

### Before Refactor
```go
// handler/callback.go
func (h *Hub) HandleExecBtn(c tele.Context) error {
    userID := c.Sender().ID
    h.Sessions.Update(userID, func(s *app.UserSession) { 
        s.Mode = app.ModeAwaitExecCmd 
    })
    return c.Send("⚡ 請輸入指令", ...)
}

// handler/text.go
case app.ModeAwaitExecCmd:
    return h.showExecConfirm(c, input)

// handler/hub.go
func (h *Hub) showExecConfirm(c tele.Context, cmd string) error {
    h.Sessions.Update(userID, func(s *app.UserSession) { 
        s.PendingCmd = cmd 
        s.Mode = app.ModeAwaitExecConfirm 
    })
    return c.Send("確認執行？", confirmMenu, ...)
}

// handler/hub.go
func (h *Hub) RunExecTask(c tele.Context, command string) error {
    ctx, done, ok := h.tryStartTask(c, "execute")
    if !ok { return c.Send("任務執行中") }
    defer done()
    
    out, err := skill.ExecShell(ctx, h.Workspace, command)
    if err != nil {
        return c.Send(fmt.Sprintf("❌ Error: %v", err))
    }
    h.recordMemory(c.Sender().ID, ...)
    h.emitRPG("exec_shell", ...)
    return c.Send(out)
}
```

### After Refactor: Layer 1 (Domains Extracted)

```go
// domain/execution/executor.go
package execution

type Executor interface {
    Execute(ctx context.Context, cmd string) (string, error)
}

// ExecResult is the output of command execution
type ExecResult struct {
    Output string
    Took   time.Duration
}

// executor.go implements Executor
type executor struct {
    runner   ExecRunner       // port interface
    validator SafetyValidator // port interface
    logger   *slog.Logger
}

func (e *executor) Execute(ctx context.Context, cmd string) (string, error) {
    if e.validator.IsDangerous(cmd) {
        return "", errors.New("dangerous command blocked")
    }
    return e.runner.Run(ctx, cmd)
}

// domain/execution/safety_validator.go
type SafetyValidator interface {
    IsDangerous(cmd string) bool
}

type safetyValidator struct {
    patterns []string // dangerous patterns
}

func (sv *safetyValidator) IsDangerous(cmd string) bool {
    for _, p := range sv.patterns {
        if strings.Contains(cmd, p) {
            return true
        }
    }
    return false
}

// ports/exec_runner.go
type ExecRunner interface {
    Run(ctx context.Context, cmd string) (string, error)
}

// adapters/skills/exec_adapter.go
type ExecAdapter struct{}

func (a *ExecAdapter) Run(ctx context.Context, cmd string) (string, error) {
    return skill.ExecShell(ctx, cmd)
}

// app/execution_service.go (wrapper)
type ExecutionService struct {
    executor execution.Executor
    memory   *MemoryStore     // cross-domain dependency
    rpg      *RPGManager
    logger   *slog.Logger
}

func (s *ExecutionService) Execute(ctx context.Context, cmd string) (string, error) {
    out, err := s.executor.Execute(ctx, cmd)
    if err == nil {
        s.memory.Add(userID, out)
        s.rpg.EmitEvent("exec", "success")
    }
    return out, err
}
```

### After Refactor: Layer 2 (Commands)

```go
// application/command/execute_command.go
package command

type ExecuteCommand struct {
    UserID int64
    Cmd    string
}

type ExecuteHandler struct {
    exec  *ExecutionService // app layer
    sess  *SessionManager
    bot   *telebot.Bot
}

func (h *ExecuteHandler) Handle(cmd ExecuteCommand) (string, error) {
    h.sess.Update(cmd.UserID, func(s *UserSession) {
        s.Mode = ModeAwaitExecConfirm
        s.PendingCmd = cmd.Cmd
    })
    
    ctx, done, ok := h.exec.TryStart("execute")
    if !ok {
        return "任務執行中", errors.New("task in progress")
    }
    defer done()
    
    return h.exec.Execute(ctx, cmd.Cmd)
}

// application/bus.go
type CommandBus interface {
    Dispatch(ctx context.Context, cmd interface{}) (interface{}, error)
}

type InProcessBus struct {
    handlers map[string]interface{}
}

func (b *InProcessBus) Register(cmd interface{}, handler interface{}) {
    t := reflect.TypeOf(cmd).Name()
    b.handlers[t] = handler
}

func (b *InProcessBus) Dispatch(ctx context.Context, cmd interface{}) (interface{}, error) {
    t := reflect.TypeOf(cmd).Name()
    handler, ok := b.handlers[t]
    if !ok {
        return nil, fmt.Errorf("unknown command: %s", t)
    }
    
    // Call handler.Handle(cmd)
    m := reflect.ValueOf(handler).MethodByName("Handle")
    results := m.Call([]reflect.Value{reflect.ValueOf(cmd)})
    
    if err := results[1].Interface(); err != nil {
        return nil, err.(error)
    }
    return results[0].Interface(), nil
}
```

### After Refactor: Layer 3 (Handler Adapter)

```go
// handler/execution_handler.go
package handler

type ExecutionHandler struct {
    bus  CommandBus
    sess SessionManager
    bot  *telebot.Bot
}

func (h *ExecutionHandler) HandleExecBtn(c tele.Context) error {
    userID := c.Sender().ID
    h.sess.Update(userID, func(s *UserSession) {
        s.Mode = ModeAwaitExecCmd
    })
    return c.Send("⚡ 請輸入指令...", ...)
}

func (h *ExecutionHandler) ConfirmExec(c tele.Context, cmd string) error {
    userID := c.Sender().ID
    
    // Dispatch command to bus
    result, err := h.bus.Dispatch(context.Background(), &command.ExecuteCommand{
        UserID: userID,
        Cmd: cmd,
    })
    
    if err != nil {
        return c.Send(fmt.Sprintf("❌ %v", err))
    }
    
    output := result.(string)
    chunks := skill.SplitMessage(output)
    for _, chunk := range chunks {
        c.Send(chunk)
    }
    return nil
}
```

**Benefits of this layering:**
1. **Testable domains:** Can test execution logic without Telegram
   ```go
   // domain/execution/executor_test.go
   exec := executor{runner, validator}
   out, err := exec.Execute(ctx, "ls -la")
   require.NoError(t, err)
   ```

2. **Testable commands:** Can test orchestration without I/O
   ```go
   // application/command/execute_test.go
   handler := ExecuteHandler{execService, sessManager, botMock}
   result, err := handler.Handle(ExecuteCommand{UserID: 1, Cmd: "echo test"})
   ```

3. **Thin handlers:** Just routing, no business logic
   ```go
   // handler/execution_handler_test.go
   h := ExecutionHandler{busMock, sessMock, botMock}
   h.ConfirmExec(ctxMock, "ls")
   // verify bus.Dispatch called with correct command
   ```

4. **Swappable implementations:** Change ExecRunner without touching handlers
   ```go
   // in wiring code
   exec := &executor{
       runner: &ContainedExecAdapter{}, // instead of skill.ExecShell
       validator: safetyValidator,
   }
   ```

---

## 7. TESTING STRATEGY

### Test Pyramid
```
        Integration Tests (E2E)     ▲
        ════════════════════         │ Fewer, Slower, More Realistic
        
        Handler Tests               │
        ════════════════════════
        
        Command/Query Tests         │
        ════════════════════════════
        
        Domain Unit Tests           │ More, Faster, Focused
        ════════════════════════════ ▼
```

### Unit Tests (Domain Logic)
```go
// domain/execution/executor_test.go
func TestExecutor_IsDangerous(t *testing.T) {
    v := &safetyValidator{patterns: []string{"rm -rf", "dd"}}
    require.True(t, v.IsDangerous("rm -rf /"))
    require.False(t, v.IsDangerous("ls -la"))
}

// domain/workflow/orchestrator_test.go
func TestWorkflow_ExecuteWithDependencies(t *testing.T) {
    w := &Workflow{
        Steps: []WorkflowStep{
            {ID: "1", Kind: "copilot", Prompt: "step 1"},
            {ID: "2", Kind: "browser", DependsOn: []string{"1"}},
        },
    }
    
    // step 2 should not run until step 1 completes
    orch := orchestrator{...}
    orch.Execute(context.Background(), w)
    
    require.Equal(t, WorkflowStepCompleted, w.Steps[1].Status)
}
```

### Command/Query Tests
```go
// application/command/execute_command_test.go
func TestExecuteCommand_IsDangerous(t *testing.T) {
    h := &ExecuteHandler{
        exec: &ExecutionServiceMock{},
        sess: NewSessionManager(),
    }
    
    _, err := h.Handle(&ExecuteCommand{
        UserID: 1,
        Cmd: "rm -rf /",
    })
    
    require.Error(t, err)
    require.Contains(t, err.Error(), "dangerous")
}

// application/query/get_status_query_test.go
func TestGetStatusQuery(t *testing.T) {
    h := &GetStatusHandler{
        taskRepo: &TaskRepositoryMock{tasks: [...]}
        workflowRepo: &WorkflowRepositoryMock{...},
    }
    
    result, err := h.Handle(&GetStatusQuery{UserID: 1})
    require.NoError(t, err)
    require.Equal(t, "executing", result.(StatusResponse).TaskStatus)
}
```

### Handler Tests
```go
// handler/execution_handler_test.go
func TestHandleExecBtn(t *testing.T) {
    h := &ExecutionHandler{
        bus: &CommandBusMock{},
        sess: NewSessionManager(),
        bot: &telebot.Bot{}, // mock
    }
    
    ctx := &telebot.ContextMock{Sender: &telebot.User{ID: 1}}
    err := h.HandleExecBtn(ctx)
    
    require.NoError(t, err)
    sess := h.sess.GetCopy(1)
    require.Equal(t, ModeAwaitExecCmd, sess.Mode)
}

// handler/execution_handler_test.go
func TestConfirmExec(t *testing.T) {
    busMock := &CommandBusMock{
        Results: map[string]interface{}{
            "ExecuteCommand": "output of command",
        },
    }
    
    h := &ExecutionHandler{
        bus: busMock,
        sess: NewSessionManager(),
        bot: &telebot.BotMock{},
    }
    
    ctx := &telebot.ContextMock{Sender: &telebot.User{ID: 1}}
    err := h.ConfirmExec(ctx, "echo test")
    
    require.NoError(t, err)
    require.True(t, busMock.WasDispatched("ExecuteCommand"))
}
```

### Integration Tests
```go
// internal/integration/execution_flow_test.go
func TestExecuteFlow_EndToEnd(t *testing.T) {
    // Setup
    memRepo := NewMemoryRepositoryMem()
    execRepo := NewExecRepositoryMem()
    bus := NewInProcessBus()
    sess := NewSessionManager()
    
    // Wire domains
    exec := &executor{
        runner: &ExecAdapterReal{},
        validator: &safetyValidator{...},
    }
    
    execSvc := &ExecutionService{
        executor: exec,
        memory: &MemoryStore{repo: memRepo},
        rpg: &RPGManager{repo: execRepo},
    }
    
    // Wire app services
    execCmd := &ExecuteHandler{
        exec: execSvc,
        sess: sess,
    }
    bus.Register(&ExecuteCommand{}, execCmd)
    
    // Execute flow
    result, err := bus.Dispatch(context.Background(), &ExecuteCommand{
        UserID: 1,
        Cmd: "echo hello",
    })
    
    // Assert
    require.NoError(t, err)
    require.Equal(t, "hello", strings.TrimSpace(result.(string)))
    
    // Verify memory was recorded
    entries, _ := memRepo.Get(1)
    require.Greater(t, len(entries), 0)
}
```

### Regression Test Suite
```bash
# Day 1: All tests pass
$ go test ./...
ok  github.com/garyellow/axle/internal/app      2.1s
ok  github.com/garyellow/axle/internal/bot      3.2s
...

# Day 20 (after refactor): Same tests still pass
$ go test ./...
ok  github.com/garyellow/axle/internal/domain  1.9s
ok  github.com/garyellow/axle/internal/application  1.1s
ok  github.com/garyellow/axle/internal/adapters  2.0s
ok  github.com/garyellow/axle/internal/integration  3.5s
...

# 100% of tests still passing ✅
# SAME TEST SUITE, NEW ARCHITECTURE ✅
```

---

## 8. TIMELINE & EFFORT BREAKDOWN

| Week | Phase | Focus | Effort | Deliverable |
|------|-------|-------|--------|-------------|
| 1-2 | 1 | Structure + simple moves | 40h | Foundation ready |
| 3 | 2a | Execution + Memory | 40h | 2 domains extracted |
| 4 | 2b | RPG + Scheduler + SubAgent | 40h | 3 more domains |
| 5 | 2c | Workflow (split 3 files) | 40h | Largest domain done |
| 6 | 3 | Commands + Queries + Bus | 40h | App services layer |
| 7 | 3b | Handler refactor start | 30h | Wire commands to bus |
| 8-9 | 4a | Split callback.go + text.go | 60h | Handlers split + tested |
| 9-10 | 4b | Skill adapters | 40h | All skills wrapped |
| 11 | 5a | Repositories | 30h | Persistence abstracted |
| 12 | 5b | Testing + docs | 40h | Full test coverage |
| 13-20 | Overlap | Continuous refining | 80h | Polish + ADRs |

**Total:** 520 hours = 13 weeks @ 40h/week for 1 developer  
**With 2 developers (overlap):** 10-12 weeks

---

## 9. SUCCESS CRITERIA (Week 20)

✅ **Codebase**
- All existing tests pass (regression verified)
- Zero new dependencies added
- No behavior changes (only structure)
- All domains isolated in `internal/domain/*`
- No cross-domain imports (imports only go up: domain → app → adapters)

✅ **Domains**
- 7 domain packages with clear entities + business logic
- Each domain can be tested without I/O
- Each domain has a public interface (exported types)
- Port interfaces for all external operations

✅ **Application Layer**
- 15-20 command handlers (one per flow)
- 5-10 query handlers (one per read)
- Simple in-process command/query bus
- Testable without Telegram or HTTP

✅ **Adapters**
- Handler files < 200 LOC each
- All skill functions wrapped in adapters
- Response DTOs for all responses
- Thin routing (dispatch → format → send)

✅ **Repositories**
- Repository interfaces for all persistence
- JSON file implementations (swappable later)
- Can add in-memory repo for testing

✅ **Testing**
- >80% code coverage (domain + app + adapters)
- Unit tests for all business logic
- Integration tests for all flows
- Handler tests for all callbacks
- Regression tests (existing tests still pass)

✅ **Documentation**
- Architecture decision records (ADRs)
- Context map showing bounded contexts
- Layer responsibilities documented
- Contributing guide (how to add new skill/domain)
- README updated with new folder structure

✅ **Team Readiness**
- New developers can understand domain boundaries from folder names
- Can add new skill (e.g., Slack) without touching domains
- Can add new domain without touching handlers
- Can swap storage backend (JSON → DB) without changing domains

---

## 10. ROLLBACK PLAN

### If Phase Gets Too Complex:

**Option 1: Pause & Stabilize**
- Stop at current phase
- Commit work
- Spend time testing + documenting
- Resume next phase when ready

Example: If Phase 4 (Handler Split) is too risky:
- Keep Phases 1-3 (domains extracted, commands working)
- Delay Phase 4 split
- Handlers can call command bus directly for now
- Still improved over starting point

**Option 2: Partial Rollback**
- Keep domain structure
- Keep command layer
- Revert adapter splitting
- Rework later when time permits

### Testing Gate Before Each Phase
```bash
# Before Phase 2 starts:
$ go test ./... -race
$ go test ./... -cover

# Must have:
# - 0 test failures
# - 0 race conditions
# - No behavior regressions
```

---

## RECOMMENDATION

1. **Start with Phase 1 immediately** — 2-3 days, establishes structure, zero risk
2. **Execute Phase 2 in parallel with Phases 3-4** — Domains extracted while app/adapters refactored
3. **Allocate 2 developers for Weeks 3-10** — Heaviest lifting
4. **Plan 1 developer for Weeks 1-2 and 11-20** — Setup/testing/polish
5. **Weekly demo/checkpoints** — Show extracted domains, working commands, refactored handlers
6. **Comprehensive testing throughout** — No shortcuts; all tests passing at each phase end

---

**Prepared by:** Axle Architecture Analysis  
**For:** Clean Architecture + DDD Refactor  
**Scope:** 20 weeks, 1-2 developers, ~5,600 LOC refactored  
**Status:** Ready to execute Phase 1
