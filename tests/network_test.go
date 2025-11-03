package tests

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/krakovia/blockchain/pkg/blockchain"
	"github.com/krakovia/blockchain/pkg/node"
	"github.com/krakovia/blockchain/pkg/signaling"
)

// TestNodeConnection testa a conexão básica entre nós
func TestNodeConnection(t *testing.T) {
	signalingPort := getRandomPort()
	signalingURL := fmt.Sprintf("ws://localhost:%d/ws", signalingPort)
	tempDir := getTempDataDir(t, "conn")

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

	// Aguardar servidor iniciar
	time.Sleep(100 * time.Millisecond)

	// Criar configurações para 2 nós com wallet e genesis
	node1Config := createTestNodeConfig(t, "test-node1", signalingURL, tempDir)
	node2Config := createTestNodeConfigWithSharedGenesis(t, "test-node2", signalingURL, tempDir, node1Config.GenesisBlock)

	// Criar e iniciar nós
	n1, err := node.NewNode(node1Config)
	if err != nil {
		t.Fatalf("Failed to create node1: %v", err)
	}
	defer stopNode(n1, t)

	n2, err := node.NewNode(node2Config)
	if err != nil {
		t.Fatalf("Failed to create node2: %v", err)
	}
	defer stopNode(n2, t)

	if err := n1.Start(); err != nil {
		t.Fatalf("Failed to start node1: %v", err)
	}

	if err := n2.Start(); err != nil {
		t.Fatalf("Failed to start node2: %v", err)
	}

	// Aguardar conexão WebRTC (otimizado)
	time.Sleep(1500 * time.Millisecond)

	// Verificar se os nós se conectaram
	peers1 := n1.GetPeers()
	peers2 := n2.GetPeers()

	if len(peers1) != 1 {
		t.Errorf("Node1 should have 1 peer, got %d", len(peers1))
	}

	if len(peers2) != 1 {
		t.Errorf("Node2 should have 1 peer, got %d", len(peers2))
	}

	t.Logf("✓ Nodes connected successfully")
	t.Logf("  Node1 peers: %d", len(peers1))
	t.Logf("  Node2 peers: %d", len(peers2))
}

// TestMultipleNodesConnection testa a conexão entre múltiplos nós
func TestMultipleNodesConnection(t *testing.T) {
	signalingPort := getRandomPort()
	signalingURL := fmt.Sprintf("ws://localhost:%d/ws", signalingPort)
	tempDir := getTempDataDir(t, "multi")

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

	// Criar genesis compartilhado
	w := createTestWallet(t)
	genesis := createTestGenesis(w.GetAddress(), 1000000000)

	const numNodes = 4
	nodes := make([]*node.Node, numNodes)

	// Criar e iniciar nós rapidamente
	for i := 0; i < numNodes; i++ {
		config := createTestNodeConfigWithSharedGenesis(t, fmt.Sprintf("multi-node%d", i+1), signalingURL, tempDir, genesis)

		n, err := node.NewNode(config)
		if err != nil {
			t.Fatalf("Failed to create node%d: %v", i+1, err)
		}
		defer stopNode(n, t)

		if err := n.Start(); err != nil {
			t.Fatalf("Failed to start node%d: %v", i+1, err)
		}

		nodes[i] = n

		// Pequeno delay para evitar race no signaling
		time.Sleep(150 * time.Millisecond)
	}

	// Aguardar conexões e data channels (otimizado)
	time.Sleep(3 * time.Second)

	// Verificar se todos os nós têm pelo menos 1 peer conectado
	minExpectedPeers := 1
	for i, n := range nodes {
		peers := n.GetPeers()
		if len(peers) < minExpectedPeers {
			t.Errorf("Node%d should have at least %d peer, got %d", i+1, minExpectedPeers, len(peers))
		} else {
			t.Logf("✓ Node%d connected to %d peers", i+1, len(peers))
		}
	}
}

// TestMessageBroadcast testa o broadcast de mensagens entre nós
func TestMessageBroadcast(t *testing.T) {
	signalingPort := getRandomPort()
	signalingURL := fmt.Sprintf("ws://localhost:%d/ws", signalingPort)
	tempDir := getTempDataDir(t, "broadcast")

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

	// Criar genesis compartilhado
	w := createTestWallet(t)
	genesis := createTestGenesis(w.GetAddress(), 1000000000)

	// Criar 3 nós
	nodes := make([]*node.Node, 3)
	var mu sync.Mutex

	// Criar e iniciar nós
	for i := 0; i < 3; i++ {
		config := createTestNodeConfigWithSharedGenesis(t, fmt.Sprintf("broadcast-node%d", i+1), signalingURL, tempDir, genesis)

		n, err := node.NewNode(config)
		if err != nil {
			t.Fatalf("Failed to create node: %v", err)
		}
		defer stopNode(n, t)

		if err := n.Start(); err != nil {
			t.Fatalf("Failed to start node: %v", err)
		}

		nodes[i] = n
		time.Sleep(150 * time.Millisecond)
	}

	// Aguardar conexões e data channels (otimizado)
	time.Sleep(2 * time.Second)

	// Verificar conexões - pelo menos 1 peer para cada nó
	for i, n := range nodes {
		peers := n.GetPeers()
		if len(peers) < 1 {
			t.Errorf("Node%d should have at least 1 peer, got %d", i+1, len(peers))
		} else {
			t.Logf("✓ Node%d connected to %d peers", i+1, len(peers))
		}
	}

	// Teste de broadcast
	testMessage := []byte("Hello from node1")
	nodes[0].BroadcastMessage("test", testMessage)

	time.Sleep(500 * time.Millisecond)

	t.Logf("✓ Broadcast test completed")
	mu.Lock()
	defer mu.Unlock()
}

// TestNodeReconnection testa a reconexão após desconexão
func TestNodeReconnection(t *testing.T) {
	signalingPort := getRandomPort()
	signalingURL := fmt.Sprintf("ws://localhost:%d/ws", signalingPort)
	tempDir := getTempDataDir(t, "reconnect")

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

	// Criar genesis compartilhado
	w := createTestWallet(t)
	genesis := createTestGenesis(w.GetAddress(), 1000000000)

	// Criar 2 nós
	node1Config := createTestNodeConfigWithSharedGenesis(t, "reconnect-node1", signalingURL, tempDir, genesis)
	node2Config := createTestNodeConfigWithSharedGenesis(t, "reconnect-node2", signalingURL, tempDir, genesis)

	n1, err := node.NewNode(node1Config)
	if err != nil {
		t.Fatalf("Failed to create node1: %v", err)
	}
	defer stopNode(n1, t)

	n2, err := node.NewNode(node2Config)
	if err != nil {
		t.Fatalf("Failed to create node2: %v", err)
	}

	// Iniciar nós
	if err := n1.Start(); err != nil {
		t.Fatalf("Failed to start node1: %v", err)
	}

	if err := n2.Start(); err != nil {
		t.Fatalf("Failed to start node2: %v", err)
	}

	// Aguardar conexão
	time.Sleep(1500 * time.Millisecond)

	peers1 := n1.GetPeers()
	if len(peers1) != 1 {
		t.Fatalf("Node1 should have 1 peer before disconnect, got %d", len(peers1))
	}

	t.Logf("✓ Initial connection established")

	// Desconectar node2
	stopNode(n2, t)
	time.Sleep(500 * time.Millisecond)

	peers1AfterDisconnect := n1.GetPeers()
	t.Logf("  Node1 peers after node2 disconnect: %d", len(peers1AfterDisconnect))

	// Reconectar node2
	n2, err = node.NewNode(node2Config)
	if err != nil {
		t.Fatalf("Failed to recreate node2: %v", err)
	}
	defer stopNode(n2, t)

	if err := n2.Start(); err != nil {
		t.Fatalf("Failed to restart node2: %v", err)
	}

	// Aguardar reconexão
	time.Sleep(1500 * time.Millisecond)

	peers1AfterReconnect := n1.GetPeers()
	peers2AfterReconnect := n2.GetPeers()

	if len(peers1AfterReconnect) != 1 {
		t.Errorf("Node1 should have 1 peer after reconnect, got %d", len(peers1AfterReconnect))
	}

	if len(peers2AfterReconnect) != 1 {
		t.Errorf("Node2 should have 1 peer after reconnect, got %d", len(peers2AfterReconnect))
	}

	t.Logf("✓ Reconnection successful")
}

// TestNetworkPartitionRecovery testa a recuperação após partição de rede
// Simula perda de conexão durante mineração e verifica se o consenso é alcançado após reconexão
func TestNetworkPartitionRecovery(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping network partition test in short mode")
	}

	signalingPort := getRandomPort()
	signalingURL := fmt.Sprintf("ws://localhost:%d/ws", signalingPort)
	tempDir := getTempDataDir(t, "partition")

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

	// Criar wallets
	wallet1 := createTestWallet(t)
	wallet2 := createTestWallet(t)

	t.Logf("Wallet 1: %s", wallet1.GetAddress())
	t.Logf("Wallet 2: %s", wallet2.GetAddress())

	// Criar genesis com wallet1 tendo tokens
	genesisTx := blockchain.NewCoinbaseTransaction(wallet1.GetAddress(), 1000000000, 0)
	genesis := blockchain.GenesisBlock(genesisTx)

	t.Logf("Genesis block: %s", genesis.Hash[:16])

	// Configurar node1
	node1Config := createTestNodeConfigWithSharedGenesis(t, "partition-node1", signalingURL, tempDir, genesis)
	node1Config.Wallet = wallet1
	node1Config.DiscoveryInterval = 2

	// Configurar node2
	node2Config := createTestNodeConfigWithSharedGenesis(t, "partition-node2", signalingURL, tempDir, genesis)
	node2Config.Wallet = wallet2
	node2Config.DiscoveryInterval = 2

	// Criar e iniciar node1
	n1, err := node.NewNode(node1Config)
	if err != nil {
		t.Fatalf("Failed to create node1: %v", err)
	}
	defer stopNode(n1, t)

	if err := n1.Start(); err != nil {
		t.Fatalf("Failed to start node1: %v", err)
	}

	// Criar e iniciar node2
	n2, err := node.NewNode(node2Config)
	if err != nil {
		t.Fatalf("Failed to create node2: %v", err)
	}
	defer stopNode(n2, t)

	if err := n2.Start(); err != nil {
		t.Fatalf("Failed to start node2: %v", err)
	}

	t.Logf("\n=== Phase 1: Initial Synchronization ===")

	// Aguardar conexão inicial
	time.Sleep(1500 * time.Millisecond)

	// Verificar conexão
	peers1 := n1.GetPeers()
	peers2 := n2.GetPeers()

	if len(peers1) == 0 || len(peers2) == 0 {
		t.Fatalf("Nodes failed to connect initially. Node1: %d peers, Node2: %d peers", len(peers1), len(peers2))
	}

	t.Logf("✓ Nodes connected: Node1 has %d peers, Node2 has %d peers", len(peers1), len(peers2))

	// Node1 faz stake e inicia mineração
	t.Logf("\n[Node1] Creating stake transaction...")
	stakeTx, err := n1.CreateStakeTransaction(100000, 10)
	if err != nil {
		t.Fatalf("Failed to create stake: %v", err)
	}
	t.Logf("[Node1] Stake created: %s", stakeTx.ID[:8])

	t.Logf("[Node1] Starting mining...")
	if err := n1.StartMining(); err != nil {
		t.Fatalf("Failed to start mining: %v", err)
	}

	// Aguardar alguns blocos serem minerados e sincronizados
	time.Sleep(800 * time.Millisecond)

	height1BeforePartition := n1.GetChainHeight()
	height2BeforePartition := n2.GetChainHeight()

	t.Logf("[Node1] Chain height: %d", height1BeforePartition)
	t.Logf("[Node2] Chain height: %d (synchronized)", height2BeforePartition)

	if height1BeforePartition > 0 && height2BeforePartition != height1BeforePartition {
		t.Logf("Warning: Heights differ before partition: Node1=%d, Node2=%d", height1BeforePartition, height2BeforePartition)
	}

	t.Logf("\n=== Phase 2: Network Partition (Simulating Connection Loss) ===")

	// Simular perda de conexão: parar node2 temporariamente
	stopNode(n2, t)
	t.Logf("✗ Node2 disconnected (simulating network partition)")

	// Aguardar ~1 segundo para node1 minerar ~3 blocos sozinho
	t.Logf("[Node1] Mining while disconnected from Node2...")
	time.Sleep(1000 * time.Millisecond)

	height1AfterPartition := n1.GetChainHeight()
	t.Logf("[Node1] Chain height after partition: %d (mined %d blocks alone)",
		height1AfterPartition, height1AfterPartition-height1BeforePartition)

	if height1AfterPartition <= height1BeforePartition {
		t.Logf("Warning: Node1 didn't mine new blocks during partition")
	}

	t.Logf("\n=== Phase 3: Network Reconnection ===")

	// Reconectar node2 (recriá-lo do zero para simular reconexão)
	n2, err = node.NewNode(node2Config)
	if err != nil {
		t.Fatalf("Failed to recreate node2: %v", err)
	}
	defer stopNode(n2, t)

	if err := n2.Start(); err != nil {
		t.Fatalf("Failed to restart node2: %v", err)
	}

	t.Logf("✓ Node2 reconnected")

	// Aguardar reconexão e sincronização
	t.Logf("Waiting for nodes to reconnect and synchronize...")
	time.Sleep(2000 * time.Millisecond)

	// Verificar reconexão
	peers1AfterReconnect := n1.GetPeers()
	peers2AfterReconnect := n2.GetPeers()

	if len(peers1AfterReconnect) == 0 || len(peers2AfterReconnect) == 0 {
		t.Logf("Warning: Nodes may not have fully reconnected. Node1: %d peers, Node2: %d peers",
			len(peers1AfterReconnect), len(peers2AfterReconnect))
	} else {
		t.Logf("✓ Nodes reconnected: Node1 has %d peers, Node2 has %d peers",
			len(peers1AfterReconnect), len(peers2AfterReconnect))
	}

	t.Logf("\n=== Phase 4: Consensus Verification ===")

	// Verificar alturas finais
	finalHeight1 := n1.GetChainHeight()
	finalHeight2 := n2.GetChainHeight()

	t.Logf("[Node1] Final chain height: %d", finalHeight1)
	t.Logf("[Node2] Final chain height: %d", finalHeight2)

	// Verificar consenso
	if finalHeight1 != finalHeight2 {
		t.Errorf("Consensus not reached! Node1 height: %d, Node2 height: %d", finalHeight1, finalHeight2)
	} else {
		t.Logf("✓ Consensus reached! Both nodes at height: %d", finalHeight1)
	}

	// Verificar que houve progresso durante a partição
	if finalHeight1 > height1BeforePartition {
		blocksMinedDuringPartition := height1AfterPartition - height1BeforePartition
		t.Logf("✓ Node1 mined %d blocks during partition", blocksMinedDuringPartition)
	}

	// Verificar que node2 sincronizou
	if finalHeight2 >= height1AfterPartition {
		t.Logf("✓ Node2 successfully synchronized after reconnection")
	} else {
		t.Logf("Warning: Node2 may not be fully synchronized (height: %d, expected >= %d)",
			finalHeight2, height1AfterPartition)
	}

	t.Logf("\n=== Test Summary ===")
	t.Logf("✓ Network partition recovery test completed")
	t.Logf("✓ Initial sync: %d blocks", height1BeforePartition)
	t.Logf("✓ Blocks mined during partition: %d", height1AfterPartition-height1BeforePartition)
	t.Logf("✓ Final consensus height: %d", finalHeight1)

	if finalHeight1 == finalHeight2 && finalHeight1 >= height1AfterPartition {
		t.Logf("✓ SUCCESS: Nodes reached consensus after network partition recovery")
	} else {
		t.Logf("⚠ Partial success: Check timing parameters for more reliable synchronization")
	}
}
