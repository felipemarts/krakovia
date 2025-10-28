package main

import (
	"fmt"

	"github.com/krakovia/blockchain/pkg/blockchain"
)

func main() {
	fmt.Println("=== Exemplo de Priorização de Validadores ===")

	// Cria lista de validadores com diferentes stakes
	validators := blockchain.ValidatorList{
		{Address: "validator_alice", Stake: 1000},  // 10% do stake total
		{Address: "validator_bob", Stake: 3000},    // 30% do stake total
		{Address: "validator_carol", Stake: 6000},  // 60% do stake total
	}

	fmt.Println("Validadores:")
	totalStake := validators.TotalStake()
	for _, v := range validators {
		prob := blockchain.GetExpectedProbability(v, totalStake)
		fmt.Printf("  %s: %d tokens (%.1f%% probabilidade esperada)\n",
			v.Address, v.Stake, prob*100)
	}
	fmt.Printf("\nStake Total: %d\n\n", totalStake)

	// Simula blocos diferentes (hashes diferentes)
	fmt.Println("Simulando 5 blocos diferentes:")
	fmt.Println(string([]byte{'-'}) + string([]byte{'-'}) + string([]byte{'-'}) +
		string([]byte{'-'}) + string([]byte{'-'}) + string([]byte{'-'}) +
		string([]byte{'-'}) + string([]byte{'-'}) + string([]byte{'-'}) +
		string([]byte{'-'}) + string([]byte{'-'}) + string([]byte{'-'}))

	// Hashes fictícios de blocos
	blockHashes := []string{
		"0000000000000000000000000000000000000000000000000000000000000001",
		"0000000000000000000000000000000000000000000000000000000000000002",
		"abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
		"fedcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321",
		"123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0",
	}

	for i, hash := range blockHashes {
		fmt.Printf("\nBloco %d (hash: %s...)\n", i+1, hash[:16])

		// Calcula prioridade dos validadores
		pq, err := blockchain.CalculateValidatorPriority(hash, validators)
		if err != nil {
			fmt.Printf("Erro: %v\n", err)
			continue
		}

		fmt.Println("  Ranking de prioridade:")
		for rank, validator := range pq.Validators {
			symbol := " "
			if rank == 0 {
				symbol = "★" // Validador selecionado
			}
			fmt.Printf("    %s %d. %s (stake: %d)\n",
				symbol, rank+1, validator.Address, validator.Stake)
		}

		// Mostra quem seria o validador selecionado
		top := pq.GetTopValidator()
		fmt.Printf("  → Validador selecionado: %s\n", top.Address)
	}

	// Demonstra determinismo
	fmt.Println("\n=== Demonstração de Determinismo ===")
	fmt.Println("Calculando prioridade 3 vezes com o mesmo hash:")

	testHash := "test_hash_for_determinism_000000000000000000000000000000000000"
	for i := 0; i < 3; i++ {
		pq, _ := blockchain.CalculateValidatorPriority(testHash, validators)
		top := pq.GetTopValidator()
		fmt.Printf("  Execução %d: %s é o top validator\n", i+1, top.Address)
	}

	// Simula distribuição de probabilidade
	fmt.Println("\n=== Simulação de Distribuição (10.000 blocos) ===")
	fmt.Println("Contando quantas vezes cada validador seria selecionado:")

	iterations := 10000
	distribution, err := blockchain.SimulateSelectionDistribution(validators, iterations)
	if err != nil {
		fmt.Printf("Erro: %v\n", err)
		return
	}

	fmt.Println("\nResultados:")
	for _, v := range validators {
		count := distribution[v.Address]
		actualProb := float64(count) / float64(iterations)
		expectedProb := blockchain.GetExpectedProbability(v, totalStake)

		fmt.Printf("  %s:\n", v.Address)
		fmt.Printf("    Selecionado: %d vezes (%.2f%%)\n", count, actualProb*100)
		fmt.Printf("    Esperado: %.2f%%\n", expectedProb*100)
		fmt.Printf("    Diferença: %.2f%%\n", (actualProb-expectedProb)*100)
	}

	// Exemplo com ValidatorSet
	fmt.Println("\n=== Exemplo com ValidatorSet ===")

	vs, _ := blockchain.NewValidatorSet(validators)
	fmt.Printf("Validator Set criado com %d validadores e %d tokens totais\n",
		vs.Count(), vs.TotalStake)

	// Adiciona um novo validador
	fmt.Println("\nAdicionando novo validador (Dave) com 4000 tokens...")
	err = vs.AddValidator(blockchain.Validator{
		Address: "validator_dave",
		Stake:   4000,
	})
	if err != nil {
		fmt.Printf("Erro: %v\n", err)
	} else {
		fmt.Printf("Sucesso! Agora temos %d validadores e %d tokens totais\n",
			vs.Count(), vs.TotalStake)
	}

	// Calcula nova prioridade
	newHash := "new_block_hash_00000000000000000000000000000000000000000000000"
	pq, _ := vs.CalculatePriority(newHash)

	fmt.Println("\nNovo ranking com Dave incluído:")
	for rank, validator := range pq.Validators {
		prob := blockchain.GetExpectedProbability(validator, vs.TotalStake)
		fmt.Printf("  %d. %s (stake: %d, prob: %.1f%%)\n",
			rank+1, validator.Address, validator.Stake, prob*100)
	}

	// Exemplo com seleção ponderada direta
	fmt.Println("\n=== Seleção Ponderada Direta ===")
	fmt.Println("Método alternativo para selecionar apenas um validador:")

	testHash2 := "weighted_selection_hash_0000000000000000000000000000000000000"
	index, err := blockchain.WeightedRandomSelection(testHash2, vs.Validators)
	if err != nil {
		fmt.Printf("Erro: %v\n", err)
	} else {
		selected := vs.Validators[index]
		fmt.Printf("Validador selecionado: %s (índice: %d)\n", selected.Address, index)
	}

	fmt.Println("\n=== Fim do Exemplo ===")
}
