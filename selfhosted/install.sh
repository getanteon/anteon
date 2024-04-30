#!/bin/bash

set -e

echo "⚡ Installing Anteon Self Hosted..."

echo "🔍 Checking prerequisites..."

# Function to check if a port is available
is_port_available() {
  local port="$1"

  if ! command -v lsof >/dev/null 2>&1; then
    echo "❌ lsof not found. Please install lsof and try again."
    exit 1
  fi

  if lsof -i :"$port" >/dev/null 2>&1; then
    echo "❌ Port $port is already in use. Free up the current port and try again."
    exit 1
  fi
}

is_port_available 8014
is_port_available 9901
is_port_available 6672
is_port_available 9086
is_port_available 8333

# Check if Git is installed
if ! command -v git >/dev/null 2>&1; then
  echo "❌ Git not found. Please install Git and try again."
  exit 1
fi

# Check if Docker is installed
if ! command -v docker >/dev/null 2>&1; then
  echo "❌ Docker not found. Please install Docker and try again."
  exit 1
fi

# Check if Docker Compose is installed
if ! command -v docker-compose >/dev/null 2>&1; then
  if ! docker compose version >/dev/null 2>&1; then
    echo "❌ Docker Compose not found. Please install Docker Compose and try again."
    exit 1
  fi
fi

# Check if Docker is running
if ! docker info >/dev/null 2>&1; then
  echo "❌ Docker is not running. Please start Docker and try again."
  exit 1
fi

echo "🚀 Starting installation of Anteon Self Hosted..."

REPO_DIR="$HOME/.anteon"

# Check if repository already exists
if [ -d "$REPO_DIR" ]; then
  echo "🔄 Repository already exists at $REPO_DIR - Attempting to update..."
  cd "$REPO_DIR"
  git checkout master
  cd "$REPO_DIR/selfhosted"
  git pull 2>&1 || {
    read -p "⚠️ Error updating repository. Clean and update? [Y/n]: " answer
    answer=${answer:-Y}
    if [[ $answer =~ ^[Yy]$ ]]; then
      git reset --hard >/dev/null 2>&1
      git clean -fd >/dev/null 2>&1
      git pull >/dev/null 2>&1
    fi
  }
else
  # Clone the repository
  echo "📦 Cloning repository to $REPO_DIR directory..."
  git clone https://github.com/getanteon/anteon.git "$REPO_DIR" >/dev/null 2>&1
  cd "$REPO_DIR"
  git checkout master >/dev/null 2>&1
  cd "$REPO_DIR/selfhosted"
fi

# Determine which compose command to use
COMPOSE_COMMAND="docker-compose"
if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
  COMPOSE_COMMAND="docker compose"
fi

echo "🚀 Deploying Anteon Self Hosted..."
$COMPOSE_COMMAND -f "$REPO_DIR/selfhosted/docker-compose.yml" up -d
docker pull busybox:1.34.1 >/dev/null 2>&1
echo ""
echo "⏳ Waiting for services to be ready..."
docker run --rm --network anteon busybox:1.34.1 /bin/sh -c "until nc -z nginx 80 && nc -z backend 8008 && nc -z hammermanager 8001 && nc -z rabbitmq-celery 5672 && nc -z rabbitmq-job 5672 && nc -z postgres 5432 && nc -z influxdb 8086 && nc -z seaweedfs 8333; do sleep 5; done"
echo "✅ Anteon Self Hosted installation complete!"
echo "📁 Installation directory: $REPO_DIR/selfhosted"
echo "🔥 To remove Anteon Self Hosted, run: cd $REPO_DIR/selfhosted && $COMPOSE_COMMAND down"
echo ""
echo "🌐 Open http://localhost:8014 in your browser to access the application."
