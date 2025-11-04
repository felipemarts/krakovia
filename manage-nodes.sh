#!/bin/bash

# Script de gerenciamento de nós Krakovia

# Cores
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

show_menu() {
    clear
    echo -e "${BLUE}=========================================${NC}"
    echo -e "${BLUE}   Krakovia Nodes - Management Menu${NC}"
    echo -e "${BLUE}=========================================${NC}"
    echo ""
    echo -e "${GREEN}1)${NC} Start 3 nodes with API"
    echo -e "${GREEN}2)${NC} Start 3 nodes with API (clean data)"
    echo -e "${GREEN}3)${NC} Stop all nodes"
    echo -e "${GREEN}4)${NC} View status of all nodes"
    echo -e "${GREEN}5)${NC} View logs (live)"
    echo -e "${GREEN}6)${NC} Check ports"
    echo -e "${GREEN}7)${NC} Clean all data"
    echo -e "${GREEN}8)${NC} API Demo (interactive)"
    echo -e "${GREEN}9)${NC} View wallet addresses"
    echo -e "${GREEN}0)${NC} Exit"
    echo ""
    read -p "Choose an option: " option
    echo ""
}

start_nodes() {
    echo -e "${BLUE}Starting 3 nodes...${NC}"
    ./start-3nodes-api.sh
}

start_nodes_clean() {
    echo -e "${BLUE}Starting 3 nodes (clean data)...${NC}"
    ./start-3nodes-api.sh --clean
}

stop_nodes() {
    echo -e "${YELLOW}Stopping all nodes...${NC}"
    pkill -f "./bin/node" 2>/dev/null || true
    pkill -f "./bin/signaling" 2>/dev/null || true
    sleep 1
    echo -e "${GREEN}All nodes stopped${NC}"
    read -p "Press Enter to continue..."
}

view_status() {
    echo -e "${BLUE}Checking status of all nodes...${NC}"
    echo ""

    for port in 8080 8081 8082; do
        node_num=$((port - 8079))
        echo -e "${YELLOW}Node ${node_num} (localhost:${port}):${NC}"

        if nc -z localhost $port 2>/dev/null; then
            response=$(curl -s http://localhost:${port}/api/status 2>/dev/null)
            if [ $? -eq 0 ]; then
                echo "$response" | jq '.' 2>/dev/null || echo "$response"
            else
                echo -e "${RED}API not responding${NC}"
            fi
        else
            echo -e "${RED}Not running${NC}"
        fi
        echo ""
    done

    read -p "Press Enter to continue..."
}

view_logs() {
    echo -e "${BLUE}Which node's logs do you want to view?${NC}"
    echo "1) Node 1"
    echo "2) Node 2"
    echo "3) Node 3"
    echo "4) Signaling"
    echo "5) All (tmux required)"
    echo ""
    read -p "Choose: " log_choice

    case $log_choice in
        1) tail -f logs/node1.log ;;
        2) tail -f logs/node2.log ;;
        3) tail -f logs/node3.log ;;
        4) tail -f logs/signaling.log ;;
        5)
            if command -v tmux &> /dev/null; then
                tmux new-session -d -s krakovia-logs "tail -f logs/node1.log"
                tmux split-window -h "tail -f logs/node2.log"
                tmux split-window -v "tail -f logs/node3.log"
                tmux select-pane -t 0
                tmux split-window -v "tail -f logs/signaling.log"
                tmux attach-session -t krakovia-logs
            else
                echo -e "${RED}tmux not installed. Install with: brew install tmux${NC}"
                read -p "Press Enter to continue..."
            fi
            ;;
        *) echo "Invalid option" ;;
    esac
}

check_ports() {
    echo -e "${BLUE}Checking ports...${NC}"
    echo ""

    ports=(9000 9001 9002 9003 8080 8081 8082)
    names=("Signaling" "Node1 P2P" "Node2 P2P" "Node3 P2P" "Node1 API" "Node2 API" "Node3 API")

    for i in "${!ports[@]}"; do
        port=${ports[$i]}
        name=${names[$i]}

        if nc -z localhost $port 2>/dev/null; then
            pid=$(lsof -ti :$port 2>/dev/null)
            echo -e "${GREEN}✅ ${name}${NC} (port ${port}) - PID: ${pid}"
        else
            echo -e "${RED}❌ ${name}${NC} (port ${port}) - Not in use"
        fi
    done

    echo ""
    read -p "Press Enter to continue..."
}

clean_data() {
    echo -e "${RED}⚠️  WARNING: This will delete all blockchain data!${NC}"
    read -p "Are you sure? (yes/no): " confirm

    if [ "$confirm" = "yes" ]; then
        echo -e "${YELLOW}Cleaning data...${NC}"
        rm -rf ./data/node1 ./data/node2 ./data/node3
        rm -f logs/*.log
        echo -e "${GREEN}✅ All data cleaned${NC}"
    else
        echo -e "${YELLOW}Cancelled${NC}"
    fi

    read -p "Press Enter to continue..."
}

api_demo() {
    ./api-demo.sh
}

view_wallets() {
    echo -e "${BLUE}Wallet Addresses:${NC}"
    echo ""
    echo -e "${YELLOW}Node 1:${NC}"
    echo "  Address: 4b2aaf060ea4e382dbd121047539dc8312a6f301e72292214c804b461f0d35c9"
    echo "  API: http://localhost:8080"
    echo ""
    echo -e "${YELLOW}Node 2:${NC}"
    echo "  Address: fa878b92dedd74e3867adbf27154fabb66fd94da899bc8af96d987771dd01098"
    echo "  API: http://localhost:8081"
    echo ""
    echo -e "${YELLOW}Node 3:${NC}"
    echo "  Address: a4f9877eba2816799b9997e345c73eb63d4c6042e4c23aa07195f6746755be75"
    echo "  API: http://localhost:8082"
    echo ""
    echo -e "${YELLOW}Genesis Recipient:${NC} Node 1 (has 1,000,000,000 tokens initially)"
    echo ""
    read -p "Press Enter to continue..."
}

# Loop principal
while true; do
    show_menu

    case $option in
        1) start_nodes ;;
        2) start_nodes_clean ;;
        3) stop_nodes ;;
        4) view_status ;;
        5) view_logs ;;
        6) check_ports ;;
        7) clean_data ;;
        8) api_demo ;;
        9) view_wallets ;;
        0) echo "Goodbye!"; exit 0 ;;
        *) echo -e "${RED}Invalid option${NC}"; sleep 1 ;;
    esac
done
