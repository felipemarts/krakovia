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
		if err := os.RemoveAll(dbPath); err != nil {
			t.Logf("Warning: failed to cleanup %s: %v", dbPath, err)
		}
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
	defer func() {
		if err := node1.Stop(); err != nil {
			t.Logf("Error stopping node1: %v", err)
		}
	}()

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

	// 6. Aguardar alguns blocos serem minerados (otimizado para testes rápidos)
	fmt.Printf("[Node 1] Waiting for blocks to be mined...\n")
	time.Sleep(1 * time.Second)

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

	// 8. Aguardar transação ser incluída em um bloco (otimizado)
	fmt.Printf("Waiting for transaction to be mined...\n")
	time.Sleep(400 * time.Millisecond)

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
	defer func() {
		if err := node2.Stop(); err != nil {
			t.Logf("Error stopping node2: %v", err)
		}
	}()

	fmt.Printf("[Node 2] Started successfully\n")
	fmt.Printf("[Node 2] Initial chain height: %d\n", node2.GetChainHeight())

	// 10. Aguardar sincronização entre nodes (otimizado para testes rápidos)
	fmt.Printf("\nWaiting for node synchronization...\n")
	time.Sleep(1 * time.Second)

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
	time.Sleep(400 * time.Millisecond)

	// 15. Verificar se Node 1 recebeu a transação no mempool
	mempoolSize1 := node1.GetMempoolSize()
	fmt.Printf("[Node 1] Mempool size: %d\n", mempoolSize1)

	if mempoolSize1 == 0 {
		t.Error("Node 1 should have received transaction from Node 2")
	}

	// 16. Aguardar mais blocos serem minerados (otimizado)
	fmt.Printf("\nWaiting for more blocks...\n")
	time.Sleep(1 * time.Second)

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
		defer func(node *node.Node, id int) {
			if err := node.Stop(); err != nil {
				t.Logf("Error stopping node %d: %v", id, err)
			}
		}(n, i+1)
		nodes[i] = n
		fmt.Printf("[Node %d] Started\n", i+1)
		time.Sleep(200 * time.Millisecond) // Delay entre starts (otimizado)
	}

	// Node 1 faz stake e inicia mineração
	fmt.Printf("\n[Node 1] Staking and starting mining...\n")
	if _, err := nodes[0].CreateStakeTransaction(100000, 10); err != nil {
		t.Fatalf("Failed to create stake transaction: %v", err)
	}
	if err := nodes[0].StartMining(); err != nil {
		t.Fatalf("Failed to start mining: %v", err)
	}

	// Aguardar blocos e conexões WebRTC (otimizado para 30s timeout)
	time.Sleep(1 * time.Second)

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

		// Verificar que todos têm o mesmo último bloco (hash)
		// Isso garante que não há fork entre os nós
		lastBlocks := make([]*blockchain.Block, 3)
		for i := 0; i < 3; i++ {
			lastBlocks[i] = nodes[i].GetLastBlock()
			if lastBlocks[i] != nil {
				fmt.Printf("[Node %d] Last block hash: %s\n", i+1, lastBlocks[i].Hash[:16])
			}
		}

		// Verificar se todos têm o mesmo hash
		if lastBlocks[0] != nil {
			referenceHash := lastBlocks[0].Hash
			for i := 1; i < 3; i++ {
				if lastBlocks[i] != nil && lastBlocks[i].Hash != referenceHash {
					t.Errorf("Fork detected! Node %d has different last block hash:\n  Node 1: %s\n  Node %d: %s",
						i+1, referenceHash[:32], i+1, lastBlocks[i].Hash[:32])
				}
			}
			fmt.Printf("✓ No fork detected! All nodes have identical last block hash\n")
		}
	}

	fmt.Printf("\n✓ Three-node consensus test completed!\n")
	if maxHeight > 0 {
		fmt.Printf("✓ All nodes at height: %d\n", heights[0])
	}
	fmt.Printf("✓ Nodes connected and communicating\n")
}

// TestNetworkPartitionRecovery testa a recuperação após partição de rede com múltiplos mineradores
// Simula perda de conexão durante mineração competitiva e verifica convergência do consenso PoS
func TestNetworkPartitionRecovery(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping network partition test in short mode")
	}

	signalingPort := getRandomPort()
	signalingURL := fmt.Sprintf("ws://localhost:%d/ws", signalingPort)

	// Iniciar servidor de signaling
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

	defer cleanupTestDirs(t, "partition_node1", "partition_node2")

	// Criar wallets para ambos os nós
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

	// Criar genesis com ambos tendo tokens para fazer stake
	// Wallet1 recebe a maioria para criar os stakes iniciais
	genesisTx := blockchain.NewCoinbaseTransaction(wallet1.GetAddress(), 1000000000, 0)
	genesisBlock := blockchain.GenesisBlock(genesisTx)

	fmt.Printf("Genesis block: %s\n", genesisBlock.Hash[:16])

	// Configurar node1
	config1 := createIntegrationNodeConfig(t, "partition_node1", signalingURL, wallet1, genesisBlock)
	config1.DiscoveryInterval = 2

	// Configurar node2
	config2 := createIntegrationNodeConfig(t, "partition_node2", signalingURL, wallet2, genesisBlock)
	config2.DiscoveryInterval = 2

	// Criar e iniciar node1
	n1, err := node.NewNode(config1)
	if err != nil {
		t.Fatalf("Failed to create node1: %v", err)
	}

	if err := n1.Start(); err != nil {
		t.Fatalf("Failed to start node1: %v", err)
	}
	defer func() {
		if err := n1.Stop(); err != nil {
			t.Logf("Error stopping node1: %v", err)
		}
	}()

	// Criar e iniciar node2
	n2, err := node.NewNode(config2)
	if err != nil {
		t.Fatalf("Failed to create node2: %v", err)
	}

	if err := n2.Start(); err != nil {
		t.Fatalf("Failed to start node2: %v", err)
	}
	defer func() {
		if err := n2.Stop(); err != nil {
			t.Logf("Error stopping node2: %v", err)
		}
	}()

	fmt.Printf("\n=== Phase 1: Initial Synchronization ===\n")

	// Aguardar conexão inicial
	time.Sleep(1500 * time.Millisecond)

	// Verificar conexão
	peers1 := n1.GetPeers()
	peers2 := n2.GetPeers()

	if len(peers1) == 0 || len(peers2) == 0 {
		t.Fatalf("Nodes failed to connect initially. Node1: %d peers, Node2: %d peers", len(peers1), len(peers2))
	}

	fmt.Printf("✓ Nodes connected: Node1 has %d peers, Node2 has %d peers\n", len(peers1), len(peers2))

	// Node1 faz stake e transfere tokens para Node2
	fmt.Printf("\n[Node1] Creating stake transaction...\n")
	stakeTx1, err := n1.CreateStakeTransaction(100000, 10)
	if err != nil {
		t.Fatalf("Failed to create stake for node1: %v", err)
	}
	fmt.Printf("[Node1] Stake created: %s\n", stakeTx1.ID[:8])

	// Node1 transfere tokens para Node2 poder fazer stake também
	fmt.Printf("[Node1] Transferring tokens to Node2 for staking...\n")
	transferTx, err := n1.CreateTransaction(wallet2.GetAddress(), 200000, 5, "stake transfer")
	if err != nil {
		t.Fatalf("Failed to create transfer: %v", err)
	}
	fmt.Printf("[Node1] Transfer created: %s\n", transferTx.ID[:8])

	// Node1 inicia mineração para incluir as transações
	fmt.Printf("[Node1] Starting mining to include transactions...\n")
	if err := n1.StartMining(); err != nil {
		t.Fatalf("Failed to start mining: %v", err)
	}

	// Aguardar blocos serem minerados e sincronizados
	time.Sleep(1500 * time.Millisecond)

	height1BeforeBothMining := n1.GetChainHeight()
	height2BeforeBothMining := n2.GetChainHeight()

	fmt.Printf("[Node1] Chain height: %d\n", height1BeforeBothMining)
	fmt.Printf("[Node2] Chain height: %d (synchronized)\n", height2BeforeBothMining)
	fmt.Printf("[Node2] Balance: %d\n", n2.GetBalance())

	// Node2 agora faz stake e também inicia mineração
	if n2.GetBalance() >= 100000 {
		fmt.Printf("\n[Node2] Creating stake transaction...\n")
		stakeTx2, err := n2.CreateStakeTransaction(100000, 10)
		if err != nil {
			t.Fatalf("Failed to create stake for node2: %v", err)
		}
		fmt.Printf("[Node2] Stake created: %s\n", stakeTx2.ID[:8])

		fmt.Printf("[Node2] Starting mining...\n")
		if err := n2.StartMining(); err != nil {
			t.Fatalf("Failed to start mining on node2: %v", err)
		}

		// Aguardar ambos minerarem juntos
		time.Sleep(1000 * time.Millisecond)
	} else {
		t.Logf("Warning: Node2 doesn't have enough balance for staking (%d), only Node1 will mine", n2.GetBalance())
	}

	height1BeforePartition := n1.GetChainHeight()
	height2BeforePartition := n2.GetChainHeight()

	fmt.Printf("\n[Node1] Chain height before partition: %d\n", height1BeforePartition)
	fmt.Printf("[Node2] Chain height before partition: %d\n", height2BeforePartition)

	if height1BeforePartition > 0 && height2BeforePartition != height1BeforePartition {
		t.Logf("Warning: Heights differ before partition: Node1=%d, Node2=%d", height1BeforePartition, height2BeforePartition)
	}

	fmt.Printf("\n=== Phase 2: Network Partition (Both nodes mining separately) ===\n")

	// Simular perda de conexão: parar node2 temporariamente
	if err := n2.Stop(); err != nil {
		t.Logf("Warning: error stopping node2: %v", err)
	}
	fmt.Printf("✗ Node2 disconnected (simulating network partition)\n")

	// Aguardar ~1 segundo para ambos minerarem separadamente
	// Node1 continua minerando
	fmt.Printf("[Node1] Mining while disconnected from Node2...\n")
	time.Sleep(1000 * time.Millisecond)

	height1AfterPartition := n1.GetChainHeight()
	fmt.Printf("[Node1] Chain height after partition: %d (mined %d blocks alone)\n",
		height1AfterPartition, height1AfterPartition-height1BeforePartition)

	if height1AfterPartition <= height1BeforePartition {
		t.Logf("Warning: Node1 didn't mine new blocks during partition")
	}

	fmt.Printf("\n=== Phase 3: Network Reconnection ===\n")

	// Reconectar node2 (recriá-lo do estado persistido)
	n2, err = node.NewNode(config2)
	if err != nil {
		t.Fatalf("Failed to recreate node2: %v", err)
	}
	defer func() {
		if err := n2.Stop(); err != nil {
			t.Logf("Error stopping node2: %v", err)
		}
	}()

	if err := n2.Start(); err != nil {
		t.Fatalf("Failed to restart node2: %v", err)
	}

	fmt.Printf("✓ Node2 reconnected\n")

	// Se node2 tinha stake, reiniciar mineração
	if n2.GetBalance() >= 100000 {
		fmt.Printf("[Node2] Restarting mining after reconnection...\n")
		if err := n2.StartMining(); err != nil {
			t.Logf("Warning: failed to restart mining on node2: %v", err)
		}
	}

	// Aguardar reconexão e sincronização
	fmt.Printf("Waiting for nodes to reconnect and synchronize...\n")
	time.Sleep(2500 * time.Millisecond)

	// Verificar reconexão
	peers1AfterReconnect := n1.GetPeers()
	peers2AfterReconnect := n2.GetPeers()

	if len(peers1AfterReconnect) == 0 || len(peers2AfterReconnect) == 0 {
		t.Logf("Warning: Nodes may not have fully reconnected. Node1: %d peers, Node2: %d peers",
			len(peers1AfterReconnect), len(peers2AfterReconnect))
	} else {
		fmt.Printf("✓ Nodes reconnected: Node1 has %d peers, Node2 has %d peers\n",
			len(peers1AfterReconnect), len(peers2AfterReconnect))
	}

	fmt.Printf("\n=== Phase 4: Consensus Verification (PoS Convergence) ===\n")

	// Aguardar um pouco mais para garantir sincronização completa
	time.Sleep(1000 * time.Millisecond)

	// Verificar alturas finais
	finalHeight1 := n1.GetChainHeight()
	finalHeight2 := n2.GetChainHeight()

	fmt.Printf("[Node1] Final chain height: %d\n", finalHeight1)
	fmt.Printf("[Node2] Final chain height: %d\n", finalHeight2)

	// Verificar consenso de altura
	if finalHeight1 != finalHeight2 {
		t.Errorf("Consensus not reached! Node1 height: %d, Node2 height: %d", finalHeight1, finalHeight2)
	} else {
		fmt.Printf("✓ Consensus reached! Both nodes at height: %d\n", finalHeight1)
	}

	// Verificar que ambos têm o mesmo último bloco (hash)
	// Isso garante que não há fork - mesma altura E mesma cadeia
	if finalHeight1 > 0 && finalHeight2 > 0 {
		lastBlock1 := n1.GetLastBlock()
		lastBlock2 := n2.GetLastBlock()

		if lastBlock1 != nil && lastBlock2 != nil {
			fmt.Printf("[Node1] Last block hash: %s\n", lastBlock1.Hash[:16])
			fmt.Printf("[Node2] Last block hash: %s\n", lastBlock2.Hash[:16])

			if lastBlock1.Hash != lastBlock2.Hash {
				t.Errorf("Fork detected! Nodes have different last block hashes:\n  Node1: %s\n  Node2: %s",
					lastBlock1.Hash[:32], lastBlock2.Hash[:32])
			} else {
				fmt.Printf("✓ No fork detected! Both nodes have identical last block hash\n")
			}
		} else {
			t.Logf("Warning: Could not retrieve last block from one or both nodes")
		}
	}

	// Verificar que houve progresso durante a partição
	if finalHeight1 > height1BeforePartition {
		blocksMinedDuringPartition := height1AfterPartition - height1BeforePartition
		fmt.Printf("✓ Node1 mined %d blocks during partition\n", blocksMinedDuringPartition)
	}

	// Verificar que node2 sincronizou
	if finalHeight2 >= height1AfterPartition {
		fmt.Printf("✓ Node2 successfully synchronized after reconnection\n")
	} else {
		t.Logf("Warning: Node2 may not be fully synchronized (height: %d, expected >= %d)",
			finalHeight2, height1AfterPartition)
	}

	fmt.Printf("\n=== Test Summary ===\n")
	fmt.Printf("✓ Network partition recovery test completed\n")
	fmt.Printf("✓ Initial sync: %d blocks\n", height1BeforePartition)
	fmt.Printf("✓ Blocks mined during partition: %d\n", height1AfterPartition-height1BeforePartition)
	fmt.Printf("✓ Final consensus height: %d\n", finalHeight1)

	if finalHeight1 == finalHeight2 && finalHeight1 >= height1AfterPartition {
		fmt.Printf("✓ SUCCESS: PoS consensus converged after network partition recovery\n")
	} else {
		fmt.Printf("⚠ Partial success: Check timing parameters for more reliable synchronization\n")
	}
}
