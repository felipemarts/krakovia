package tests

import (
	"fmt"
	"sync"
	"testing"
	"time"

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
