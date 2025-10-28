package blockchain

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"testing"
)

func TestValidatorListTotalStake(t *testing.T) {
	validators := ValidatorList{
		{Address: "addr1", Stake: 100},
		{Address: "addr2", Stake: 200},
		{Address: "addr3", Stake: 300},
	}

	total := validators.TotalStake()
	if total != 600 {
		t.Errorf("Expected total stake 600, got %d", total)
	}
}

func TestValidatorListValidate(t *testing.T) {
	// Lista válida
	validList := ValidatorList{
		{Address: "addr1", Stake: 100},
		{Address: "addr2", Stake: 200},
	}

	if err := validList.ValidateList(); err != nil {
		t.Errorf("Valid list failed validation: %v", err)
	}

	// Lista vazia
	emptyList := ValidatorList{}
	if err := emptyList.ValidateList(); err == nil {
		t.Error("Empty list should fail validation")
	}

	// Validador com endereço vazio
	invalidAddr := ValidatorList{
		{Address: "", Stake: 100},
	}
	if err := invalidAddr.ValidateList(); err == nil {
		t.Error("List with empty address should fail validation")
	}

	// Validador com stake zero
	zeroStake := ValidatorList{
		{Address: "addr1", Stake: 0},
	}
	if err := zeroStake.ValidateList(); err == nil {
		t.Error("List with zero stake should fail validation")
	}

	// Endereços duplicados
	duplicates := ValidatorList{
		{Address: "addr1", Stake: 100},
		{Address: "addr1", Stake: 200},
	}
	if err := duplicates.ValidateList(); err == nil {
		t.Error("List with duplicate addresses should fail validation")
	}
}

func TestCalculateValidatorPriority(t *testing.T) {
	validators := ValidatorList{
		{Address: "validator1", Stake: 100},
		{Address: "validator2", Stake: 200},
		{Address: "validator3", Stake: 300},
	}

	hash := "0000000000000000000000000000000000000000000000000000000000000001"

	pq, err := CalculateValidatorPriority(hash, validators)
	if err != nil {
		t.Fatalf("Failed to calculate priority: %v", err)
	}

	// Verifica que todos os validadores estão presentes
	if len(pq.Validators) != 3 {
		t.Errorf("Expected 3 validators in priority queue, got %d", len(pq.Validators))
	}

	// Verifica que há prioridades atribuídas
	if len(pq.Priorities) != 3 {
		t.Errorf("Expected 3 priorities, got %d", len(pq.Priorities))
	}

	// Verifica que a fila de prioridade retorna resultados consistentes
	top := pq.GetTopValidator()
	if top == nil {
		t.Fatal("Top validator should not be nil")
	}
	if top.Address != pq.Validators[0].Address {
		t.Error("Top validator doesn't match first in queue")
	}
}

func TestCalculateValidatorPriorityDeterminism(t *testing.T) {
	validators := ValidatorList{
		{Address: "validator1", Stake: 100},
		{Address: "validator2", Stake: 200},
		{Address: "validator3", Stake: 300},
	}

	hash := "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"

	// Calcula duas vezes com o mesmo hash
	pq1, err1 := CalculateValidatorPriority(hash, validators)
	if err1 != nil {
		t.Fatalf("Failed to calculate priority (1): %v", err1)
	}

	pq2, err2 := CalculateValidatorPriority(hash, validators)
	if err2 != nil {
		t.Fatalf("Failed to calculate priority (2): %v", err2)
	}

	// Verifica que a ordem é exatamente a mesma
	for i := 0; i < len(pq1.Validators); i++ {
		if pq1.Validators[i].Address != pq2.Validators[i].Address {
			t.Errorf("Order differs at position %d: %s vs %s",
				i, pq1.Validators[i].Address, pq2.Validators[i].Address)
		}
		if pq1.Priorities[i] != pq2.Priorities[i] {
			t.Errorf("Priorities differ at position %d: %d vs %d",
				i, pq1.Priorities[i], pq2.Priorities[i])
		}
	}
}

func TestCalculateValidatorPriorityDifferentHashes(t *testing.T) {
	validators := ValidatorList{
		{Address: "validator1", Stake: 100},
		{Address: "validator2", Stake: 200},
		{Address: "validator3", Stake: 300},
	}

	hash1 := "0000000000000000000000000000000000000000000000000000000000000001"
	hash2 := "0000000000000000000000000000000000000000000000000000000000000002"

	pq1, _ := CalculateValidatorPriority(hash1, validators)
	pq2, _ := CalculateValidatorPriority(hash2, validators)

	// Verifica que a ordem pode ser diferente com hashes diferentes
	// (não é garantido, mas é altamente provável)
	sameOrder := true
	for i := 0; i < len(pq1.Validators); i++ {
		if pq1.Validators[i].Address != pq2.Validators[i].Address {
			sameOrder = false
			break
		}
	}

	// Com hashes diferentes, é extremamente improvável que a ordem seja exatamente a mesma
	if sameOrder {
		t.Log("Warning: Same order with different hashes (unlikely but possible)")
	}
}

func TestCalculateValidatorPriorityProportionality(t *testing.T) {
	// Testa se validadores com mais stake aparecem mais frequentemente no topo
	validators := ValidatorList{
		{Address: "small", Stake: 10},  // 10% do stake
		{Address: "large", Stake: 90},  // 90% do stake
	}

	iterations := 10000
	distribution, err := SimulateSelectionDistribution(validators, iterations)
	if err != nil {
		t.Fatalf("Failed to simulate distribution: %v", err)
	}

	smallCount := distribution["small"]
	largeCount := distribution["large"]

	// Calcula as proporções
	smallRatio := float64(smallCount) / float64(iterations)
	largeRatio := float64(largeCount) / float64(iterations)

	t.Logf("Small validator (10%% stake): selected %.2f%% of the time", smallRatio*100)
	t.Logf("Large validator (90%% stake): selected %.2f%% of the time", largeRatio*100)

	// Verifica que o validador com mais stake é selecionado mais frequentemente
	if largeCount <= smallCount {
		t.Error("Validator with more stake should be selected more frequently")
	}

	// Verifica que a proporção está razoavelmente próxima do esperado (com margem de erro)
	expectedLargeRatio := 0.90
	tolerance := 0.05 // 5% de tolerância

	if math.Abs(largeRatio-expectedLargeRatio) > tolerance {
		t.Errorf("Large validator ratio %.2f is too far from expected %.2f (tolerance %.2f)",
			largeRatio, expectedLargeRatio, tolerance)
	}
}

func TestCalculateValidatorPriorityEqualStake(t *testing.T) {
	// Com stakes iguais, todos devem ter aproximadamente a mesma chance
	validators := ValidatorList{
		{Address: "validator1", Stake: 100},
		{Address: "validator2", Stake: 100},
		{Address: "validator3", Stake: 100},
	}

	iterations := 9000 // Divisível por 3
	distribution, err := SimulateSelectionDistribution(validators, iterations)
	if err != nil {
		t.Fatalf("Failed to simulate distribution: %v", err)
	}

	expectedCount := iterations / 3
	tolerance := float64(iterations) * 0.1 // 10% de tolerância

	for addr, count := range distribution {
		t.Logf("Validator %s: selected %d times (expected ~%d)", addr, count, expectedCount)

		diff := math.Abs(float64(count) - float64(expectedCount))
		if diff > tolerance {
			t.Errorf("Validator %s selection count %d is too far from expected %d",
				addr, count, expectedCount)
		}
	}
}

func TestPriorityQueueGetTopValidator(t *testing.T) {
	validators := ValidatorList{
		{Address: "validator1", Stake: 100},
		{Address: "validator2", Stake: 200},
	}

	hash := "0000000000000000000000000000000000000000000000000000000000000001"
	pq, _ := CalculateValidatorPriority(hash, validators)

	top := pq.GetTopValidator()
	if top == nil {
		t.Fatal("Top validator is nil")
	}

	// O top validator deve ser o primeiro da lista
	if top.Address != pq.Validators[0].Address {
		t.Error("GetTopValidator doesn't match first validator in queue")
	}
}

func TestPriorityQueueGetTopN(t *testing.T) {
	validators := ValidatorList{
		{Address: "validator1", Stake: 100},
		{Address: "validator2", Stake: 200},
		{Address: "validator3", Stake: 300},
		{Address: "validator4", Stake: 400},
	}

	hash := "0000000000000000000000000000000000000000000000000000000000000001"
	pq, _ := CalculateValidatorPriority(hash, validators)

	// Pega top 2
	top2 := pq.GetTopN(2)
	if len(top2) != 2 {
		t.Errorf("Expected 2 validators, got %d", len(top2))
	}

	// Pega top 10 (mais que o disponível)
	top10 := pq.GetTopN(10)
	if len(top10) != 4 {
		t.Errorf("Expected 4 validators (all), got %d", len(top10))
	}

	// Pega top 0
	top0 := pq.GetTopN(0)
	if len(top0) != 0 {
		t.Errorf("Expected 0 validators, got %d", len(top0))
	}
}

func TestPriorityQueueGetValidatorRank(t *testing.T) {
	validators := ValidatorList{
		{Address: "validator1", Stake: 100},
		{Address: "validator2", Stake: 200},
		{Address: "validator3", Stake: 300},
	}

	hash := "0000000000000000000000000000000000000000000000000000000000000001"
	pq, _ := CalculateValidatorPriority(hash, validators)

	// Verifica que todos os validadores têm um rank válido
	for _, v := range validators {
		rank := pq.GetValidatorRank(v.Address)
		if rank < 0 || rank >= len(validators) {
			t.Errorf("Invalid rank %d for validator %s", rank, v.Address)
		}
	}

	// Verifica validador inexistente
	rank := pq.GetValidatorRank("nonexistent")
	if rank != -1 {
		t.Errorf("Expected rank -1 for nonexistent validator, got %d", rank)
	}
}

func TestPriorityQueueIsTopValidator(t *testing.T) {
	validators := ValidatorList{
		{Address: "validator1", Stake: 100},
		{Address: "validator2", Stake: 200},
	}

	hash := "0000000000000000000000000000000000000000000000000000000000000001"
	pq, _ := CalculateValidatorPriority(hash, validators)

	topAddr := pq.Validators[0].Address
	if !pq.IsTopValidator(topAddr) {
		t.Error("First validator should be top validator")
	}

	otherAddr := pq.Validators[1].Address
	if pq.IsTopValidator(otherAddr) {
		t.Error("Second validator should not be top validator")
	}
}

func TestWeightedRandomSelection(t *testing.T) {
	validators := ValidatorList{
		{Address: "validator1", Stake: 100},
		{Address: "validator2", Stake: 200},
		{Address: "validator3", Stake: 300},
	}

	hash := "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"

	index, err := WeightedRandomSelection(hash, validators)
	if err != nil {
		t.Fatalf("Failed to select validator: %v", err)
	}

	if index < 0 || index >= len(validators) {
		t.Errorf("Invalid index %d", index)
	}

	// Testa determinismo
	index2, _ := WeightedRandomSelection(hash, validators)
	if index != index2 {
		t.Error("Weighted selection is not deterministic")
	}
}

func TestWeightedRandomSelectionDistribution(t *testing.T) {
	validators := ValidatorList{
		{Address: "small", Stake: 10},
		{Address: "large", Stake: 90},
	}

	counts := make(map[string]int)
	counts["small"] = 0
	counts["large"] = 0

	iterations := 10000
	for i := 0; i < iterations; i++ {
		// Cria um hash diferente para cada iteração
		seed := []byte(fmt.Sprintf("test-%d", i))
		hash := sha256.Sum256(seed)
		hashStr := hex.EncodeToString(hash[:])

		index, err := WeightedRandomSelection(hashStr, validators)
		if err != nil {
			t.Fatalf("Failed to select validator: %v", err)
		}

		counts[validators[index].Address]++
	}

	smallRatio := float64(counts["small"]) / float64(iterations)
	largeRatio := float64(counts["large"]) / float64(iterations)

	t.Logf("Small validator (10%% stake): selected %.2f%% of the time", smallRatio*100)
	t.Logf("Large validator (90%% stake): selected %.2f%% of the time", largeRatio*100)

	// Verifica proporcionalidade
	expectedLargeRatio := 0.90
	tolerance := 0.05

	if math.Abs(largeRatio-expectedLargeRatio) > tolerance {
		t.Errorf("Large validator ratio %.2f is too far from expected %.2f",
			largeRatio, expectedLargeRatio)
	}
}

func TestValidatorSet(t *testing.T) {
	validators := ValidatorList{
		{Address: "validator1", Stake: 100},
		{Address: "validator2", Stake: 200},
	}

	vs, err := NewValidatorSet(validators)
	if err != nil {
		t.Fatalf("Failed to create validator set: %v", err)
	}

	if vs.TotalStake != 300 {
		t.Errorf("Expected total stake 300, got %d", vs.TotalStake)
	}

	if vs.Count() != 2 {
		t.Errorf("Expected 2 validators, got %d", vs.Count())
	}
}

func TestValidatorSetAddRemove(t *testing.T) {
	validators := ValidatorList{
		{Address: "validator1", Stake: 100},
	}

	vs, _ := NewValidatorSet(validators)

	// Adiciona validador
	err := vs.AddValidator(Validator{Address: "validator2", Stake: 200})
	if err != nil {
		t.Errorf("Failed to add validator: %v", err)
	}

	if vs.Count() != 2 {
		t.Errorf("Expected 2 validators after add, got %d", vs.Count())
	}

	if vs.TotalStake != 300 {
		t.Errorf("Expected total stake 300, got %d", vs.TotalStake)
	}

	// Tenta adicionar duplicado
	err = vs.AddValidator(Validator{Address: "validator2", Stake: 100})
	if err == nil {
		t.Error("Should not allow duplicate validator")
	}

	// Remove validador
	err = vs.RemoveValidator("validator2")
	if err != nil {
		t.Errorf("Failed to remove validator: %v", err)
	}

	if vs.Count() != 1 {
		t.Errorf("Expected 1 validator after remove, got %d", vs.Count())
	}

	if vs.TotalStake != 100 {
		t.Errorf("Expected total stake 100, got %d", vs.TotalStake)
	}

	// Tenta remover inexistente
	err = vs.RemoveValidator("nonexistent")
	if err == nil {
		t.Error("Should fail to remove nonexistent validator")
	}
}

func TestValidatorSetUpdateStake(t *testing.T) {
	validators := ValidatorList{
		{Address: "validator1", Stake: 100},
		{Address: "validator2", Stake: 200},
	}

	vs, _ := NewValidatorSet(validators)

	// Atualiza stake
	err := vs.UpdateStake("validator1", 150)
	if err != nil {
		t.Errorf("Failed to update stake: %v", err)
	}

	v := vs.Validators.GetValidator("validator1")
	if v.Stake != 150 {
		t.Errorf("Expected stake 150, got %d", v.Stake)
	}

	if vs.TotalStake != 350 {
		t.Errorf("Expected total stake 350, got %d", vs.TotalStake)
	}

	// Tenta atualizar para zero
	err = vs.UpdateStake("validator1", 0)
	if err == nil {
		t.Error("Should not allow zero stake")
	}

	// Tenta atualizar inexistente
	err = vs.UpdateStake("nonexistent", 100)
	if err == nil {
		t.Error("Should fail to update nonexistent validator")
	}
}

func TestGetExpectedProbability(t *testing.T) {
	validator := Validator{Address: "test", Stake: 300}
	totalStake := uint64(1000)

	prob := GetExpectedProbability(validator, totalStake)
	expected := 0.3

	if math.Abs(prob-expected) > 0.0001 {
		t.Errorf("Expected probability %.4f, got %.4f", expected, prob)
	}
}

func TestCalculateValidatorPriorityInvalidInput(t *testing.T) {
	validators := ValidatorList{
		{Address: "validator1", Stake: 100},
	}

	// Hash vazio
	_, err := CalculateValidatorPriority("", validators)
	if err == nil {
		t.Error("Should fail with empty hash")
	}

	// Lista vazia
	_, err = CalculateValidatorPriority("hash", ValidatorList{})
	if err == nil {
		t.Error("Should fail with empty validator list")
	}
}

func BenchmarkCalculateValidatorPriority(b *testing.B) {
	validators := ValidatorList{
		{Address: "validator1", Stake: 100},
		{Address: "validator2", Stake: 200},
		{Address: "validator3", Stake: 300},
		{Address: "validator4", Stake: 400},
		{Address: "validator5", Stake: 500},
	}

	hash := "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = CalculateValidatorPriority(hash, validators)
	}
}

func BenchmarkCalculateValidatorPriority100(b *testing.B) {
	validators := make(ValidatorList, 100)
	for i := 0; i < 100; i++ {
		validators[i] = Validator{
			Address: fmt.Sprintf("validator%d", i),
			Stake:   uint64((i + 1) * 100),
		}
	}

	hash := "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = CalculateValidatorPriority(hash, validators)
	}
}

func BenchmarkWeightedRandomSelection(b *testing.B) {
	validators := ValidatorList{
		{Address: "validator1", Stake: 100},
		{Address: "validator2", Stake: 200},
		{Address: "validator3", Stake: 300},
		{Address: "validator4", Stake: 400},
		{Address: "validator5", Stake: 500},
	}

	hash := "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = WeightedRandomSelection(hash, validators)
	}
}
