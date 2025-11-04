#!/bin/bash

# Script para iniciar 3 nós Krakovia com API habilitada
# Cada nó terá sua própria interface web em portas diferentes

set -e

# Cores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${BLUE}=========================================${NC}"
echo -e "${BLUE}   Krakovia Network - 3 Nodes with API${NC}"
echo -e "${BLUE}=========================================${NC}"
echo ""

# Função para limpar processos ao sair
cleanup() {
    echo -e "\n${YELLOW}Shutting down all nodes...${NC}"

    # Matar todos os processos filhos
    jobs -p | xargs -r kill 2>/dev/null || true

    # Esperar um pouco para os processos terminarem gracefully
    sleep 2

    echo -e "${GREEN}All nodes stopped${NC}"
    exit 0
}

# Registrar handler para Ctrl+C
trap cleanup SIGINT SIGTERM

# Função para verificar se uma porta está em uso
check_port() {
    if nc -z localhost $1 2>/dev/null; then
        return 0
    else
        return 1
    fi
}

# Verificar se o signaling server está rodando
echo -e "${BLUE}[1/4] Checking signaling server...${NC}"
if ! check_port 9000; then
    echo -e "${YELLOW}⚠️  Signaling server not running on port 9000${NC}"
    echo -e "${YELLOW}   Starting signaling server...${NC}"
    ./bin/signaling -addr :9000 > logs/signaling.log 2>&1 &
    SIGNALING_PID=$!
    sleep 2

    if check_port 9000; then
        echo -e "${GREEN}✅ Signaling server started (PID: $SIGNALING_PID)${NC}"
    else
        echo -e "${RED}❌ Failed to start signaling server${NC}"
        exit 1
    fi
else
    echo -e "${GREEN}✅ Signaling server already running${NC}"
fi

echo ""

# Criar diretório de logs se não existir
mkdir -p logs

# Limpar dados antigos se solicitado
if [[ "$1" == "--clean" ]]; then
    echo -e "${BLUE}[2/4] Cleaning old data...${NC}"
    rm -rf ./data/node1 ./data/node2 ./data/node3
    rm -f logs/node*.log
    echo -e "${GREEN}✅ Data cleaned${NC}"
    echo ""
else
    echo -e "${BLUE}[2/4] Keeping existing data (use --clean to reset)${NC}"
    echo ""
fi

# Verificar se as portas das APIs estão disponíveis
echo -e "${BLUE}[3/4] Checking API ports availability...${NC}"
PORTS_OK=true

for port in 8080 8081 8082; do
    if check_port $port; then
        echo -e "${RED}❌ Port $port is already in use${NC}"
        PORTS_OK=false
    else
        echo -e "${GREEN}✅ Port $port is available${NC}"
    fi
done

if [ "$PORTS_OK" = false ]; then
    echo -e "\n${RED}Some ports are in use. Please free them or stop existing nodes.${NC}"
    exit 1
fi

echo ""

# Iniciar os 3 nós
echo -e "${BLUE}[4/4] Starting nodes...${NC}"
echo ""

# Node 1
echo -e "${GREEN}Starting Node 1...${NC}"
echo -e "  - P2P Port: 9001"
echo -e "  - API Port: 8080"
echo -e "  - URL: ${BLUE}http://localhost:8080${NC}"
./bin/node -config configs/node1-api.json > logs/node1.log 2>&1 &
NODE1_PID=$!
sleep 1

# Node 2
echo -e "${GREEN}Starting Node 2...${NC}"
echo -e "  - P2P Port: 9002"
echo -e "  - API Port: 8081"
echo -e "  - URL: ${BLUE}http://localhost:8081${NC}"
./bin/node -config configs/node2-api.json > logs/node2.log 2>&1 &
NODE2_PID=$!
sleep 1

# Node 3
echo -e "${GREEN}Starting Node 3...${NC}"
echo -e "  - P2P Port: 9003"
echo -e "  - API Port: 8082"
echo -e "  - URL: ${BLUE}http://localhost:8082${NC}"
./bin/node -config configs/node3-api.json > logs/node3.log 2>&1 &
NODE3_PID=$!
sleep 2

echo ""
echo -e "${BLUE}=========================================${NC}"
echo -e "${GREEN}✅ All nodes started successfully!${NC}"
echo -e "${BLUE}=========================================${NC}"
echo ""
echo -e "${YELLOW}Access the nodes:${NC}"
echo -e "  • Node 1: ${BLUE}http://localhost:8080${NC}"
echo -e "  • Node 2: ${BLUE}http://localhost:8081${NC}"
echo -e "  • Node 3: ${BLUE}http://localhost:8082${NC}"
echo ""
echo -e "${YELLOW}Credentials:${NC}"
echo -e "  • Username: ${GREEN}admin${NC}"
echo -e "  • Password: ${GREEN}krakovia123${NC}"
echo ""
echo -e "${YELLOW}Logs:${NC}"
echo -e "  • Signaling: logs/signaling.log"
echo -e "  • Node 1: logs/node1.log"
echo -e "  • Node 2: logs/node2.log"
echo -e "  • Node 3: logs/node3.log"
echo ""
echo -e "${YELLOW}Monitor logs:${NC}"
echo -e "  tail -f logs/node1.log"
echo -e "  tail -f logs/node2.log"
echo -e "  tail -f logs/node3.log"
echo ""
echo -e "${RED}Press Ctrl+C to stop all nodes${NC}"
echo ""

# Aguardar indefinidamente (os processos rodam em background)
wait
