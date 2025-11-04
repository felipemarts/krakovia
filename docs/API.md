# API HTTP do Nó Krakovia

Este documento descreve a API HTTP REST implementada para interagir com os nós da blockchain Krakovia.

## Visão Geral

A API HTTP fornece uma interface REST para:
- Visualizar informações do nó e blockchain
- Consultar saldo e stake da carteira
- Realizar operações (transferências, stake, unstake)
- Controlar mineração
- Interface web amigável para interação

## Configuração

### Habilitando a API

Para habilitar a API HTTP, adicione a seguinte seção no arquivo de configuração JSON do nó:

```json
{
  "api": {
    "enabled": true,
    "address": ":8080",
    "username": "admin",
    "password": "sua_senha_segura"
  }
}
```

**Parâmetros:**
- `enabled`: Habilita/desabilita a API (padrão: false)
- `address`: Endereço e porta do servidor (padrão: ":8080")
- `username`: Usuário para autenticação (obrigatório)
- `password`: Senha para autenticação (obrigatório)

### Exemplo de Configuração Completa

Veja o arquivo [configs/node-with-api.example.json](../configs/node-with-api.example.json) para um exemplo completo.

## Iniciando o Nó com API

```bash
# Compilar
go build -o bin/node ./cmd/node

# Iniciar com configuração que inclui API
./bin/node -config configs/node-with-api.example.json
```

Você verá uma mensagem indicando que a API foi iniciada:

```
HTTP API: http://localhost:8080
```

## Interface Web

Acesse `http://localhost:8080` no navegador para usar a interface web.

A interface fornece:
- Dashboard com status do nó em tempo real
- Informações da carteira (saldo, stake, nonce)
- Último bloco minerado
- Formulários para transferências
- Formulários para stake/unstake
- Controles de mineração
- Lista de peers conectados

**Autenticação:** A primeira operação protegida (transferência, stake, etc.) solicitará usuário e senha configurados no JSON.

## Endpoints da API

### Endpoints Públicos (sem autenticação)

#### GET /api/status
Retorna status geral do nó.

**Resposta:**
```json
{
  "node_id": "node1",
  "chain_height": 150,
  "mempool_size": 5,
  "peers_count": 3,
  "mining": true,
  "blocks_memory": 200
}
```

#### GET /api/blockchain/info
Retorna informações da blockchain.

**Resposta:**
```json
{
  "height": 150,
  "last_hash": "a3f5c8b2d9e1f4a6...",
  "last_height": 150,
  "timestamp": 1735862400
}
```

#### GET /api/blockchain/last-block
Retorna informações do último bloco.

**Resposta:**
```json
{
  "hash": "a3f5c8b2d9e1f4a6...",
  "height": 150,
  "timestamp": 1735862400,
  "previous_hash": "b4e6d9a1c2f5e8...",
  "validator_addr": "a3f5c8b2d9...",
  "transactions_count": 10,
  "merkle_root": "c5f7e0b3d8..."
}
```

#### GET /api/mempool
Retorna informações do mempool.

**Resposta:**
```json
{
  "size": 5,
  "transactions": []
}
```

#### GET /api/peers
Retorna lista de peers conectados.

**Resposta:**
```json
{
  "count": 3,
  "peers": [
    {
      "id": "node2",
      "ready": true
    },
    {
      "id": "node3",
      "ready": true
    }
  ]
}
```

### Endpoints Protegidos (requerem autenticação)

Todos os endpoints abaixo requerem autenticação HTTP Basic com as credenciais configuradas.

#### GET /api/wallet/balance
Retorna saldo e stake da carteira do nó.

**Resposta:**
```json
{
  "balance": 1000000000,
  "stake": 100000,
  "nonce": 42
}
```

#### GET /api/wallet/address
Retorna informações da carteira.

**Resposta:**
```json
{
  "node_id": "node1"
}
```

#### POST /api/wallet/transfer
Cria uma transação de transferência.

**Request Body:**
```json
{
  "to": "a3f5c8b2d9e1f4a6c7b8d9e0f1a2b3c4d5e6f7a8",
  "amount": 1000,
  "fee": 10,
  "data": "Pagamento por serviços"
}
```

**Resposta:**
```json
{
  "transaction_id": "b4e6d9a1c2f5e8...",
  "status": "submitted"
}
```

#### POST /api/wallet/stake
Cria uma transação de stake.

**Request Body:**
```json
{
  "amount": 10000,
  "fee": 10
}
```

**Resposta:**
```json
{
  "transaction_id": "c5f7e0b3d8...",
  "status": "submitted"
}
```

#### POST /api/wallet/unstake
Cria uma transação de unstake.

**Request Body:**
```json
{
  "amount": 5000,
  "fee": 10
}
```

**Resposta:**
```json
{
  "transaction_id": "d6g8f1c4e9...",
  "status": "submitted"
}
```

#### POST /api/mining/start
Inicia a mineração no nó.

**Resposta:**
```json
{
  "status": "mining_started"
}
```

#### POST /api/mining/stop
Para a mineração no nó.

**Resposta:**
```json
{
  "status": "mining_stopped"
}
```

#### GET /api/mining/status
Retorna status da mineração.

**Resposta:**
```json
{
  "mining": true
}
```

## Exemplos com cURL

### Consultar Status (sem autenticação)
```bash
curl http://localhost:8080/api/status
```

### Consultar Saldo (com autenticação)
```bash
curl -u admin:krakovia123 http://localhost:8080/api/wallet/balance
```

### Fazer Transferência (com autenticação)
```bash
curl -u admin:krakovia123 \
  -X POST \
  -H "Content-Type: application/json" \
  -d '{"to":"a3f5c8b2d9e1f4a6c7b8d9e0f1a2b3c4d5e6f7a8","amount":1000,"fee":10,"data":"Pagamento"}' \
  http://localhost:8080/api/wallet/transfer
```

### Fazer Stake (com autenticação)
```bash
curl -u admin:krakovia123 \
  -X POST \
  -H "Content-Type: application/json" \
  -d '{"amount":10000,"fee":10}' \
  http://localhost:8080/api/wallet/stake
```

### Iniciar Mineração (com autenticação)
```bash
curl -u admin:krakovia123 \
  -X POST \
  http://localhost:8080/api/mining/start
```

## Segurança

### Autenticação

A API usa **HTTP Basic Authentication** para proteger endpoints sensíveis:
- Operações de leitura (status, blockchain info) são públicas
- Operações de carteira e mineração requerem autenticação
- Credenciais são configuradas no arquivo JSON do nó

**Importante:**
- Use sempre senhas fortes
- Em produção, use HTTPS para proteger as credenciais
- Não exponha a API diretamente na internet sem proteção adicional

### Recomendações de Segurança

1. **HTTPS em Produção**: Configure um reverse proxy (nginx, caddy) com HTTPS
2. **Firewall**: Limite acesso à porta da API apenas a IPs confiáveis
3. **Senhas Fortes**: Use senhas longas e complexas
4. **Rate Limiting**: Configure rate limiting no reverse proxy
5. **Monitoramento**: Monitore logs para detectar tentativas de acesso não autorizado

## Códigos de Status HTTP

- `200 OK`: Requisição bem-sucedida
- `400 Bad Request`: Parâmetros inválidos
- `401 Unauthorized`: Autenticação necessária ou credenciais inválidas
- `405 Method Not Allowed`: Método HTTP não permitido
- `500 Internal Server Error`: Erro interno do servidor

## Erros

Erros são retornados no formato JSON:

```json
{
  "error": "Mensagem descritiva do erro"
}
```

## Interface Web - Recursos

### Dashboard
- Atualização automática a cada 5 segundos
- Status do nó em tempo real
- Informações da blockchain
- Dados da carteira

### Operações
- **Transferências**: Enviar tokens para outro endereço
- **Stake**: Participar do consenso PoS
- **Unstake**: Retirar tokens do stake
- **Mineração**: Iniciar/parar mineração de blocos

### Visualização
- Lista de peers conectados
- Último bloco minerado
- Tamanho do mempool
- Altura da chain

## Desenvolvimento

### Estrutura do Código

```
pkg/api/
├── server.go  # Servidor HTTP e endpoints
└── ui.go      # Interface HTML/JavaScript
```

### Adicionando Novos Endpoints

1. Adicione o handler em `server.go`
2. Registre a rota em `registerRoutes()`
3. Use `requireAuth()` se precisar de autenticação
4. Atualize a interface em `ui.go` se necessário

### Interface com o Nó

A API usa a interface `NodeInterface` para se comunicar com o nó, evitando dependência circular:

```go
type NodeInterface interface {
    GetChainHeight() uint64
    GetBalance() uint64
    CreateTransaction(...) (*blockchain.Transaction, error)
    // ... outros métodos
}
```

## Troubleshooting

### API não inicia

- Verifique se `enabled: true` está no JSON
- Verifique se a porta está disponível
- Veja os logs para erros específicos

### Autenticação falha

- Confirme username/password no JSON
- Use codificação Base64 correta (cURL faz automaticamente com `-u`)
- Limpe cache do navegador

### Interface web não carrega

- Acesse a URL raiz `http://localhost:8080/`
- Verifique se o servidor iniciou corretamente
- Veja console do navegador para erros JavaScript

### Transações não são criadas

- Verifique se você tem saldo suficiente
- Confirme que a taxa está especificada
- Veja os logs do nó para detalhes do erro

## Próximos Passos

Funcionalidades planejadas:
- [ ] WebSocket para notificações em tempo real
- [ ] Histórico de transações
- [ ] Gráficos e estatísticas
- [ ] Explorador de blocos integrado
- [ ] Suporte a múltiplas carteiras
- [ ] API de consulta de transações específicas

## Contribuindo

Contribuições são bem-vindas! Veja o arquivo principal [README.md](../README.md) para guidelines.
