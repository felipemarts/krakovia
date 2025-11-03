# Teste de Integração - Krakovia Blockchain

Este documento descreve os testes de integração que validam a sincronização, propagação de transações, mineração PoS e convergência da blockchain.

## 📋 O Que é Testado

### 1. Sincronização de Blockchain
- ✅ Node 2 inicia depois do Node 1
- ✅ Node 2 solicita blocos faltantes do Node 1
- ✅ Node 2 sincroniza completamente com Node 1
- ✅ Ambos nodes chegam à mesma altura de blockchain

### 2. Propagação de Transações
- ✅ Transação criada no Node 1 é propagada para Node 2
- ✅ Transação criada no Node 2 é propagada para Node 1
- ✅ Mempool mantém consistência entre nodes
- ✅ Transações são removidas do mempool após mineração

### 3. Mineração Proof of Stake (PoS)
- ✅ Node com stake suficiente pode minerar
- ✅ Blocos são criados no intervalo configurado
- ✅ Recompensas de mineração são atribuídas corretamente
- ✅ Consenso de validador funciona

### 4. Convergência da Blockchain
- ✅ Múltiplos nodes convergem para mesma blockchain
- ✅ Não há forks persistentes
- ✅ Estado final é consistente entre todos nodes

## 🚀 Como Executar

### Opção 1: Demo Interativa (Recomendado)

A demo interativa mostra visualmente o que está acontecendo:

```bash
./run-demo.sh
```

**O que a demo faz:**
1. Inicia servidor de signaling
2. Cria 2 wallets
3. Cria bloco genesis (1 bilhão de tokens para wallet 1)
4. Inicia Node 1
5. Node 1 faz stake de 100,000 tokens
6. Node 1 inicia mineração
7. Node 1 cria transação de 50,000 tokens para wallet 2
8. Inicia Node 2 (sincroniza automaticamente)
9. Node 2 cria transação de resposta para wallet 1
10. Mostra estatísticas finais e verifica convergência

**Saída esperada:**
```
==============================================
  Krakovia Blockchain - Integration Demo
==============================================

[Setup] Creating wallets...
  Wallet 1: a3f5c8b2d9e1f4a6c7b8...
  Wallet 2: b4g6d9c3e0f2a5b7c8d9...

[Setup] Creating genesis block...
  Genesis hash: 00000000a1b2c3d4...
  Initial supply: 1,000,000,000 tokens to wallet 1

[Node 1] ✓ Started
  Balance: 1000000000
  Height: 0

[Node 1] ✓ Stake transaction: a1b2c3d4...
[Node 1] ✓ Mining started
[Node 1] ✓ Mined 3 blocks

[Node 1] ✓ Transaction: e5f6a7b8...
  From: wallet 1
  To: wallet 2
  Amount: 50,000
  Fee: 5

[Node 2] ✓ Started
  Initial height: 0

[Node 2] ✓ Synchronized
  Height: 4
  Balance: 50000 (received from transaction)

==============================================
              Final Statistics
==============================================

[Node 1]
  Height: 6
  Balance: 999849995
  Stake: 100000
  Mempool: 0 transactions
  Peers: 1

[Node 2]
  Height: 6
  Balance: 45000
  Mempool: 0 transactions
  Peers: 1

==============================================
              Verification
==============================================
✓ Chains synchronized (height 6)
✓ Transaction propagated (wallet 2 balance: 45000)
✓ Staking working (stake: 100000)
✓ PoS mining working (6 blocks)
```

### Opção 2: Teste Automatizado

Para rodar o teste de integração completo:

```bash
cd tests
./run_integration.sh
```

**O que o teste faz:**
- Compila o projeto
- Inicia servidor de signaling automaticamente
- Roda testes Go com verificações
- Para o servidor ao finalizar
- Limpa diretórios temporários

### Opção 3: Teste Manual com Go

Se você já tem o servidor de signaling rodando:

```bash
# Terminal 1: Servidor de signaling
go run cmd/signaling/main.go

# Terminal 2: Rodar teste
cd tests
go test -v -run TestNodeIntegration
```

## 📊 Cenários de Teste

### Teste 1: Integração de 2 Nodes (`TestNodeIntegration`)

**Duração:** ~20 segundos

**Cenário:**
1. Node 1 inicia com genesis block
2. Node 1 minera alguns blocos
3. Node 1 cria transação
4. Node 2 inicia (deve sincronizar)
5. Verificar que ambos têm mesma altura
6. Node 2 cria transação
7. Verificar propagação para Node 1

**Validações:**
- ✅ Sincronização completa
- ✅ Transações propagadas
- ✅ Blocos minerados
- ✅ Convergência de estado

### Teste 2: Consenso com 3 Nodes (`TestThreeNodeConsensus`)

**Duração:** ~15 segundos

**Cenário:**
1. 3 nodes iniciam com mesmo genesis
2. Node 1 inicia mineração
3. Aguardar propagação de blocos
4. Verificar que todos têm mesma altura

**Validações:**
- ✅ Todos nodes convergem
- ✅ Sem forks
- ✅ Consenso mantido

## 🔍 O Que Observar

### Logs Importantes

**Conexão de Peer:**
```
Peer test_node2 connected to node test_node1
[test_node2] Requested sync from test_node1 (from height 1)
```

**Sincronização:**
```
[test_node1] Sent 5 blocks to test_node2 (height 1-5)
[test_node2] Successfully synced 5 blocks, current height: 5
```

**Propagação de Bloco:**
```
[test_node1] Broadcasting block 3 to all peers
[test_node2] Received block 3 (hash: a1b2c3d4) from test_node1
[test_node2] Block 3 added to chain successfully
```

**Propagação de Transação:**
```
[test_node1] Broadcasting transaction a1b2c3d4 to all peers
[test_node2] Received transaction a1b2c3d4 from test_node1
[test_node2] Transaction a1b2c3d4 added to mempool
```

## 🐛 Troubleshooting

### Erro: "Failed to connect to signaling server"

**Solução:** Certifique-se que o servidor de signaling está rodando na porta 9000:
```bash
# Em outro terminal
go run cmd/signaling/main.go
```

### Erro: "Timeout waiting for connection"

**Possível causa:** Firewall bloqueando conexões WebRTC

**Solução:** Verifique configurações de firewall ou use `localhost` ao invés de IP externo

### Nodes não sincronizam

**Debug:**
1. Verificar logs do servidor de signaling
2. Confirmar que nodes estão conectados como peers
3. Verificar se sincronização foi solicitada:
   ```
   [node] Requested sync from peer (from height X)
   ```

### Transações não propagam

**Verificar:**
1. Data channel está aberto entre peers
2. Handlers de mensagem estão registrados
3. Serialização/deserialização está funcionando

## 📈 Métricas de Sucesso

| Métrica | Valor Esperado |
|---------|---------------|
| Tempo de sincronização | < 5 segundos |
| Propagação de transação | < 1 segundo |
| Intervalo entre blocos | ~250ms |
| Taxa de convergência | 100% |

## 🔧 Configurações de Teste

```go
// configs/node1.example.json
{
  "id": "node1",
  "address": ":9001",
  "max_peers": 50,
  "min_peers": 5,
  "discovery_interval": 30,
  "wallet": { ... },
  "genesis": { ... }
}
```

**Parâmetros importantes:**
- `discovery_interval`: Frequência de descoberta de peers (segundos)
- `max_peers`: Máximo de conexões simultâneas
- `min_peers`: Mínimo de peers antes de solicitar mais

## 📝 Estrutura dos Testes

```
tests/
├── integration_test.go          # Testes automatizados
├── run_integration.sh           # Script para rodar testes
└── discovery_test.go            # Testes de descoberta de peers

cmd/
└── integration-demo/
    └── main.go                  # Demo interativa

run-demo.sh                      # Script para rodar demo
```

## ✅ Checklist de Validação

Após executar os testes, verificar:

- [ ] Nodes se conectam via WebRTC
- [ ] Sincronização acontece automaticamente
- [ ] Blocos são propagados para todos peers
- [ ] Transações são propagadas para todos peers
- [ ] Mempool é atualizado corretamente
- [ ] Mineração PoS funciona
- [ ] Recompensas são distribuídas
- [ ] Estado final é consistente
- [ ] Não há leaks de recursos (goroutines, conexões)

## 🎯 Próximos Passos

Testes adicionais a implementar:

1. **Teste de Reorganização**
   - Simular fork temporário
   - Verificar resolução de conflitos

2. **Teste de Rede Particionada**
   - Dividir rede em 2 grupos
   - Reconectar e verificar convergência

3. **Teste de Carga**
   - Múltiplas transações simultâneas
   - Verificar throughput e latência

4. **Teste de Falha**
   - Node crashando e reconectando
   - Verificar recuperação de estado

5. **Teste de Validação**
   - Tentar adicionar blocos inválidos
   - Verificar rejeição

## 📚 Referências

- [INTEGRATION.md](INTEGRATION.md) - Documentação da integração
- [README.md](README.md) - Visão geral do projeto
- [pkg/node/node.go](pkg/node/node.go) - Implementação do node
- [pkg/blockchain/](pkg/blockchain/) - Módulo blockchain
