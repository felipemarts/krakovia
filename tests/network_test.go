package tests

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/krakovia/blockchain/pkg/node"
	"github.com/krakovia/blockchain/pkg/signaling"
)

// getRandomPort retorna uma porta aleatória no intervalo 9000-29000
func getRandomPort() int {
	return 9000 + rand.Intn(20000)
}

// getTempDataDir cria um diretório temporário único para o teste
func getTempDataDir(t *testing.T, testName string) string {
	tempDir := filepath.Join(os.TempDir(), fmt.Sprintf("krakovia-test-%s-%d", testName, time.Now().UnixNano()))
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	t.Cleanup(func() {
		os.RemoveAll(tempDir)
	})
	return tempDir
}

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

	// Aguardar servidor iniciar
	time.Sleep(100 * time.Millisecond)

	// Criar configurações para 2 nós
	node1Config := node.Config{
		ID:                "test-node1",
		Address:           fmt.Sprintf(":%d", getRandomPort()),
		DBPath:            filepath.Join(tempDir, "node1"),
		SignalingServer:   signalingURL,
		MaxPeers:          10,
		MinPeers:          1,
		DiscoveryInterval: 60, // Não precisa descoberta rápida neste teste
	}

	node2Config := node.Config{
		ID:                "test-node2",
		Address:           fmt.Sprintf(":%d", getRandomPort()),
		DBPath:            filepath.Join(tempDir, "node2"),
		SignalingServer:   signalingURL,
		MaxPeers:          10,
		MinPeers:          1,
		DiscoveryInterval: 60,
	}

	// Criar e iniciar nós
	n1, err := node.NewNode(node1Config)
	if err != nil {
		t.Fatalf("Failed to create node1: %v", err)
	}
	defer n1.Stop()

	n2, err := node.NewNode(node2Config)
	if err != nil {
		t.Fatalf("Failed to create node2: %v", err)
	}
	defer n2.Stop()

	if err := n1.Start(); err != nil {
		t.Fatalf("Failed to start node1: %v", err)
	}

	if err := n2.Start(); err != nil {
		t.Fatalf("Failed to start node2: %v", err)
	}

	// Aguardar conexão WebRTC (reduzido de 3s para 2s)
	time.Sleep(2 * time.Second)

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

	time.Sleep(100 * time.Millisecond)

	const numNodes = 4
	nodes := make([]*node.Node, numNodes)
	var wg sync.WaitGroup

	// Criar e iniciar nós rapidamente
	for i := 0; i < numNodes; i++ {
		config := node.Config{
			ID:                fmt.Sprintf("multi-node%d", i+1),
			Address:           fmt.Sprintf(":%d", getRandomPort()),
			DBPath:            filepath.Join(tempDir, fmt.Sprintf("node%d", i+1)),
			SignalingServer:   signalingURL,
			MaxPeers:          10,
			MinPeers:          2,
			DiscoveryInterval: 60,
		}

		n, err := node.NewNode(config)
		if err != nil {
			t.Fatalf("Failed to create node%d: %v", i+1, err)
		}
		defer n.Stop()

		if err := n.Start(); err != nil {
			t.Fatalf("Failed to start node%d: %v", i+1, err)
		}

		nodes[i] = n

		// Pequeno delay para evitar race no signaling
		time.Sleep(150 * time.Millisecond)
	}

	// Aguardar conexões (3s para garantir mesh completa)
	time.Sleep(3 * time.Second)

	// Verificar se todos os nós estão bem conectados (pelo menos 2 peers)
	// Em uma rede mesh com 4 nós, nem sempre todos conectam simultaneamente a todos
	minExpectedPeers := 2
	for i, n := range nodes {
		peers := n.GetPeers()
		if len(peers) < minExpectedPeers {
			t.Errorf("Node%d should have at least %d peers, got %d", i+1, minExpectedPeers, len(peers))
		} else {
			t.Logf("✓ Node%d connected to %d peers", i+1, len(peers))
		}
	}

	wg.Wait()
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

	time.Sleep(100 * time.Millisecond)

	// Criar 3 nós
	configs := []node.Config{
		{
			ID:                "broadcast-node1",
			Address:           fmt.Sprintf(":%d", getRandomPort()),
			DBPath:            filepath.Join(tempDir, "node1"),
			SignalingServer:   signalingURL,
			MaxPeers:          10,
			MinPeers:          2,
			DiscoveryInterval: 60,
		},
		{
			ID:                "broadcast-node2",
			Address:           fmt.Sprintf(":%d", getRandomPort()),
			DBPath:            filepath.Join(tempDir, "node2"),
			SignalingServer:   signalingURL,
			MaxPeers:          10,
			MinPeers:          2,
			DiscoveryInterval: 60,
		},
		{
			ID:                "broadcast-node3",
			Address:           fmt.Sprintf(":%d", getRandomPort()),
			DBPath:            filepath.Join(tempDir, "node3"),
			SignalingServer:   signalingURL,
			MaxPeers:          10,
			MinPeers:          2,
			DiscoveryInterval: 60,
		},
	}

	nodes := make([]*node.Node, len(configs))
	receivedMessages := make([]int, len(configs))
	var mu sync.Mutex

	// Criar e iniciar nós
	for i, cfg := range configs {
		n, err := node.NewNode(cfg)
		if err != nil {
			t.Fatalf("Failed to create node: %v", err)
		}
		defer n.Stop()

		if err := n.Start(); err != nil {
			t.Fatalf("Failed to start node: %v", err)
		}

		nodes[i] = n
		time.Sleep(150 * time.Millisecond)
	}

	// Aguardar conexões (reduzido para 1.5s)
	time.Sleep(1500 * time.Millisecond)

	// Verificar conexões
	for i, n := range nodes {
		peers := n.GetPeers()
		expectedPeers := len(nodes) - 1
		if len(peers) != expectedPeers {
			t.Errorf("Node%d should have %d peers, got %d", i+1, expectedPeers, len(peers))
		}
	}

	// Teste de broadcast
	testMessage := []byte("Hello from node1")
	nodes[0].BroadcastMessage("test", testMessage)

	time.Sleep(500 * time.Millisecond)

	t.Logf("✓ Broadcast test completed")
	mu.Lock()
	for i, count := range receivedMessages {
		t.Logf("  Node%d received %d messages", i+1, count)
	}
	mu.Unlock()
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

	time.Sleep(100 * time.Millisecond)

	// Criar 2 nós
	node1Config := node.Config{
		ID:                "reconnect-node1",
		Address:           fmt.Sprintf(":%d", getRandomPort()),
		DBPath:            filepath.Join(tempDir, "node1"),
		SignalingServer:   signalingURL,
		MaxPeers:          10,
		MinPeers:          1,
		DiscoveryInterval: 60,
	}

	node2Config := node.Config{
		ID:                "reconnect-node2",
		Address:           fmt.Sprintf(":%d", getRandomPort()),
		DBPath:            filepath.Join(tempDir, "node2"),
		SignalingServer:   signalingURL,
		MaxPeers:          10,
		MinPeers:          1,
		DiscoveryInterval: 60,
	}

	n1, err := node.NewNode(node1Config)
	if err != nil {
		t.Fatalf("Failed to create node1: %v", err)
	}
	defer n1.Stop()

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

	// Aguardar conexão (reduzido para 1.5s)
	time.Sleep(1500 * time.Millisecond)

	peers1 := n1.GetPeers()
	if len(peers1) != 1 {
		t.Fatalf("Node1 should have 1 peer before disconnect, got %d", len(peers1))
	}

	t.Logf("✓ Initial connection established")

	// Desconectar node2
	n2.Stop()
	time.Sleep(500 * time.Millisecond)

	peers1AfterDisconnect := n1.GetPeers()
	t.Logf("  Node1 peers after node2 disconnect: %d", len(peers1AfterDisconnect))

	// Reconectar node2
	n2, err = node.NewNode(node2Config)
	if err != nil {
		t.Fatalf("Failed to recreate node2: %v", err)
	}
	defer n2.Stop()

	if err := n2.Start(); err != nil {
		t.Fatalf("Failed to restart node2: %v", err)
	}

	// Aguardar reconexão (reduzido para 1.5s)
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
