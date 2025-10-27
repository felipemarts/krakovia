package tests

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/krakovia/blockchain/pkg/node"
	"github.com/krakovia/blockchain/pkg/signaling"
)

const (
	signalingAddr = ":9100"
	signalingURL  = "ws://localhost:9100/ws"
)

// TestNodeConnection testa a conexão básica entre nós
func TestNodeConnection(t *testing.T) {
	// Iniciar servidor de signaling
	server := signaling.NewServer()
	go func() {
		if err := server.Start(signalingAddr); err != nil {
			t.Logf("Signaling server error: %v", err)
		}
	}()

	// Dar tempo para o servidor iniciar
	time.Sleep(500 * time.Millisecond)

	// Criar configurações para 2 nós
	node1Config := node.Config{
		ID:                "test-node1",
		Address:           ":9101",
		DBPath:            "./test-data/node1",
		SignalingServer:   signalingURL,
		MaxPeers:          10,
		MinPeers:          1,
		DiscoveryInterval: 30,
	}

	node2Config := node.Config{
		ID:                "test-node2",
		Address:           ":9102",
		DBPath:            "./test-data/node2",
		SignalingServer:   signalingURL,
		MaxPeers:          10,
		MinPeers:          1,
		DiscoveryInterval: 30,
	}

	// Criar nós
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

	// Iniciar nós
	if err := n1.Start(); err != nil {
		t.Fatalf("Failed to start node1: %v", err)
	}

	if err := n2.Start(); err != nil {
		t.Fatalf("Failed to start node2: %v", err)
	}

	// Aguardar conexão WebRTC ser estabelecida
	time.Sleep(3 * time.Second)

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
	// Iniciar servidor de signaling
	server := signaling.NewServer()
	go func() {
		if err := server.Start(":9200"); err != nil {
			t.Logf("Signaling server error: %v", err)
		}
	}()

	// Dar tempo para o servidor iniciar
	time.Sleep(500 * time.Millisecond)

	const numNodes = 4
	nodes := make([]*node.Node, numNodes)
	var wg sync.WaitGroup

	// Criar e iniciar nós com delay entre cada um
	for i := 0; i < numNodes; i++ {
		config := node.Config{
			ID:                fmt.Sprintf("test-node%d", i+1),
			Address:           fmt.Sprintf(":920%d", i+1),
			DBPath:            fmt.Sprintf("./test-data/node%d", i+1),
			SignalingServer:   "ws://localhost:9200/ws",
			MaxPeers:          10,
			MinPeers:          2,
			DiscoveryInterval: 30,
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

		// Dar tempo para o nó se conectar aos anteriores
		time.Sleep(1 * time.Second)
	}

	// Aguardar todas as conexões WebRTC serem estabelecidas
	time.Sleep(3 * time.Second)

	// Verificar se todos os nós estão conectados entre si
	expectedPeers := numNodes - 1
	for i, n := range nodes {
		peers := n.GetPeers()
		if len(peers) != expectedPeers {
			t.Errorf("Node%d should have %d peers, got %d", i+1, expectedPeers, len(peers))
		} else {
			t.Logf("✓ Node%d connected to %d peers", i+1, len(peers))
		}
	}

	wg.Wait()
}

// TestMessageBroadcast testa o broadcast de mensagens entre nós
func TestMessageBroadcast(t *testing.T) {
	// Iniciar servidor de signaling
	server := signaling.NewServer()
	go func() {
		if err := server.Start(":9300"); err != nil {
			t.Logf("Signaling server error: %v", err)
		}
	}()

	time.Sleep(500 * time.Millisecond)

	// Criar 3 nós
	configs := []node.Config{
		{
			ID:                "broadcast-node1",
			Address:           ":9301",
			DBPath:            "./test-data/broadcast-node1",
			SignalingServer:   "ws://localhost:9300/ws",
			MaxPeers:          10,
			MinPeers:          2,
			DiscoveryInterval: 30,
		},
		{
			ID:                "broadcast-node2",
			Address:           ":9302",
			DBPath:            "./test-data/broadcast-node2",
			SignalingServer:   "ws://localhost:9300/ws",
			MaxPeers:          10,
			MinPeers:          2,
			DiscoveryInterval: 30,
		},
		{
			ID:                "broadcast-node3",
			Address:           ":9303",
			DBPath:            "./test-data/broadcast-node3",
			SignalingServer:   "ws://localhost:9300/ws",
			MaxPeers:          10,
			MinPeers:          2,
			DiscoveryInterval: 30,
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

		// Configurar handler de mensagens (será implementado no futuro)
		// Por enquanto, apenas verificamos a conexão
	}

	// Aguardar conexões
	time.Sleep(3 * time.Second)

	// Verificar conexões
	for i, n := range nodes {
		peers := n.GetPeers()
		expectedPeers := len(nodes) - 1
		if len(peers) != expectedPeers {
			t.Errorf("Node%d should have %d peers, got %d", i+1, expectedPeers, len(peers))
		}
	}

	// Teste de broadcast (será expandido quando implementarmos handlers de mensagens)
	testMessage := []byte("Hello from node1")
	nodes[0].BroadcastMessage("test", testMessage)

	time.Sleep(1 * time.Second)

	t.Logf("✓ Broadcast test completed")
	mu.Lock()
	for i, count := range receivedMessages {
		t.Logf("  Node%d received %d messages", i+1, count)
	}
	mu.Unlock()
}

// TestNodeReconnection testa a reconexão após desconexão
func TestNodeReconnection(t *testing.T) {
	// Iniciar servidor de signaling
	server := signaling.NewServer()
	go func() {
		if err := server.Start(":9400"); err != nil {
			t.Logf("Signaling server error: %v", err)
		}
	}()

	time.Sleep(500 * time.Millisecond)

	// Criar 2 nós
	node1Config := node.Config{
		ID:                "reconnect-node1",
		Address:           ":9401",
		DBPath:            "./test-data/reconnect-node1",
		SignalingServer:   "ws://localhost:9400/ws",
		MaxPeers:          10,
		MinPeers:          1,
		DiscoveryInterval: 30,
	}

	node2Config := node.Config{
		ID:                "reconnect-node2",
		Address:           ":9402",
		DBPath:            "./test-data/reconnect-node2",
		SignalingServer:   "ws://localhost:9400/ws",
		MaxPeers:          10,
		MinPeers:          1,
		DiscoveryInterval: 30,
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

	// Aguardar conexão
	time.Sleep(3 * time.Second)

	peers1 := n1.GetPeers()
	if len(peers1) != 1 {
		t.Fatalf("Node1 should have 1 peer before disconnect, got %d", len(peers1))
	}

	t.Logf("✓ Initial connection established")

	// Desconectar node2
	n2.Stop()
	time.Sleep(2 * time.Second)

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

	// Aguardar reconexão
	time.Sleep(3 * time.Second)

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
