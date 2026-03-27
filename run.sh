#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log()  { echo -e "${GREEN}[run]${NC} $*"; }
warn() { echo -e "${YELLOW}[run]${NC} $*"; }
err()  { echo -e "${RED}[run]${NC} $*"; }

usage() {
    echo "Usage: $0 <command>"
    echo ""
    echo "Commands:"
    echo "  infra       Start infrastructure (MySQL, Redis, NATS) via docker compose"
    echo "  migrate     Run SQL migrations against MySQL"
    echo "  engine      Build and run the engine server"
    echo "  adapter     Build and run the adapter-stub"
    echo "  all         Start infra, run migrations, then run engine + adapter"
    echo "  docker      Run everything via docker compose (including engine)"
    echo "  stop        Stop docker compose infrastructure"
    echo "  clean       Stop infra and remove volumes"
    echo ""
    exit 1
}

wait_for_mysql() {
    log "Waiting for MySQL to be ready..."
    for i in $(seq 1 30); do
        if docker compose exec -T mysql mysqladmin ping -h 127.0.0.1 -u root -ppassword --silent 2>/dev/null; then
            log "MySQL is ready"
            return 0
        fi
        sleep 1
    done
    err "MySQL did not become ready in time"
    return 1
}

wait_for_nats() {
    log "Waiting for NATS to be ready..."
    for i in $(seq 1 15); do
        if curl -sf http://localhost:8222/healthz >/dev/null 2>&1; then
            log "NATS is ready"
            return 0
        fi
        sleep 1
    done
    err "NATS did not become ready in time"
    return 1
}

cmd_infra() {
    log "Starting infrastructure (MySQL, Redis, NATS)..."
    docker compose up -d mysql redis nats
    wait_for_mysql
    wait_for_nats
    log "Infrastructure is up"
}

cmd_migrate() {
    log "Running migrations..."
    for f in migrations/*.up.sql; do
        log "  Applying $(basename "$f")..."
        docker compose exec -T mysql mysql -u root -ppassword event_engine < "$f" 2>/dev/null || true
    done
    log "Migrations complete"
}

cmd_engine() {
    log "Building engine..."
    go build -o bin/engine ./cmd/engine
    log "Starting engine..."
    exec ./bin/engine
}

cmd_adapter() {
    log "Building adapter-stub..."
    go build -o bin/adapter-stub ./cmd/adapter-stub
    log "Starting adapter-stub..."
    exec ./bin/adapter-stub
}

cmd_all() {
    cmd_infra
    cmd_migrate

    log "Starting engine in background..."
    go build -o bin/engine ./cmd/engine
    ./bin/engine &
    ENGINE_PID=$!

    sleep 2

    log "Starting adapter-stub in background..."
    go build -o bin/adapter-stub ./cmd/adapter-stub
    ./bin/adapter-stub &
    ADAPTER_PID=$!

    trap "log 'Stopping processes...'; kill $ENGINE_PID $ADAPTER_PID 2>/dev/null; wait" EXIT INT TERM

    log "Everything is running:"
    log "  Engine:    http://localhost:8080"
    log "  Dashboard: http://localhost:8080/"
    log "  Health:    http://localhost:8080/health"
    log "  Metrics:   http://localhost:8080/metrics"
    log "  NATS Mon:  http://localhost:8222"
    log ""
    log "Press Ctrl+C to stop"

    wait
}

cmd_docker() {
    log "Starting everything via docker compose..."
    docker compose up --build
}

cmd_stop() {
    log "Stopping docker compose services..."
    docker compose down
}

cmd_clean() {
    log "Stopping services and removing volumes..."
    docker compose down -v
}

[[ $# -lt 1 ]] && usage

case "$1" in
    infra)   cmd_infra ;;
    migrate) cmd_migrate ;;
    engine)  cmd_engine ;;
    adapter) cmd_adapter ;;
    all)     cmd_all ;;
    docker)  cmd_docker ;;
    stop)    cmd_stop ;;
    clean)   cmd_clean ;;
    *)       usage ;;
esac
