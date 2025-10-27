.PHONY: help build test clean run-signaling run-node1 run-node2 run-node3 deps

help:
	@echo "Krakovia Blockchain - Comandos disponíveis:"
	@echo ""
	@echo "  make deps            - Baixar dependências"
	@echo "  make build           - Compilar binários"
	@echo "  make test            - Executar testes"
	@echo "  make clean           - Limpar arquivos gerados"
	@echo "  make run-signaling   - Iniciar servidor de signaling"
	@echo "  make run-node1       - Iniciar nó 1"
	@echo "  make run-node2       - Iniciar nó 2"
	@echo "  make run-node3       - Iniciar nó 3"
	@echo ""

deps:
	@echo "Baixando dependências..."
	go mod download
	go mod tidy

build:
	@echo "Compilando binários..."
	go build -o bin/signaling cmd/signaling/main.go
	go build -o bin/node cmd/node/main.go

test:
	@echo "Executando testes..."
	go test ./tests -v -timeout 60s

test-clean:
	@echo "Limpando dados de teste..."
	rm -rf tests/test-data

clean: test-clean
	@echo "Limpando arquivos gerados..."
	rm -rf bin/
	rm -rf data/
	go clean

run-signaling:
	@echo "Iniciando servidor de signaling..."
	go run cmd/signaling/main.go -addr :9000

run-node1:
	@echo "Iniciando nó 1..."
	go run cmd/node/main.go -config configs/node1.json

run-node2:
	@echo "Iniciando nó 2..."
	go run cmd/node/main.go -config configs/node2.json

run-node3:
	@echo "Iniciando nó 3..."
	go run cmd/node/main.go -config configs/node3.json

# Comandos de desenvolvimento
dev-signaling:
	@echo "Modo desenvolvimento - Servidor de signaling"
	air -c .air-signaling.toml

dev-node:
	@echo "Modo desenvolvimento - Nó"
	air -c .air-node.toml
