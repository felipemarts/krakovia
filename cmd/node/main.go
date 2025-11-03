package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/krakovia/blockchain/internal/config"
	"github.com/krakovia/blockchain/pkg/blockchain"
	"github.com/krakovia/blockchain/pkg/node"
	"github.com/krakovia/blockchain/pkg/wallet"
)

func main() {
	configPath := flag.String("config", "", "Path to JSON config file (required)")
	flag.Parse()

	if *configPath == "" {
		log.Fatal("Config file is required. Use -config flag")
	}

	// Carregar configuração do arquivo JSON
	cfg, err := config.LoadNodeConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Criar diretório de dados se não existir
	dbDir := filepath.Dir(cfg.DBPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		log.Fatal("Failed to create data directory:", err)
	}

	// Carregar ou criar wallet a partir da configuração
	w, err := wallet.NewWalletFromPrivateKey(cfg.Wallet.PrivateKey)
	if err != nil {
		log.Fatalf("Failed to load wallet: %v", err)
	}

	// Verificar se a wallet corresponde à configuração
	if w.GetAddress() != cfg.Wallet.Address {
		log.Fatal("Wallet address mismatch! Check your configuration file.")
	}

	fmt.Printf("Wallet loaded: %s\n", w.GetAddress())

	// Criar bloco gênesis
	var genesisBlock *blockchain.Block
	if cfg.Genesis != nil {
		// Criar transação coinbase para o genesis
		genesisTx := blockchain.NewCoinbaseTransaction(
			cfg.Genesis.RecipientAddr,
			cfg.Genesis.Amount,
			0, // block height 0
		)

		// Criar genesis block
		genesisBlock = blockchain.GenesisBlock(genesisTx)

		fmt.Printf("Genesis block created: %s\n", genesisBlock.Hash[:16])
		fmt.Printf("Genesis recipient: %s\n", cfg.Genesis.RecipientAddr)
		fmt.Printf("Genesis amount: %d\n", cfg.Genesis.Amount)
	} else {
		// Criar genesis padrão se não fornecido
		genesisTx := blockchain.NewCoinbaseTransaction(
			w.GetAddress(),
			1000000000, // 1 bilhão de tokens iniciais
			0,
		)
		genesisBlock = blockchain.GenesisBlock(genesisTx)
		fmt.Printf("Default genesis block created: %s\n", genesisBlock.Hash[:16])
	}

	// Configuração da blockchain
	chainConfig := blockchain.DefaultChainConfig()

	// Configurar nó
	nodeConfig := node.Config{
		ID:                cfg.ID,
		Address:           cfg.Address,
		DBPath:            cfg.DBPath,
		SignalingServer:   cfg.SignalingServer,
		MaxPeers:          cfg.MaxPeers,
		MinPeers:          cfg.MinPeers,
		DiscoveryInterval: cfg.DiscoveryInterval,
		Wallet:            w,
		GenesisBlock:      genesisBlock,
		ChainConfig:       chainConfig,
	}

	// Criar nó
	n, err := node.NewNode(nodeConfig)
	if err != nil {
		log.Fatal("Failed to create node:", err)
	}

	// Iniciar nó
	if err := n.Start(); err != nil {
		log.Fatal("Failed to start node:", err)
	}

	fmt.Printf("\n=================================\n")
	fmt.Printf("Node %s started successfully!\n", cfg.ID)
	fmt.Printf("Address: %s\n", cfg.Address)
	fmt.Printf("Database: %s\n", cfg.DBPath)
	fmt.Printf("Signaling: %s\n", cfg.SignalingServer)
	fmt.Printf("=================================\n")

	// Mostrar informações da blockchain
	fmt.Printf("\n--- Blockchain Info ---\n")
	fmt.Printf("Wallet Address: %s\n", w.GetAddress())
	fmt.Printf("Initial Balance: %d\n", n.GetBalance())
	fmt.Printf("Chain Height: %d\n", n.GetChainHeight())
	fmt.Printf("Genesis Hash: %s\n", genesisBlock.Hash[:16]+"...")
	fmt.Printf("=======================\n\n")

	// Aguardar sinal de interrupção
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\nShutting down node...")
	if err := n.Stop(); err != nil {
		log.Fatal("Failed to stop node:", err)
	}

	fmt.Println("Node stopped successfully")
}
