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
  - `GetLastBlock()` - **Retorna último bloco para validação de consenso**
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

## Testes de Integração

### Suite de Testes Completa

A blockchain possui uma suite abrangente de testes de integração localizados em `tests/`:

#### 1. Testes Básicos de Integração

**`TestNodeIntegration`** - Testa integração completa entre 2 nós:
- Conexão WebRTC entre nós
- Mineração de blocos com PoS
- Criação e propagação de transações
- Sincronização de mempool
- Verificação de saldo e estado

**`TestThreeNodeConsensus`** - Testa consenso entre 3 nós:
- Múltiplos nós conectados
- Mineração concorrente
- Verificação de altura consensual
- **Validação de hash do último bloco** (garante ausência de fork)
- Todos os nós convergem para a mesma blockchain

#### 2. Testes de Rede

**`TestNodeConnection`** - Conexão básica WebRTC

**`TestMultipleNodesConnection`** - Conectividade entre múltiplos nós

**`TestMessageBroadcast`** - Broadcast de mensagens

**`TestNodeReconnection`** - Reconexão após desconexão

#### 3. Testes de Partição de Rede e Recuperação

**`TestNetworkPartitionRecovery`** - **Partição com 2 nós mineradores**:
- **Fase 1:** Sincronização inicial
  - Ambos os nós se conectam
  - Node1 faz stake e transfere tokens para Node2
  - Node2 também faz stake
  - **Ambos podem minerar** (competição PoS)

- **Fase 2:** Partição de rede
  - Node2 se desconecta (simula perda de pacotes)
  - Node1 continua minerando sozinho
  - Simula rede instável

- **Fase 3:** Reconexão
  - Node2 se reconecta
  - Reinicia mineração se tiver stake

- **Fase 4:** Verificação de consenso
  - ✅ Verifica altura igual
  - ✅ **Verifica hash do último bloco** (garante mesma cadeia)
  - ✅ Confirma que não há fork
  - ✅ Testa convergência PoS em redes instáveis

**`TestThreeNodePartitionWithStakeResolution`** - **Resolução de fork baseada em stake**:
- **Cenário:** 3 nós com stakes diferentes
  - Node1: 100k stake
  - Node2: 150k stake (maior individual)
  - Node3: 100k stake

- **Fase 1:** Setup inicial
  - Node1 distribui tokens
  - Todos fazem stake e iniciam mineração

- **Fase 2:** Partição de rede
  - **Node3 se isola** (100k stake sozinho)
  - **Node1+Node2 continuam conectados** (250k stake combinado)
  - Ambos grupos minerando → **FORK**

- **Fase 3:** Reconexão
  - Node3 se reconecta à rede

- **Fase 4:** Resolução de fork
  - ✅ **Node3 descarta sua chain isolada**
  - ✅ **Node3 adota a chain de Node1+Node2** (maior stake total)
  - ✅ Verifica que todos têm o mesmo hash
  - ✅ **Testa regra fundamental do PoS:** chain com maior stake total prevalece

#### 4. Testes de Protocolo Gossip

**Propagação e validação:**
- `TestGossipPropagation` - Propagação de mensagens gossip
- `TestGossipDeduplication` - Detecção de duplicatas
- `TestGossipRateLimitAttack` - Proteção contra flood
- `TestGossipInvalidMessages` - Rejeição de mensagens inválidas

#### 5. Testes de Discovery

**Gerenciamento de peers:**
- `TestPeerLimitEnforcement` - Limites de peers respeitados
- `TestPeerDiscovery` - Descoberta periódica
- `TestMinimumPeersMaintenance` - Manutenção de peers mínimos

### Validações Críticas

**Consenso por Altura + Hash:**
```go
// Antes (INSUFICIENTE - pode ter fork):
if height1 == height2 { ✓ } // Mesma altura, mas chains diferentes!

// Agora (CORRETO - garante mesma chain):
if height1 == height2 && hash1 == hash2 { ✓ } // Mesma altura E mesma blockchain
```

**Resolução de Fork baseada em Stake:**
```
Chain A (Node1+Node2): stake total = 250k ✓ VENCEDOR
Chain B (Node3):       stake total = 100k ✗ DESCARTADA
```

### Executar Testes

```bash
# Todos os testes de integração
go test -v ./tests/

# Teste específico de partição de rede
go test -v -run TestNetworkPartitionRecovery ./tests/

# Teste de resolução de fork por stake
go test -v -run TestThreeNodePartitionWithStakeResolution ./tests/

# Testes de consenso
go test -v -run TestThreeNodeConsensus ./tests/

# Testes rápidos (pula testes longos)
go test -v -short ./tests/
```

### Cobertura de Testes

- ✅ Conectividade WebRTC
- ✅ Descoberta e gerenciamento de peers
- ✅ Protocolo gossip e propagação
- ✅ Mineração PoS e consenso
- ✅ Sincronização de blockchain
- ✅ Partição de rede e recuperação
- ✅ **Resolução de fork baseada em stake total**
- ✅ **Validação de consenso por hash (detecta forks)**
- ✅ Transações e mempool
- ✅ Validação de blocos e transações

## Problemas Conhecidos

- [x] ~~Sincronização inicial não implementada~~ ✅ Implementada e testada
- [x] ~~Persistência da blockchain não implementada~~ ✅ Implementada (LevelDB)
- [ ] Sem CLI para interação em runtime
- [ ] Logs muito verbosos (considerar níveis de log)
- [ ] Testes ocasionalmente não mineram blocos (timeout muito curto para PoS)

## Conclusão

A integração está funcional e pronta para testes! O sistema agora é uma blockchain P2P completa com:
- Rede descentralizada via WebRTC
- Consenso Proof of Stake
- Propagação de blocos e transações
- Validação e verificação
- Gerenciamento de estado
