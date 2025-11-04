# Krakovia API - Início Rápido

Este guia mostra como configurar e usar a API HTTP do nó Krakovia em poucos minutos.

## 1. Configurar o Nó

Crie ou edite seu arquivo de configuração JSON adicionando a seção `api`:

```json
{
  "id": "node1",
  "address": ":9001",
  "db_path": "./data/node1",
  "signaling_server": "ws://localhost:9000/ws",
  "wallet": {
    "private_key": "sua_chave_privada",
    "public_key": "sua_chave_publica",
    "address": "seu_endereco"
  },
  "api": {
    "enabled": true,
    "address": ":8080",
    "username": "admin",
    "password": "krakovia123"
  }
}
```

**OU** use o arquivo de exemplo já configurado:
```bash
cp configs/node-with-api.example.json configs/node1.json
```

## 2. Iniciar o Servidor de Signaling

Em um terminal:
```bash
./bin/signaling -addr :9000
```

## 3. Iniciar o Nó com API

Em outro terminal:
```bash
./bin/node -config configs/node1.json
```

Você verá:
```
HTTP API: http://localhost:8080
```

## 4. Acessar a Interface Web

Abra seu navegador e acesse:
```
http://localhost:8080
```

Você verá o dashboard com:
- ✅ Status do nó em tempo real
- ✅ Informações da carteira (saldo, stake)
- ✅ Último bloco minerado
- ✅ Formulários para transferências e stake
- ✅ Controles de mineração

## 5. Realizar Operações

### Via Interface Web

1. Acesse `http://localhost:8080`
2. Clique em "Iniciar Mineração" (pode requerer login)
3. Use os formulários para:
   - Fazer transferências
   - Fazer stake
   - Fazer unstake

**Credenciais:** Use o username e password configurados no JSON

### Via API (cURL)

#### Consultar Status (sem autenticação)
```bash
curl http://localhost:8080/api/status
```

#### Iniciar Mineração (com autenticação)
```bash
curl -u admin:krakovia123 -X POST http://localhost:8080/api/mining/start
```

#### Consultar Saldo (com autenticação)
```bash
curl -u admin:krakovia123 http://localhost:8080/api/wallet/balance
```

#### Fazer Transferência (com autenticação)
```bash
curl -u admin:krakovia123 \
  -X POST \
  -H "Content-Type: application/json" \
  -d '{
    "to": "endereco_destino_hex",
    "amount": 1000,
    "fee": 10,
    "data": "Pagamento"
  }' \
  http://localhost:8080/api/wallet/transfer
```

## 6. Múltiplos Nós com API

Para executar múltiplos nós com APIs em portas diferentes:

**Node 1 - configs/node1.json:**
```json
{
  "id": "node1",
  "address": ":9001",
  "api": {
    "enabled": true,
    "address": ":8080",
    "username": "admin",
    "password": "pass1"
  }
}
```

**Node 2 - configs/node2.json:**
```json
{
  "id": "node2",
  "address": ":9002",
  "api": {
    "enabled": true,
    "address": ":8081",
    "username": "admin",
    "password": "pass2"
  }
}
```

Execute:
```bash
# Terminal 1
./bin/node -config configs/node1.json

# Terminal 2
./bin/node -config configs/node2.json
```

Acesse:
- Node 1: http://localhost:8080
- Node 2: http://localhost:8081

## Endpoints Principais

| Endpoint | Método | Auth | Descrição |
|----------|--------|------|-----------|
| `/` | GET | Não | Interface web |
| `/api/status` | GET | Não | Status do nó |
| `/api/blockchain/info` | GET | Não | Info da blockchain |
| `/api/blockchain/last-block` | GET | Não | Último bloco |
| `/api/wallet/balance` | GET | Sim | Saldo da carteira |
| `/api/wallet/transfer` | POST | Sim | Fazer transferência |
| `/api/wallet/stake` | POST | Sim | Fazer stake |
| `/api/wallet/unstake` | POST | Sim | Fazer unstake |
| `/api/mining/start` | POST | Sim | Iniciar mineração |
| `/api/mining/stop` | POST | Sim | Parar mineração |
| `/api/peers` | GET | Não | Lista de peers |

## Segurança

⚠️ **Importante para Produção:**

1. **Use HTTPS**: Configure um reverse proxy (nginx, caddy)
2. **Senhas Fortes**: Troque as senhas padrão
3. **Firewall**: Limite acesso apenas a IPs confiáveis
4. **Não Exponha**: Não deixe a API aberta na internet sem proteção

Exemplo de configuração nginx:
```nginx
server {
    listen 443 ssl;
    server_name node.seudominio.com;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

## Troubleshooting

### Porta já em uso
Se a porta 8080 já estiver em uso, troque no JSON:
```json
"api": {
  "address": ":8090"
}
```

### Autenticação falha
- Verifique username/password no JSON
- No navegador, use Ctrl+Shift+Delete para limpar cache
- No cURL, use `-u username:password`

### Interface não atualiza
- Recarregue a página (F5)
- Verifique o console do navegador (F12)
- Confirme que o nó está rodando

## Próximos Passos

- 📖 Veja a [documentação completa da API](docs/API.md)
- 🔧 Configure múltiplos nós para testar rede
- 💻 Explore os endpoints via cURL ou Postman
- 🌐 Desenvolva sua própria interface usando a API

## Recursos Adicionais

- [README Principal](README.md) - Documentação completa do projeto
- [API Documentation](docs/API.md) - Referência completa da API
- [Blockchain System](docs/BLOCKCHAIN_SYSTEM.md) - Arquitetura da blockchain

## Suporte

Encontrou um problema? Abra uma issue no GitHub!
