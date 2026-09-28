# 📧 StdMailBot — High-Performance Telegram Temp Mail Bot

<p align="center">
  <img src="https://graph.org/file/00ea4effe5d2dfbb8d5be.jpg" alt="StdMailBot Banner" width="450"/>
</p>

<p align="center">
  <a href="https://golang.org"><img src="https://img.shields.io/badge/Language-Go%201.22+-00ADD8?style=for-the-badge&logo=go" alt="Go"/></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-AGPL%20v3-blue?style=for-the-badge" alt="License"/></a>
  <a href="https://t.me/STDBOTS"><img src="https://img.shields.io/badge/Channel-%40STDBOTS-2CA5E0?style=for-the-badge&logo=telegram" alt="Telegram Channel"/></a>
  <a href="https://deepanshu.in"><img src="https://img.shields.io/badge/Author-STD%20DEEPANSHU-FF4500?style=for-the-badge" alt="Author"/></a>
</p>

<p align="center">
  <a href="https://dashboard.heroku.com/new?template=https://github.com/StdBots/StdMailBot">
    <img src="https://img.shields.io/badge/Deploy%20To%20Heroku-7056bf?style=for-the-badge&logo=heroku" alt="Deploy to Heroku"/>
  </a>
  <a href="https://railway.app/template/new?template=https://github.com/StdBots/StdMailBot">
    <img src="https://img.shields.io/badge/Deploy%20On%20Railway-0B0D0E?style=for-the-badge&logo=railway" alt="Deploy on Railway"/>
  </a>
</p>

---

## ⚡ Overview

**StdMailBot** is a high-speed, disposable temporary email bot built with **Go (Golang)** and **MongoDB**. It gives Telegram users instant disposable email addresses with real-time inbox monitoring, automatic OTP code detection, login/verification link extraction, and background inbox auto-watch mode.

Developed by **[STD DEEPANSHU](https://deepanshu.in)** as part of the **[STD BOTS Ecosystem](https://t.me/STDBOTS)**.

---

## ✨ Features

- ⚡ **Go 1.22+ Architecture:** Ultra-fast execution, low memory footprint (<25MB RAM).
- 🔄 **Dual High-Speed Engine:** Supports `Temp-Mail.io` API v3 + `1secmail.com` API with seamless automatic failover.
- 🔑 **Smart OTP Extractor:** Automatically extracts 4 to 8 digit OTP codes with one-tap copy formatting.
- 🔗 **Magic Link Extractor:** Automatically extracts and isolates Login, Activation, and Verification URLs.
- 🔔 **Inbox Watch Mode (Auto-Notify):** Background Goroutine polls inbox and immediately alerts the user with incoming emails.
- 🍃 **MongoDB Persistence:** Active mailbox address, server token, and message history persist across bot restarts.
- 📱 **Modern Interactive UI:** 1-Tap copyable monospace emails, dynamic inline buttons, and refresh controls.
- 📢 **Force Subscribe (FSUB):** Verify channel membership before users can generate mailboxes.
- 📣 **Admin Broadcast Engine:** Worker pool broadcasting with flood wait protection.
- 🔒 **7-Layer Credit Protection:** AGPL-3.0 integrity validation, zero-width watermarks, and brand protection.

---

## 🚀 One-Click Deployments

### 🟣 Deploy to Heroku
Click the button below to deploy your instance to Heroku in 60 seconds:

[![Deploy](https://www.herokucdn.com/deploy/button.svg)](https://dashboard.heroku.com/new?template=https://github.com/StdBots/StdMailBot)

### 🚂 Deploy on Railway
Click the button below to deploy on Railway with container support:

[![Deploy on Railway](https://railway.app/button.svg)](https://railway.app/template/new?template=https://github.com/StdBots/StdMailBot)

### 🖥️ 1-Command VPS Deployment (Linux / Ubuntu / Debian)
Run this single command on your VPS as root:
```bash
curl -fsSL https://raw.githubusercontent.com/StdBots/StdMailBot/main/scripts/install_vps.sh | bash
```

---

## 🐳 Docker & Manual VPS Setup

```bash
# 1. Clone repository
git clone https://github.com/StdBots/StdMailBot.git
cd StdMailBot

# 2. Configure environment
cp .env.example .env
nano .env

# 3. Start with Docker Compose
docker compose up -d --build
```

---

## ⚙️ Environment Variables

| Variable | Description | Required | Default |
|---|---|---|---|
| `BOT_TOKEN` | Telegram Bot Token from [@BotFather](https://t.me/BotFather) | **Yes** | — |
| `OWNER_ID` | Telegram User ID of the primary administrator | **Yes** | `7394590844` |
| `MONGO_URI` | MongoDB Connection String (Atlas or Local) | **Yes** | `mongodb://localhost:27017/stdmailbot` |
| `DB_NAME` | Database name | No | `stdmailbot` |
| `FORCE_SUB_CHANNEL` | Channel username without `@` for force-sub | No | `StdBots` |
| `LOG_CHANNEL_ID` | Telegram Channel ID for logging events | No | `0` |
| `DEFAULT_MAIL_ENGINE`| Mail engine (`tempmail` / `secmail`) | No | `tempmail` |
| `ENV` | Environment mode (`development`/`production`) | No | `production` |

---

## 🤖 Commands

| Command | Description |
|---|---|
| `/start` | Launch bot, view features and active mailbox |
| `/getmail` | Generate a new disposable email address |
| `/inbox` | Check inbox for latest incoming emails |
| `/watch` | Toggle 5-minute background auto-polling for incoming emails |
| `/changemail` | Discard current email and generate a fresh one |
| `/deletemail` | Delete current mailbox session |
| `/help` | Detailed help guide |
| `/stats` | Global bot analytics (Admin only) |
| `/broadcast` | Broadcast message to all registered users (Admin only) |

---

## 📄 License & Attribution

Licensed under the [GNU Affero General Public License v3 (AGPL-3.0)](LICENSE).

Mandatory Attribution: Derivative works, forks, and hosted instances must preserve all visible and embedded credits pointing to **STD DEEPANSHU** ([https://deepanshu.in](https://deepanshu.in)) and **STD BOTS** ([@STDBOTS](https://t.me/STDBOTS)).
