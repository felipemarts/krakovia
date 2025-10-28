package blockchain

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/big"
	"sort"
)

// Validator representa um validador com seu endereço e stake
type Validator struct {
	Address string // Endereço do validador
	Stake   uint64 // Quantidade de tokens em stake
}

// ValidatorList é uma lista de validadores
type ValidatorList []Validator

// TotalStake retorna o stake total de todos os validadores
func (vl ValidatorList) TotalStake() uint64 {
	var total uint64
	for _, v := range vl {
		total += v.Stake
	}
	return total
}

// ValidateList valida a lista de validadores
func (vl ValidatorList) ValidateList() error {
	if len(vl) == 0 {
		return fmt.Errorf("validator list is empty")
	}

	seen := make(map[string]bool)
	for i, v := range vl {
		if v.Address == "" {
			return fmt.Errorf("validator %d has empty address", i)
		}
		if v.Stake == 0 {
			return fmt.Errorf("validator %d (%s) has zero stake", i, v.Address)
		}
		if seen[v.Address] {
			return fmt.Errorf("duplicate validator address: %s", v.Address)
		}
		seen[v.Address] = true
	}

	return nil
}

// GetValidator retorna um validador pelo endereço
func (vl ValidatorList) GetValidator(address string) *Validator {
	for i := range vl {
		if vl[i].Address == address {
			return &vl[i]
		}
	}
	return nil
}

// PriorityQueue representa uma fila de prioridade de validadores ordenada por prioridade
type PriorityQueue struct {
	Validators []Validator // Validadores ordenados por prioridade (maior prioridade primeiro)
	Priorities []uint64    // Prioridades correspondentes
}

// CalculateValidatorPriority calcula a prioridade dos validadores baseado no hash do bloco anterior
// Retorna uma lista ordenada por prioridade (maior prioridade primeiro)
//
// Algoritmo:
// 1. Usa o hash do bloco anterior como seed para determinismo
// 2. Para cada validador, calcula um score baseado em:
//    - Hash(previousBlockHash + validatorAddress) como fonte de aleatoriedade
//    - Multiplica pelo stake do validador para dar peso proporcional
// 3. Ordena por score (maior primeiro)
//
// Isso garante:
// - Determinismo: mesmo hash sempre produz mesma ordem
// - Proporcionalidade: validadores com mais stake têm maior probabilidade de prioridade
// - Randomização: a ordem varia com cada bloco
func CalculateValidatorPriority(previousBlockHash string, validators ValidatorList) (*PriorityQueue, error) {
	// Valida entrada
	if previousBlockHash == "" {
		return nil, fmt.Errorf("previous block hash is empty")
	}

	if err := validators.ValidateList(); err != nil {
		return nil, fmt.Errorf("invalid validator list: %w", err)
	}

	// Calcula prioridade para cada validador
	type validatorScore struct {
		validator Validator
		score     *big.Int
	}

	scores := make([]validatorScore, len(validators))

	for i, validator := range validators {
		// Combina hash do bloco anterior com endereço do validador
		data := previousBlockHash + validator.Address
		hash := sha256.Sum256([]byte(data))

		// Converte hash para big.Int (256 bits)
		hashInt := new(big.Int).SetBytes(hash[:])

		// Multiplica pelo stake para dar peso proporcional
		stake := new(big.Int).SetUint64(validator.Stake)
		score := new(big.Int).Mul(hashInt, stake)

		scores[i] = validatorScore{
			validator: validator,
			score:     score,
		}
	}

	// Ordena por score (maior primeiro)
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score.Cmp(scores[j].score) > 0
	})

	// Cria a fila de prioridade
	pq := &PriorityQueue{
		Validators: make([]Validator, len(scores)),
		Priorities: make([]uint64, len(scores)),
	}

	for i, vs := range scores {
		pq.Validators[i] = vs.validator
		// Trunca score para uint64 para facilitar visualização (usa os 64 bits mais significativos)
		pq.Priorities[i] = truncateBigIntToUint64(vs.score)
	}

	return pq, nil
}

// CalculateValidatorPriorityWithSeed é uma variante que aceita um seed customizado
// Útil para testes ou casos específicos
func CalculateValidatorPriorityWithSeed(seed []byte, validators ValidatorList) (*PriorityQueue, error) {
	if len(seed) == 0 {
		return nil, fmt.Errorf("seed is empty")
	}

	if err := validators.ValidateList(); err != nil {
		return nil, fmt.Errorf("invalid validator list: %w", err)
	}

	type validatorScore struct {
		validator Validator
		score     *big.Int
	}

	scores := make([]validatorScore, len(validators))

	for i, validator := range validators {
		// Combina seed com endereço do validador
		data := append(seed, []byte(validator.Address)...)
		hash := sha256.Sum256(data)

		hashInt := new(big.Int).SetBytes(hash[:])
		stake := new(big.Int).SetUint64(validator.Stake)
		score := new(big.Int).Mul(hashInt, stake)

		scores[i] = validatorScore{
			validator: validator,
			score:     score,
		}
	}

	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score.Cmp(scores[j].score) > 0
	})

	pq := &PriorityQueue{
		Validators: make([]Validator, len(scores)),
		Priorities: make([]uint64, len(scores)),
	}

	for i, vs := range scores {
		pq.Validators[i] = vs.validator
		pq.Priorities[i] = truncateBigIntToUint64(vs.score)
	}

	return pq, nil
}

// GetTopValidator retorna o validador com maior prioridade
func (pq *PriorityQueue) GetTopValidator() *Validator {
	if len(pq.Validators) == 0 {
		return nil
	}
	return &pq.Validators[0]
}

// GetTopN retorna os N validadores com maior prioridade
func (pq *PriorityQueue) GetTopN(n int) []Validator {
	if n <= 0 || len(pq.Validators) == 0 {
		return []Validator{}
	}

	if n > len(pq.Validators) {
		n = len(pq.Validators)
	}

	result := make([]Validator, n)
	copy(result, pq.Validators[:n])
	return result
}

// GetValidatorRank retorna a posição (rank) de um validador na fila de prioridade
// Retorna -1 se o validador não for encontrado
func (pq *PriorityQueue) GetValidatorRank(address string) int {
	for i, v := range pq.Validators {
		if v.Address == address {
			return i
		}
	}
	return -1
}

// IsTopValidator verifica se um endereço é o validador com maior prioridade
func (pq *PriorityQueue) IsTopValidator(address string) bool {
	return pq.GetValidatorRank(address) == 0
}

// truncateBigIntToUint64 converte um big.Int para uint64
// Usa módulo para evitar overflow e manter alguma informação
func truncateBigIntToUint64(n *big.Int) uint64 {
	// Pega os bytes do número
	bytes := n.Bytes()

	// Se tiver menos de 8 bytes, converte diretamente
	if len(bytes) <= 8 {
		return n.Uint64()
	}

	// Para números grandes, usa hash SHA-256 para comprimir a informação
	// mantendo uma distribuição uniforme
	hash := sha256.Sum256(bytes)
	return binary.BigEndian.Uint64(hash[:8])
}

// WeightedRandomSelection implementa uma seleção aleatória ponderada usando o hash como seed
// Retorna o índice do validador selecionado
// Este método é uma alternativa ao CalculateValidatorPriority para casos onde você quer
// selecionar apenas um validador diretamente
func WeightedRandomSelection(previousBlockHash string, validators ValidatorList) (int, error) {
	if previousBlockHash == "" {
		return -1, fmt.Errorf("previous block hash is empty")
	}

	if err := validators.ValidateList(); err != nil {
		return -1, fmt.Errorf("invalid validator list: %w", err)
	}

	totalStake := validators.TotalStake()
	if totalStake == 0 {
		return -1, fmt.Errorf("total stake is zero")
	}

	// Usa o hash como seed para gerar um número aleatório
	hashBytes, err := hex.DecodeString(previousBlockHash)
	if err != nil {
		return -1, fmt.Errorf("invalid hash format: %w", err)
	}

	// Converte hash para big.Int
	hashInt := new(big.Int).SetBytes(hashBytes)

	// Gera um número aleatório entre 0 e totalStake
	totalStakeBig := new(big.Int).SetUint64(totalStake)
	randomValue := new(big.Int).Mod(hashInt, totalStakeBig).Uint64()

	// Seleciona o validador baseado no valor aleatório
	var cumulative uint64
	for i, validator := range validators {
		cumulative += validator.Stake
		if randomValue < cumulative {
			return i, nil
		}
	}

	// Não deve chegar aqui, mas retorna o último como fallback
	return len(validators) - 1, nil
}

// SimulateSelectionDistribution simula a seleção de validadores N vezes
// usando diferentes hashes (simulando N blocos) e retorna a contagem de
// quantas vezes cada validador foi selecionado como top priority
// Útil para verificar a distribuição de probabilidade
func SimulateSelectionDistribution(validators ValidatorList, iterations int) (map[string]int, error) {
	if err := validators.ValidateList(); err != nil {
		return nil, fmt.Errorf("invalid validator list: %w", err)
	}

	if iterations <= 0 {
		return nil, fmt.Errorf("iterations must be positive")
	}

	distribution := make(map[string]int)
	for _, v := range validators {
		distribution[v.Address] = 0
	}

	// Simula diferentes hashes de blocos
	for i := 0; i < iterations; i++ {
		// Cria um hash único para cada iteração
		seed := []byte(fmt.Sprintf("simulation-%d", i))
		hash := sha256.Sum256(seed)
		fakeHash := hex.EncodeToString(hash[:])

		// Calcula prioridade
		pq, err := CalculateValidatorPriority(fakeHash, validators)
		if err != nil {
			return nil, err
		}

		// Incrementa contador do validador com maior prioridade
		topValidator := pq.GetTopValidator()
		if topValidator != nil {
			distribution[topValidator.Address]++
		}
	}

	return distribution, nil
}

// GetExpectedProbability retorna a probabilidade esperada de um validador
// ser selecionado baseado em seu stake
func GetExpectedProbability(validator Validator, totalStake uint64) float64 {
	if totalStake == 0 {
		return 0
	}
	return float64(validator.Stake) / float64(totalStake)
}

// ValidatorSet representa um conjunto de validadores com métodos auxiliares
type ValidatorSet struct {
	Validators ValidatorList
	TotalStake uint64
}

// NewValidatorSet cria um novo conjunto de validadores
func NewValidatorSet(validators ValidatorList) (*ValidatorSet, error) {
	if err := validators.ValidateList(); err != nil {
		return nil, err
	}

	return &ValidatorSet{
		Validators: validators,
		TotalStake: validators.TotalStake(),
	}, nil
}

// CalculatePriority calcula a prioridade usando o hash do bloco anterior
func (vs *ValidatorSet) CalculatePriority(previousBlockHash string) (*PriorityQueue, error) {
	return CalculateValidatorPriority(previousBlockHash, vs.Validators)
}

// AddValidator adiciona um validador ao conjunto
func (vs *ValidatorSet) AddValidator(validator Validator) error {
	if validator.Address == "" {
		return fmt.Errorf("validator address is empty")
	}
	if validator.Stake == 0 {
		return fmt.Errorf("validator stake is zero")
	}

	// Verifica se já existe
	if vs.Validators.GetValidator(validator.Address) != nil {
		return fmt.Errorf("validator already exists: %s", validator.Address)
	}

	vs.Validators = append(vs.Validators, validator)
	vs.TotalStake += validator.Stake
	return nil
}

// RemoveValidator remove um validador do conjunto
func (vs *ValidatorSet) RemoveValidator(address string) error {
	for i, v := range vs.Validators {
		if v.Address == address {
			vs.TotalStake -= v.Stake
			vs.Validators = append(vs.Validators[:i], vs.Validators[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("validator not found: %s", address)
}

// UpdateStake atualiza o stake de um validador
func (vs *ValidatorSet) UpdateStake(address string, newStake uint64) error {
	if newStake == 0 {
		return fmt.Errorf("new stake cannot be zero")
	}

	for i := range vs.Validators {
		if vs.Validators[i].Address == address {
			oldStake := vs.Validators[i].Stake
			vs.Validators[i].Stake = newStake
			vs.TotalStake = vs.TotalStake - oldStake + newStake
			return nil
		}
	}
	return fmt.Errorf("validator not found: %s", address)
}

// Count retorna o número de validadores
func (vs *ValidatorSet) Count() int {
	return len(vs.Validators)
}
