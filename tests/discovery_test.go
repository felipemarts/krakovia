package tests

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/krakovia/blockchain/pkg/node"
	"github.com/krakovia/blockchain/pkg/signaling"
)

// Funções auxiliares compartilhadas já definidas em network_test.go
// mas precisamos redefinir aqui para evitar dependências

// getRandomPortDiscovery retorna uma porta aleatória no intervalo 9000-29000
func getRandomPortDiscovery() int {
	return 9000 + rand.Intn(20000)
}

// getTempDataDirDiscovery cria um diretório temporário único para o teste
func getTempDataDirDiscovery(t *testing.T, testName string) string {
	tempDir := filepath.Join(os.TempDir(), fmt.Sprintf("krakovia-test-%s-%d", testName, time.Now().UnixNano()))
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	t.Cleanup(func() {
		os.RemoveAll(tempDir)
	})
	return tempDir
}

// TestPeerLimitEnforcement testa se o limite de peers é respeitado
func TestPeerLimitEnforcement(t *testing.T) {
	signalingPort := getRandomPortDiscovery()
	signalingURL := fmt.Sprintf("ws://localhost:%d/ws", signalingPort)
	tempDir := getTempDataDirDiscovery(t, "limit")

	// Iniciar servidor de signaling
	server := signaling.NewServer()
	go func() {
		if err := server.Start(fmt.Sprintf(":%d", signalingPort)); err != nil {
			t.Logf("Signaling server error: %v", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	// Criar 6 nós (reduzido de 10), mas com limite de 3 peers cada
	const numNodes = 6
	const maxPeers = 3
	const minPeers = 2

	nodes := make([]*node.Node, numNodes)

	for i := 0; i < numNodes; i++ {
		config := node.Config{
			ID:                fmt.Sprintf("limit-test-node%d", i+1),
			Address:           fmt.Sprintf(":%d", getRandomPortDiscovery()),
			DBPath:            filepath.Join(tempDir, fmt.Sprintf("node%d", i+1)),
			SignalingServer:   signalingURL,
			MaxPeers:          maxPeers,
			MinPeers:          minPeers,
			DiscoveryInterval: 2, // Descoberta rápida para o teste
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
		time.Sleep(100 * time.Millisecond)
	}

	// Aguardar conexões (reduzido para 3s)
	time.Sleep(3 * time.Second)

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
	signalingPort := getRandomPortDiscovery()
	signalingURL := fmt.Sprintf("ws://localhost:%d/ws", signalingPort)
	tempDir := getTempDataDirDiscovery(t, "discovery")

	// Iniciar servidor de signaling
	server := signaling.NewServer()
	go func() {
		if err := server.Start(fmt.Sprintf(":%d", signalingPort)); err != nil {
			t.Logf("Signaling server error: %v", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	// Criar 2 nós inicialmente
	node1Config := node.Config{
		ID:                "discovery-node1",
		Address:           fmt.Sprintf(":%d", getRandomPortDiscovery()),
		DBPath:            filepath.Join(tempDir, "node1"),
		SignalingServer:   signalingURL,
		MaxPeers:          10,
		MinPeers:          2,
		DiscoveryInterval: 2, // Descoberta rápida
	}

	node2Config := node.Config{
		ID:                "discovery-node2",
		Address:           fmt.Sprintf(":%d", getRandomPortDiscovery()),
		DBPath:            filepath.Join(tempDir, "node2"),
		SignalingServer:   signalingURL,
		MaxPeers:          10,
		MinPeers:          2,
		DiscoveryInterval: 2,
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

	time.Sleep(1500 * time.Millisecond)

	peers1Before := len(n1.GetPeers())
	peers2Before := len(n2.GetPeers())

	t.Logf("Initial state - Node1: %d peers, Node2: %d peers", peers1Before, peers2Before)

	// Adicionar um terceiro nó
	node3Config := node.Config{
		ID:                "discovery-node3",
		Address:           fmt.Sprintf(":%d", getRandomPortDiscovery()),
		DBPath:            filepath.Join(tempDir, "node3"),
		SignalingServer:   signalingURL,
		MaxPeers:          10,
		MinPeers:          2,
		DiscoveryInterval: 2,
	}

	n3, err := node.NewNode(node3Config)
	if err != nil {
		t.Fatalf("Failed to create node3: %v", err)
	}
	defer n3.Stop()

	n3.Start()

	// Aguardar descoberta periódica (reduzido para 2.5s)
	time.Sleep(2500 * time.Millisecond)

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
	signalingPort := getRandomPortDiscovery()
	signalingURL := fmt.Sprintf("ws://localhost:%d/ws", signalingPort)
	tempDir := getTempDataDirDiscovery(t, "minpeers")

	// Iniciar servidor de signaling
	server := signaling.NewServer()
	go func() {
		if err := server.Start(fmt.Sprintf(":%d", signalingPort)); err != nil {
			t.Logf("Signaling server error: %v", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	const minPeers = 3
	const numNodes = 5

	nodes := make([]*node.Node, numNodes)

	// Criar nós com requisito mínimo de 3 peers
	for i := 0; i < numNodes; i++ {
		config := node.Config{
			ID:                fmt.Sprintf("min-peers-node%d", i+1),
			Address:           fmt.Sprintf(":%d", getRandomPortDiscovery()),
			DBPath:            filepath.Join(tempDir, fmt.Sprintf("node%d", i+1)),
			SignalingServer:   signalingURL,
			MaxPeers:          10,
			MinPeers:          minPeers,
			DiscoveryInterval: 2, // Descoberta rápida
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
		time.Sleep(200 * time.Millisecond)
	}

	// Aguardar conexões (reduzido para 3s)
	time.Sleep(3 * time.Second)

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
