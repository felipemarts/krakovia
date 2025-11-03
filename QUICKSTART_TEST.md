# Quickstart - Teste de Integração

## 🚀 Executar Demo Interativa (RECOMENDADO)

A maneira mais fácil de ver a integração funcionando:

```bash
./run-demo.sh
```

Isso irá:
1. ✅ Compilar o projeto
2. ✅ Iniciar servidor de signaling automaticamente
3. ✅ Rodar demo com 2 nodes
4. ✅ Mostrar sincronização, mineração e propagação em tempo real
5. ✅ Exibir estatísticas e verificação

**Duração:** ~30 segundos

**O que você verá:**
- Criação de wallets e genesis block
- Node 1 minerando blocos
- Node 1 criando transações
- Node 2 sincronizando automaticamente
- Propagação de transações entre nodes
- Estatísticas finais mostrando convergência

---

## 🧪 Executar Testes Automatizados

Para rodar os testes de integração completos:

```bash
cd tests
./run_integration.sh
```

Isso irá:
1. ✅ Compilar o projeto
2. ✅ Iniciar servidor de signaling
3. ✅ Rodar testes Go com verificações
4. ✅ Limpar recursos automaticamente

**Duração:** ~25 segundos

---

## 📋 O Que é Testado

### ✅ Sincronização
- Node 2 inicia depois do Node 1
- Node 2 sincroniza blockchain automaticamente
- Ambos nodes chegam à mesma altura

### ✅ Propagação de Transações
- Transações criadas em um node aparecem em outros
- Mempool mantém consistência

### ✅ Mineração PoS
- Node com stake minera blocos
- Recompensas são distribuídas
- Blocos propagam para a rede

### ✅ Convergência
- Múltiplos nodes mantêm mesmo estado
- Não há forks persistentes

---

## 🎯 Exemplo de Saída

```
==============================================
  Krakovia Blockchain - Integration Demo
==============================================

[Setup] Creating wallets...
  Wallet 1: a3f5c8b2d9e1f4a6c7b8...
  Wallet 2: b4g6d9c3e0f2a5b7c8d9...

[Node 1] ✓ Mining started
[Node 1] ✓ Mined 3 blocks

[Node 2] ✓ Synchronized
  Height: 4
  Balance: 50000 (received from transaction)

==============================================
              Verification
==============================================
✓ Chains synchronized (height 6)
✓ Transaction propagated
✓ Staking working
✓ PoS mining working
```

---

## 📚 Documentação Completa

Para mais detalhes, veja:
- [INTEGRATION_TEST.md](INTEGRATION_TEST.md) - Documentação completa dos testes
- [INTEGRATION.md](INTEGRATION.md) - Arquitetura da integração
