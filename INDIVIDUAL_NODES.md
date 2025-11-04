# Iniciando Nós Individualmente

Você pode iniciar cada nó separadamente usando os scripts individuais.

## 🚀 Scripts Disponíveis

| Script | Nó | API | P2P | Wallet |
|--------|-----|-----|-----|--------|
| `./start-node1.sh` | node1 | :8080 | :9001 | 4b2aaf06... (1B tokens) |
| `./start-node2.sh` | node2 | :8081 | :9002 | fa878b92... |
| `./start-node3.sh` | node3 | :8082 | :9003 | a4f9877e... |

## 📋 Uso

### Iniciar Node 1
```bash
./start-node1.sh
```

### Iniciar Node 2
```bash
./start-node2.sh
```

### Iniciar Node 3
```bash
./start-node3.sh
```

### Iniciar com Dados Limpos
```bash
./start-node1.sh --clean
./start-node2.sh --clean
./start-node3.sh --clean
```

## 🔧 O que cada script faz:

1. ✅ Verifica se o signaling server está rodando (inicia se necessário)
2. ✅ Verifica se a porta da API está disponível
3. ✅ Cria diretório de logs se não existir
4. ✅ Limpa dados antigos (se `--clean`)
5. ✅ Inicia o nó
6. ✅ Mostra URL de acesso e credenciais

## 🌐 Acessar os Nós

Após iniciar, acesse no navegador:

- **Node 1:** http://localhost:8080
- **Node 2:** http://localhost:8081
- **Node 3:** http://localhost:8082

**Credenciais (todos):**
- Usuário: `admin`
- Senha: `admin`

## 📊 Exemplo: Iniciando em Terminais Separados

```bash
# Terminal 1 - Node 1
./start-node1.sh

# Terminal 2 - Node 2
./start-node2.sh

# Terminal 3 - Node 3
./start-node3.sh
```

## ⚙️ Configurações da Blockchain (JSON)

Cada nó está configurado com:
- **Block Time:** 5000ms (5 segundos)
- **Max Block Size:** 1000 transações
- **Block Reward:** 50 tokens
- **Min Validator Stake:** 1000 tokens

Para alterar, edite o arquivo `configs/nodeX-api.json`:

```json
{
  "genesis": {
    "block_time": 5000,
    "max_block_size": 1000,
    "block_reward": 50,
    "min_validator_stake": 1000
  }
}
```

**Nota:** `block_time` está em milissegundos.

## 🛑 Parar um Nó

Pressione **Ctrl+C** no terminal onde o nó está rodando.

Ou mate o processo:
```bash
# Encontrar o PID
ps aux | grep "node -config configs/node1-api.json"

# Matar o processo
kill <PID>
```

## 📝 Logs

Os logs são salvos em:
- `logs/node1.log`
- `logs/node2.log`
- `logs/node3.log`
- `logs/signaling.log`

Ver logs em tempo real:
```bash
tail -f logs/node1.log
tail -f logs/node2.log
tail -f logs/node3.log
```

## 🔍 Verificar Status

```bash
# Ver processos
ps aux | grep "./bin/node"

# Ver portas em uso
lsof -i :8080  # Node 1 API
lsof -i :8081  # Node 2 API
lsof -i :8082  # Node 3 API
lsof -i :9000  # Signaling
```

## 💡 Dicas

1. **Inicie o Node 1 primeiro** - ele tem os tokens iniciais (1 bilhão)
2. **Aguarde 30 segundos** para os nós se descobrirem
3. **Use a interface web** para facilitar operações
4. **Monitore os logs** para ver o que está acontecendo

## 🎯 Cenários Comuns

### Testar Sincronização
```bash
# Terminal 1
./start-node1.sh

# Terminal 2 (após 30 segundos)
./start-node2.sh

# No Node 1, inicie mineração
# Veja o Node 2 sincronizar automaticamente
```

### Testar Transferência
```bash
# Inicie Node 1 e Node 2
./start-node1.sh  # Terminal 1
./start-node2.sh  # Terminal 2

# Acesse http://localhost:8080
# Faça uma transferência para o endereço do Node 2:
# fa878b92dedd74e3867adbf27154fabb66fd94da899bc8af96d987771dd01098
```

### Testar Stake e Consenso
```bash
# Inicie todos os nós
./start-node1.sh  # Terminal 1
./start-node2.sh  # Terminal 2
./start-node3.sh  # Terminal 3

# Em cada nó, faça stake via interface web
# Inicie mineração em todos
# Observe qual nó está validando blocos (baseado em stake)
```

## 📚 Mais Informações

- [3NODES_GUIDE.md](3NODES_GUIDE.md) - Guia completo dos 3 nós
- [QUICK_START.md](QUICK_START.md) - Início rápido
- [API_QUICKSTART.md](API_QUICKSTART.md) - Guia da API

---

**Pronto!** Agora você pode iniciar cada nó individualmente! 🎉
