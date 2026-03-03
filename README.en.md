# Axle

<div align="center">

**🌐 Language**  
[繁體中文](README.md) | [English](README.en.md)

[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)](https://go.dev)
[![License](https://img.shields.io/badge/License-MIT-green)](LICENSE)
[![Platform](https://img.shields.io/badge/Platform-macOS%20%7C%20Linux-lightgrey)]()

</div>

> 🔧 A secure, local Golang AI Agent — control your dev tools via Telegram, no extra API key needed.

---

## 📖 Documentation

| Doc | Description |
|-----|-------------|
| 🚀 [Quick Start](docs/tutorial-en.md#1-quick-start) | Install and start chatting in 5 minutes |
| 📚 [Full Tutorial (Beginner → Advanced)](docs/tutorial-en.md) | Telegram usage · Git · Scheduling · Self-upgrade |
| 🔧 [Full Feature Reference](docs/features-zh.md) | Detailed description of every feature |
| 🏗️ [Architecture & Dev Guide](docs/architecture.md) | Design, security, contributing |

---

## ✨ Feature Highlights

<table>
<tr>
<td width="50%">

**🔌 Default Features (out of the box)**
- 📁 Read / ✏️ Write code files
- 🤖 Copilot Assistant (streaming)
- ⚡ Shell execution (danger detection)
- 🔄 Switch AI models (17 options)
- 📊 System status · 🛑 Cancel task

</td>
<td width="50%">

**🧩 Optional Extensions (enable on demand)**
- 🔀 Git · 🐙 GitHub integration
- 📧 Email · 📅 Calendar
- 🔍 Web search · 🌐 URL fetch
- 📄 PDF summary · 👥 Sub-agents
- ⏰ Scheduled tasks · 🎮 RPG Dashboard

</td>
</tr>
<tr>
<td colspan="2">

**✨ Signature Features**
- 🔧 **Self-Upgrade**: Describe a feature → AI plans → auto-code → compile → test → restart
- 🎮 **RPG Dashboard**: Pixel-art browser dashboard with level system and achievements
- 🗣️ **Natural Language Routing**: Type anything to auto-enter AI chat — no button clicks needed

</td>
</tr>
</table>

---

## 🚀 Quick Start

### Prerequisites

| Tool | Purpose | Required |
|------|---------|:--------:|
| Go 1.22+ | Compile & run | ✅ Yes |
| Telegram Bot Token | Bot interface ([@BotFather](https://t.me/BotFather)) | ✅ Yes |
| GitHub Copilot CLI (`copilot`) | AI core features | ✅ Yes |
| GitHub CLI (`gh`) | GitHub integration | Optional |
| icalBuddy | macOS calendar (`brew install ical-buddy`) | Optional |

### Three-step Install

```bash
# 1. Clone
git clone https://github.com/Gary-GaGa/axle.git && cd axle

# 2. Build
go build -o axle ./cmd/axle

# 3. Start (interactive setup on first run)
./axle
```

On first launch, Axle interactively asks for your **Telegram Bot Token** and **Telegram User ID**, saves them to `~/.axle/credentials.json` and never asks again.

> 💡 For advanced configuration (multi-user, Email, GitHub) see the [Full Tutorial](docs/tutorial-en.md)

---

## 🔒 Security

- **Whitelist Stealth Mode**: Unauthorized users get zero response — the bot is invisible to them
- **Sandboxed File System**: All operations confined to workspace, `../` escape blocked
- **3-Level Danger Detection**: ⛔ Blocked → ⚠️ Double-confirm → ✅ Normal confirm
- **Human-in-the-Loop**: All shell commands require manual button confirmation

> See [Architecture — Security](docs/architecture.md#security) for full details

---

## 🎮 RPG Dashboard

After launch, open `http://localhost:8080` to see a pixel-art real-time dashboard: character panel, quest scroll, battle log, skill stats. Level grows from 🟤 Apprentice to 🔴 Immortal Engine. XP persists across restarts.

---

## 📝 Dev Commands

```bash
go build -race -o axle ./cmd/axle  # build
go vet ./...                        # static analysis
go test -race ./internal/...        # tests
```

---

## License

MIT
