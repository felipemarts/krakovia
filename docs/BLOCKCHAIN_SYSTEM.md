# Sistema Blockchain - Krakovia

## Visão Geral

O sistema blockchain da Krakovia implementa uma estrutura tradicional de blocos e transações com criptografia ECDSA (Elliptic Curve Digital Signature Algorithm) para segurança e autenticação.

## Componentes Principais

### 1. Wallet (Carteira)

Localização: `pkg/wallet/wallet.go`

A carteira gerencia as chaves criptográficas de um usuário e permite assinar transações.

#### Estrutura

```go
type Wallet struct {
    PrivateKey *ecdsa.PrivateKey  // Chave privada ECDSA
    PublicKey  *ecdsa.PublicKey   // Chave pública ECDSA
}
```

#### Funcionalidades

- **Geração de Carteira**: Cria um novo par de chaves ECDSA usando a curva P-256
- **Derivação de Endereço**: Gera endereço a partir do hash SHA-256 da chave pública
- **Assinatura de Dados**: Assina dados usando a chave privada
- **Verificação de Assinatura**: Verifica assinaturas usando a chave pública

#### Formato das Chaves

- **Chave Privada**: 32 bytes (64 caracteres hexadecimais)
- **Chave Pública**: 64 bytes (128 caracteres hexadecimais) - X (32 bytes) + Y (32 bytes)
- **Endereço**: 32 bytes (64 caracteres hexadecimais) - SHA-256 da chave pública
- **Assinatura**: 64 bytes (128 caracteres hexadecimais) - r (32 bytes) + s (32 bytes)

#### Exemplo de Uso

```go
// Criar nova carteira
wallet, err := wallet.NewWallet()

// Obter informações
privateKeyHex := wallet.GetPrivateKeyHex()
publicKeyHex := wallet.GetPublicKeyHex()
address := wallet.GetAddress()

// Assinar dados
signature, err := wallet.Sign([]byte("mensagem"))

// Verificar assinatura
valid, err := wallet.Verify(publicKeyHex, []byte("mensagem"), signature)
```

### 2. Transaction (Transação)

Localização: `pkg/blockchain/transaction.go`

Transações representam transferências de valor entre endereços na blockchain.

#### Estrutura

```go
type Transaction struct {
    ID        string  // Hash da transação (SHA-256)
    From      string  // Endereço do remetente
    To        string  // Endereço do destinatário
    Amount    uint64  // Quantidade transferida
    Fee       uint64  // Taxa da transação
    Timestamp int64   // Timestamp Unix
    Signature string  // Assinatura ECDSA
    PublicKey string  // Chave pública do remetente
    Nonce     uint64  // Nonce para prevenir replay attacks
    Data      string  // Dados adicionais (opcional)
}
```

#### Tipos de Transações

1. **Transação Regular**: Transferência de tokens entre dois endereços
2. **Transação Coinbase**: Recompensa de bloco para o validador (sem remetente)

#### Validações

A transação passa por múltiplas validações:

1. **Verificação de Assinatura**: Valida que a assinatura corresponde ao remetente
2. **Verificação de Hash**: Confirma que o ID corresponde ao hash calculado
3. **Verificação de Endereço**: Valida que o endereço From corresponde à chave pública
4. **Regras de Negócio**:
   - Amount > 0
   - From ≠ To
   - Timestamp não muito no futuro (±5 minutos)

#### Merkle Tree

As transações de um bloco formam uma Merkle Tree, cuja raiz é armazenada no header do bloco. Isso permite verificação eficiente de inclusão de transações.

```
        Root Hash
       /          \
    Hash01      Hash23
    /    \      /    \
  Tx0   Tx1   Tx2   Tx3
```

### 3. Block (Bloco)

Localização: `pkg/blockchain/block.go`

Blocos agrupam transações e formam a cadeia da blockchain.

#### Estrutura

```go
type BlockHeader struct {
    Version       uint32  // Versão do protocolo
    Height        uint64  // Altura do bloco na chain
    Timestamp     int64   // Timestamp Unix
    PreviousHash  string  // Hash do bloco anterior
    MerkleRoot    string  // Raiz da árvore de Merkle
    ValidatorAddr string  // Endereço do validador
    Signature     string  // Assinatura do validador
    PublicKey     string  // Chave pública do validador
    Nonce         uint64  // Nonce (ordenação/desempate)
}

type Block struct {
    Header       BlockHeader
    Transactions TransactionSlice
    Hash         string
}
```

#### Validações de Bloco

1. **Verificação de Hash**: Hash do bloco está correto
2. **Verificação de Merkle Root**: Raiz de Merkle corresponde às transações
3. **Verificação de Transações**:
   - Primeira transação é coinbase
   - Demais transações não são coinbase
   - Sem transações duplicadas
   - Todas as transações são válidas
4. **Verificação de Encadeamento**:
   - PreviousHash corresponde ao hash do bloco anterior
   - Height é sequencial
   - Timestamp é posterior ao bloco anterior

#### Bloco Gênesis

O bloco gênesis é o primeiro bloco da blockchain e possui características especiais:

- Height = 0
- PreviousHash = "" (vazio)
- Contém apenas uma transação coinbase
- Hash é determinístico e conhecido

### 4. Configuração

Localização: `internal/config/config.go`

#### Estrutura de Configuração

```json
{
  "id": "node1",
  "address": ":9001",
  "db_path": "./data/node1",
  "signaling_server": "ws://localhost:9000/ws",
  "max_peers": 50,
  "min_peers": 5,
  "discovery_interval": 30,
  "wallet": {
    "private_key": "...",
    "public_key": "...",
    "address": "..."
  },
  "genesis": {
    "timestamp": 1609459200,
    "recipient_addr": "...",
    "amount": 1000000000,
    "hash": "..."
  }
}
```

#### Campos

- **wallet**: Chaves da carteira do nó (usadas para assinar blocos e transações)
- **genesis**: Configuração do bloco gênesis (opcional, usado apenas para nós iniciais)

## Segurança

### Criptografia ECDSA

- **Curva**: P-256 (secp256r1)
- **Tamanho da Chave**: 256 bits
- **Algoritmo de Hash**: SHA-256
- **Biblioteca**: crypto/ecdsa do Go (padrão)

### Proteções Implementadas

1. **Assinatura Digital**: Todas as transações regulares são assinadas
2. **Hash de Integridade**: Blocos e transações têm hashes verificáveis
3. **Merkle Tree**: Verificação eficiente de inclusão de transações
4. **Nonce**: Previne replay attacks
5. **Timestamp Validation**: Rejeita transações com timestamps muito no futuro
6. **Address Derivation**: Endereços são derivados deterministicamente da chave pública

### Proteções Faltando (TODO)

- [ ] Assinatura de blocos por validadores
- [ ] Verificação de stake do validador
- [ ] Verificação de duplo gasto (necessita state management)
- [ ] Validação de saldo (necessita state management)

## Fluxo de uma Transação

```
1. Usuário cria transação
   ↓
2. Usuário assina transação com sua carteira
   ↓
3. Transação é validada (assinatura, campos, regras)
   ↓
4. Transação é propagada pela rede (Gossip)
   ↓
5. Validador inclui transação em um bloco
   ↓
6. Bloco é validado pela rede
   ↓
7. Bloco é adicionado à blockchain
   ↓
8. Transação é considerada confirmada
```

## Fluxo de um Bloco

```
1. Validador coleta transações pendentes
   ↓
2. Validador cria transação coinbase (recompensa)
   ↓
3. Validador cria bloco com transações
   ↓
4. Validador calcula Merkle Root
   ↓
5. Validador calcula hash do bloco
   ↓
6. Validador assina bloco (TODO)
   ↓
7. Bloco é propagado pela rede (Gossip)
   ↓
8. Nós validam o bloco
   ↓
9. Bloco é adicionado à chain local
   ↓
10. Estado é atualizado (TODO)
```

## Utilitários

### Gerador de Carteiras

Localização: `cmd/wallet-gen/main.go`

Gera novas carteiras ECDSA:

```bash
# Gerar uma carteira
make wallet-gen

# Gerar múltiplas carteiras
go run cmd/wallet-gen/main.go -count 5

# Salvar em arquivo
go run cmd/wallet-gen/main.go -output wallets.json
```

## Testes

### Testes Unitários

```bash
# Testar carteiras
go test ./pkg/wallet -v

# Testar blockchain
go test ./pkg/blockchain -v

# Testar tudo
make test-all
```

### Cobertura de Testes

- **Wallet**: 12 testes + 5 benchmarks
- **Transaction**: 23 testes + 4 benchmarks
- **Block**: 26 testes + 3 benchmarks

Total: **61 testes + 12 benchmarks**

## Performance

### Benchmarks (aproximados)

```
BenchmarkNewWallet         - ~5000 ops/sec (200 µs/op)
BenchmarkSign              - ~8000 ops/sec (125 µs/op)
BenchmarkVerify            - ~3000 ops/sec (333 µs/op)
BenchmarkTransactionSign   - ~8000 ops/sec (125 µs/op)
BenchmarkBlockCalculateHash - ~50000 ops/sec (20 µs/op)
BenchmarkMerkleRoot(100tx) - ~1000 ops/sec (1000 µs/op)
```

## Próximos Passos

### Curto Prazo

1. Implementar assinatura de blocos por validadores
2. Criar transaction pool (mempool)
3. Implementar validação de duplo gasto
4. Adicionar gerenciamento de estado (saldos)

### Médio Prazo

1. Implementar consenso PoS
2. Adicionar sincronização de blockchain
3. Implementar fork resolution
4. Adicionar checkpoints

### Longo Prazo

1. Suporte a smart contracts
2. Sharding
3. State pruning
4. Light clients

## Referências

- [ECDSA - Elliptic Curve Digital Signature Algorithm](https://en.wikipedia.org/wiki/Elliptic_Curve_Digital_Signature_Algorithm)
- [Merkle Tree](https://en.wikipedia.org/wiki/Merkle_tree)
- [Bitcoin Whitepaper](https://bitcoin.org/bitcoin.pdf)
- [Ethereum Yellow Paper](https://ethereum.github.io/yellowpaper/paper.pdf)
- [Go crypto/ecdsa](https://pkg.go.dev/crypto/ecdsa)

## Arquivos Relacionados

- `pkg/wallet/wallet.go` - Implementação de carteiras
- `pkg/wallet/wallet_test.go` - Testes de carteiras
- `pkg/blockchain/transaction.go` - Implementação de transações
- `pkg/blockchain/transaction_test.go` - Testes de transações
- `pkg/blockchain/block.go` - Implementação de blocos
- `pkg/blockchain/block_test.go` - Testes de blocos
- `internal/config/config.go` - Estrutura de configuração
- `cmd/wallet-gen/main.go` - Gerador de carteiras
- `configs/node1.example.json` - Exemplo de configuração
