# Guia de Mineração

## 🔨 Como Iniciar a Mineração

Existem **3 formas** de iniciar a mineração no Krakovia:

### 1. Via Script com Flag `--mine` (Mais Fácil)

Inicia o nó já minerando automaticamente:

```bash
./start-node1.sh --mine
./start-node2.sh --mine
./start-node3.sh --mine
```

Pode combinar com `--clean`:
```bash
./start-node1.sh --clean --mine
```

### 2. Via Interface Web

1. Acesse http://localhost:8080
2. Faça login (admin / admin)
3. Clique em **"Iniciar Mineração"**

### 3. Via API (cURL)

```bash
curl -u admin:admin -X POST http://localhost:8080/api/mining/start
```

## ⏱️ Tempo Entre Blocos

O tempo entre blocos está configurado no JSON do nó:

```json
{
  "genesis": {
    "block_time": 5000  // 5 segundos (em milissegundos)
  }
}
```

**Valores comuns:**
- `2000` = 2 segundos (rápido)
- `5000` = 5 segundos (padrão)
- `10000` = 10 segundos (lento)
- `15000` = 15 segundos (Bitcoin-like)

## 🎯 Cenários de Uso

### Teste Rápido: 1 Nó Minerando

```bash
# Inicia node1 já minerando
./start-node1.sh --mine

# Acesse: http://localhost:8080
# Veja blocos sendo gerados a cada 5 segundos
```

### Rede Completa: 3 Nós Minerando

```bash
# Terminal 1
./start-node1.sh --mine

# Terminal 2
./start-node2.sh --mine

# Terminal 3
./start-node3.sh --mine
```

### Teste de Sincronização

```bash
# Terminal 1 - Inicia minerando
./start-node1.sh --mine

# Terminal 2 - Inicia SEM minerar (aguarda 30s)
./start-node2.sh

# Node 2 vai sincronizar os blocos do Node 1
# Depois inicie mineração no Node 2 via interface web
```

## 🏆 Consenso PoS (Proof of Stake)

### Como Funciona

1. **Sem Stake:** Nó pode minerar, mas com prioridade baixa
2. **Com Stake:** Quanto mais stake, maior chance de validar blocos
3. **Stake Mínimo:** 1000 tokens (configurável no JSON)

### Fazer Stake

**Via Interface Web:**
1. Acesse o nó
2. Na seção "Stake"
3. Digite a quantidade (ex: 10000)
4. Clique em "Fazer Stake"

**Via API:**
```bash
curl -u admin:admin \
  -X POST \
  -H "Content-Type: application/json" \
  -d '{"amount":10000,"fee":10}' \
  http://localhost:8080/api/wallet/stake
```

### Verificar Stake

**Via Interface Web:**
- Veja na seção "Carteira" → "Stake"

**Via API:**
```bash
curl -u admin:admin http://localhost:8080/api/wallet/balance | jq .stake
```

## 📊 Monitorar Mineração

### Ver Logs em Tempo Real

```bash
tail -f logs/node1.log | grep -i "block\|mining"
```

### Ver Altura da Chain

```bash
# Via API
curl -s http://localhost:8080/api/status | jq .chain_height

# Via Interface Web
# Olhe "Status do Nó" → "Altura"
```

### Ver Último Bloco

```bash
curl -s http://localhost:8080/api/blockchain/last-block | jq
```

## ⚙️ Configurações de Mineração

Edite `configs/nodeX-api.json`:

```json
{
  "genesis": {
    "block_time": 5000,              // Tempo entre blocos (ms)
    "max_block_size": 1000,          // Transações por bloco
    "block_reward": 50,              // Recompensa do minerador
    "min_validator_stake": 1000      // Stake mínimo
  }
}
```

### Exemplos de Configuração

#### Mineração Rápida (2 segundos)
```json
{
  "genesis": {
    "block_time": 2000,
    "block_reward": 25
  }
}
```

#### Mineração Lenta (15 segundos, como Bitcoin)
```json
{
  "genesis": {
    "block_time": 15000,
    "block_reward": 100
  }
}
```

#### Blocos Grandes
```json
{
  "genesis": {
    "max_block_size": 5000,
    "block_time": 10000
  }
}
```

## 🛑 Parar Mineração

### Via Script
Pressione **Ctrl+C** no terminal

### Via Interface Web
Clique em **"Parar Mineração"**

### Via API
```bash
curl -u admin:admin -X POST http://localhost:8080/api/mining/stop
```

## 💡 Dicas

1. **Inicie com `--mine`** para já começar minerando
2. **Ajuste `block_time`** para sua necessidade:
   - Testes rápidos: 2000ms
   - Produção: 5000-10000ms
3. **Use stake** para influenciar quem valida blocos
4. **Monitore os logs** para ver o que está acontecendo
5. **Node 1 tem 1 bilhão** de tokens inicialmente

## 🔍 Troubleshooting

### Nó não está minerando

**Verifique se iniciou com `-mine`:**
```bash
./start-node1.sh --mine
```

**Ou inicie via API:**
```bash
curl -u admin:admin -X POST http://localhost:8080/api/mining/start
```

**Verifique status:**
```bash
curl -s http://localhost:8080/api/status | jq .mining
```

### Blocos não estão sendo gerados

1. **Verifique se está minerando:**
   ```bash
   curl -s http://localhost:8080/api/status | jq .mining
   ```

2. **Veja os logs:**
   ```bash
   tail -f logs/node1.log
   ```

3. **Verifique mempool:**
   ```bash
   curl -s http://localhost:8080/api/mempool | jq
   ```

### Blocos muito lentos

Edite `configs/nodeX-api.json`:
```json
{
  "genesis": {
    "block_time": 2000  // Reduzir para 2 segundos
  }
}
```

Depois reinicie:
```bash
pkill -f "./bin/node"
./start-node1.sh --clean --mine
```

## 📚 Mais Informações

- [INDIVIDUAL_NODES.md](INDIVIDUAL_NODES.md) - Scripts individuais
- [3NODES_GUIDE.md](3NODES_GUIDE.md) - Guia completo
- [API_QUICKSTART.md](API_QUICKSTART.md) - Guia da API

---

**Resumo Rápido:**
```bash
# Iniciar minerando
./start-node1.sh --mine

# Ver blocos sendo gerados
curl -s http://localhost:8080/api/status | jq .chain_height

# Acessar interface
open http://localhost:8080
```

🎉 Divirta-se minerando!
