package tests

import (
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"github.com/krakovia/blockchain/internal/config"
	"github.com/krakovia/blockchain/pkg/blockchain"
	"github.com/krakovia/blockchain/pkg/node"
	"github.com/krakovia/blockchain/pkg/wallet"
)

// checkSignalingServer verifica se o servidor de signaling está disponível
func checkSignalingServer(t *testing.T) {
	// Tentar conectar na porta 9000
	conn, err := net.DialTimeout("tcp", "localhost:9000", 2*time.Second)
	if err != nil {
		t.Skip("Skipping test: signaling server not available at localhost:9000. " +
			"Start the signaling server with: go run cmd/signaling/main.go")
		return
	}
	_ = conn.Close()
	fmt.Println("✓ Signaling server is available")
}

// TestNodeSynchronization testa se o Node2 sincroniza corretamente com o Node1
func TestNodeSynchronization(t *testing.T) {
	// Verificar se o servidor de signaling está disponível
	checkSignalingServer(t)

	// Criar diretórios temporários
	dbPath1 := "./test-data/test_sync_node1_db"
	dbPath2 := "./test-data/test_sync_node2_db"
	defer func() { _ = os.RemoveAll(dbPath1) }()
	defer func() { _ = os.RemoveAll(dbPath2) }()

	// Criar wallets
	w1, err := wallet.NewWallet()
	if err != nil {
		t.Fatalf("Failed to create wallet 1: %v", err)
	}

	w2, err := wallet.NewWallet()
	if err != nil {
		t.Fatalf("Failed to create wallet 2: %v", err)
	}

	// Criar bloco genesis (mesmo para ambos)
	genesisTx := blockchain.NewCoinbaseTransaction(w1.GetAddress(), 1000000, 0)
	genesisBlock := blockchain.GenesisBlockWithTimestamp(genesisTx, time.Now().Unix())

	chainConfig := blockchain.ChainConfig{
		BlockTime:         200 * time.Millisecond,
		MaxBlockSize:      1000,
		BlockReward:       50,
		MinValidatorStake: 100,
	}

	checkpointConfig := &config.CheckpointConfig{
		Enabled:      true,
		Interval:     10,
		KeepInMemory: 20,
		KeepOnDisk:   2,
		CSVDelimiter: ",",
		Compression:  false,
	}

	// Criar Node1
	t.Log("Creating Node1...")
	node1Config := node.Config{
		ID:                "sync-test-node1",
		Address:           ":19001",
		DBPath:            dbPath1,
		SignalingServer:   "ws://localhost:9000/ws",
		MaxPeers:          50,
		MinPeers:          1,
		DiscoveryInterval: 5,
		Wallet:            w1,
		GenesisBlock:      genesisBlock,
		ChainConfig:       chainConfig,
		CheckpointConfig:  checkpointConfig,
		InitialStakeAddr:  w1.GetAddress(),
		InitialStake:      1000,
	}

	node1, err := node.NewNode(node1Config)
	if err != nil {
		t.Fatalf("Failed to create node1: %v", err)
	}
	defer func() { _ = node1.Stop() }()

	if err := node1.Start(); err != nil {
		t.Fatalf("Failed to start node1: %v", err)
	}

	// Minerar alguns blocos no Node1
	t.Log("Mining 10 blocks on Node1...")
	for i := 0; i < 10; i++ {
		block, err := node1.GetMiner().TryMineBlock()
		if err != nil {
			t.Fatalf("Failed to mine block %d on node1: %v", i+1, err)
		}

		if err := node1.GetChain().AddBlock(block); err != nil {
			t.Fatalf("Failed to add block %d to node1 chain: %v", i+1, err)
		}

		if err := blockchain.SaveBlockToDB(node1.GetDB(), block); err != nil {
			t.Fatalf("Failed to save block %d to node1 disk: %v", i+1, err)
		}

		t.Logf("Node1: Mined block %d (height: %d)", i+1, block.Header.Height)
		time.Sleep(300 * time.Millisecond)
	}

	node1Height := node1.GetChainHeight()
	if node1Height != 10 {
		t.Errorf("Expected node1 height 10, got %d", node1Height)
	}
	t.Logf("Node1 final height: %d ✓", node1Height)

	// Aguardar um pouco antes de criar Node2
	time.Sleep(1 * time.Second)

	// Criar Node2
	t.Log("Creating Node2...")
	node2Config := node.Config{
		ID:                "sync-test-node2",
		Address:           ":19002",
		DBPath:            dbPath2,
		SignalingServer:   "ws://localhost:9000/ws",
		MaxPeers:          50,
		MinPeers:          1,
		DiscoveryInterval: 5,
		Wallet:            w2,
		GenesisBlock:      genesisBlock,
		ChainConfig:       chainConfig,
		CheckpointConfig:  checkpointConfig,
	}

	node2, err := node.NewNode(node2Config)
	if err != nil {
		t.Fatalf("Failed to create node2: %v", err)
	}
	defer func() { _ = node2.Stop() }()

	// Verificar altura inicial do Node2 (deve ser 0)
	initialHeight2 := node2.GetChainHeight()
	if initialHeight2 != 0 {
		t.Errorf("Expected node2 initial height 0, got %d", initialHeight2)
	}

	if err := node2.Start(); err != nil {
		t.Fatalf("Failed to start node2: %v", err)
	}

	// Aguardar sincronização (com timeout de 30s)
	t.Log("Waiting for Node2 to synchronize with Node1...")
	syncTimeout := time.After(30 * time.Second)
	syncTicker := time.NewTicker(500 * time.Millisecond)
	defer syncTicker.Stop()

	synced := false
	for !synced {
		select {
		case <-syncTimeout:
			t.Fatalf("Timeout: Node2 failed to sync with Node1 after 30s (node2 height: %d, node1 height: %d)",
				node2.GetChainHeight(), node1.GetChainHeight())
		case <-syncTicker.C:
			node1CurrentHeight := node1.GetChainHeight()
			node2CurrentHeight := node2.GetChainHeight()

			t.Logf("Sync progress: Node1=%d, Node2=%d", node1CurrentHeight, node2CurrentHeight)

			// Considerar sincronizado quando Node2 alcançar ou superar a altura inicial do Node1
			if node2CurrentHeight >= node1Height {
				synced = true
				t.Logf("Node2 synchronized! Height: %d ✓", node2CurrentHeight)
			}
		}
	}

	// Verificar que Node2 sincronizou corretamente
	finalHeight2 := node2.GetChainHeight()
	if finalHeight2 < node1Height {
		t.Errorf("Node2 did not fully sync: expected at least %d, got %d", node1Height, finalHeight2)
	}

	// Verificar se os peers estão conectados
	node1Peers := node1.GetPeers()
	node2Peers := node2.GetPeers()

	if len(node1Peers) == 0 {
		t.Error("Node1 has no peers connected")
	} else {
		t.Logf("Node1 has %d peer(s) connected ✓", len(node1Peers))
	}

	if len(node2Peers) == 0 {
		t.Error("Node2 has no peers connected")
	} else {
		t.Logf("Node2 has %d peer(s) connected ✓", len(node2Peers))
	}

	t.Log("✓ Node synchronization test passed!")
}

// TestNodeSynchronizationWithMoreBlocks testa sincronização com mais blocos (25 blocos)
func TestNodeSynchronizationWithMoreBlocks(t *testing.T) {
	// Verificar se o servidor de signaling está disponível
	checkSignalingServer(t)

	// Criar diretórios temporários
	dbPath1 := "./test-data/test_sync_more_node1_db"
	dbPath2 := "./test-data/test_sync_more_node2_db"
	defer func() { _ = os.RemoveAll(dbPath1) }()
	defer func() { _ = os.RemoveAll(dbPath2) }()

	// Criar wallets
	w1, err := wallet.NewWallet()
	if err != nil {
		t.Fatalf("Failed to create wallet 1: %v", err)
	}

	w2, err := wallet.NewWallet()
	if err != nil {
		t.Fatalf("Failed to create wallet 2: %v", err)
	}

	// Criar bloco genesis
	genesisTx := blockchain.NewCoinbaseTransaction(w1.GetAddress(), 1000000, 0)
	genesisBlock := blockchain.GenesisBlockWithTimestamp(genesisTx, time.Now().Unix())

	chainConfig := blockchain.ChainConfig{
		BlockTime:         200 * time.Millisecond,
		MaxBlockSize:      1000,
		BlockReward:       50,
		MinValidatorStake: 100,
	}

	checkpointConfig := &config.CheckpointConfig{
		Enabled:      true,
		Interval:     10,
		KeepInMemory: 20,
		KeepOnDisk:   2,
		CSVDelimiter: ",",
		Compression:  false,
	}

	// Criar Node1
	t.Log("Creating Node1...")
	node1Config := node.Config{
		ID:                "sync-test-more-node1",
		Address:           ":19011",
		DBPath:            dbPath1,
		SignalingServer:   "ws://localhost:9000/ws",
		MaxPeers:          50,
		MinPeers:          1,
		DiscoveryInterval: 5,
		Wallet:            w1,
		GenesisBlock:      genesisBlock,
		ChainConfig:       chainConfig,
		CheckpointConfig:  checkpointConfig,
		InitialStakeAddr:  w1.GetAddress(),
		InitialStake:      1000,
	}

	node1, err := node.NewNode(node1Config)
	if err != nil {
		t.Fatalf("Failed to create node1: %v", err)
	}
	defer func() { _ = node1.Stop() }()

	if err := node1.Start(); err != nil {
		t.Fatalf("Failed to start node1: %v", err)
	}

	// Minerar 25 blocos no Node1
	blocksToMine := 25
	t.Logf("Mining %d blocks on Node1...", blocksToMine)
	for i := 0; i < blocksToMine; i++ {
		block, err := node1.GetMiner().TryMineBlock()
		if err != nil {
			t.Fatalf("Failed to mine block %d on node1: %v", i+1, err)
		}

		if err := node1.GetChain().AddBlock(block); err != nil {
			t.Fatalf("Failed to add block %d to node1 chain: %v", i+1, err)
		}

		if err := blockchain.SaveBlockToDB(node1.GetDB(), block); err != nil {
			t.Fatalf("Failed to save block %d to node1 disk: %v", i+1, err)
		}

		if (i+1)%5 == 0 {
			t.Logf("Node1: Mined %d blocks...", i+1)
		}
		time.Sleep(300 * time.Millisecond)
	}

	node1Height := node1.GetChainHeight()
	if node1Height != uint64(blocksToMine) {
		t.Errorf("Expected node1 height %d, got %d", blocksToMine, node1Height)
	}
	t.Logf("Node1 final height: %d ✓", node1Height)

	// Aguardar um pouco antes de criar Node2
	time.Sleep(1 * time.Second)

	// Criar Node2
	t.Log("Creating Node2...")
	node2Config := node.Config{
		ID:                "sync-test-more-node2",
		Address:           ":19012",
		DBPath:            dbPath2,
		SignalingServer:   "ws://localhost:9000/ws",
		MaxPeers:          50,
		MinPeers:          1,
		DiscoveryInterval: 5,
		Wallet:            w2,
		GenesisBlock:      genesisBlock,
		ChainConfig:       chainConfig,
		CheckpointConfig:  checkpointConfig,
	}

	node2, err := node.NewNode(node2Config)
	if err != nil {
		t.Fatalf("Failed to create node2: %v", err)
	}
	defer func() { _ = node2.Stop() }()

	if err := node2.Start(); err != nil {
		t.Fatalf("Failed to start node2: %v", err)
	}

	// Aguardar sincronização (com timeout de 45s para 25 blocos)
	t.Log("Waiting for Node2 to synchronize with Node1...")
	syncTimeout := time.After(45 * time.Second)
	syncTicker := time.NewTicker(500 * time.Millisecond)
	defer syncTicker.Stop()

	synced := false
	for !synced {
		select {
		case <-syncTimeout:
			t.Fatalf("Timeout: Node2 failed to sync with Node1 after 45s (node2 height: %d, node1 height: %d)",
				node2.GetChainHeight(), node1.GetChainHeight())
		case <-syncTicker.C:
			node2CurrentHeight := node2.GetChainHeight()

			if node2CurrentHeight%5 == 0 && node2CurrentHeight > 0 {
				t.Logf("Sync progress: Node2 height=%d", node2CurrentHeight)
			}

			if node2CurrentHeight >= node1Height {
				synced = true
				t.Logf("Node2 synchronized! Height: %d ✓", node2CurrentHeight)
			}
		}
	}

	// Verificar sincronização
	finalHeight2 := node2.GetChainHeight()
	if finalHeight2 < node1Height {
		t.Errorf("Node2 did not fully sync: expected at least %d, got %d", node1Height, finalHeight2)
	}

	t.Log("✓ Node synchronization with more blocks test passed!")
}
