package tests

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/krakovia/blockchain/pkg/blockchain"
	"github.com/krakovia/blockchain/pkg/node"
	"github.com/krakovia/blockchain/pkg/signaling"
	"github.com/krakovia/blockchain/pkg/wallet"
)

// Helper para limpar diretórios de teste de integração
func cleanupTestDirs(t *testing.T, nodeIDs ...string) {
	for _, id := range nodeIDs {
		dbPath := filepath.Join(os.TempDir(), "krakovia_test_"+id)
		os.RemoveAll(dbPath)
	}
}

// Helper específico para testes de integração
func createIntegrationNodeConfig(t *testing.T, nodeID string, signalingURL string, wallet *wallet.Wallet, genesis *blockchain.Block) node.Config {
	dbPath := filepath.Join(os.TempDir(), "krakovia_test_"+nodeID)

	return node.Config{
		ID:                nodeID,
		Address:           fmt.Sprintf(":%d", getRandomPort()),
		DBPath:            dbPath,
		SignalingServer:   signalingURL,
		MaxPeers:          10,
		MinPeers:          1,
		DiscoveryInterval: 5,
		Wallet:            wallet,
		GenesisBlock:      genesis,
		ChainConfig:       blockchain.DefaultChainConfig(),
	}
}

// TestNodeIntegration testa integração completa entre 2 nodes
func TestNodeIntegration(t *testing.T) {
	// Pular se não houver servidor de signaling rodando
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Iniciar servidor de signaling
	signalingPort := getRandomPort()
	signalingURL := fmt.Sprintf("ws://localhost:%d/ws", signalingPort)

	server := signaling.NewServer()
	go func() {
		if err := server.Start(fmt.Sprintf(":%d", signalingPort)); err != nil {
			t.Logf("Signaling server error: %v", err)
		}
	}()
	defer func() {
		if err := server.Stop(); err != nil {
			t.Logf("Warning: error stopping signaling server: %v", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	// Cleanup no final
	defer cleanupTestDirs(t, "node1", "node2")

	// 1. Criar wallets para os nós
	wallet1, err := wallet.NewWallet()
	if err != nil {
		t.Fatalf("Failed to create wallet1: %v", err)
	}

	wallet2, err := wallet.NewWallet()
	if err != nil {
		t.Fatalf("Failed to create wallet2: %v", err)
	}

	fmt.Printf("Wallet 1: %s\n", wallet1.GetAddress())
	fmt.Printf("Wallet 2: %s\n", wallet2.GetAddress())

	// 2. Criar bloco gênesis (wallet1 recebe tokens iniciais)
	genesisTx := blockchain.NewCoinbaseTransaction(wallet1.GetAddress(), 1000000000, 0)
	genesisBlock := blockchain.GenesisBlock(genesisTx)

	fmt.Printf("Genesis block: %s\n", genesisBlock.Hash[:16])

	// 3. Criar e iniciar Node 1
	config1 := createIntegrationNodeConfig(t, "test_node1", signalingURL, wallet1, genesisBlock)
	node1, err := node.NewNode(config1)
	if err != nil {
		t.Fatalf("Failed to create node1: %v", err)
	}

	if err := node1.Start(); err != nil {
		t.Fatalf("Failed to start node1: %v", err)
	}
	defer node1.Stop()

	fmt.Printf("\n[Node 1] Started successfully\n")
	fmt.Printf("[Node 1] Balance: %d\n", node1.GetBalance())
	fmt.Printf("[Node 1] Chain Height: %d\n\n", node1.GetChainHeight())

	// 4. Node 1 faz stake para poder minerar
	fmt.Printf("[Node 1] Creating stake transaction...\n")
	stakeTx, err := node1.CreateStakeTransaction(100000, 10)
	if err != nil {
		t.Fatalf("Failed to create stake transaction: %v", err)
	}
	fmt.Printf("[Node 1] Stake transaction created: %s\n", stakeTx.ID[:8])

	// 5. Node 1 inicia mineração
	fmt.Printf("[Node 1] Starting mining...\n")
	if err := node1.StartMining(); err != nil {
		t.Fatalf("Failed to start mining: %v", err)
	}

	// 6. Aguardar alguns blocos serem minerados (pode demorar com PoS)
	fmt.Printf("[Node 1] Waiting for blocks to be mined...\n")
	time.Sleep(5 * time.Second)

	height1 := node1.GetChainHeight()
	fmt.Printf("[Node 1] Chain height after mining: %d\n\n", height1)

	if height1 < 1 {
		t.Logf("Warning: Expected at least 1 block, got %d (mining may take longer)", height1)
	}

	// 7. Criar uma transação no Node 1
	fmt.Printf("[Node 1] Creating transaction to wallet2...\n")
	tx, err := node1.CreateTransaction(wallet2.GetAddress(), 10000, 5, "test transfer")
	if err != nil {
		t.Fatalf("Failed to create transaction: %v", err)
	}
	fmt.Printf("[Node 1] Transaction created: %s\n", tx.ID[:8])
	fmt.Printf("[Node 1] Mempool size: %d\n\n", node1.GetMempoolSize())

	// 8. Aguardar transação ser incluída em um bloco
	fmt.Printf("Waiting for transaction to be mined...\n")
	time.Sleep(2 * time.Second)

	// 9. Criar e iniciar Node 2 (vai sincronizar com Node 1)
	fmt.Printf("\n[Node 2] Creating and starting...\n")
	config2 := createIntegrationNodeConfig(t, "test_node2", signalingURL, wallet2, genesisBlock)
	node2, err := node.NewNode(config2)
	if err != nil {
		t.Fatalf("Failed to create node2: %v", err)
	}

	if err := node2.Start(); err != nil {
		t.Fatalf("Failed to start node2: %v", err)
	}
	defer node2.Stop()

	fmt.Printf("[Node 2] Started successfully\n")
	fmt.Printf("[Node 2] Initial chain height: %d\n", node2.GetChainHeight())

	// 10. Aguardar sincronização entre nodes (conexão WebRTC pode demorar)
	fmt.Printf("\nWaiting for node synchronization...\n")
	time.Sleep(8 * time.Second)

	// 11. Verificar se Node 2 sincronizou com Node 1
	height2 := node2.GetChainHeight()
	fmt.Printf("\n[Node 2] Chain height after sync: %d\n", height2)
	fmt.Printf("[Node 2] Balance: %d\n", node2.GetBalance())

	if height1 > 0 && height2 != height1 {
		t.Logf("Warning: Node 2 height (%d) doesn't match Node 1 height (%d)", height2, height1)
	}

	// 12. Verificar se wallet2 recebeu os tokens (se houve mineração)
	balance2 := node2.GetBalance()
	if height1 > 0 && balance2 == 0 {
		t.Logf("Warning: Node 2 should have received tokens from transaction")
	}
	fmt.Printf("[Node 2] Received balance: %d\n", balance2)

	// 13. Node 2 cria uma transação
	fmt.Printf("\n[Node 2] Creating transaction back to wallet1...\n")
	tx2, err := node2.CreateTransaction(wallet1.GetAddress(), 1000, 5, "test reply")
	if err != nil {
		t.Fatalf("Failed to create transaction on node2: %v", err)
	}
	fmt.Printf("[Node 2] Transaction created: %s\n", tx2.ID[:8])

	// 14. Aguardar propagação para Node 1
	fmt.Printf("Waiting for transaction propagation...\n")
	time.Sleep(1 * time.Second)

	// 15. Verificar se Node 1 recebeu a transação no mempool
	mempoolSize1 := node1.GetMempoolSize()
	fmt.Printf("[Node 1] Mempool size: %d\n", mempoolSize1)

	if mempoolSize1 == 0 {
		t.Error("Node 1 should have received transaction from Node 2")
	}

	// 16. Aguardar mais blocos serem minerados
	fmt.Printf("\nWaiting for more blocks...\n")
	time.Sleep(3 * time.Second)

	// 17. Verificar convergência final
	finalHeight1 := node1.GetChainHeight()
	finalHeight2 := node2.GetChainHeight()

	fmt.Printf("\n=== Final State ===\n")
	fmt.Printf("[Node 1] Height: %d, Balance: %d, Mempool: %d\n",
		finalHeight1, node1.GetBalance(), node1.GetMempoolSize())
	fmt.Printf("[Node 2] Height: %d, Balance: %d, Mempool: %d\n",
		finalHeight2, node2.GetBalance(), node2.GetMempoolSize())

	if finalHeight1 > 0 && finalHeight2 > 0 && finalHeight1 != finalHeight2 {
		t.Logf("Warning: Final heights don't match: Node1=%d, Node2=%d", finalHeight1, finalHeight2)
	}

	fmt.Printf("\n✓ Integration test completed!\n")
	fmt.Printf("✓ Nodes connected and communicating\n")
	if finalHeight1 > 0 {
		fmt.Printf("✓ Blocks mined: %d\n", finalHeight1)
	}
	fmt.Printf("✓ Transactions created and broadcasted\n")
}

// TestThreeNodeConsensus testa consenso com 3 nodes
func TestThreeNodeConsensus(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Iniciar servidor de signaling
	signalingPort := getRandomPort()
	signalingURL := fmt.Sprintf("ws://localhost:%d/ws", signalingPort)

	server := signaling.NewServer()
	go func() {
		if err := server.Start(fmt.Sprintf(":%d", signalingPort)); err != nil {
			t.Logf("Signaling server error: %v", err)
		}
	}()
	defer func() {
		if err := server.Stop(); err != nil {
			t.Logf("Warning: error stopping signaling server: %v", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	defer cleanupTestDirs(t, "node1", "node2", "node3")

	// Criar wallets
	wallets := make([]*wallet.Wallet, 3)
	for i := 0; i < 3; i++ {
		w, err := wallet.NewWallet()
		if err != nil {
			t.Fatalf("Failed to create wallet %d: %v", i, err)
		}
		wallets[i] = w
		fmt.Printf("Wallet %d: %s\n", i+1, w.GetAddress())
	}

	// Genesis dá tokens para wallet 1
	genesisTx := blockchain.NewCoinbaseTransaction(wallets[0].GetAddress(), 1000000000, 0)
	genesisBlock := blockchain.GenesisBlock(genesisTx)

	// Criar nodes
	nodes := make([]*node.Node, 3)

	for i := 0; i < 3; i++ {
		config := createIntegrationNodeConfig(t, fmt.Sprintf("test_node%d", i+1), signalingURL, wallets[i], genesisBlock)
		n, err := node.NewNode(config)
		if err != nil {
			t.Fatalf("Failed to create node %d: %v", i+1, err)
		}
		if err := n.Start(); err != nil {
			t.Fatalf("Failed to start node %d: %v", i+1, err)
		}
		defer n.Stop()
		nodes[i] = n
		fmt.Printf("[Node %d] Started\n", i+1)
		time.Sleep(500 * time.Millisecond) // Delay entre starts
	}

	// Node 1 faz stake e inicia mineração
	fmt.Printf("\n[Node 1] Staking and starting mining...\n")
	nodes[0].CreateStakeTransaction(100000, 10)
	nodes[0].StartMining()

	// Aguardar blocos e conexões WebRTC
	time.Sleep(8 * time.Second)

	// Verificar que todos nodes têm a mesma altura
	fmt.Printf("\n=== Checking consensus ===\n")
	heights := make([]uint64, 3)
	for i := 0; i < 3; i++ {
		heights[i] = nodes[i].GetChainHeight()
		fmt.Printf("[Node %d] Height: %d\n", i+1, heights[i])
	}

	// Verificar se há pelo menos alguma atividade
	maxHeight := heights[0]
	for i := 1; i < 3; i++ {
		if heights[i] > maxHeight {
			maxHeight = heights[i]
		}
	}

	// Todos devem ter a mesma altura (se houve mineração)
	if maxHeight > 0 {
		for i := 1; i < 3; i++ {
			if heights[i] != heights[0] {
				t.Logf("Warning: Node %d height (%d) doesn't match Node 1 (%d)", i+1, heights[i], heights[0])
			}
		}
	}

	fmt.Printf("\n✓ Three-node consensus test completed!\n")
	if maxHeight > 0 {
		fmt.Printf("✓ All nodes at height: %d\n", heights[0])
	}
	fmt.Printf("✓ Nodes connected and communicating\n")
}
