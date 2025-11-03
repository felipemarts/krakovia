# Genesis Generator

Ferramenta de linha de comando para gerar configuração de bloco genesis com parâmetros personalizados.

## Instalação

```bash
go build -o bin/genesis-gen ./cmd/genesis-gen/
```

## Uso

```bash
./bin/genesis-gen [flags]
```

### Flags Disponíveis

- `-recipient <address>` (obrigatório): Endereço que receberá a alocação inicial de tokens
- `-amount <uint64>`: Quantidade inicial de tokens (padrão: 1000000000)
- `-block-time <int64>`: Tempo entre blocos em milissegundos (padrão: 5000ms, mínimo: 1000ms)
- `-max-block-size <int>`: Máximo de transações por bloco (padrão: 1000)
- `-block-reward <uint64>`: Recompensa por bloco minerado (padrão: 50)
- `-min-stake <uint64>`: Stake mínimo para ser validador (padrão: 1000)
- `-timestamp <int64>`: Timestamp Unix do bloco genesis (padrão: tempo atual)
- `-output <string>`: Caminho do arquivo de saída (padrão: stdout)

### Exemplos

#### Gerar genesis básico (saída no terminal)

```bash
./bin/genesis-gen -recipient a3f5c8b2d9e1f4a6c7b8d9e0f1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0
```

#### Gerar genesis com configurações customizadas

```bash
./bin/genesis-gen \
  -recipient a3f5c8b2d9e1f4a6c7b8d9e0f1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0 \
  -amount 1000000000 \
  -block-time 5000 \
  -min-stake 1000 \
  -block-reward 50 \
  -max-block-size 2000 \
  -output genesis.json
```

#### Gerar genesis para produção (tempo de bloco mais longo)

```bash
./bin/genesis-gen \
  -recipient <seu-endereço> \
  -amount 10000000000 \
  -block-time 15000 \
  -min-stake 10000 \
  -block-reward 100 \
  -output configs/genesis-production.json
```

## Saída

O comando gera um arquivo JSON com a seguinte estrutura:

```json
{
  "timestamp": 1762179261,
  "recipient_addr": "a3f5c8b2d9e1f4a6c7b8d9e0f1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0",
  "amount": 1000000000,
  "hash": "50b0d172d55b70dd73b8a5e37e9f129e0d9a23f8d378494e44e239b2086a2367",
  "block_time": 5000,
  "max_block_size": 1000,
  "block_reward": 50,
  "min_validator_stake": 1000
}
```

## Validação de Tempo de Bloco

A blockchain implementa uma validação de tempo mínimo entre blocos:

- **Tempo Mínimo**: 80% do `block_time` configurado
- **Exemplo**: Se `block_time` é 5000ms (5s), blocos consecutivos devem ter timestamps com diferença de pelo menos 4000ms (4s)
- **Validação**: A validação só ocorre quando `VerifyChain()` é chamada com a configuração da chain

Esta validação garante que:
1. Blocos não sejam criados muito rapidamente
2. A rede tenha tempo para propagar e validar blocos
3. Evita ataques de flooding de blocos

## Uso na Configuração do Node

Copie o JSON gerado para a seção `genesis` do seu arquivo de configuração do node:

```json
{
  "id": "node1",
  "address": ":9001",
  "db_path": "./data/node1",
  "signaling_server": "ws://localhost:9000/ws",
  "wallet": {
    "private_key": "...",
    "public_key": "...",
    "address": "..."
  },
  "genesis": {
    "timestamp": 1762179261,
    "recipient_addr": "a3f5c8b2d9e1f4a6c7b8d9e0f1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0",
    "amount": 1000000000,
    "hash": "50b0d172d55b70dd73b8a5e37e9f129e0d9a23f8d378494e44e239b2086a2367",
    "block_time": 5000,
    "max_block_size": 1000,
    "block_reward": 50,
    "min_validator_stake": 1000
  }
}
```

## Notas Importantes

1. **Endereço Único**: Use um endereço válido gerado com `wallet-gen`
2. **Consistency**: Todos os nós da rede devem usar exatamente o mesmo genesis
3. **Hash**: O hash do genesis é calculado automaticamente e deve ser o mesmo em todos os nós
4. **Timestamp**: Use o mesmo timestamp para garantir que todos os nós tenham o mesmo genesis
5. **Block Time**: Escolha um tempo de bloco apropriado para sua rede (recomendado: 5-15 segundos para produção)
