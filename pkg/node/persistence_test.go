package node

import (
	"os"
	"testing"
	"time"

	"github.com/krakovia/blockchain/internal/config"
	"github.com/krakovia/blockchain/pkg/blockchain"
	"github.com/krakovia/blockchain/pkg/wallet"
	"github.com/syndtr/goleveldb/leveldb"
)

// TestBlockchainPersistence testa se a blockchain é persistida e carregada corretamente
func TestBlockchainPersistence(t *testing.T) {
	// Criar diretório temporário para o teste
	dbPath := "./test_persistence_db"
	defer os.RemoveAll(dbPath)

	// Criar wallet de teste
	w, err := wallet.NewWallet()
	if err != nil {
		t.Fatalf("Failed to generate wallet: %v", err)
	}

	// Criar transação genesis
	genesisTx := blockchain.NewCoinbaseTransaction(w.GetAddress(), 1000000, 0)

	// Criar bloco genesis
	genesisBlock := blockchain.GenesisBlockWithTimestamp(genesisTx, time.Now().Unix())

	// Configurações da chain
	chainConfig := blockchain.ChainConfig{
		BlockTime:         200 * time.Millisecond,
		MaxBlockSize:      1000,
		BlockReward:       50,
		MinValidatorStake: 100,
	}

	// Configuração do checkpoint
	checkpointConfig := &config.CheckpointConfig{
		Enabled:      true,
		Interval:     10,
		KeepInMemory: 20,
		KeepOnDisk:   2,
		CSVDelimiter: ",",
		Compression:  false,
	}

	// Criar primeiro node
	nodeConfig1 := Config{
		ID:               "test-node-1",
		Address:          ":9999",
		DBPath:           dbPath,
		SignalingServer:  "ws://localhost:9000/ws",
		MaxPeers:         50,
		MinPeers:         0,
		Wallet:           w,
		GenesisBlock:     genesisBlock,
		ChainConfig:      chainConfig,
		CheckpointConfig: checkpointConfig,
		InitialStakeAddr: w.GetAddress(),
		InitialStake:     1000,
	}

	node1, err := NewNode(nodeConfig1)
	if err != nil {
		t.Fatalf("Failed to create first node: %v", err)
	}

	// Verificar altura inicial (deve ser 0 - apenas genesis)
	initialHeight := node1.GetChainHeight()
	if initialHeight != 0 {
		t.Errorf("Expected initial height 0, got %d", initialHeight)
	}

	// Minerar alguns blocos
	t.Log("Mining 5 blocks...")
	for i := 0; i < 5; i++ {
		block, err := node1.miner.TryMineBlock()
		if err != nil {
			t.Fatalf("Failed to mine block %d: %v", i+1, err)
		}

		// Adicionar bloco à chain
		if err := node1.chain.AddBlock(block); err != nil {
			t.Fatalf("Failed to add block %d to chain: %v", i+1, err)
		}

		// Salvar bloco no disco (simular o que acontece no callback OnBlockCreated)
		if err := blockchain.SaveBlockToDB(node1.db, block); err != nil {
			t.Fatalf("Failed to save block %d to disk: %v", i+1, err)
		}

		t.Logf("Mined and saved block %d (height: %d)", i+1, block.Header.Height)
		time.Sleep(300 * time.Millisecond) // Aguardar tempo mínimo entre blocos
	}

	// Verificar altura após mineração
	heightAfterMining := node1.GetChainHeight()
	expectedHeight := uint64(5)
	if heightAfterMining != expectedHeight {
		t.Errorf("Expected height %d after mining, got %d", expectedHeight, heightAfterMining)
	}

	// Obter saldo antes de parar
	balanceBefore := node1.GetBalance()
	t.Logf("Balance after mining: %d", balanceBefore)

	// Parar o primeiro node
	t.Log("Stopping first node...")
	if err := node1.Stop(); err != nil {
		t.Fatalf("Failed to stop first node: %v", err)
	}

	// Aguardar um pouco para garantir que tudo foi fechado
	time.Sleep(500 * time.Millisecond)

	// Verificar se os dados foram salvos no banco de dados
	t.Log("Verifying data was saved to disk...")
	db, err := leveldb.OpenFile(dbPath, nil)
	if err != nil {
		t.Fatalf("Failed to open database for verification: %v", err)
	}

	// Verificar altura salva
	chainHeightData, err := db.Get([]byte("metadata-chain-height"), nil)
	if err != nil {
		t.Fatalf("Failed to get chain height from DB: %v", err)
	}
	t.Logf("Chain height in DB: %s", string(chainHeightData))

	// Verificar se os blocos foram salvos
	for height := uint64(1); height <= expectedHeight; height++ {
		block, err := blockchain.LoadBlockFromDB(db, height)
		if err != nil {
			t.Errorf("Failed to load block %d from DB: %v", height, err)
		} else {
			t.Logf("Successfully loaded block %d from DB (hash: %s)", height, block.Hash[:8])
		}
	}
	db.Close()

	// Aguardar um pouco
	time.Sleep(500 * time.Millisecond)

	// Criar segundo node com mesma configuração (simular reinicialização)
	t.Log("Creating second node (simulating restart)...")
	nodeConfig2 := Config{
		ID:               "test-node-2",
		Address:          ":9998",
		DBPath:           dbPath,
		SignalingServer:  "ws://localhost:9000/ws",
		MaxPeers:         50,
		MinPeers:         0,
		Wallet:           w,
		GenesisBlock:     genesisBlock,
		ChainConfig:      chainConfig,
		CheckpointConfig: checkpointConfig,
		InitialStakeAddr: w.GetAddress(),
		InitialStake:     1000,
	}

	node2, err := NewNode(nodeConfig2)
	if err != nil {
		t.Fatalf("Failed to create second node: %v", err)
	}
	defer node2.Stop()

	// Verificar se a altura foi restaurada
	heightAfterRestart := node2.GetChainHeight()
	if heightAfterRestart != expectedHeight {
		t.Errorf("Expected height %d after restart, got %d", expectedHeight, heightAfterRestart)
	}
	t.Logf("Height after restart: %d (expected: %d) ✓", heightAfterRestart, expectedHeight)

	// Verificar se o saldo foi restaurado
	balanceAfter := node2.GetBalance()
	if balanceAfter != balanceBefore {
		t.Errorf("Expected balance %d after restart, got %d", balanceBefore, balanceAfter)
	}
	t.Logf("Balance after restart: %d (expected: %d) ✓", balanceAfter, balanceBefore)

	// Verificar se os blocos estão acessíveis
	for height := uint64(1); height <= expectedHeight; height++ {
		block, exists := node2.chain.GetBlockByHeight(height)
		if !exists {
			t.Errorf("Block at height %d not found in chain after restart", height)
		} else {
			t.Logf("Block %d accessible in chain after restart (hash: %s) ✓", height, block.Hash[:8])
		}
	}

	// Minerar mais blocos no node restaurado
	t.Log("Mining 3 more blocks on restored node...")
	for i := 0; i < 3; i++ {
		block, err := node2.miner.TryMineBlock()
		if err != nil {
			t.Fatalf("Failed to mine block on restored node: %v", err)
		}

		if err := node2.chain.AddBlock(block); err != nil {
			t.Fatalf("Failed to add block to restored chain: %v", err)
		}

		if err := blockchain.SaveBlockToDB(node2.db, block); err != nil {
			t.Fatalf("Failed to save block to disk on restored node: %v", err)
		}

		t.Logf("Mined block %d on restored node (height: %d)", i+1, block.Header.Height)
		time.Sleep(300 * time.Millisecond)
	}

	// Verificar altura final
	finalHeight := node2.GetChainHeight()
	expectedFinalHeight := expectedHeight + 3
	if finalHeight != expectedFinalHeight {
		t.Errorf("Expected final height %d, got %d", expectedFinalHeight, finalHeight)
	}
	t.Logf("Final height: %d (expected: %d) ✓", finalHeight, expectedFinalHeight)

	t.Log("✓ All persistence tests passed!")
}

// TestBlockchainPersistenceWithMultipleRestarts testa múltiplas reinicializações
func TestBlockchainPersistenceWithMultipleRestarts(t *testing.T) {
	dbPath := "./test_multiple_restarts_db"
	defer os.RemoveAll(dbPath)

	w, err := wallet.NewWallet()
	if err != nil {
		t.Fatalf("Failed to generate wallet: %v", err)
	}

	genesisTx := blockchain.NewCoinbaseTransaction(w.GetAddress(), 1000000, 0)
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

	// Simular 3 reinicializações, minerando blocos em cada uma
	totalBlocksMined := 0
	restarts := 3
	blocksPerRestart := 3

	for restart := 0; restart < restarts; restart++ {
		t.Logf("=== Restart %d/%d ===", restart+1, restarts)

		nodeConfig := Config{
			ID:               "test-node",
			Address:          ":9997",
			DBPath:           dbPath,
			SignalingServer:  "ws://localhost:9000/ws",
			MaxPeers:         50,
			MinPeers:         0,
			Wallet:           w,
			GenesisBlock:     genesisBlock,
			ChainConfig:      chainConfig,
			CheckpointConfig: checkpointConfig,
			InitialStakeAddr: w.GetAddress(),
			InitialStake:     1000,
		}

		node, err := NewNode(nodeConfig)
		if err != nil {
			t.Fatalf("Failed to create node on restart %d: %v", restart+1, err)
		}

		// Verificar altura esperada
		expectedHeight := uint64(totalBlocksMined)
		currentHeight := node.GetChainHeight()
		if currentHeight != expectedHeight {
			t.Errorf("Restart %d: Expected height %d, got %d", restart+1, expectedHeight, currentHeight)
		}
		t.Logf("Restart %d: Current height is %d (expected: %d) ✓", restart+1, currentHeight, expectedHeight)

		// Minerar alguns blocos
		for i := 0; i < blocksPerRestart; i++ {
			block, err := node.miner.TryMineBlock()
			if err != nil {
				t.Fatalf("Restart %d: Failed to mine block: %v", restart+1, err)
			}

			if err := node.chain.AddBlock(block); err != nil {
				t.Fatalf("Restart %d: Failed to add block: %v", restart+1, err)
			}

			if err := blockchain.SaveBlockToDB(node.db, block); err != nil {
				t.Fatalf("Restart %d: Failed to save block: %v", restart+1, err)
			}

			totalBlocksMined++
			t.Logf("Restart %d: Mined block %d (height: %d)", restart+1, i+1, block.Header.Height)
			time.Sleep(300 * time.Millisecond)
		}

		// Parar node
		if err := node.Stop(); err != nil {
			t.Fatalf("Failed to stop node on restart %d: %v", restart+1, err)
		}

		time.Sleep(500 * time.Millisecond)
	}

	t.Logf("✓ Successfully completed %d restarts with %d total blocks mined!", restarts, totalBlocksMined)
}

// TestEmptyDatabaseLoad testa o carregamento quando o banco de dados está vazio
func TestEmptyDatabaseLoad(t *testing.T) {
	dbPath := "./test_empty_db"
	defer os.RemoveAll(dbPath)

	w, err := wallet.NewWallet()
	if err != nil {
		t.Fatalf("Failed to generate wallet: %v", err)
	}

	genesisTx := blockchain.NewCoinbaseTransaction(w.GetAddress(), 1000000, 0)
	genesisBlock := blockchain.GenesisBlockWithTimestamp(genesisTx, time.Now().Unix())

	chainConfig := blockchain.DefaultChainConfig()

	nodeConfig := Config{
		ID:              "test-node",
		Address:         ":9996",
		DBPath:          dbPath,
		SignalingServer: "ws://localhost:9000/ws",
		MaxPeers:        50,
		MinPeers:        0,
		Wallet:          w,
		GenesisBlock:    genesisBlock,
		ChainConfig:     chainConfig,
	}

	node, err := NewNode(nodeConfig)
	if err != nil {
		t.Fatalf("Failed to create node with empty DB: %v", err)
	}
	defer node.Stop()

	// Deve começar com altura 0 (apenas genesis)
	height := node.GetChainHeight()
	if height != 0 {
		t.Errorf("Expected height 0 for new node, got %d", height)
	}

	t.Log("✓ Empty database load test passed!")
}
