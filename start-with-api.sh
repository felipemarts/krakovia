#!/bin/bash

# Script para iniciar o nó Krakovia com API habilitada

echo "========================================="
echo "   Krakovia Node - Starting with API"
echo "========================================="
echo ""

# Verificar se o servidor de signaling está rodando
echo "Checking if signaling server is running on port 9000..."
if ! nc -z localhost 9000 2>/dev/null; then
    echo "⚠️  Warning: Signaling server not detected on port 9000"
    echo "   You may need to start it in another terminal:"
    echo "   ./bin/signaling -addr :9000"
    echo ""
    read -p "Continue anyway? (y/n) " -n 1 -r
    echo ""
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
else
    echo "✅ Signaling server is running"
fi

echo ""
echo "Starting node with HTTP API..."
echo "API will be available at: http://localhost:8080"
echo "Username: admin"
echo "Password: krakovia123"
echo ""
echo "Press Ctrl+C to stop"
echo ""

# Limpar dados antigos se solicitado
if [[ "$1" == "--clean" ]]; then
    echo "Cleaning old data..."
    rm -rf ./data/node1
    echo "✅ Data cleaned"
    echo ""
fi

# Iniciar o nó
./bin/node -config configs/node-with-api.example.json
