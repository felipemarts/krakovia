package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/krakovia/blockchain/internal/config"
	"github.com/krakovia/blockchain/pkg/blockchain"
)

func main() {
	var (
		recipientAddr     string
		amount            uint64
		blockTime         int64
		maxBlockSize      int
		blockReward       uint64
		minValidatorStake uint64
		outputFile        string
		timestamp         int64
	)

	flag.StringVar(&recipientAddr, "recipient", "", "Recipient address for initial allocation (required)")
	flag.Uint64Var(&amount, "amount", 1000000000, "Initial token amount")
	flag.Int64Var(&blockTime, "block-time", 5000, "Time between blocks in milliseconds (min: 1000ms)")
	flag.IntVar(&maxBlockSize, "max-block-size", 1000, "Maximum transactions per block")
	flag.Uint64Var(&blockReward, "block-reward", 50, "Reward per block mined")
	flag.Uint64Var(&minValidatorStake, "min-stake", 1000, "Minimum stake to be a validator")
	flag.StringVar(&outputFile, "output", "", "Output file path (default: stdout)")
	flag.Int64Var(&timestamp, "timestamp", 0, "Genesis block timestamp (default: current time)")
	flag.Parse()

	if recipientAddr == "" {
		log.Fatal("Recipient address is required. Use -recipient flag")
	}

	if blockTime < 1000 {
		log.Fatal("Block time must be at least 1000ms (1 second)")
	}

	if amount == 0 {
		log.Fatal("Amount must be greater than 0")
	}

	// Usa timestamp atual se não fornecido
	if timestamp == 0 {
		timestamp = time.Now().Unix()
	}

	// Cria a transação coinbase do genesis
	genesisTx := blockchain.NewCoinbaseTransaction(recipientAddr, amount, 0)

	// Cria o bloco genesis
	genesisBlock := blockchain.GenesisBlock(genesisTx)

	// Cria a configuração do genesis
	genesisConfig := config.GenesisBlock{
		Timestamp:         timestamp,
		RecipientAddr:     recipientAddr,
		Amount:            amount,
		Hash:              genesisBlock.Hash,
		BlockTime:         blockTime,
		MaxBlockSize:      maxBlockSize,
		BlockReward:       blockReward,
		MinValidatorStake: minValidatorStake,
	}

	// Serializa para JSON
	output, err := json.MarshalIndent(genesisConfig, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal genesis config: %v", err)
	}

	// Exibe ou salva
	if outputFile != "" {
		err = os.WriteFile(outputFile, output, 0644)
		if err != nil {
			log.Fatalf("Failed to write output file: %v", err)
		}
		fmt.Printf("Genesis configuration written to %s\n", outputFile)
		fmt.Printf("\nGenesis Block Hash: %s\n", genesisBlock.Hash)
	} else {
		fmt.Println(string(output))
	}

	// Exibe resumo
	fmt.Printf("\n=== Genesis Block Configuration ===\n")
	fmt.Printf("Recipient Address: %s\n", recipientAddr)
	fmt.Printf("Initial Amount: %d tokens\n", amount)
	fmt.Printf("Block Time: %dms (%.1fs)\n", blockTime, float64(blockTime)/1000)
	fmt.Printf("Max Block Size: %d transactions\n", maxBlockSize)
	fmt.Printf("Block Reward: %d tokens\n", blockReward)
	fmt.Printf("Min Validator Stake: %d tokens\n", minValidatorStake)
	fmt.Printf("Timestamp: %d (%s)\n", timestamp, time.Unix(timestamp, 0).Format(time.RFC3339))
	fmt.Printf("Genesis Hash: %s\n", genesisBlock.Hash)
}
