package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// NodeConfig representa a configuração de um nó
type NodeConfig struct {
	ID              string `json:"id"`
	Address         string `json:"address"`
	DBPath          string `json:"db_path"`
	SignalingServer string `json:"signaling_server"`
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
