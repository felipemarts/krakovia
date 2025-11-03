#!/bin/bash

# Script para rodar teste de integração da Krakovia Blockchain
# Este script:
# 1. Inicia o servidor de signaling
# 2. Roda os testes de integração
# 3. Para o servidor de signaling

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

echo "==================================="
echo "Krakovia Blockchain Integration Test"
echo "==================================="
echo ""

# Cores para output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Função para cleanup
cleanup() {
    echo ""
    echo -e "${YELLOW}Cleaning up...${NC}"

    # Matar servidor de signaling
    if [ ! -z "$SIGNALING_PID" ]; then
        echo "Stopping signaling server (PID: $SIGNALING_PID)..."
        kill $SIGNALING_PID 2>/dev/null || true
        wait $SIGNALING_PID 2>/dev/null || true
    fi

    # Limpar diretórios temporários
    rm -rf /tmp/krakovia_test_* 2>/dev/null || true

    echo -e "${GREEN}Cleanup completed${NC}"
}

# Registrar cleanup ao sair
trap cleanup EXIT INT TERM

# 1. Compilar o projeto
echo -e "${YELLOW}Building project...${NC}"
cd "$PROJECT_ROOT"
go build -o bin/signaling ./cmd/signaling/ || {
    echo -e "${RED}Failed to build signaling server${NC}"
    exit 1
}
echo -e "${GREEN}✓ Build completed${NC}"
echo ""

# 2. Iniciar servidor de signaling em background
echo -e "${YELLOW}Starting signaling server...${NC}"
./bin/signaling > /tmp/signaling.log 2>&1 &
SIGNALING_PID=$!

# Aguardar servidor iniciar
sleep 2

# Verificar se servidor está rodando
if ! kill -0 $SIGNALING_PID 2>/dev/null; then
    echo -e "${RED}Failed to start signaling server${NC}"
    cat /tmp/signaling.log
    exit 1
fi

echo -e "${GREEN}✓ Signaling server started (PID: $SIGNALING_PID)${NC}"
echo ""

# 3. Rodar testes de integração
echo -e "${YELLOW}Running integration tests...${NC}"
echo ""

cd "$PROJECT_ROOT/tests"

# Rodar teste com timeout de 60 segundos
timeout 60 go test -v -run TestNodeIntegration || TEST_RESULT=$?

echo ""

if [ "${TEST_RESULT:-0}" -eq 0 ]; then
    echo -e "${GREEN}================================${NC}"
    echo -e "${GREEN}✓ All tests passed!${NC}"
    echo -e "${GREEN}================================${NC}"
    exit 0
else
    echo -e "${RED}================================${NC}"
    echo -e "${RED}✗ Tests failed${NC}"
    echo -e "${RED}================================${NC}"

    # Mostrar logs do signaling em caso de erro
    echo ""
    echo -e "${YELLOW}Signaling server logs:${NC}"
    cat /tmp/signaling.log

    exit 1
fi
