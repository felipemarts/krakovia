# Krakovia Blockchain

Blockchain Proof of Stake (PoS) implementada em Golang com LevelDB e WebRTC para comunicação P2P.

## Estrutura do Projeto

```
krakovia/
├── cmd/
│   ├── node/          # Executável do nó da blockchain
│   └── signaling/     # Executável do servidor de signaling
├── pkg/
│   ├── node/          # Lógica do nó
│   ├── network/       # Comunicação WebRTC e peers
│   ├── signaling/     # Servidor de signaling WebSocket
│   └── storage/       # (futuro) Gerenciamento do LevelDB
├── internal/
│   └── config/        # (futuro) Configurações internas
└── data/              # Dados dos nós (criado em runtime)
```

## Componentes

### 1. Node (Nó)
- Gerencia o estado local da blockchain
- Mantém conexões com peers via WebRTC
- Armazena dados no LevelDB
- Broadcast de mensagens para a rede

### 2. Network (Rede)
- **WebRTCClient**: Cliente WebRTC para conexões P2P
- **Peer**: Representa uma conexão peer-to-peer
- Gerenciamento de data channels
- Troca de mensagens entre peers

### 3. Signaling Server
- Servidor WebSocket para coordenar conexões WebRTC
- Facilita a troca de SDPs e ICE candidates
- Mantém lista de peers conectados
- Encaminha mensagens de signaling

## Como Usar

### 1. Instalar Dependências

```bash
go mod download
```

### 2. Iniciar o Servidor de Signaling

```bash
go run cmd/signaling/main.go -addr :9000
```

O servidor de signaling ficará disponível em `ws://localhost:9000/ws`

### 3. Configurar Nós

Crie arquivos de configuração JSON para cada nó. Exemplos em `configs/`:

```json
// configs/node1.json
{
  "id": "node1",
  "address": ":9001",
  "db_path": "./data/node1",
  "signaling_server": "ws://localhost:9000/ws",
  "max_peers": 50,
  "min_peers": 5,
  "discovery_interval": 30
}
```

**Parâmetros de Configuração:**
- `id`: Identificador único do nó (obrigatório)
- `address`: Endereço e porta do nó (obrigatório)
- `db_path`: Caminho para banco de dados LevelDB (obrigatório)
- `signaling_server`: URL do servidor de signaling (obrigatório)
- `max_peers`: Número máximo de peers conectados (padrão: 50)
- `min_peers`: Número mínimo de peers desejado (padrão: 5)
- `discovery_interval`: Intervalo de descoberta em segundos (padrão: 30)

### 4. Iniciar Nós

Em terminais separados, inicie múltiplos nós:

```bash
# Nó 1
go run cmd/node/main.go -config configs/node1.json

# Nó 2
go run cmd/node/main.go -config configs/node2.json

# Nó 3
go run cmd/node/main.go -config configs/node3.json
```

### Parâmetros

**Servidor de Signaling:**
- `-addr`: Endereço do servidor (padrão: `:9000`)

**Nó:**
- `-config`: Caminho para arquivo JSON de configuração (obrigatório)

## Fluxo de Conexão

1. **Nó se conecta ao Signaling Server**
   - Envia mensagem de registro com seu ID
   - Recebe lista de peers já conectados

2. **Descoberta e Seleção de Peers**
   - Sistema de descoberta analisa peers disponíveis
   - Seleciona peers baseado em min/max_peers configurados
   - Conecta apenas aos peers necessários (não a todos)

3. **Estabelecimento de Conexão P2P**
   - Nó cria PeerConnection WebRTC para peers selecionados
   - Troca de ofertas/respostas via signaling server
   - Troca de ICE candidates
   - Estabelece data channel direto entre peers

4. **Comunicação P2P**
   - Mensagens são enviadas diretamente via data channel
   - Sem intermediação do servidor de signaling
   - Broadcast de mensagens para todos os peers conectados

5. **Descoberta Periódica**
   - A cada `discovery_interval` segundos:
     - Verifica se precisa de mais peers (< min_peers)
     - Desconecta peers excedentes (> max_peers)
     - Solicita nova lista de peers disponíveis
   - Mantém rede balanceada automaticamente

## Próximos Passos

- [ ] Implementar consenso PoS
- [ ] Adicionar criação e validação de blocos
- [ ] Implementar transações
- [ ] Adicionar carteiras e chaves
- [ ] Sistema de stake e validadores
- [ ] Finalização de blocos
- [ ] Sincronização de blockchain entre nós

## Tecnologias

- **Go 1.21+**: Linguagem de programação
- **Pion WebRTC**: Biblioteca WebRTC para Go
- **Gorilla WebSocket**: WebSocket para servidor de signaling
- **LevelDB**: Banco de dados local para persistência

## Desenvolvimento

### Build
- `make build`: Gera os binários em `bin/` (`bin/node` e `bin/signaling`).
- Alternativa direta: `go build ./cmd/...`.

### Testes
- `make test`: Executa os testes localizados em `./tests` com verbosidade.
- Para rodar todos os testes do módulo: `go test ./...`.

### Lint
- `golangci-lint run --timeout=5m`: Executa as verificações estáticas (requer `golangci-lint` instalado).
- Você pode instalar via `brew install golangci-lint` (macOS) ou consultar as instruções oficiais em https://golangci-lint.run/usage/install/.

## Licença

MIT
