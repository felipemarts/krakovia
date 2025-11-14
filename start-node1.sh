#!/bin/bash

# Script para iniciar Node 1 com API

set -e

# Cores
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}=========================================${NC}"
echo -e "${BLUE}   Starting Krakovia Node 1${NC}"
echo -e "${BLUE}=========================================${NC}"
echo ""

# Criar diretório de logs
mkdir -p logs

# Verificar se o signaling server está rodando
if ! nc -z localhost 9000 2>/dev/null; then
    echo -e "${YELLOW}⚠️  Signaling server not running${NC}"
    echo -e "${YELLOW}   Starting signaling server...${NC}"
    ./bin/signaling -addr :9000 > logs/signaling.log 2>&1 &
    sleep 2
    echo -e "${GREEN}✅ Signaling server started${NC}"
fi

# Verificar se a porta da API está disponível
if nc -z localhost 8080 2>/dev/null; then
    echo -e "${RED}❌ Port 8080 is already in use${NC}"
    exit 1
fi

echo -e "${GREEN}Starting Node 1...${NC}"
echo -e "  - Node ID: ${BLUE}node1${NC}"
echo -e "  - P2P Port: ${BLUE}9001${NC}"
echo -e "  - API Port: ${BLUE}8080${NC}"
echo -e "  - API URL: ${BLUE}http://localhost:8080${NC}"
echo -e "  - Username: ${GREEN}admin${NC}"
echo -e "  - Password: ${GREEN}admin${NC}"
echo ""

# Limpar dados antigos se solicitado
if [[ "$1" == "--clean" ]]; then
    echo -e "${YELLOW}Cleaning old data...${NC}"
    rm -rf ./data/node1
    echo -e "${GREEN}✅ Data cleaned${NC}"
    echo ""
fi

echo -e "${YELLOW}Press Ctrl+C to stop${NC}"
echo ""

# Verificar se deve iniciar com mineração
MINE_FLAG=""
if [[ "$1" == "--mine" ]] || [[ "$2" == "--mine" ]]; then
    MINE_FLAG="-mine"
    echo -e "${GREEN}🔨 Auto-mining enabled${NC}"
    echo ""
fi

# Iniciar o nó
./bin/node -config configs/node1-api.json $MINE_FLAG
