package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// GenesisBlock representa a configuração do bloco gênesis
type GenesisBlock struct {
	Timestamp     int64  `json:"timestamp"`      // Timestamp do bloco gênesis
	RecipientAddr string `json:"recipient_addr"` // Endereço que receberá a recompensa inicial
	Amount        uint64 `json:"amount"`         // Quantidade de tokens iniciais
	Hash          string `json:"hash"`           // Hash esperado do bloco gênesis
}

// WalletConfig representa as chaves da carteira do nó
type WalletConfig struct {
	PrivateKey string `json:"private_key"` // Chave privada ECDSA em formato hexadecimal
	PublicKey  string `json:"public_key"`  // Chave pública ECDSA em formato hexadecimal
	Address    string `json:"address"`     // Endereço derivado da chave pública
}

// NodeConfig representa a configuração de um nó
type NodeConfig struct {
	ID                string        `json:"id"`
	Address           string        `json:"address"`
	DBPath            string        `json:"db_path"`
	SignalingServer   string        `json:"signaling_server"`
	MaxPeers          int           `json:"max_peers"`          // Máximo de peers conectados (0 = ilimitado)
	MinPeers          int           `json:"min_peers"`          // Mínimo de peers desejado
	DiscoveryInterval int           `json:"discovery_interval"` // Intervalo de descoberta em segundos
	Wallet            WalletConfig  `json:"wallet"`             // Configuração da carteira
	Genesis           *GenesisBlock `json:"genesis,omitempty"`  // Configuração do bloco gênesis (opcional)
}

// LoadNodeConfig carrega a configuração de um arquivo JSON
func LoadNodeConfig(filepath string) (*NodeConfig, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config NodeConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validações básicas
	if config.ID == "" {
		return nil, fmt.Errorf("node ID is required")
	}
	if config.Address == "" {
		return nil, fmt.Errorf("node address is required")
	}
	if config.DBPath == "" {
		return nil, fmt.Errorf("database path is required")
	}
	if config.SignalingServer == "" {
		return nil, fmt.Errorf("signaling server address is required")
	}

	// Validações da carteira
	if config.Wallet.PrivateKey == "" {
		return nil, fmt.Errorf("wallet private key is required")
	}
	if config.Wallet.PublicKey == "" {
		return nil, fmt.Errorf("wallet public key is required")
	}
	if config.Wallet.Address == "" {
		return nil, fmt.Errorf("wallet address is required")
	}

	// Validações do bloco gênesis (se fornecido)
	if config.Genesis != nil {
		if config.Genesis.RecipientAddr == "" {
			return nil, fmt.Errorf("genesis recipient address is required")
		}
		if config.Genesis.Amount == 0 {
			return nil, fmt.Errorf("genesis amount must be greater than 0")
		}
		if config.Genesis.Hash == "" {
			return nil, fmt.Errorf("genesis hash is required")
		}
	}

	// Valores padrão
	if config.MaxPeers == 0 {
		config.MaxPeers = 50 // Padrão: 50 peers
	}
	if config.MinPeers == 0 {
		config.MinPeers = 5 // Padrão: 5 peers mínimo
	}
	if config.DiscoveryInterval == 0 {
		config.DiscoveryInterval = 30 // Padrão: 30 segundos
	}

	// Validar limites
	if config.MinPeers > config.MaxPeers {
		return nil, fmt.Errorf("min_peers (%d) cannot be greater than max_peers (%d)", config.MinPeers, config.MaxPeers)
	}

	return &config, nil
}

// SaveNodeConfig salva a configuração em um arquivo JSON
func SaveNodeConfig(filepath string, config *NodeConfig) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(filepath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
