#!/bin/sh
# ============================================
# Health Check Script for go_ProFiBus
# ============================================
# This script performs comprehensive health checks
# Usage: ./healthcheck.sh [host] [port]
# ============================================

set -e

# Configuration
HOST="${1:-localhost}"
PORT="${2:-8080}"
TIMEOUT=10
MAX_RETRIES=3

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo "${RED}[ERROR]${NC} $1"
}

# Check if curl is available
if ! command -v curl >/dev/null 2>&1; then
    log_error "curl is not installed"
    exit 1
fi

# Function to check endpoint
check_endpoint() {
    local endpoint=$1
    local expected_status=${2:-200}
    local description=$3

    log_info "Checking $description..."

    response=$(curl -s -o /dev/null -w "%{http_code}" \
        --connect-timeout $TIMEOUT \
        --max-time $((TIMEOUT * 2)) \
        "http://${HOST}:${PORT}${endpoint}" 2>/dev/null || echo "000")

    if [ "$response" = "$expected_status" ]; then
        log_info "$description: ${GREEN}OK${NC} (HTTP $response)"
        return 0
    else
        log_error "$description: ${RED}FAILED${NC} (HTTP $response, expected $expected_status)"
        return 1
    fi
}

# Function to check endpoint with retry
check_with_retry() {
    local endpoint=$1
    local expected_status=${2:-200}
    local description=$3
    local retries=0

    while [ $retries -lt $MAX_RETRIES ]; do
        if check_endpoint "$endpoint" "$expected_status" "$description"; then
            return 0
        fi

        retries=$((retries + 1))
        if [ $retries -lt $MAX_RETRIES ]; then
            log_warn "Retrying ($retries/$MAX_RETRIES)..."
            sleep 2
        fi
    done

    log_error "$description failed after $MAX_RETRIES attempts"
    return 1
}

# Main health checks
main() {
    log_info "Starting health checks for go_ProFiBus at ${HOST}:${PORT}"
    echo "========================================"

    local failed=0

    # 1. Check main health endpoint
    if ! check_with_retry "/health" "200" "Health endpoint"; then
        failed=$((failed + 1))
    fi

    # 2. Check ping endpoint
    if ! check_with_retry "/ping" "200" "Ping endpoint"; then
        failed=$((failed + 1))
    fi

    # 3. Check API readiness
    if ! check_endpoint "/api/v1/pipelines" "200" "API readiness"; then
        log_warn "API might not be ready yet (this is okay during startup)"
    fi

    echo "========================================"

    if [ $failed -eq 0 ]; then
        log_info "${GREEN}All health checks passed!${NC}"
        exit 0
    else
        log_error "${RED}$failed health check(s) failed!${NC}"
        exit 1
    fi
}

# Run main function
main
