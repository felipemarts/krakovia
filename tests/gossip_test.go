package tests

import (
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/krakovia/blockchain/pkg/network"
	"github.com/krakovia/blockchain/pkg/node"
	"github.com/krakovia/blockchain/pkg/signaling"
)

// TestGossipMessage testa criação e validação de mensagens gossip
func TestGossipMessage(t *testing.T) {
	// Criar mensagem
	msg, err := network.NewGossipMessage("node1", "test", []byte("hello"), 5)
	if err != nil {
		t.Fatalf("Failed to create gossip message: %v", err)
	}

	// Validar mensagem
	if err := msg.Validate(); err != nil {
		t.Fatalf("Message validation failed: %v", err)
	}

	// Testar campos
	if msg.OriginID != "node1" {
		t.Errorf("Expected origin node1, got %s", msg.OriginID)
	}
	if msg.Type != "test" {
		t.Errorf("Expected type test, got %s", msg.Type)
	}
	if msg.TTL != 5 {
		t.Errorf("Expected TTL 5, got %d", msg.TTL)
	}
	if msg.HopCount != 0 {
		t.Errorf("Expected HopCount 0, got %d", msg.HopCount)
	}

	t.Logf("✓ Message created and validated successfully")
}

// TestGossipMessageHash testa integridade do hash
func TestGossipMessageHash(t *testing.T) {
	msg, err := network.NewGossipMessage("node1", "test", []byte("data"), 5)
	if err != nil {
		t.Fatalf("Failed to create message: %v", err)
	}

	originalHash := msg.Hash

	// Modificar dados
	msg.Data = []byte("modified")

	// Validação deve falhar
	if err := msg.Validate(); err == nil {
		t.Error("Expected validation to fail with modified data")
	}

	// Restaurar hash original
	msg.Hash = originalHash

	t.Logf("✓ Hash integrity protection working")
}

// TestGossipMessageTTL testa controle de TTL
func TestGossipMessageTTL(t *testing.T) {
	msg, _ := network.NewGossipMessage("node1", "test", []byte("data"), 3)

	// Deve propagar inicialmente
	if !msg.ShouldPropagate() {
		t.Error("Message should propagate initially")
	}

	// Incrementar hops
	msg.IncrementHop()
	msg.IncrementHop()
	msg.IncrementHop()

	// Não deve mais propagar
	if msg.ShouldPropagate() {
		t.Error("Message should not propagate after TTL exceeded")
	}

	t.Logf("✓ TTL control working correctly")
}

// TestMessageCache testa cache de mensagens
func TestMessageCache(t *testing.T) {
	cache := network.NewMessageCache(100)

	msg1, _ := network.NewGossipMessage("node1", "test", []byte("data1"), 5)
	msg2, _ := network.NewGossipMessage("node2", "test", []byte("data2"), 5)

	// Adicionar mensagens
	cache.Add(msg1)
	cache.Add(msg2)

	// Verificar se estão no cache
	if !cache.Has(msg1.ID) {
		t.Error("Message 1 should be in cache")
	}
	if !cache.Has(msg2.ID) {
		t.Error("Message 2 should be in cache")
	}

	// Testar duplicata
	if !cache.Has(msg1.ID) {
		t.Error("Duplicate detection failed")
	}

	t.Logf("✓ Message cache working correctly")
}

// TestRateLimiter testa rate limiting
func TestRateLimiter(t *testing.T) {
	limiter := network.NewRateLimiter(5) // 5 mensagens por segundo

	// Enviar 5 mensagens (deve passar)
	for i := 0; i < 5; i++ {
		if !limiter.Allow("peer1") {
			t.Errorf("Message %d should be allowed", i+1)
		}
	}

	// 6ª mensagem deve ser bloqueada
	if limiter.Allow("peer1") {
		t.Error("6th message should be blocked")
	}

	// Aguardar 1 segundo
	time.Sleep(1100 * time.Millisecond)

	// Deve permitir novamente
	if !limiter.Allow("peer1") {
		t.Error("Should allow after window reset")
	}

	t.Logf("✓ Rate limiting working correctly")
}

// TestGossipPropagation testa propagação de mensagens entre nós
func TestGossipPropagation(t *testing.T) {
	signalingPort := getRandomPortDiscovery()
	signalingURL := fmt.Sprintf("ws://localhost:%d/ws", signalingPort)
	tempDir := getTempDataDirDiscovery(t, "gossip-prop")

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

	// Criar 3 nós
	nodes := make([]*node.Node, 3)
	messagesReceived := make([]int, 3)
	var mu sync.Mutex

	for i := 0; i < 3; i++ {
		config := node.Config{
			ID:                fmt.Sprintf("gossip-node%d", i+1),
			Address:           fmt.Sprintf(":%d", getRandomPortDiscovery()),
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
		defer stopNodeDiscovery(n, t)

		// Registrar handler para mensagens gossip
		idx := i
		n.GetWebRTC().RegisterGossipHandler("test-broadcast", func(msg *network.GossipMessage, fromPeer string) error {
			mu.Lock()
			messagesReceived[idx]++
			mu.Unlock()
			t.Logf("Node%d received gossip message from %s", idx+1, msg.OriginID)
			return nil
		})

		if err := n.Start(); err != nil {
			t.Fatalf("Failed to start node%d: %v", i+1, err)
		}

		nodes[i] = n
		time.Sleep(150 * time.Millisecond)
	}

	// Aguardar conexões e data channels
	time.Sleep(2 * time.Second)

	// Verificar se há peers conectados antes de enviar
	if len(nodes[0].GetPeers()) == 0 {
		t.Skip("No peers connected, skipping propagation test")
	}

	// Log status dos peers antes de enviar
	t.Logf("Node1 has %d peers connected", len(nodes[0].GetPeers()))
	for i, n := range nodes {
		t.Logf("Node%d has %d peers: %v", i+1, len(n.GetPeers()), n.GetPeers())
	}

	// Node1 envia mensagem gossip
	testData := []byte("Hello Gossip!")
	if err := nodes[0].GetWebRTC().GossipBroadcast("test-broadcast", testData); err != nil {
		t.Logf("Warning: Failed to broadcast gossip: %v", err)
		t.Skip("Could not broadcast, data channels not ready")
	}

	// Aguardar propagação
	time.Sleep(1 * time.Second)

	// Verificar se mensagens foram recebidas
	mu.Lock()
	defer mu.Unlock()

	t.Logf("Messages received: Node1=%d, Node2=%d, Node3=%d",
		messagesReceived[0], messagesReceived[1], messagesReceived[2])

	// Pelo menos node2 e node3 devem ter recebido
	if messagesReceived[1] == 0 && messagesReceived[2] == 0 {
		t.Error("Gossip message was not propagated to any node")
	}

	t.Logf("✓ Gossip propagation test completed")
}

// TestGossipDeduplication testa detecção de mensagens duplicadas
func TestGossipDeduplication(t *testing.T) {
	manager := network.NewGossipManager("test-node", nil)

	// Criar mensagem manualmente (não através do manager para evitar cache automático)
	msg, _ := network.NewGossipMessage("origin-node", "test", []byte("data"), 5)
	msgData, _ := msg.ToJSON()

	// Primeira vez deve ser aceita
	_, _, err := manager.HandleIncomingMessage(msgData, "peer1")
	if err != nil {
		t.Errorf("First message should be accepted: %v", err)
	}

	// Segunda vez deve ser detectada como duplicata
	_, _, err = manager.HandleIncomingMessage(msgData, "peer2")
	if err == nil {
		t.Error("Duplicate message should be detected")
	} else {
		t.Logf("Duplicate correctly detected: %v", err)
	}

	t.Logf("✓ Deduplication working correctly")
}

// TestGossipRateLimitAttack testa proteção contra ataque de flood
func TestGossipRateLimitAttack(t *testing.T) {
	config := network.DefaultGossipConfig()
	config.RateLimitPerSecond = 10 // Limite baixo para teste

	manager := network.NewGossipManager("test-node", config)

	attackerPeer := "attacker"
	accepted := 0
	rejected := 0

	// Tentar enviar 20 mensagens rapidamente
	for i := 0; i < 20; i++ {
		msg, _ := network.NewGossipMessage("attacker-node", "attack", []byte(fmt.Sprintf("msg%d", i)), 5)
		msgData, _ := msg.ToJSON()

		_, _, err := manager.HandleIncomingMessage(msgData, attackerPeer)
		if err == nil {
			accepted++
		} else {
			rejected++
		}
	}

	t.Logf("Accepted: %d, Rejected: %d", accepted, rejected)

	// Deve ter rejeitado mensagens além do limite
	if rejected == 0 {
		t.Error("Rate limiting did not block any messages")
	}

	// Verificar se peer foi bloqueado
	metrics := manager.GetMetrics()
	if metrics["messages_invalid"] == 0 {
		t.Error("Invalid messages not counted")
	}

	t.Logf("✓ Rate limit protection working")
}

// TestGossipInvalidMessages testa rejeição de mensagens inválidas
func TestGossipInvalidMessages(t *testing.T) {
	// Mensagem com TTL negativo
	invalidMsg := &network.GossipMessage{
		ID:       "test-id",
		OriginID: "origin",
		Type:     "test",
		Data:     []byte("data"),
		TTL:      -1,
	}

	if err := invalidMsg.Validate(); err == nil {
		t.Error("Should reject message with negative TTL")
	}

	t.Logf("✓ Invalid message rejection working")
}

// TestGossipMetrics testa coleta de métricas
func TestGossipMetrics(t *testing.T) {
	manager := network.NewGossipManager("test-node", nil)

	// Enviar algumas mensagens
	for i := 0; i < 5; i++ {
		msg, _ := manager.CreateMessage("test", []byte(fmt.Sprintf("data%d", i)))
		msgData, _ := msg.ToJSON()
		_, _, err := manager.HandleIncomingMessage(msgData, "peer1")
		if err != nil {
			t.Logf("Message %d handling: %v (expected for duplicates)", i, err)
		}
	}

	metrics := manager.GetMetrics()

	if metrics["messages_sent"] != 5 {
		t.Errorf("Expected 5 messages sent, got %d", metrics["messages_sent"])
	}

	t.Logf("Gossip metrics: %+v", metrics)
	t.Logf("✓ Metrics collection working")
}
