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
	"github.com/krakovia/blockchain/pkg/node"
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

	// Configurar nó
	nodeConfig := node.Config{
		ID:                cfg.ID,
		Address:           cfg.Address,
		DBPath:            cfg.DBPath,
		SignalingServer:   cfg.SignalingServer,
		MaxPeers:          cfg.MaxPeers,
		MinPeers:          cfg.MinPeers,
		DiscoveryInterval: cfg.DiscoveryInterval,
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
	fmt.Printf("=================================\n\n")

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
