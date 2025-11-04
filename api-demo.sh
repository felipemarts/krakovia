#!/bin/bash

# Script de demonstração da API dos nós Krakovia
# Exemplos de como usar a API via cURL

set -e

# Cores
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Credenciais
USERNAME="admin"
PASSWORD="krakovia123"

echo -e "${BLUE}=========================================${NC}"
echo -e "${BLUE}   Krakovia API - Demo Commands${NC}"
echo -e "${BLUE}=========================================${NC}"
echo ""

# Função para fazer requisição
api_request() {
    local node=$1
    local endpoint=$2
    local method=${3:-GET}
    local data=${4:-}

    local port=$((8080 + node - 1))
    local url="http://localhost:${port}${endpoint}"

    echo -e "${YELLOW}[Node ${node}] ${method} ${endpoint}${NC}"

    if [ "$method" = "GET" ]; then
        if [[ "$endpoint" == "/api/wallet/"* ]] || [[ "$endpoint" == "/api/mining/"* ]]; then
            # Endpoints protegidos
            curl -s -u ${USERNAME}:${PASSWORD} "${url}" | jq '.' 2>/dev/null || curl -s -u ${USERNAME}:${PASSWORD} "${url}"
        else
            # Endpoints públicos
            curl -s "${url}" | jq '.' 2>/dev/null || curl -s "${url}"
        fi
    else
        curl -s -u ${USERNAME}:${PASSWORD} -X ${method} -H "Content-Type: application/json" -d "${data}" "${url}" | jq '.' 2>/dev/null || curl -s -u ${USERNAME}:${PASSWORD} -X ${method} -H "Content-Type: application/json" -d "${data}" "${url}"
    fi

    echo ""
}

# Menu
show_menu() {
    echo -e "${GREEN}Choose an action:${NC}"
    echo "  1) View status of all nodes"
    echo "  2) View balance of all nodes"
    echo "  3) View last block from Node 1"
    echo "  4) View peers from all nodes"
    echo "  5) Start mining on Node 1"
    echo "  6) Stop mining on Node 1"
    echo "  7) Transfer tokens from Node 1 to Node 2"
    echo "  8) Stake tokens on Node 1"
    echo "  9) View blockchain info from all nodes"
    echo "  0) Exit"
    echo ""
    read -p "Enter option: " option
    echo ""
}

# Ações
action_status_all() {
    echo -e "${BLUE}=== Status of All Nodes ===${NC}"
    echo ""
    for i in 1 2 3; do
        api_request $i "/api/status"
        echo ""
    done
}

action_balance_all() {
    echo -e "${BLUE}=== Balance of All Nodes ===${NC}"
    echo ""
    for i in 1 2 3; do
        api_request $i "/api/wallet/balance"
        echo ""
    done
}

action_last_block() {
    echo -e "${BLUE}=== Last Block from Node 1 ===${NC}"
    echo ""
    api_request 1 "/api/blockchain/last-block"
}

action_peers_all() {
    echo -e "${BLUE}=== Peers from All Nodes ===${NC}"
    echo ""
    for i in 1 2 3; do
        api_request $i "/api/peers"
        echo ""
    done
}

action_start_mining() {
    echo -e "${BLUE}=== Starting Mining on Node 1 ===${NC}"
    echo ""
    api_request 1 "/api/mining/start" "POST"
}

action_stop_mining() {
    echo -e "${BLUE}=== Stopping Mining on Node 1 ===${NC}"
    echo ""
    api_request 1 "/api/mining/stop" "POST"
}

action_transfer() {
    echo -e "${BLUE}=== Transfer from Node 1 to Node 2 ===${NC}"
    echo ""

    # Endereço do Node 2
    NODE2_ADDR="fa878b92dedd74e3867adbf27154fabb66fd94da899bc8af96d987771dd01098"

    read -p "Enter amount to transfer: " amount
    read -p "Enter fee (default 10): " fee
    fee=${fee:-10}

    local data="{\"to\":\"${NODE2_ADDR}\",\"amount\":${amount},\"fee\":${fee},\"data\":\"Transfer from Node 1 to Node 2\"}"

    api_request 1 "/api/wallet/transfer" "POST" "${data}"
}

action_stake() {
    echo -e "${BLUE}=== Stake on Node 1 ===${NC}"
    echo ""

    read -p "Enter amount to stake: " amount
    read -p "Enter fee (default 10): " fee
    fee=${fee:-10}

    local data="{\"amount\":${amount},\"fee\":${fee}}"

    api_request 1 "/api/wallet/stake" "POST" "${data}"
}

action_blockchain_info() {
    echo -e "${BLUE}=== Blockchain Info from All Nodes ===${NC}"
    echo ""
    for i in 1 2 3; do
        api_request $i "/api/blockchain/info"
        echo ""
    done
}

# Loop principal
while true; do
    show_menu

    case $option in
        1) action_status_all ;;
        2) action_balance_all ;;
        3) action_last_block ;;
        4) action_peers_all ;;
        5) action_start_mining ;;
        6) action_stop_mining ;;
        7) action_transfer ;;
        8) action_stake ;;
        9) action_blockchain_info ;;
        0) echo "Goodbye!"; exit 0 ;;
        *) echo -e "${YELLOW}Invalid option${NC}" ;;
    esac

    echo ""
    read -p "Press Enter to continue..."
    echo ""
    echo ""
done
