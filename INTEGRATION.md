# Integração Blockchain + Node de Rede

## Resumo da Integração

A integração entre o módulo blockchain e o node de rede foi concluída com sucesso! Agora os nodes na rede P2P WebRTC podem:

- ✅ Minerar blocos usando consenso Proof of Stake (PoS)
- ✅ Propagar blocos pela rede WebRTC
- ✅ Propagar transações pela rede
- ✅ Manter mempool sincronizado
- ✅ Validar transações e blocos recebidos
- ✅ Gerenciar saldo e stake

## Arquitetura

```
cmd/node/main.go
    ↓
pkg/node/node.go (Node Integrado)
    ├── WebRTC Client (Rede P2P)
    ├── Peer Discovery
    ├── LevelDB (Persistência)
    ├── Chain (Blockchain)
    ├── Mempool (Pool de Transações)
    ├── Miner (Minerador PoS)
    └── Wallet (Identidade)
```

## Mudanças Principais

### 1. pkg/node/node.go

**Adicionado:**
- Componentes blockchain: `wallet`, `chain`, `mempool`, `miner`
- Handlers de mensagens para blocos e transações
- Métodos de broadcast pela rede WebRTC
- API para interação com blockchain:
  - `StartMining()` / `StopMining()`
  - `CreateTransaction()`
  - `CreateStakeTransaction()` / `CreateUnstakeTransaction()`
  - `GetBalance()`, `GetStake()`, `GetNonce()`
  - `GetChainHeight()`, `GetMempoolSize()`
  - `PrintStats()`

**Modificado:**
- `Config` agora inclui: `Wallet`, `GenesisBlock`, `ChainConfig`
- `NewNode()` inicializa blockchain completa
- `AddPeer()` configura handlers de mensagens para cada peer
- `Stop()` para mineração antes de fechar

### 2. cmd/node/main.go

**Adicionado:**
- Carregamento de wallet a partir da configuração
- Criação do bloco gênesis
- Validação de configuração da wallet
- Exibição de informações da blockchain ao iniciar

### 3. Configuração (configs/node1.example.json)

**Estrutura:**
```json
{
  "id": "node1",
  "address": ":9001",
  "db_path": "./data/node1",
  "signaling_server": "ws://localhost:9000/ws",
  "max_peers": 50,
  "min_peers": 5,
  "discovery_interval": 30,
  "wallet": {
    "private_key": "...",
    "public_key": "...",
    "address": "..."
  },
  "genesis": {
    "timestamp": 1609459200,
    "recipient_addr": "...",
    "amount": 1000000000,
    "hash": "..."
  }
}
```

## Fluxo de Mensagens

### Propagação de Blocos

1. Minerador cria bloco via `Miner.TryMineBlock()`
2. Callback `onBlockCreated` é chamado
3. `Node.broadcastBlock()` serializa e envia para todos peers
4. Peers recebem via `HandlePeerMessage("block", data)`
5. `handleBlockMessage()` deserializa e valida
6. Bloco é adicionado à chain local
7. Transações são removidas do mempool
8. Bloco é propagado para outros peers (exceto remetente)

### Propagação de Transações

1. Usuário cria transação via `Node.CreateTransaction()`
2. Transação é assinada pelo `Miner`
3. Callback `onTxCreated` é chamado
4. `Node.broadcastTransaction()` serializa e envia para todos peers
5. Peers recebem via `HandlePeerMessage("transaction", data)`
6. `handleTransactionMessage()` deserializa e valida
7. Transação é adicionada ao mempool local
8. Transação é propagada para outros peers (exceto remetente)

## Como Usar

### 1. Iniciar Servidor de Signaling

```bash
go run cmd/signaling/main.go
```

### 2. Gerar Wallets (se necessário)

```bash
go run cmd/wallet-gen/main.go
```

### 3. Iniciar Nodes

```bash
# Node 1
./bin/node -config configs/node1.json

# Node 2 (em outro terminal)
./bin/node -config configs/node2.json
```

### 4. Interagir com o Node

Os métodos disponíveis para interação:

```go
// Iniciar/parar mineração
node.StartMining()
node.StopMining()

// Criar transação
tx, err := node.CreateTransaction(
    to,      // endereço destino
    amount,  // quantidade
    fee,     // taxa
    data,    // dados adicionais
)

// Stake
stx, err := node.CreateStakeTransaction(amount, fee)
ustx, err := node.CreateUnstakeTransaction(amount, fee)

// Consultas
balance := node.GetBalance()
stake := node.GetStake()
height := node.GetChainHeight()
mempoolSize := node.GetMempoolSize()

// Estatísticas
node.PrintStats()
stats := node.GetBlockchainStats()
```

## Próximos Passos

### Funcionalidades a Implementar

1. **CLI Interativo**
   - Comando para iniciar/parar mineração
   - Criar transações via linha de comando
   - Consultar saldo e estatísticas
   - Ver blockchain e mempool

2. **API HTTP/WebSocket**
   - Endpoints REST para operações básicas
   - WebSocket para eventos em tempo real
   - Dashboard web

3. **Sincronização de Blockchain**
   - Implementar `handleSyncRequest()` e `handleSyncResponse()`
   - Download de blocos faltantes
   - Resolução de forks

4. **Persistência**
   - Salvar blockchain no LevelDB
   - Carregar estado ao iniciar
   - Checkpoint periódico

5. **Melhorias de Consenso**
   - Timeouts para mineração
   - Penalidades para validadores offline
   - Recompensas dinâmicas

## Testes

Para testar a integração:

1. Inicie 3+ nodes
2. Verifique conexão entre peers
3. Inicie mineração em um node
4. Observe propagação de blocos
5. Crie transações e observe propagação
6. Verifique sincronização de mempool

## Problemas Conhecidos

- [ ] Sincronização inicial não implementada
- [ ] Persistência da blockchain não implementada
- [ ] Sem CLI para interação em runtime
- [ ] Logs muito verbosos (considerar níveis de log)

## Conclusão

A integração está funcional e pronta para testes! O sistema agora é uma blockchain P2P completa com:
- Rede descentralizada via WebRTC
- Consenso Proof of Stake
- Propagação de blocos e transações
- Validação e verificação
- Gerenciamento de estado
