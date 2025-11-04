package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// GenesisBlock representa a configuração do bloco gênesis
type GenesisBlock struct {
	Timestamp         int64  `json:"timestamp"`           // Timestamp do bloco gênesis
	RecipientAddr     string `json:"recipient_addr"`      // Endereço que receberá a recompensa inicial
	Amount            uint64 `json:"amount"`              // Quantidade de tokens iniciais
	InitialStake      uint64 `json:"initial_stake"`       // Stake inicial do recipient (0 = sem stake inicial)
	Hash              string `json:"hash"`                // Hash esperado do bloco gênesis
	BlockTime         int64  `json:"block_time"`          // Tempo entre blocos em milissegundos
	MaxBlockSize      int    `json:"max_block_size"`      // Máximo de transações por bloco
	BlockReward       uint64 `json:"block_reward"`        // Recompensa por bloco minerado
	MinValidatorStake uint64 `json:"min_validator_stake"` // Stake mínimo para ser validador
}

// WalletConfig representa as chaves da carteira do nó
type WalletConfig struct {
	PrivateKey string `json:"private_key"` // Chave privada ECDSA em formato hexadecimal
	PublicKey  string `json:"public_key"`  // Chave pública ECDSA em formato hexadecimal
	Address    string `json:"address"`     // Endereço derivado da chave pública
}

// CheckpointConfig representa a configuração do sistema de checkpoints
type CheckpointConfig struct {
	Enabled       bool `json:"enabled"`          // Habilita o sistema de checkpoints
	Interval      int  `json:"interval"`         // Checkpoint a cada X blocos
	KeepInMemory  int  `json:"keep_in_memory"`   // Manter últimos X blocos em memória
	KeepOnDisk    int  `json:"keep_on_disk"`     // Manter últimos X checkpoints no disco
	CSVDelimiter  string `json:"csv_delimiter"`  // Delimitador do CSV (padrão: ",")
	Compression   bool `json:"compression"`      // Comprimir CSV no LevelDB
}

// APIConfig representa a configuração do servidor HTTP da API
type APIConfig struct {
	Enabled  bool   `json:"enabled"`  // Habilita/desabilita a API HTTP
	Address  string `json:"address"`  // Endereço do servidor (ex: :8080)
	Username string `json:"username"` // Usuário para autenticação
	Password string `json:"password"` // Senha para autenticação
}

// NodeConfig representa a configuração de um nó
type NodeConfig struct {
	ID                string            `json:"id"`
	Address           string            `json:"address"`
	DBPath            string            `json:"db_path"`
	SignalingServer   string            `json:"signaling_server"`
	MaxPeers          int               `json:"max_peers"`          // Máximo de peers conectados (0 = ilimitado)
	MinPeers          int               `json:"min_peers"`          // Mínimo de peers desejado
	DiscoveryInterval int               `json:"discovery_interval"` // Intervalo de descoberta em segundos
	Wallet            WalletConfig      `json:"wallet"`             // Configuração da carteira
	Genesis           *GenesisBlock     `json:"genesis,omitempty"`  // Configuração do bloco gênesis (opcional)
	Checkpoint        *CheckpointConfig `json:"checkpoint,omitempty"` // Configuração de checkpoints (opcional)
	API               *APIConfig        `json:"api,omitempty"`      // Configuração da API HTTP (opcional)
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

		// Valores padrão para configurações da chain
		if config.Genesis.BlockTime == 0 {
			config.Genesis.BlockTime = 5000 // Padrão: 5 segundos (5000ms)
		}
		if config.Genesis.MaxBlockSize == 0 {
			config.Genesis.MaxBlockSize = 1000
		}
		if config.Genesis.BlockReward == 0 {
			config.Genesis.BlockReward = 50
		}
		if config.Genesis.MinValidatorStake == 0 {
			config.Genesis.MinValidatorStake = 1000
		}

		// Validações
		if config.Genesis.BlockTime < 1000 {
			return nil, fmt.Errorf("block time must be at least 1000ms (1 second)")
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

	// Configuração de checkpoint (valores padrão se não fornecido)
	if config.Checkpoint != nil {
		if config.Checkpoint.Enabled {
			// Valores padrão
			if config.Checkpoint.Interval == 0 {
				config.Checkpoint.Interval = 100 // Padrão: checkpoint a cada 100 blocos
			}
			if config.Checkpoint.KeepInMemory == 0 {
				config.Checkpoint.KeepInMemory = 200 // Padrão: manter últimos 200 blocos em memória
			}
			if config.Checkpoint.KeepOnDisk == 0 {
				config.Checkpoint.KeepOnDisk = 2 // Padrão: manter últimos 2 checkpoints no disco
			}
			if config.Checkpoint.CSVDelimiter == "" {
				config.Checkpoint.CSVDelimiter = "," // Padrão: vírgula
			}

			// Validações
			if config.Checkpoint.Interval < 1 {
				return nil, fmt.Errorf("checkpoint interval must be at least 1")
			}
			if config.Checkpoint.KeepInMemory < config.Checkpoint.Interval {
				return nil, fmt.Errorf("keep_in_memory (%d) must be at least equal to interval (%d)", config.Checkpoint.KeepInMemory, config.Checkpoint.Interval)
			}
			if config.Checkpoint.KeepOnDisk < 1 {
				return nil, fmt.Errorf("keep_on_disk must be at least 1")
			}
		}
	}

	// Configuração da API (valores padrão e validações)
	if config.API != nil {
		if config.API.Enabled {
			// Valores padrão
			if config.API.Address == "" {
				config.API.Address = ":8080" // Padrão: porta 8080
			}
			if config.API.Username == "" {
				return nil, fmt.Errorf("API username is required when API is enabled")
			}
			if config.API.Password == "" {
				return nil, fmt.Errorf("API password is required when API is enabled")
			}
		}
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
