#!/usr/bin/env bash
set -euo pipefail

# Astra-System Installer
# Cross-platform bootstrap (macOS + Linux)
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/xdfkenny/Astra-System/main/installer/scripts/install.sh | bash
#   curl -fsSL ... | bash -s -- remove    # Uninstall Astra-System
#
# Options (via env vars):
#   CHANNEL=stable        Release channel (stable|beta|canary)
#   INSTALL_DIR=/opt/astra  Install directory
#   DATA_DIR=/var/lib/astra Data directory
#   GHCR_TOKEN=xxx        GitHub Container Registry token (read:packages)

CHANNEL="${CHANNEL:-stable}"
GHCR_USER="${GHCR_USER:-xdfkenny}"
GHCR_TOKEN="${GHCR_TOKEN:-}"
INSTALL_DIR="${INSTALL_DIR:-}"
DATA_DIR="${DATA_DIR:-}"
REPO_OWNER="${REPO_OWNER:-xdfkenny}"
REPO_NAME="${REPO_NAME:-Astra-System}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

info()  { printf "${GREEN}  ✓ %s${NC}\n" "$*"; }
warn()  { printf "${YELLOW}  ! %s${NC}\n" "$*"; }
err()   { printf "${RED}  ✗ %s${NC}\n" "$*"; }
header(){ printf "\n${CYAN}→ %s${NC}\n" "$*"; }

detect_platform() {
    case "$(uname -s)" in
        Darwin*)  echo "darwin" ;;
        Linux*)   echo "linux" ;;
        MINGW*|MSYS*|CYGWIN*) echo "windows" ;;
        *)        echo "unknown" ;;
    esac
}

PLATFORM=$(detect_platform)
ARCH=$(uname -m)
case "$ARCH" in
    x86_64|amd64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
esac

if [ "$PLATFORM" = "windows" ]; then
    echo "Windows detected — please use the PowerShell bootstrap or the .exe installer."
    echo "  https://github.com/${REPO_OWNER}/${REPO_NAME}/releases"
    exit 1
fi

if [ "$PLATFORM" = "unknown" ]; then
    err "Unsupported platform: $(uname -s)"
    exit 1
fi

# ─── Docker compose detection ────────────────────────────────────

DOCKER_COMPOSE="docker compose"
if command -v docker-compose &>/dev/null; then
    DOCKER_COMPOSE="docker-compose"
fi

# ─── Uninstall ───────────────────────────────────────────────────

do_remove() {
    header "Removing Astra-System"
    echo ""

    local compose_dir
    if [ -n "$DATA_DIR" ]; then
        compose_dir="${DATA_DIR}/compose"
    elif [ "$PLATFORM" = "darwin" ]; then
        compose_dir="/usr/local/var/astra-system/compose"
    else
        compose_dir="/var/lib/astra-system/compose"
    fi

    if [ -f "$compose_dir/docker-compose.yml" ]; then
        info "Stopping and removing containers..."
        $DOCKER_COMPOSE -p astra-system -f "$compose_dir/docker-compose.yml" down --volumes 2>/dev/null || true
    fi

    if [ "$PLATFORM" = "darwin" ]; then
        info "Removing LaunchDaemon..."
        sudo launchctl unload /Library/LaunchDaemons/com.astra-system.updater.plist 2>/dev/null || true
        sudo rm -f /Library/LaunchDaemons/com.astra-system.updater.plist
    else
        info "Removing systemd timer..."
        sudo systemctl stop astra-updater.timer 2>/dev/null || true
        sudo systemctl disable astra-updater.timer 2>/dev/null || true
        sudo rm -f /etc/systemd/system/astra-updater.service /etc/systemd/system/astra-updater.timer
        sudo systemctl daemon-reload 2>/dev/null || true
    fi

    local data_dir
    if [ -n "$DATA_DIR" ]; then
        data_dir="$DATA_DIR"
    elif [ "$PLATFORM" = "darwin" ]; then
        data_dir="/usr/local/var/astra-system"
    else
        data_dir="/var/lib/astra-system"
    fi

    info "Removing data directory..."
    sudo rm -rf "$data_dir" 2>/dev/null || rm -rf "$data_dir"

    local install_dir
    if [ -n "$INSTALL_DIR" ]; then
        install_dir="$INSTALL_DIR"
    elif [ "$PLATFORM" = "darwin" ]; then
        install_dir="/Applications/Astra-System"
    else
        install_dir="/opt/astra-system"
    fi

    info "Removing install directory..."
    sudo rm -rf "$install_dir" 2>/dev/null || rm -rf "$install_dir"

    echo ""
    info "Astra-System has been removed"
    exit 0
}

if [ "${1:-}" = "remove" ] || [ "${1:-}" = "uninstall" ]; then
    do_remove
fi

# ─── Install flow ────────────────────────────────────────────────

echo "╔═══════════════════════════════════════════╗"
echo "║     Astra-System Installer v0.2.0         ║"
echo "║  Production-grade Self-Checkout Platform  ║"
printf "║  Platform: %s/%s                    ║\n" "$PLATFORM" "$ARCH"
echo "╚═══════════════════════════════════════════╝"
echo ""

check_docker_binary() {
    if ! command -v docker &>/dev/null; then
        err "Docker is not installed"
        return 2
    fi
}

check_docker_running() {
    local version
    version=$(docker version --format '{{.Server.Version}}' 2>/dev/null || true)
    if [ -n "$version" ]; then
        info "Docker $version is running"
        return 0
    fi
    if [ "$PLATFORM" = "darwin" ]; then
        warn "Docker Desktop is installed but not running. Starting it..."
        open -a Docker 2>/dev/null || true
    fi
    return 1
}

wait_for_docker() {
    local timeout=${1:-120}
    local elapsed=0
    warn "Waiting for Docker to start..."
    while [ $elapsed -lt $timeout ]; do
        if docker version --format '{{.Server.Version}}' &>/dev/null; then
            info "Docker is ready"
            return 0
        fi
        sleep 3
        elapsed=$((elapsed + 3))
        if [ $((elapsed % 15)) -eq 0 ]; then
            warn "Still waiting for Docker... (${elapsed}s)"
        fi
    done
    err "Docker did not start within ${timeout}s"
    return 1
}

install_docker_macos() {
    warn "Downloading Docker Desktop for Mac..."
    local arch_flag
    if [ "$ARCH" = "arm64" ]; then
        arch_flag="arm64"
    else
        arch_flag="amd64"
    fi
    local url="https://desktop.docker.com/mac/main/${arch_flag}/Docker.dmg"
    local dmg="/tmp/Docker.dmg"
    curl -fsSL -o "$dmg" "$url"
    warn "Installing Docker Desktop..."
    sudo hdiutil attach "$dmg" -quiet -nobrowse
    sudo cp -R "/Volumes/Docker/Docker.app" /Applications/
    sudo hdiutil detach "/Volumes/Docker" -quiet
    warn "Starting Docker Desktop for the first time..."
    open -a Docker
    rm -f "$dmg"
    echo ""
    printf "${YELLOW}  ╔══════════════════════════════════════════════════════════╗${NC}\n"
    printf "${YELLOW}  ║  RESTART REQUIRED                                       ║${NC}\n"
    printf "${YELLOW}  ║  1. Docker Desktop is now installed                     ║${NC}\n"
    printf "${YELLOW}  ║  2. It will start automatically — wait for the whale    ║${NC}\n"
    printf "${YELLOW}  ║  3. Re-run this script to complete Astra-System setup   ║${NC}\n"
    printf "${YELLOW}  ╚══════════════════════════════════════════════════════════╝${NC}\n"
}

install_docker_linux() {
    warn "Installing Docker Engine..."
    if command -v apt-get &>/dev/null; then
        curl -fsSL https://get.docker.com | sudo sh
        sudo usermod -aG docker "$USER"
        warn "You may need to log out and back in for group changes to take effect."
    elif command -v yum &>/dev/null; then
        curl -fsSL https://get.docker.com | sudo sh
        sudo usermod -aG docker "$USER"
    elif command -v pacman &>/dev/null; then
        sudo pacman -S --noconfirm docker
        sudo systemctl enable docker
        sudo usermod -aG docker "$USER"
    else
        err "No supported package manager found. Install Docker manually: https://docs.docker.com/engine/install/"
        return 1
    fi
    sudo systemctl start docker
    info "Docker Engine installed"
}

# ─── GHCR Auth ──────────────────────────────────────────────────

check_ghcr_auth() {
    docker manifest inspect "ghcr.io/${GHCR_USER}/astra-system/gateway:latest" &>/dev/null
}

do_ghcr_login() {
    if [ -z "$GHCR_TOKEN" ]; then
        printf "${YELLOW}  ! GHCR_TOKEN not set${NC}\n"
        printf "  Astra-System images are hosted on GitHub Container Registry.\n"
        printf "  You may need a GitHub PAT with 'read:packages' scope.\n"
        printf "  Create one at: https://github.com/settings/tokens/new?scopes=read:packages\n"
        printf "  Then re-run with: ${CYAN}GHCR_TOKEN=ghp_xxx ./install.sh${NC}\n"
        printf "  ${YELLOW}Continuing without auth — may fail if images require auth.${NC}\n"
        return 1
    fi
    echo "$GHCR_TOKEN" | docker login ghcr.io -u "$GHCR_USER" --password-stdin &>/dev/null || {
        err "GHCR login failed"
        return 1
    }
    info "Logged in to ghcr.io"
}

# ─── Generate Docker Compose ────────────────────────────────────

generate_compose() {
    local compose_dir="$1"
    mkdir -p "$compose_dir"

    cat > "$compose_dir/docker-compose.yml" << 'COMPOSE'
x-env: &env
  DB_HOST: postgres
  DB_PORT: "5432"
  DB_USER: ${POSTGRES_USER:-astra}
  DB_PASSWORD: ${POSTGRES_PASSWORD}
  DB_NAME: ${POSTGRES_DB:-astra_service}
  REDIS_ADDR: redis:6379
  NATS_URL: nats:4222

services:
  postgres:
    image: postgres:16-alpine
    container_name: astra-postgres
    restart: unless-stopped
    environment:
      POSTGRES_USER: ${POSTGRES_USER:-astra}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
      POSTGRES_DB: ${POSTGRES_DB:-astra_service}
    ports:
      - "5432:5432"
    volumes:
      - astra_postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${POSTGRES_USER:-astra} -d ${POSTGRES_DB:-astra_service}"]
      interval: 10s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    container_name: astra-redis
    restart: unless-stopped
    ports:
      - "6379:6379"
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 3s
      retries: 5

  nats:
    image: nats:2-alpine
    container_name: astra-nats
    restart: unless-stopped
    ports:
      - "4222:4222"
      - "8222:8222"
    healthcheck:
      test: ["CMD-SHELL", "wget -qO- http://localhost:8222/healthz || exit 1"]
      interval: 10s
      timeout: 5s
      retries: 5

  gateway:
    image: ghcr.io/xdfkenny/astra-system/gateway:latest
    container_name: astra-gateway
    restart: unless-stopped
    ports:
      - "8080:8080"
    environment:
      <<: *env
    depends_on:
      postgres: { condition: service_healthy }
      redis: { condition: service_healthy }
      nats: { condition: service_healthy }

  menu-service:
    image: ghcr.io/xdfkenny/astra-system/menu-service:latest
    container_name: astra-menu
    restart: unless-stopped
    ports:
      - "8085:8085"
    environment:
      <<: *env
    depends_on:
      postgres: { condition: service_healthy }
      nats: { condition: service_healthy }

  cart-service:
    image: ghcr.io/xdfkenny/astra-system/cart-service:latest
    container_name: astra-cart
    restart: unless-stopped
    ports:
      - "8081:8081"
    environment:
      <<: *env
    depends_on:
      postgres: { condition: service_healthy }
      nats: { condition: service_healthy }

  order-service:
    image: ghcr.io/xdfkenny/astra-system/order-service:latest
    container_name: astra-order
    restart: unless-stopped
    ports:
      - "8083:8083"
    environment:
      <<: *env
    depends_on:
      postgres: { condition: service_healthy }
      nats: { condition: service_healthy }

  inventory-service:
    image: ghcr.io/xdfkenny/astra-system/inventory-service:latest
    container_name: astra-inventory
    restart: unless-stopped
    ports:
      - "8082:8082"
    environment:
      <<: *env
    depends_on:
      postgres: { condition: service_healthy }
      nats: { condition: service_healthy }

  sync-service:
    image: ghcr.io/xdfkenny/astra-system/sync-service:latest
    container_name: astra-sync
    restart: unless-stopped
    ports:
      - "8087:8087"
    environment:
      <<: *env
    depends_on:
      postgres: { condition: service_healthy }
      nats: { condition: service_healthy }

  payment-orchestrator:
    image: ghcr.io/xdfkenny/astra-system/payment-orchestrator:latest
    container_name: astra-payment
    restart: unless-stopped
    ports:
      - "8086:8086"
    environment:
      <<: *env
    depends_on:
      postgres: { condition: service_healthy }
      nats: { condition: service_healthy }

  kiosk:
    image: ghcr.io/xdfkenny/astra-system/kiosk:latest
    container_name: astra-kiosk
    restart: unless-stopped
    ports:
      - "${KIOSK_PORT:-80}:80"
    depends_on:
      - gateway

volumes:
  astra_postgres_data:
COMPOSE
    info "docker-compose.yml generated"
}

write_env_file() {
    local compose_dir="$1"
    local pg_pass="${POSTGRES_PASSWORD:-astra_$(date +%s)}"
    cat > "$compose_dir/.env" << EOF
POSTGRES_USER=astra
POSTGRES_PASSWORD=${pg_pass}
POSTGRES_DB=astra_service
KIOSK_PORT=${KIOSK_PORT:-80}
EOF
    info "Environment file written"
}

# ─── Updater Service ────────────────────────────────────────────

install_updater_launchd() {
    local data_dir="$1"
    local plist="/Library/LaunchDaemons/com.astra-system.updater.plist"
    if [ ! -w "$(dirname "$plist")" ] && [ "$(id -u)" != "0" ]; then
        warn "Cannot write to /Library/LaunchDaemons — skip auto-update setup."
        warn "Re-run with sudo or run: sudo astra-installer --silent --data-dir ${data_dir}"
        return
    fi
    sudo tee "$plist" > /dev/null << EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.astra-system.updater</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/astra-installer</string>
        <string>--silent</string>
        <string>--data-dir</string>
        <string>${data_dir}</string>
        <string>--channel</string>
        <string>${CHANNEL}</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>StartInterval</key>
    <integer>21600</integer>
    <key>StandardOutPath</key>
    <string>${data_dir}/logs/updater.log</string>
    <key>StandardErrorPath</key>
    <string>${data_dir}/logs/updater.log</string>
    <key>KeepAlive</key>
    <false/>
</dict>
</plist>
EOF
    sudo chmod 644 "$plist"
    sudo launchctl load "$plist"
    info "LaunchDaemon installed — checks for updates every 6 hours"
}

install_updater_systemd() {
    local data_dir="$1"
    local service="/etc/systemd/system/astra-updater.service"
    local timer="/etc/systemd/system/astra-updater.timer"

    if [ ! -w "$(dirname "$service")" ] && [ "$(id -u)" != "0" ]; then
        warn "Cannot write to /etc/systemd/system — skip auto-update setup."
        warn "Re-run with sudo or run: sudo astra-installer --silent --data-dir ${data_dir}"
        return
    fi

    sudo tee "$service" > /dev/null << EOF
[Unit]
Description=Astra-System Update Agent
After=network-online.target docker.service
Wants=network-online.target docker.service

[Service]
Type=oneshot
ExecStart=/usr/local/bin/astra-installer --silent --data-dir ${data_dir} --channel ${CHANNEL}
EOF

    sudo tee "$timer" > /dev/null << EOF
[Unit]
Description=Astra-System Update Timer (every 6 hours)

[Timer]
OnBootSec=5min
OnUnitActiveSec=6h
Persistent=true

[Install]
WantedBy=timers.target
EOF

    sudo systemctl daemon-reload
    sudo systemctl enable astra-updater.timer
    sudo systemctl start astra-updater.timer
    info "Systemd timer installed — checks for updates every 6 hours"
}

# ─── Main ───────────────────────────────────────────────────────

main() {
    if [ -z "$INSTALL_DIR" ]; then
        if [ "$PLATFORM" = "darwin" ]; then
            INSTALL_DIR="/Applications/Astra-System"
        else
            INSTALL_DIR="/opt/astra-system"
        fi
    fi
    if [ -z "$DATA_DIR" ]; then
        if [ "$PLATFORM" = "darwin" ]; then
            DATA_DIR="/usr/local/var/astra-system"
        else
            DATA_DIR="/var/lib/astra-system"
        fi
    fi

    local COMPOSE_DIR="${DATA_DIR}/compose"
    mkdir -p "${COMPOSE_DIR}" "${DATA_DIR}/logs" "${DATA_DIR}/config"

    # Step 1: Check Docker
    header "Checking prerequisites"
    set +e
    check_docker_binary
    local docker_status=$?
    set -e

    if [ $docker_status -eq 2 ]; then
        echo ""
        read -r -p "  Install Docker now? (Y/n): " choice
        if [ "$choice" != "n" ] && [ "$choice" != "N" ]; then
            if [ "$PLATFORM" = "darwin" ]; then
                install_docker_macos
            else
                install_docker_linux
            fi
            echo ""
            warn "Please restart your computer / log out, then re-run this script."
            exit 0
        else
            err "Docker is required. Exiting."
            exit 1
        fi
    else
        set +e
        check_docker_running
        local running_status=$?
        set -e
        if [ $running_status -eq 1 ]; then
            wait_for_docker
        fi
    fi

    # Step 2: GHCR auth
    header "Checking registry authentication"
    if ! check_ghcr_auth; then
        do_ghcr_login
    else
        info "Already authenticated to GHCR"
    fi

    # Step 3: Generate compose
    header "Generating Docker Compose configuration"
    generate_compose "$COMPOSE_DIR"
    write_env_file "$COMPOSE_DIR"

    # Step 4: Pull images
    header "Pulling Docker images"
    $DOCKER_COMPOSE -p astra-system -f "${COMPOSE_DIR}/docker-compose.yml" pull --quiet
    info "Images downloaded"

    # Step 5: Start services
    header "Starting services"
    $DOCKER_COMPOSE -p astra-system -f "${COMPOSE_DIR}/docker-compose.yml" up -d
    info "Services started"

    # Step 6: Wait for health
    header "Waiting for services to become healthy"
    local deadline=$((SECONDS + 120))
    while [ $SECONDS -lt $deadline ]; do
        local unhealthy
        unhealthy=$($DOCKER_COMPOSE -p astra-system -f "${COMPOSE_DIR}/docker-compose.yml" ps --format "{{.Status}}" 2>/dev/null | grep -cE "Exit|unhealthy" || true)
        local all_up
        all_up=$($DOCKER_COMPOSE -p astra-system -f "${COMPOSE_DIR}/docker-compose.yml" ps --format "{{.Status}}" 2>/dev/null | grep -cE "Up|healthy" || true)
        local total
        total=$($DOCKER_COMPOSE -p astra-system -f "${COMPOSE_DIR}/docker-compose.yml" ps --format "{{.Status}}" 2>/dev/null | wc -l | tr -d ' ')
        if [ "$total" -gt 0 ] && [ "$all_up" -eq "$total" ] && [ "$unhealthy" -eq 0 ]; then
            info "All services healthy"
            break
        fi
        sleep 3
    done
    if [ $SECONDS -ge $deadline ]; then
        warn "Some services may not be ready yet"
        $DOCKER_COMPOSE -p astra-system -f "${COMPOSE_DIR}/docker-compose.yml" ps
    fi

    # Step 7: Register update agent
    header "Registering auto-update service"
    if [ "$PLATFORM" = "darwin" ]; then
        install_updater_launchd "$DATA_DIR"
    else
        install_updater_systemd "$DATA_DIR"
    fi

    # Done
    echo ""
    printf "${GREEN}  ╔═══════════════════════════════════════════╗${NC}\n"
    printf "${GREEN}  ║  Astra-System is now running!            ║${NC}\n"
    printf "${GREEN}  ║                                          ║${NC}\n"
    printf "${GREEN}  ║  Kiosk:    http://localhost:80           ║${NC}\n"
    printf "${GREEN}  ║  Dashboard: http://localhost:8080        ║${NC}\n"
    printf "${GREEN}  ╚═══════════════════════════════════════════╝${NC}\n"
    echo ""
    info "Data directory: ${DATA_DIR}"
    info "Compose file:   ${COMPOSE_DIR}/docker-compose.yml"
    info "Logs:           ${DATA_DIR}/logs"
    info "Auto-updates:   every 6 hours (${CHANNEL} channel)"
}

main "$@"
