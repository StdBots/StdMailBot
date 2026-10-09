#!/bin/bash
# ==============================================================================
# STD BOTS ECOSYSTEM - VPS Automated Deployment Script
# Bot: StdMailBot (Telegram Temp Mail Bot)
# Author: STD DEEPANSHU (https://deepanshu.in) | @STDBOTS
# ==============================================================================

set -e

echo "🚀 Starting StdMailBot VPS Setup..."

# Check root privileges
if [ "$EUID" -ne 0 ]; then
  echo "❌ Please run as root or with sudo."
  exit 1
fi

# Update package lists
echo "📦 Updating packages..."
apt-get update -y && apt-get install -y curl git make

# Install Docker if not present
if ! command -v docker &> /dev/null; then
    echo "🐳 Docker not found. Installing Docker..."
    curl -fsSL https://get.docker.com -o get-docker.sh
    sh get-docker.sh
    rm -f get-docker.sh
    systemctl enable docker
    systemctl start docker
fi

# Install Docker Compose plugin if not present
if ! docker compose version &> /dev/null; then
    echo "🐳 Installing Docker Compose plugin..."
    apt-get install -y docker-compose-plugin
fi

# Setup directory
INSTALL_DIR="/opt/stdbots/StdMailBot"
echo "📁 Setting up installation directory at $INSTALL_DIR..."
mkdir -p "$INSTALL_DIR"

if [ -d "$INSTALL_DIR/.git" ]; then
    echo "🔄 Existing installation found. Pulling latest updates..."
    cd "$INSTALL_DIR"
    git pull
else
    echo "📥 Cloning StdMailBot repository..."
    git clone https://github.com/StdBots/StdMailBot.git "$INSTALL_DIR"
    cd "$INSTALL_DIR"
fi

# Setup environment file
if [ ! -f "$INSTALL_DIR/.env" ]; then
    echo "⚙️ Creating .env file from template..."
    cp "$INSTALL_DIR/.env.example" "$INSTALL_DIR/.env"
    echo ""
    echo "⚠️ Action Required: Edit your credentials in $INSTALL_DIR/.env"
    echo "Run: nano $INSTALL_DIR/.env"
    echo "Then start the bot with: cd $INSTALL_DIR && docker compose up -d"
    exit 0
fi

# Launch Docker Compose
echo "🚀 Starting StdMailBot container stack..."
docker compose up -d --build

echo "✅ StdMailBot is now running successfully on your VPS!"
echo "📜 View logs with: docker compose -f $INSTALL_DIR/docker-compose.yml logs -f"
