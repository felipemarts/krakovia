# Krakovia - Guia de 3 Nós com API

Este guia mostra como executar uma rede completa de 3 nós Krakovia, cada um com sua própria API HTTP e interface web.

## 📋 Visão Geral

Este setup cria uma rede blockchain completa com:
- **3 nós** conectados via P2P (WebRTC)
- **3 interfaces web** (uma por nó)
- **Sincronização automática** entre os nós
- **Checkpoints** habilitados
- **APIs REST** completas

## 🚀 Início Rápido

### Opção 1: Script Automático (Recomendado)

```bash
./start-3nodes-api.sh
```

### Opção 2: Limpar Dados e Começar do Zero

```bash
./start-3nodes-api.sh --clean
```

Isso irá:
1. ✅ Verificar se o signaling server está rodando (inicia automaticamente se não estiver)
2. ✅ Verificar disponibilidade das portas
3. ✅ Iniciar os 3 nós em background
4. ✅ Mostrar URLs de acesso

## 🌐 Acessando os Nós

Após iniciar, você pode acessar cada nó no navegador:

| Nó | Interface Web | P2P Port | Wallet Address |
|----|--------------|----------|----------------|
| Node 1 | http://localhost:8080 | 9001 | `4b2aaf06...` |
| Node 2 | http://localhost:8081 | 9002 | `fa878b92...` |
| Node 3 | http://localhost:8082 | 9003 | `a4f9877e...` |

**Credenciais (todos os nós):**
- Usuário: `admin`
- Senha: `krakovia123`

## 📊 Estrutura da Rede

```
                    Signaling Server
                    (localhost:9000)
                           |
        +------------------+------------------+
        |                  |                  |
    Node 1             Node 2             Node 3
   (API :8080)       (API :8081)       (API :8082)
   (P2P :9001)       (P2P :9002)       (P2P :9003)
        |                  |                  |
        +------------------+------------------+
                    P2P Network (WebRTC)
```

## 🎮 Usando a Rede

### 1. Monitorar os Nós

Acompanhe os logs em tempo real:

```bash
# Node 1
tail -f logs/node1.log

# Node 2
tail -f logs/node2.log

# Node 3
tail -f logs/node3.log

# Signaling
tail -f logs/signaling.log
```

### 2. Interface Web

Abra os 3 nós em abas diferentes do navegador:
- http://localhost:8080 (Node 1)
- http://localhost:8081 (Node 2)
- http://localhost:8082 (Node 3)

Você verá:
- Status do nó em tempo real
- Saldo e stake
- Último bloco
- Peers conectados
- Formulários para transferências e stake

### 3. Script de Demonstração da API

Use o script interativo para testar a API:

```bash
./api-demo.sh
```

Menu de opções:
1. Ver status de todos os nós
2. Ver saldo de todos os nós
3. Ver último bloco do Node 1
4. Ver peers de todos os nós
5. Iniciar mineração no Node 1
6. Parar mineração no Node 1
7. Transferir tokens do Node 1 para Node 2
8. Fazer stake no Node 1
9. Ver info da blockchain de todos os nós

### 4. API via cURL

#### Consultar Status (sem autenticação)

```bash
# Node 1
curl http://localhost:8080/api/status | jq

# Node 2
curl http://localhost:8081/api/status | jq

# Node 3
curl http://localhost:8082/api/status | jq
```

#### Consultar Saldo (com autenticação)

```bash
# Node 1
curl -u admin:krakovia123 http://localhost:8080/api/wallet/balance | jq

# Node 2
curl -u admin:krakovia123 http://localhost:8081/api/wallet/balance | jq

# Node 3
curl -u admin:krakovia123 http://localhost:8082/api/wallet/balance | jq
```

#### Iniciar Mineração (Node 1)

```bash
curl -u admin:krakovia123 -X POST http://localhost:8080/api/mining/start
```

#### Transferir Tokens (Node 1 → Node 2)

```bash
curl -u admin:krakovia123 \
  -X POST \
  -H "Content-Type: application/json" \
  -d '{
    "to": "fa878b92dedd74e3867adbf27154fabb66fd94da899bc8af96d987771dd01098",
    "amount": 1000,
    "fee": 10,
    "data": "Transfer from Node 1 to Node 2"
  }' \
  http://localhost:8080/api/wallet/transfer | jq
```

#### Fazer Stake (Node 1)

```bash
curl -u admin:krakovia123 \
  -X POST \
  -H "Content-Type: application/json" \
  -d '{
    "amount": 10000,
    "fee": 10
  }' \
  http://localhost:8080/api/wallet/stake | jq
```

## 🧪 Cenários de Teste

### Cenário 1: Mineração e Sincronização

1. Inicie os 3 nós
2. Inicie mineração no Node 1 (interface web ou API)
3. Observe os blocos sendo propagados para os outros nós
4. Verifique que a altura da chain é a mesma em todos os nós

### Cenário 2: Transferências

1. Verifique o saldo inicial do Node 1 (1 bilhão de tokens)
2. Faça uma transferência do Node 1 para Node 2
3. Aguarde o bloco ser minerado
4. Verifique que o saldo do Node 2 aumentou

### Cenário 3: Stake e Consenso

1. Faça stake no Node 1, Node 2 e Node 3
2. Inicie mineração em todos os nós
3. Observe qual nó está sendo selecionado para validar blocos
4. Nós com mais stake devem ser selecionados mais frequentemente

### Cenário 4: Partição de Rede

1. Pare o Node 2 (Ctrl+C no terminal ou kill)
2. Continue minerando no Node 1
3. Reinicie o Node 2
4. Observe a sincronização automática via checkpoint

## 📁 Arquivos de Configuração

Cada nó tem seu próprio arquivo de configuração:

- `configs/node-with-api.example.json` - Node 1
- `configs/node2-api.json` - Node 2
- `configs/node3-api.json` - Node 3

### Diferenças entre os nós:

| Parâmetro | Node 1 | Node 2 | Node 3 |
|-----------|--------|--------|--------|
| ID | node1 | node2 | node3 |
| P2P Port | 9001 | 9002 | 9003 |
| API Port | 8080 | 8081 | 8082 |
| DB Path | ./data/node1 | ./data/node2 | ./data/node3 |
| Wallet | Diferente | Diferente | Diferente |

**Nota:** Todos os nós usam o mesmo genesis block para garantir compatibilidade.

## 🛑 Parando os Nós

### Opção 1: Parar Todos (se usou o script)

Pressione **Ctrl+C** no terminal onde executou `start-3nodes-api.sh`

### Opção 2: Parar Individualmente

```bash
# Encontrar os PIDs
ps aux | grep "./bin/node"

# Matar um nó específico
kill <PID>
```

### Opção 3: Script de Parada

```bash
pkill -f "./bin/node"
pkill -f "./bin/signaling"
```

## 🔍 Monitoramento

### Ver todos os processos

```bash
ps aux | grep -E "(node|signaling)" | grep -v grep
```

### Ver uso de portas

```bash
lsof -i :9000  # Signaling
lsof -i :9001  # Node 1 P2P
lsof -i :9002  # Node 2 P2P
lsof -i :9003  # Node 3 P2P
lsof -i :8080  # Node 1 API
lsof -i :8081  # Node 2 API
lsof -i :8082  # Node 3 API
```

### Ver estatísticas da chain

```bash
# Via API (todos os nós devem ter a mesma altura)
curl -s http://localhost:8080/api/blockchain/info | jq .height
curl -s http://localhost:8081/api/blockchain/info | jq .height
curl -s http://localhost:8082/api/blockchain/info | jq .height
```

## 🐛 Troubleshooting

### Erro: "Port already in use"

```bash
# Verificar quem está usando a porta
lsof -i :8080

# Matar o processo
kill <PID>
```

### Erro: "Signaling server not responding"

```bash
# Verificar se está rodando
ps aux | grep signaling

# Reiniciar manualmente
./bin/signaling -addr :9000
```

### Nós não se conectam

1. Verifique se o signaling server está rodando
2. Verifique os logs: `tail -f logs/node*.log`
3. Aguarde 30 segundos (intervalo de descoberta)
4. Verifique peers: `curl http://localhost:8080/api/peers`

### Chain não sincroniza

1. Verifique se todos os nós têm o mesmo genesis block
2. Limpe os dados e reinicie: `./start-3nodes-api.sh --clean`
3. Verifique os logs para erros de validação

### Interface web não carrega

1. Verifique se a API está rodando: `curl http://localhost:8080/api/status`
2. Limpe o cache do navegador (Ctrl+Shift+Delete)
3. Tente em modo anônimo

## 📊 Logs e Dados

### Estrutura de diretórios

```
krakovia/
├── logs/
│   ├── signaling.log    # Logs do servidor de signaling
│   ├── node1.log        # Logs do Node 1
│   ├── node2.log        # Logs do Node 2
│   └── node3.log        # Logs do Node 3
│
└── data/
    ├── node1/           # LevelDB do Node 1
    ├── node2/           # LevelDB do Node 2
    └── node3/           # LevelDB do Node 3
```

### Limpar dados

```bash
# Remover todos os dados
rm -rf ./data/node1 ./data/node2 ./data/node3

# Remover logs
rm -f logs/*.log

# Ou usar o script
./start-3nodes-api.sh --clean
```

## 🎯 Próximos Passos

- Experimente fazer stake em todos os nós
- Teste transferências entre os nós
- Observe a seleção de validadores baseada em stake
- Monitore a sincronização de checkpoints
- Desenvolva sua própria aplicação usando a API

## 📚 Recursos Adicionais

- [API Documentation](docs/API.md) - Referência completa da API
- [API Quick Start](API_QUICKSTART.md) - Guia rápido da API
- [README Principal](README.md) - Documentação completa do projeto
- [Blockchain System](docs/BLOCKCHAIN_SYSTEM.md) - Arquitetura da blockchain

## 💡 Dicas

1. **Abra as 3 interfaces web lado a lado** para ver a rede funcionando em tempo real
2. **Use o script de demo** (`./api-demo.sh`) para testar rapidamente
3. **Monitore os logs** em terminais separados para entender o que está acontecendo
4. **Faça stake em todos os nós** para ver o consenso PoS em ação
5. **Teste partições de rede** parando e reiniciando nós

---

Divirta-se explorando a rede Krakovia! 🚀
