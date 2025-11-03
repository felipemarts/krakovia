package tests

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/krakovia/blockchain/pkg/blockchain"
	"github.com/krakovia/blockchain/pkg/node"
	"github.com/krakovia/blockchain/pkg/wallet"
)

// getRandomPort retorna uma porta aleatória no intervalo 9000-29000
func getRandomPort() int {
	return 9000 + rand.Intn(20000)
}

// stopNode para o nó de forma segura, logando se houver erro
func stopNode(n *node.Node, t *testing.T) {
	if err := n.Stop(); err != nil {
		t.Logf("Warning: error stopping node: %v", err)
	}
}

// getTempDataDir cria um diretório temporário único para o teste
func getTempDataDir(t *testing.T, testName string) string {
	tempDir := filepath.Join(os.TempDir(), fmt.Sprintf("krakovia-test-%s-%d", testName, time.Now().UnixNano()))
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Warning: failed to remove temp dir: %v", err)
		}
	})
	return tempDir
}

// createTestWallet cria uma wallet de teste
func createTestWallet(t *testing.T) *wallet.Wallet {
	w, err := wallet.NewWallet()
	if err != nil {
		t.Fatalf("Failed to create test wallet: %v", err)
	}
	return w
}

// createTestGenesis cria um bloco genesis de teste
func createTestGenesis(recipientAddr string, amount uint64) *blockchain.Block {
	genesisTx := blockchain.NewCoinbaseTransaction(recipientAddr, amount, 0)
	return blockchain.GenesisBlock(genesisTx)
}

// createTestNodeConfig cria uma configuração de node com wallet e genesis
func createTestNodeConfig(t *testing.T, nodeID, signalingURL, tempDir string) node.Config {
	w := createTestWallet(t)
	genesis := createTestGenesis(w.GetAddress(), 1000000000)

	return node.Config{
		ID:                nodeID,
		Address:           fmt.Sprintf(":%d", getRandomPort()),
		DBPath:            filepath.Join(tempDir, nodeID),
		SignalingServer:   signalingURL,
		MaxPeers:          10,
		MinPeers:          1,
		DiscoveryInterval: 60,
		Wallet:            w,
		GenesisBlock:      genesis,
		ChainConfig:       blockchain.DefaultChainConfig(),
	}
}

// createTestNodeConfigWithSharedGenesis cria config com genesis compartilhado
func createTestNodeConfigWithSharedGenesis(t *testing.T, nodeID, signalingURL, tempDir string, genesis *blockchain.Block) node.Config {
	w := createTestWallet(t)

	return node.Config{
		ID:                nodeID,
		Address:           fmt.Sprintf(":%d", getRandomPort()),
		DBPath:            filepath.Join(tempDir, nodeID),
		SignalingServer:   signalingURL,
		MaxPeers:          10,
		MinPeers:          1,
		DiscoveryInterval: 60,
		Wallet:            w,
		GenesisBlock:      genesis,
		ChainConfig:       blockchain.DefaultChainConfig(),
	}
}

// waitForCondition aguarda até que uma condição seja satisfeita ou timeout
func waitForCondition(condition func() bool, timeout time.Duration, checkInterval time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return true
		}
		time.Sleep(checkInterval)
	}
	return false
}
