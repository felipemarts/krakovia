# 🚀 Krakovia - Quick Start (3 Nós com API)

## ⚡ Início Ultra-Rápido (1 comando)

```bash
./manage-nodes.sh
```

Selecione a opção **1** (Start 3 nodes with API)

## 🌐 Acesse os Nós

Abra no navegador:
- **Node 1:** http://localhost:8080
- **Node 2:** http://localhost:8081
- **Node 3:** http://localhost:8082

**Login:** `admin` / `krakovia123`

## 📋 Scripts Disponíveis

| Script | Descrição |
|--------|-----------|
| `./manage-nodes.sh` | 📊 Menu interativo completo |
| `./start-3nodes-api.sh` | ▶️ Iniciar 3 nós |
| `./api-demo.sh` | 🧪 Demo interativo da API |
| `./start-with-api.sh` | ▶️ Iniciar 1 nó |

## 🎯 Operações Comuns

### Via Menu (Recomendado)
```bash
./manage-nodes.sh
```

### Via Script Direto
```bash
# Iniciar nós
./start-3nodes-api.sh

# Limpar dados e iniciar
./start-3nodes-api.sh --clean

# Parar tudo
pkill -f "./bin/node"
```

### Via API (cURL)
```bash
# Ver status de todos
for port in 8080 8081 8082; do
  curl -s http://localhost:$port/api/status | jq
done

# Iniciar mineração (Node 1)
curl -u admin:krakovia123 -X POST http://localhost:8080/api/mining/start

# Ver saldo (Node 1)
curl -u admin:krakovia123 http://localhost:8080/api/wallet/balance | jq
```

## 📊 Monitoramento

```bash
# Logs em tempo real
tail -f logs/node1.log
tail -f logs/node2.log
tail -f logs/node3.log

# Status das portas
lsof -i :8080 :8081 :8082
```

## 🎮 Teste Rápido

1. **Inicie:** `./start-3nodes-api.sh`
2. **Acesse:** http://localhost:8080
3. **Mine:** Clique em "Iniciar Mineração"
4. **Observe:** Os 3 nós sincronizando
5. **Transfira:** Use o formulário para transferir tokens

## 🛑 Parar

```bash
# Via menu
./manage-nodes.sh
# → Opção 3 (Stop all nodes)

# Ou direto
pkill -f "./bin/node"
```

## 📚 Documentação

- **[3NODES_GUIDE.md](3NODES_GUIDE.md)** - Guia completo de 3 nós
- **[API_QUICKSTART.md](API_QUICKSTART.md)** - Guia da API
- **[docs/API.md](docs/API.md)** - Referência completa da API

## 🆘 Problemas?

```bash
# Ver o que está rodando
ps aux | grep node

# Limpar tudo
./start-3nodes-api.sh --clean

# Verificar portas
./manage-nodes.sh → Opção 6
```

---

**Pronto!** 🎉 Sua rede blockchain com 3 nós e APIs está funcionando!
