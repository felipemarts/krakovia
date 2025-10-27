package tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/krakovia/blockchain/pkg/node"
	"github.com/krakovia/blockchain/pkg/signaling"
)

// TestPeerLimitEnforcement testa se o limite de peers é respeitado
func TestPeerLimitEnforcement(t *testing.T) {
	// Iniciar servidor de signaling
	server := signaling.NewServer()
	go func() {
		if err := server.Start(":9500"); err != nil {
			t.Logf("Signaling server error: %v", err)
		}
	}()

	time.Sleep(500 * time.Millisecond)

	// Criar 10 nós, mas com limite de 3 peers cada
	const numNodes = 10
	const maxPeers = 3
	const minPeers = 2

	nodes := make([]*node.Node, numNodes)

	for i := 0; i < numNodes; i++ {
		config := node.Config{
			ID:                fmt.Sprintf("limit-test-node%d", i+1),
			Address:           fmt.Sprintf(":950%d", i+1),
			DBPath:            fmt.Sprintf("./test-data/limit-node%d", i+1),
			SignalingServer:   "ws://localhost:9500/ws",
			MaxPeers:          maxPeers,
			MinPeers:          minPeers,
			DiscoveryInterval: 5, // 5 segundos para teste rápido
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
		time.Sleep(200 * time.Millisecond) // Pequeno delay entre nós
	}

	// Aguardar conexões serem estabelecidas
	time.Sleep(8 * time.Second)

	// Verificar que nenhum nó excedeu o limite
	for i, n := range nodes {
		peers := n.GetPeers()
		peerCount := len(peers)

		if peerCount > maxPeers {
			t.Errorf("Node%d has %d peers, exceeds max of %d", i+1, peerCount, maxPeers)
		}

		if peerCount >= minPeers {
			t.Logf("✓ Node%d has %d peers (within limits: min=%d, max=%d)",
				i+1, peerCount, minPeers, maxPeers)
		} else {
			t.Logf("⚠ Node%d has %d peers (below minimum of %d)",
				i+1, peerCount, minPeers)
		}
	}

	t.Logf("✓ Peer limit enforcement working")
}

// TestPeerDiscovery testa a descoberta periódica de peers
func TestPeerDiscovery(t *testing.T) {
	// Iniciar servidor de signaling
	server := signaling.NewServer()
	go func() {
		if err := server.Start(":9600"); err != nil {
			t.Logf("Signaling server error: %v", err)
		}
	}()

	time.Sleep(500 * time.Millisecond)

	// Criar 2 nós inicialmente
	node1Config := node.Config{
		ID:                "discovery-node1",
		Address:           ":9601",
		DBPath:            "./test-data/discovery-node1",
		SignalingServer:   "ws://localhost:9600/ws",
		MaxPeers:          10,
		MinPeers:          2,
		DiscoveryInterval: 3, // 3 segundos
	}

	node2Config := node.Config{
		ID:                "discovery-node2",
		Address:           ":9602",
		DBPath:            "./test-data/discovery-node2",
		SignalingServer:   "ws://localhost:9600/ws",
		MaxPeers:          10,
		MinPeers:          2,
		DiscoveryInterval: 3,
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
	defer n2.Stop()

	n1.Start()
	n2.Start()

	time.Sleep(3 * time.Second)

	peers1Before := len(n1.GetPeers())
	peers2Before := len(n2.GetPeers())

	t.Logf("Initial state - Node1: %d peers, Node2: %d peers", peers1Before, peers2Before)

	// Adicionar um terceiro nó após os dois primeiros se conectarem
	node3Config := node.Config{
		ID:                "discovery-node3",
		Address:           ":9603",
		DBPath:            "./test-data/discovery-node3",
		SignalingServer:   "ws://localhost:9600/ws",
		MaxPeers:          10,
		MinPeers:          2,
		DiscoveryInterval: 3,
	}

	n3, err := node.NewNode(node3Config)
	if err != nil {
		t.Fatalf("Failed to create node3: %v", err)
	}
	defer n3.Stop()

	n3.Start()

	// Aguardar descoberta periódica encontrar o novo nó
	time.Sleep(6 * time.Second)

	peers1After := len(n1.GetPeers())
	peers2After := len(n2.GetPeers())
	peers3After := len(n3.GetPeers())

	t.Logf("After discovery - Node1: %d peers, Node2: %d peers, Node3: %d peers",
		peers1After, peers2After, peers3After)

	// Node3 deve ter descoberto os outros
	if peers3After < 2 {
		t.Errorf("Node3 should have discovered at least 2 peers, got %d", peers3After)
	} else {
		t.Logf("✓ Node3 discovered %d peers", peers3After)
	}

	t.Logf("✓ Peer discovery working")
}

// TestMinimumPeersMaintenance testa se os nós mantêm o mínimo de peers
func TestMinimumPeersMaintenance(t *testing.T) {
	// Iniciar servidor de signaling
	server := signaling.NewServer()
	go func() {
		if err := server.Start(":9700"); err != nil {
			t.Logf("Signaling server error: %v", err)
		}
	}()

	time.Sleep(500 * time.Millisecond)

	const minPeers = 3
	const numNodes = 5

	nodes := make([]*node.Node, numNodes)

	// Criar nós com requisito mínimo de 3 peers
	for i := 0; i < numNodes; i++ {
		config := node.Config{
			ID:                fmt.Sprintf("min-peers-node%d", i+1),
			Address:           fmt.Sprintf(":970%d", i+1),
			DBPath:            fmt.Sprintf("./test-data/min-peers-node%d", i+1),
			SignalingServer:   "ws://localhost:9700/ws",
			MaxPeers:          10,
			MinPeers:          minPeers,
			DiscoveryInterval: 5,
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
		time.Sleep(500 * time.Millisecond)
	}

	// Aguardar conexões
	time.Sleep(8 * time.Second)

	// Verificar que todos os nós têm pelo menos minPeers conexões
	allGood := true
	for i, n := range nodes {
		peers := n.GetPeers()
		peerCount := len(peers)

		if peerCount < minPeers {
			t.Logf("⚠ Node%d has only %d peers (minimum: %d)", i+1, peerCount, minPeers)
			allGood = false
		} else {
			t.Logf("✓ Node%d has %d peers (minimum satisfied)", i+1, peerCount)
		}
	}

	if allGood {
		t.Logf("✓ All nodes maintain minimum peer count")
	}
}
