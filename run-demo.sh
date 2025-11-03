#!/bin/bash

# Script para rodar demo interativa da Krakovia Blockchain

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Cores
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
CYAN='\033[0;36m'
NC='\033[0m'

echo -e "${CYAN}"
echo "=============================================="
echo "  Krakovia Blockchain - Interactive Demo"
echo "=============================================="
echo -e "${NC}"

# Cleanup
cleanup() {
    echo ""
    echo -e "${YELLOW}Cleaning up...${NC}"
    if [ ! -z "$SIGNALING_PID" ]; then
        echo "Stopping signaling server..."
        kill $SIGNALING_PID 2>/dev/null || true
        wait $SIGNALING_PID 2>/dev/null || true
    fi
    rm -rf /tmp/krakovia_demo_* 2>/dev/null || true
    echo -e "${GREEN}Cleanup completed${NC}"
}

trap cleanup EXIT INT TERM

# Build
echo -e "${YELLOW}Building project...${NC}"
cd "$SCRIPT_DIR"

go build -o bin/signaling ./cmd/signaling/ || {
    echo -e "${RED}Failed to build signaling server${NC}"
    exit 1
}

go build -o bin/integration-demo ./cmd/integration-demo/ || {
    echo -e "${RED}Failed to build demo${NC}"
    exit 1
}

echo -e "${GREEN}✓ Build completed${NC}"
echo ""

# Start signaling server
echo -e "${YELLOW}Starting signaling server...${NC}"
./bin/signaling > /tmp/signaling_demo.log 2>&1 &
SIGNALING_PID=$!

sleep 2

if ! kill -0 $SIGNALING_PID 2>/dev/null; then
    echo -e "${RED}Failed to start signaling server${NC}"
    cat /tmp/signaling_demo.log
    exit 1
fi

echo -e "${GREEN}✓ Signaling server started (PID: $SIGNALING_PID)${NC}"
echo ""

# Run demo
echo -e "${YELLOW}Starting demo...${NC}"
echo ""

./bin/integration-demo || {
    echo -e "${RED}Demo failed${NC}"
    echo ""
    echo -e "${YELLOW}Signaling server logs:${NC}"
    cat /tmp/signaling_demo.log
    exit 1
}
