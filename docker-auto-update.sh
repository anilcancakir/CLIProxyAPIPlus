#!/usr/bin/env bash
set -euo pipefail

# Log file location
LOG_FILE="/var/log/docker-auto-update.log"

# Logging function with timestamp
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" | tee -a "$LOG_FILE"
}

# Update compose-based service (cliproxy, antigravity)
# Always pull latest image and recreate if changed
update_compose_service() {
    local service_dir="$1"
    local service_name
    service_name=$(basename "$service_dir")

    log "Updating $service_name at $service_dir..."

    if [[ ! -d "$service_dir" ]]; then
        log "ERROR: Directory $service_dir does not exist, skipping"
        return 0
    fi

    cd "$service_dir" || {
        log "ERROR: Cannot cd to $service_dir"
        return 0
    }

    log "Pulling latest images for $service_name..."
    docker compose pull 2>&1 | tee -a "$LOG_FILE"

    log "Recreating $service_name with latest image..."
    docker compose up -d --remove-orphans 2>&1 | tee -a "$LOG_FILE"

    log "✓ $service_name update completed"
}

# Update git-based service (kiro-gateway)
# Always git pull + build + recreate
update_git_service() {
    local service_dir="$1"
    local service_name
    service_name=$(basename "$service_dir")

    log "Updating $service_name at $service_dir..."

    if [[ ! -d "$service_dir" ]]; then
        log "ERROR: Directory $service_dir does not exist, skipping"
        return 0
    fi

    cd "$service_dir" || {
        log "ERROR: Cannot cd to $service_dir"
        return 0
    }

    log "Pulling git changes for $service_name..."
    git pull 2>&1 | tee -a "$LOG_FILE"

    log "Building $service_name with latest base images..."
    docker compose build --pull 2>&1 | tee -a "$LOG_FILE"

    log "Recreating $service_name..."
    docker compose up -d --remove-orphans 2>&1 | tee -a "$LOG_FILE"

    log "✓ $service_name update completed"
}

# Main execution
log "=========================================="
log "Starting Docker auto-update routine"
log "=========================================="

# Update each service (with error isolation)
update_compose_service /opt/cliproxy || true
update_compose_service /opt/antigravity || true
update_git_service /opt/kiro-gateway || true

# Clean up dangling images
log "Cleaning up dangling images..."
docker image prune -f 2>&1 | tee -a "$LOG_FILE"
log "✓ Image cleanup completed"

log "=========================================="
log "Docker auto-update routine completed"
log "=========================================="
