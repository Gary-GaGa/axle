# Axle — Full Tutorial

[← Back to Home](../README.en.md) | [繁體中文](tutorial-zh.md)

---

## Table of Contents

- [1. Quick Start](#1-quick-start)
- [2. Basic Operations](#2-basic-operations)
- [3. Advanced Usage](#3-advanced-usage)
- [4. Self-Upgrade](#4-self-upgrade)

---

## 1. Quick Start

### 1.1 Create a Telegram Bot

1. Open Telegram and search for **[@BotFather](https://t.me/BotFather)**
2. Send `/newbot`, follow the prompts to name your bot (Username must end in `bot`)
3. BotFather returns a Token like `1234567890:ABCdef...` — **keep it safe**

### 1.2 Get Your Telegram User ID

1. Search for **[@userinfobot](https://t.me/userinfobot)** in Telegram
2. Send any message — it replies with your numeric User ID

### 1.3 Install Copilot CLI

Axle's AI features depend on GitHub Copilot CLI:

```bash
# macOS
brew install gh
gh auth login         # Log in to GitHub
brew install copilot  # Install Copilot CLI

# Verify
copilot --version
```

> Requires an active GitHub Copilot subscription (Individual or Enterprise)

### 1.4 Install & Start Axle

```bash
# Clone
git clone https://github.com/Gary-GaGa/axle.git
cd axle

# Build
go build -o axle ./cmd/axle

# Start
./axle
```

**First-run setup:**

```
? Enter Telegram Bot Token: 1234567890:ABCdef...
? Enter allowed Telegram User IDs (comma-separated): 98765432
✅ Credentials saved to ~/.axle/credentials.json
🚀 Axle v0.10.0 started!
```

Then find your Bot on Telegram and send `/start`.

### 1.5 First Conversation

After `/start`, you'll see the main menu:

```
⚔️ Axle Engine Started v0.10.0
Mode: Solo | Model: claude-sonnet-4.6

[📁 Read Code]   [✏️ Write File]
[🤖 Copilot]     [⚡ Shell Exec]
[🔄 Models]      [📂 Switch Dir]
[📊 Status]      [➕ More]
```

Click **🤖 Copilot**, type a question, and start chatting.

---

## 2. Basic Operations

### 2.1 Read Code

1. Click **📁 Read Code**
2. Enter a path relative to the current workspace, e.g. `main.go`
3. Bot returns the formatted file (max 30 KB)

> 💡 Enable the **📂 Directory Browse** extension to explore structure first

### 2.2 Execute Shell Commands

1. Click **⚡ Shell Exec**
2. Enter a command like `go test ./...`
3. Bot shows the danger level and confirmation buttons:
   - ✅ Safe → confirm and run
   - ⚠️ Warning → extra confirmation required
   - ⛔ Blocked → rejected outright
4. After clicking **✅ Confirm**, output streams back in real time

### 2.3 Switch AI Model

1. Click **🔄 Models**
2. Choose vendor (Anthropic / OpenAI / Google)
3. Choose model (with cost multiplier shown)

**Recommendations:**

| Use Case | Model |
|----------|-------|
| Daily chat | claude-sonnet-4.6 (1x, balanced) |
| Complex architecture | claude-opus-4.5 (3x, strong reasoning) |
| Quick answers | claude-haiku-4.5 (0.25x, fast) |
| Cost-saving | gpt-4.1 (free) |

### 2.4 Enable Extensions

Click **➕ More** to manage optional extensions:

```
✅ 📂 Directory Browse    ⬜ 🔎 Search Code
⬜ 🔍 Web Search          ⬜ 🌐 Web Fetch
✅ 🔀 Git                  ⬜ 🐙 GitHub
⬜ 📧 Email                ⬜ 📅 Calendar
```

Toggle any extension — enabled ones appear in the main menu.

### 2.5 Switch Working Directory

1. Click **📂 Switch Project**
2. Enter an absolute path, e.g. `/Users/gary/myproject`
3. All operations will run in that directory
4. Type `reset` to return to default

---

## 3. Advanced Usage

### 3.1 Git Integration

Enable **🔀 Git** to access:

```
[📊 Status]  [📝 Diff]
[📋 Log]     [🚀 Commit & Push]
```

**Commit & Push flow:**
1. Click **🚀 Commit & Push**
2. Enter a commit message
3. Axle runs `git add -A && git commit -m "..." && git push`

### 3.2 GitHub Integration

Enable **🐙 GitHub** (requires `gh auth login`):

```bash
gh auth login   # if not already logged in
gh auth status  # verify
```

Features: list PRs, list Issues, CI status, create PR.

### 3.3 Scheduled Tasks

Enable **⏰ Schedule** to create cron jobs:

1. Click **➕ Add Schedule**
2. Enter a cron expression, e.g. `0 9 * * 1-5` (weekdays 9 AM)
3. Enter the command, or `@briefing` for daily report

**Common cron expressions:**

```
0 9 * * 1-5     Weekdays at 9 AM
*/30 * * * *    Every 30 minutes
0 18 * * *      Daily at 6 PM
0 0 * * 0       Sundays at midnight
```

Schedules persist to `~/.axle/schedules.json`.

### 3.4 Sub-Agents

Enable **👥 Sub-Agents** to delegate tasks:

1. Describe the task in natural language
2. Axle spawns an independent sub-agent to handle it
3. Sub-agent reports back when complete
4. **List Agents** shows all active/completed agents

### 3.5 Plugin System

Enable **🧩 Extra Skills**. Axle creates an example plugin at `~/.axle/plugins/`. Add your own:

```yaml
# ~/.axle/plugins/weather.yaml
name: weather
description: Get weather for a city
type: shell
command: curl -s wttr.in/${INPUT}?format=3
```

### 3.6 Daily Briefing

Enable **📢 Daily Briefing**:

- **Manual**: click the button anytime
- **Automatic**: create a schedule with command `@briefing`

Briefing contains: system stats + Git status + today's calendar + disk usage.

### 3.7 RPG Dashboard

After Axle starts, open `http://localhost:8080`:

- Each skill execution earns XP (Shell Strike +15 · Copilot +25)
- Level up triggers animations; XP persists across restarts
- Skill usage stats serve as a work-activity report

---

## 4. Self-Upgrade

> ⚠️ This feature modifies Axle's own source code and recompiles. Use under version control.

### 4.1 Enable

Turn on **🔧 Self-Upgrade** in **➕ More**.

### 4.2 Describe the Feature

1. Click **🔧 Self-Upgrade**
2. Describe what you want in natural language:
   ```
   Example: Add a weather lookup feature — user types a city name, 
   Axle fetches from wttr.in and returns the weather.
   ```
3. Axle analyzes the codebase via Copilot CLI and produces an upgrade plan

### 4.3 Review & Confirm

The plan is shown before any code is changed:

```
📋 Upgrade Plan:
1. Add internal/bot/skill/weather.go — FetchWeather(city) function
2. Add WeatherSkill to extension list
3. Register HandleWeatherBtn in callback.go

[✅ Confirm]  [❌ Cancel]
```

### 4.4 Automated Execution

After confirmation:

```
[1/5] 🔧 Backup binary → axle.bak
[2/5] 💻 Apply code changes (Copilot CLI)
[3/5] 📦 go build (compile check)
[4/5] 🧪 go test (test verification)
[5/5] 🔀 git commit → version v0.10.0 → v0.10.1
⚡ Restarting...
```

### 4.5 Automatic Rollback

If build or tests fail:
- Automatically restores `axle.bak`
- Bot reports the failure reason
- Version number is NOT incremented

---

## 📌 Quick Reference

### Commands

| Command | Description |
|---------|-------------|
| `/start` | Show main menu |
| `/cancel` | Cancel running task |

### Storage Paths

| Path | Contents |
|------|----------|
| `~/.axle/credentials.json` | Token · User ID · Email config |
| `~/.axle/rpg_state.json` | RPG level and XP |
| `~/.axle/schedules.json` | Scheduled tasks |
| `~/.axle/plugins/` | Custom plugins |
| `~/.axle/memory/` | AI memory |

### Limits

| Item | Limit |
|------|-------|
| Concurrent tasks | 1 |
| Shell timeout | 60 seconds |
| Copilot timeout | 5 minutes |
| Max file read | 30 KB |
| Max shell output | 1 MB |

---

[← Back to Home](../README.en.md) | [Feature Reference](features-zh.md) | [Architecture](architecture.md)
