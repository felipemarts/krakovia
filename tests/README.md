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

### discovery_test.go

Testes do sistema de descoberta de peers:

1. **TestPeerLimitEnforcement**
   - Verifica se o limite máximo de peers (MaxPeers) é respeitado
   - Cria 6 nós com limite de 3 peers cada
   - Valida que nenhum nó excede o limite configurado

2. **TestPeerDiscovery**
   - Testa descoberta periódica de novos peers
   - Adiciona novo nó após rede estabelecida
   - Verifica se nós existentes descobrem o novo peer automaticamente

3. **TestMinimumPeersMaintenance**
   - Verifica se nós mantêm o mínimo de peers configurado (MinPeers)
   - Testa se a descoberta periódica preenche conexões insuficientes
   - Valida que todos os nós atingem o mínimo de conexões

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

**Os testes agora usam alocação dinâmica de portas!**

Cada execução de teste aloca portas aleatórias no intervalo **9000-29000**, evitando completamente conflitos de porta entre:
- Múltiplas execuções de testes simultâneas
- Serviços já rodando no sistema
- Execuções consecutivas sem cleanup completo

Isso significa que você pode executar os testes quantas vezes quiser sem se preocupar com portas ocupadas.

## Dados de Teste

**Os testes agora usam diretórios temporários únicos!**

Cada execução cria um diretório único em `/tmp/krakovia-test-{nome}-{timestamp}/` que é automaticamente limpo ao final do teste usando `t.Cleanup()`.

Não há mais necessidade de limpar manualmente dados de teste entre execuções.

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
