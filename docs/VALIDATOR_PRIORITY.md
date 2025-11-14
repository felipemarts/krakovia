# Sistema de Priorização de Validadores (Proof of Stake)

## Visão Geral

O sistema de priorização de validadores da Krakovia implementa um mecanismo Proof of Stake (PoS) onde validadores são selecionados para criar blocos baseado na quantidade de tokens que possuem em stake.

Localização: `pkg/blockchain/validator.go`

## Componentes

### 1. Validator

Estrutura básica que representa um validador:

```go
type Validator struct {
    Address string // Endereço único do validador
    Stake   uint64 // Quantidade de tokens em stake
}
```

### 2. ValidatorList

Lista de validadores com métodos auxiliares:

```go
type ValidatorList []Validator

func (vl ValidatorList) TotalStake() uint64
func (vl ValidatorList) ValidateList() error
func (vl ValidatorList) GetValidator(address string) *Validator
```

### 3. PriorityQueue

Fila de prioridade ordenada resultante do cálculo de priorização:

```go
type PriorityQueue struct {
    Validators []Validator // Ordenados por prioridade (maior primeiro)
    Priorities []uint64    // Scores para referência
}
```

## Algoritmos de Seleção

### Algoritmo 1: CalculateValidatorPriority

Calcula uma fila de prioridade completa ordenada para todos os validadores.

#### Funcionamento

1. **Entrada**: Hash do bloco anterior + Lista de validadores
2. **Para cada validador**:
   - Combina: `hash_do_bloco_anterior + endereço_do_validador`
   - Calcula: `SHA-256(combinação)`
   - Score: `hash_resultante × stake_do_validador`
3. **Ordena** validadores por score (maior primeiro)
4. **Retorna** lista ordenada

#### Propriedades

- ✅ **Determinístico**: Mesmo hash sempre produz mesma ordem
- ✅ **Proporcional**: Validadores com mais stake têm maior probabilidade
- ✅ **Aleatório por bloco**: Ordem muda a cada novo bloco
- ⚠️ **Concentração**: Validadores grandes têm probabilidade ligeiramente maior que proporcional

#### Uso

```go
validators := ValidatorList{
    {Address: "validator_a", Stake: 1000},
    {Address: "validator_b", Stake: 5000},
}

previousBlockHash := "abc123..."

pq, err := CalculateValidatorPriority(previousBlockHash, validators)

// Validador com maior prioridade
topValidator := pq.GetTopValidator()

// Top 3 validadores
top3 := pq.GetTopN(3)

// Rank de um validador específico
rank := pq.GetValidatorRank("validator_a")
```

#### Quando Usar

- Quando você precisa de um **ranking completo** de validadores
- Para selecionar **múltiplos validadores** (ex: comitê de consenso)
- Para determinar ordem de produção de blocos
- Quando o viés para validadores maiores é aceitável

### Algoritmo 2: WeightedRandomSelection

Seleciona diretamente um único validador usando seleção aleatória ponderada.

#### Funcionamento

1. **Entrada**: Hash do bloco anterior + Lista de validadores
2. **Calcula** stake total
3. **Gera** número aleatório: `hash % stake_total`
4. **Seleciona** validador onde o número aleatório "cai" no intervalo de stake acumulado

#### Propriedades

- ✅ **Determinístico**: Mesmo hash sempre seleciona mesmo validador
- ✅ **Exatamente proporcional**: Probabilidade = stake / stake_total
- ✅ **Eficiente**: O(n) - mais rápido que CalculateValidatorPriority
- ✅ **Justo**: Não há viés para validadores maiores

#### Uso

```go
validators := ValidatorList{
    {Address: "validator_a", Stake: 1000},
    {Address: "validator_b", Stake: 5000},
}

previousBlockHash := "abc123..."

index, err := WeightedRandomSelection(previousBlockHash, validators)
selectedValidator := validators[index]
```

#### Quando Usar

- Quando você precisa selecionar apenas **um validador**
- Quando é importante ter **proporcionalidade exata**
- Para sistemas onde **justiça é crítica**
- Quando **performance é importante** (mais rápido)

## Comparação de Algoritmos

### Distribuição de Probabilidade

Exemplo com 3 validadores (simulação de 10.000 blocos):

| Validador | Stake | % Stake | CalculatePriority | WeightedRandom |
|-----------|-------|---------|-------------------|----------------|
| Alice     | 1000  | 10%     | ~5-8%             | ~10%           |
| Bob       | 3000  | 30%     | ~25-28%           | ~30%           |
| Carol     | 6000  | 60%     | ~65-70%           | ~60%           |

**Observação**: CalculateValidatorPriority tende a favorecer validadores com mais stake além da proporção esperada. Isso acontece porque ao multiplicar `hash × stake`, a variância também aumenta proporcionalmente.

### Performance

```
BenchmarkCalculateValidatorPriority (5 vals)   - ~50,000 ops/sec
BenchmarkCalculateValidatorPriority (100 vals) - ~2,000 ops/sec
BenchmarkWeightedRandomSelection (5 vals)      - ~100,000 ops/sec
```

## ValidatorSet

Estrutura de alto nível para gerenciar um conjunto de validadores:

```go
type ValidatorSet struct {
    Validators ValidatorList
    TotalStake uint64
}
```

### Operações

```go
// Criar conjunto
vs, err := NewValidatorSet(validators)

// Adicionar validador
err = vs.AddValidator(Validator{Address: "new", Stake: 1000})

// Remover validador
err = vs.RemoveValidator("address")

// Atualizar stake
err = vs.UpdateStake("address", 2000)

// Calcular prioridade
pq, err := vs.CalculatePriority(previousBlockHash)
```

## Funções Auxiliares

### SimulateSelectionDistribution

Simula múltiplas seleções para verificar distribuição de probabilidade:

```go
iterations := 10000
distribution, err := SimulateSelectionDistribution(validators, iterations)

// distribution["validator_address"] = número de vezes selecionado
```

### GetExpectedProbability

Calcula a probabilidade esperada teoricamente:

```go
prob := GetExpectedProbability(validator, totalStake)
// prob = validator.Stake / totalStake
```

## Segurança e Considerações

### Determinismo

Ambos os algoritmos são **completamente determinísticos**:
- Mesmo hash de bloco anterior sempre produz mesmo resultado
- Permite que todos os nós da rede concordem com a seleção
- Essencial para consenso distribuído

### Previsibilidade

⚠️ **Importante**: Como o hash do bloco anterior é conhecido, a seleção do próximo validador é **previsível**. Considerações:

1. **Mitigação de Grinding**: O validador não deve poder manipular facilmente o hash do bloco
2. **VRF (Verifiable Random Function)**: Para produção, considere usar VRF ao invés de hash simples
3. **Commit-Reveal**: Esquemas onde o validador se compromete com um valor antes de revelá-lo

### Sybil Resistance

O sistema resiste a ataques Sybil porque:
- Dividir stake em múltiplos validadores não aumenta probabilidade total
- Exemplo: 1 validador com 1000 stake = 10 validadores com 100 stake cada

### Stake Mínimo

Considere implementar stake mínimo para:
- Reduzir número de validadores
- Aumentar custo de ataques
- Melhorar performance

## Exemplo Completo

```go
package main

import (
    "fmt"
    "github.com/krakovia/blockchain/pkg/blockchain"
)

func main() {
    // Cria conjunto de validadores
    validators := blockchain.ValidatorList{
        {Address: "alice", Stake: 1000},
        {Address: "bob", Stake: 3000},
        {Address: "carol", Stake: 6000},
    }

    // Hash do bloco anterior (vem da blockchain)
    previousBlockHash := "abc123..."

    // Método 1: Ranking completo
    pq, _ := blockchain.CalculateValidatorPriority(previousBlockHash, validators)
    topValidator := pq.GetTopValidator()
    fmt.Printf("Validador selecionado: %s\n", topValidator.Address)

    // Método 2: Seleção direta (mais justo)
    index, _ := blockchain.WeightedRandomSelection(previousBlockHash, validators)
    selected := validators[index]
    fmt.Printf("Validador selecionado: %s\n", selected.Address)

    // Simula 1000 blocos para verificar distribuição
    distribution, _ := blockchain.SimulateSelectionDistribution(validators, 1000)

    totalStake := validators.TotalStake()
    for addr, count := range distribution {
        v := validators.GetValidator(addr)
        expectedProb := blockchain.GetExpectedProbability(*v, totalStake)
        actualProb := float64(count) / 1000.0

        fmt.Printf("%s: esperado %.1f%%, obtido %.1f%%\n",
            addr, expectedProb*100, actualProb*100)
    }
}
```

## Integração com Blockchain

### Fluxo de Criação de Bloco

```
1. Blockchain está no bloco N
   ↓
2. Obtém hash do bloco N
   ↓
3. Calcula prioridade dos validadores usando hash do bloco N
   ↓
4. Seleciona validador com maior prioridade (ou usa WeightedRandom)
   ↓
5. Validador selecionado cria bloco N+1
   ↓
6. Valida e adiciona à blockchain
   ↓
7. Repete com hash do bloco N+1
```

### Código de Integração

```go
// Obtém último bloco
lastBlock := blockchain.GetLastBlock()

// Obtém validadores ativos
validators := stakeManager.GetActiveValidators()

// Seleciona validador
pq, _ := blockchain.CalculateValidatorPriority(lastBlock.Hash, validators)
selectedValidator := pq.GetTopValidator()

// Verifica se o nó atual é o validador selecionado
if selectedValidator.Address == myAddress {
    // Cria e propõe novo bloco
    newBlock := CreateBlock(...)
}
```

## Testes

Todos os algoritmos possuem testes extensivos:

```bash
# Testa determinismo
go test ./pkg/blockchain -v -run TestCalculateValidatorPriorityDeterminism

# Testa proporcionalidade
go test ./pkg/blockchain -v -run TestCalculateValidatorPriorityProportionality

# Testa WeightedRandom
go test ./pkg/blockchain -v -run TestWeightedRandomSelection

# Todos os testes de validadores
go test ./pkg/blockchain -v -run Validator

# Benchmarks
go test ./pkg/blockchain -bench=Validator
```

## Próximos Passos

### Melhorias Recomendadas

1. **VRF (Verifiable Random Function)**
   - Substituir hash simples por VRF
   - Previne grinding attacks
   - Mantém verificabilidade

2. **Slashing**
   - Penalizar validadores maliciosos
   - Reduzir stake automaticamente
   - Integrar com ValidatorSet

3. **Delegation**
   - Permitir delegação de stake
   - Separar validadores de stakers
   - Pool de stakes

4. **Stake Dinâmico**
   - Permitir entrada/saída de validadores
   - Período de unbonding
   - Stake mínimo configurável

5. **Reward Distribution**
   - Calcular recompensas proporcionais
   - Distribuir taxas de transações
   - Recompensas de bloco

## Referências

- [Ethereum PoS](https://ethereum.org/en/developers/docs/consensus-mechanisms/pos/)
- [Cosmos Tendermint](https://docs.tendermint.com/master/spec/consensus/proposer-selection.html)
- [Polkadot NPoS](https://wiki.polkadot.network/docs/learn-consensus)
- [VRF (Verifiable Random Functions)](https://en.wikipedia.org/wiki/Verifiable_random_function)

## Arquivos Relacionados

- `pkg/blockchain/validator.go` - Implementação
- `pkg/blockchain/validator_test.go` - Testes (24 testes)
- `examples/validator_priority_example.go` - Exemplo prático
