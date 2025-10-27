# Testes de Rede da Krakovia Blockchain

Esta pasta contém testes de integração para a camada de rede da blockchain.

## Estrutura de Testes

### network_test.go

Testes de conexão WebRTC entre nós:

1. **TestNodeConnection**
   - Testa conexão básica entre 2 nós
   - Verifica se os nós se descobrem via signaling server
   - Confirma estabelecimento de conexão P2P

2. **TestMultipleNodesConnection**
   - Testa conexão entre 4 nós simultaneamente
   - Verifica mesh network completo (todos conectados a todos)
   - Valida descoberta automática de peers

3. **TestMessageBroadcast**
   - Testa broadcast de mensagens entre nós
   - Verifica se mensagens chegam a todos os peers
   - Valida serialização/deserialização de mensagens

4. **TestNodeReconnection**
   - Testa reconexão após desconexão
   - Simula falha de nó e recuperação
   - Verifica estabilidade da rede

## Como Executar os Testes

### Executar todos os testes
```bash
go test ./tests -v
```

### Executar um teste específico
```bash
go test ./tests -v -run TestNodeConnection
```

### Executar testes com timeout maior
```bash
go test ./tests -v -timeout 30s
```

### Limpar dados de teste
```bash
rm -rf tests/test-data
```

## Portas Utilizadas nos Testes

Os testes usam portas diferentes para evitar conflitos:

- **TestNodeConnection**: 9100-9102
- **TestMultipleNodesConnection**: 9200-9205
- **TestMessageBroadcast**: 9300-9303
- **TestNodeReconnection**: 9400-9402

Certifique-se de que essas portas estão disponíveis antes de executar os testes.

## Dados de Teste

Os testes criam dados temporários em `tests/test-data/`. Estes arquivos são ignorados pelo git (veja .gitignore) e podem ser removidos manualmente se necessário.

## Observações

- Os testes aguardam alguns segundos para estabelecer conexões WebRTC (ICE negotiation)
- Conexões podem levar de 2-5 segundos dependendo da rede
- Se os testes falharem por timeout, aumente o tempo de espera
- Cada teste inicia seu próprio servidor de signaling em porta específica

## Próximos Testes

- [ ] Testes de latência de rede
- [ ] Testes de throughput
- [ ] Testes de falha de signaling server
- [ ] Testes de stress com muitos nós
- [ ] Testes de segurança e validação de mensagens
