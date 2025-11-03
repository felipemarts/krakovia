package blockchain

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/krakovia/blockchain/pkg/wallet"
)

// Helper: cria um bloco gênesis para testes
func createTestGenesis(t *testing.T, allocations map[string]uint64) *Block {
	t.Helper()

	// Para simplificar, usa apenas a primeira alocação como genesis
	// Em um genesis real, você pode ter múltiplas alocações
	var addr string
	var amount uint64
	for a, amt := range allocations {
		addr = a
		amount = amt
		break
	}

	// Cria transação coinbase para o endereço
	tx := NewCoinbaseTransaction(addr, amount, 0)

	// Cria bloco gênesis
	genesis := GenesisBlock(tx)

	return genesis
}

// Helper: cria um nó de teste
func createTestNode(t *testing.T, id string, genesis *Block) (*Node, *wallet.Wallet) {
	t.Helper()

	w, err := wallet.NewWallet()
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	config := DefaultChainConfig()
	chain, err := NewChain(genesis, config)
	if err != nil {
		t.Fatalf("Failed to create chain: %v", err)
	}

	mempool := NewMempool()

	node := NewNode(id, w, chain, mempool)

	return node, w
}

// Helper: conecta nós em rede completa (todos conectados entre si)
func connectNodesFullMesh(nodes []*Node) {
	for i := 0; i < len(nodes); i++ {
		for j := i + 1; j < len(nodes); j++ {
			nodes[i].ConnectPeer(nodes[j])
			nodes[j].ConnectPeer(nodes[i])
		}
	}
}

// Helper: aguarda convergência (todas as chains na mesma altura)
func waitForConvergence(t *testing.T, nodes []*Node, targetHeight uint64, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		allConverged := true

		for _, node := range nodes {
			if node.GetChain().GetHeight() < targetHeight {
				allConverged = false
				break
			}
		}

		if allConverged {
			return
		}

		time.Sleep(50 * time.Millisecond)
	}

	// Timeout - mostra estado atual
	t.Logf("Convergence timeout. Node states:")
	for _, node := range nodes {
		stats := node.GetNodeStats()
		t.Logf("  %s: height=%d, stake=%d, balance=%d",
			stats.ID, stats.Height, stats.Stake, stats.Balance)
	}

	t.Fatalf("Nodes did not converge to height %d within %v", targetHeight, timeout)
}

// Teste 1: Criação básica de nó
func TestNodeCreation(t *testing.T) {
	allocations := map[string]uint64{
		"genesis_addr": 10000,
	}
	genesis := createTestGenesis(t, allocations)

	node, w := createTestNode(t, "node1", genesis)

	if node.GetID() != "node1" {
		t.Errorf("Expected ID 'node1', got '%s'", node.GetID())
	}

	if node.GetChain().GetHeight() != 0 {
		t.Errorf("Expected height 0, got %d", node.GetChain().GetHeight())
	}

	if node.GetMiner().GetAddress() != w.GetAddress() {
		t.Errorf("Miner address mismatch")
	}
}

// Teste 2: Conexão entre peers
func TestNodePeerConnection(t *testing.T) {
	allocations := map[string]uint64{
		"genesis_addr": 10000,
	}
	genesis := createTestGenesis(t, allocations)

	node1, _ := createTestNode(t, "node1", genesis)
	node2, _ := createTestNode(t, "node2", genesis)
	node3, _ := createTestNode(t, "node3", genesis)

	// Conecta em linha: node1 <-> node2 <-> node3
	node1.ConnectPeer(node2)
	node2.ConnectPeer(node1)
	node2.ConnectPeer(node3)
	node3.ConnectPeer(node2)

	if len(node1.GetPeers()) != 1 {
		t.Errorf("Node1 should have 1 peer, got %d", len(node1.GetPeers()))
	}

	if len(node2.GetPeers()) != 2 {
		t.Errorf("Node2 should have 2 peers, got %d", len(node2.GetPeers()))
	}

	if len(node3.GetPeers()) != 1 {
		t.Errorf("Node3 should have 1 peer, got %d", len(node3.GetPeers()))
	}
}

// Teste 3: Propagação de transação
func TestTransactionPropagation(t *testing.T) {
	allocations := map[string]uint64{
		"genesis_addr": 10000,
	}
	genesis := createTestGenesis(t, allocations)

	node1, w1 := createTestNode(t, "node1", genesis)
	node2, _ := createTestNode(t, "node2", genesis)
	node3, _ := createTestNode(t, "node3", genesis)

	// Dá saldo inicial para node1
	addr1 := w1.GetAddress()
	genesis.Transactions[0] = NewCoinbaseTransaction(addr1, 10000, 0)
	hash, _ := genesis.CalculateHash()
	genesis.Hash = hash

	// Recria chains com novo gênesis
	config := DefaultChainConfig()
	node1.chain, _ = NewChain(genesis, config)
	node2.chain, _ = NewChain(genesis, config)
	node3.chain, _ = NewChain(genesis, config)

	// Conecta em malha completa
	connectNodesFullMesh([]*Node{node1, node2, node3})

	// Node1 cria transação
	tx, err := node1.CreateTransaction(node2.GetMiner().GetAddress(), 100, 1, "test")
	if err != nil {
		t.Fatalf("Failed to create transaction: %v", err)
	}

	// Aguarda propagação
	time.Sleep(100 * time.Millisecond)

	// Verifica que todos os nós têm a transação
	if _, exists := node1.GetMempool().GetTransaction(tx.ID); !exists {
		t.Error("Node1 should have the transaction")
	}
	if _, exists := node2.GetMempool().GetTransaction(tx.ID); !exists {
		t.Error("Node2 should have the transaction")
	}
	if _, exists := node3.GetMempool().GetTransaction(tx.ID); !exists {
		t.Error("Node3 should have the transaction")
	}
}

// Teste 4: Propagação de bloco
func TestBlockPropagation(t *testing.T) {
	w1, _ := wallet.NewWallet()
	allocations := map[string]uint64{
		w1.GetAddress(): 10000,
	}
	genesis := createTestGenesis(t, allocations)

	config := DefaultChainConfig()
	chain1, _ := NewChain(genesis, config)
	chain2, _ := NewChain(genesis, config)

	mempool1 := NewMempool()
	mempool2 := NewMempool()

	node1 := NewNode("node1", w1, chain1, mempool1)
	node2 := NewNode("node2", w1, chain2, mempool2)

	// Cria bloco bootstrap com stake
	stakeData := NewStakeData(1000)
	dataStr, _ := stakeData.Serialize()
	stakeTx := NewTransaction(w1.GetAddress(), w1.GetAddress(), 1000, 1, 0, dataStr)
	_ = stakeTx.Sign(w1)

	time.Sleep(250 * time.Millisecond)
	coinbase := NewCoinbaseTransaction(w1.GetAddress(), config.BlockReward, 1)
	txs := TransactionSlice{coinbase, stakeTx}
	block1 := NewBlock(1, genesis.Hash, txs, w1.GetAddress())
	hash, _ := block1.CalculateHash()
	block1.Hash = hash
	_ = chain1.AddBlock(block1)
	_ = chain2.AddBlock(block1)

	// Conecta nós
	node1.ConnectPeer(node2)
	node2.ConnectPeer(node1)

	// Node1 minera um segundo bloco
	time.Sleep(250 * time.Millisecond)
	block2, err := node1.GetMiner().TryMineBlock()
	if err != nil {
		t.Fatalf("Failed to mine block: %v", err)
	}

	// Adiciona à chain do node1
	if err := node1.GetChain().AddBlock(block2); err != nil {
		t.Fatalf("Failed to add block to node1: %v", err)
	}

	// Propaga
	node1.BroadcastBlock(block2)

	// Aguarda propagação
	time.Sleep(100 * time.Millisecond)

	// Verifica que node2 recebeu o bloco 2
	if node2.GetChain().GetHeight() != 2 {
		t.Errorf("Node2 should have height 2, got %d", node2.GetChain().GetHeight())
	}
}

// Teste 5: Mineração com um validador
func TestSingleValidatorMining(t *testing.T) {
	w, err := wallet.NewWallet()
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}
	addr := w.GetAddress()

	allocations := map[string]uint64{
		addr: 10000,
	}
	genesis := createTestGenesis(t, allocations)

	config := DefaultChainConfig()
	config.BlockTime = 200 * time.Millisecond

	chain, err := NewChain(genesis, config)
	if err != nil {
		t.Fatalf("Failed to create chain: %v", err)
	}

	mempool := NewMempool()
	node := NewNode("validator1", w, chain, mempool)

	// Cria uma transação de stake e adiciona ao mempool
	stakeData := NewStakeData(1000)
	dataStr, _ := stakeData.Serialize()
	stakeTx := NewTransaction(addr, addr, 1000, 1, 0, dataStr)
	_ = stakeTx.Sign(w)
	_ = mempool.AddTransaction(stakeTx)

	// Cria manualmente o primeiro bloco com a transação de stake
	// (sem validação de "é minha vez" porque precisa do stake primeiro)
	time.Sleep(250 * time.Millisecond)

	coinbase := NewCoinbaseTransaction(addr, config.BlockReward, 1)
	txs := TransactionSlice{coinbase, stakeTx}

	block1 := NewBlock(1, genesis.Hash, txs, addr)
	hash, _ := block1.CalculateHash()
	block1.Hash = hash

	err = chain.AddBlock(block1)
	if err != nil {
		t.Fatalf("Failed to add first block with stake: %v", err)
	}

	mempool.RemoveTransactions([]string{stakeTx.ID})

	// Verifica que agora tem stake
	if node.GetStake() < 1000 {
		t.Fatalf("Expected stake >= 1000 after block, got %d", node.GetStake())
	}

	// Inicia mineração automática
	node.StartMining()
	defer node.StopMining()

	// Aguarda alguns blocos
	time.Sleep(500 * time.Millisecond)

	if chain.GetHeight() < 3 {
		t.Errorf("Expected at least 3 blocks mined, got %d", chain.GetHeight())
	}
}

// Teste 6: Mineração com múltiplos validadores
func TestMultipleValidatorMining(t *testing.T) {
	// Cria 3 validadores com stakes diferentes
	validators := make([]*Node, 3)
	wallets := make([]*wallet.Wallet, 3)

	// Cria wallets
	for i := 0; i < 3; i++ {
		w, err := wallet.NewWallet()
		if err != nil {
			t.Fatalf("Failed to create wallet %d: %v", i, err)
		}
		wallets[i] = w
	}

	// Aloca saldo inicial apenas para o primeiro (vai distribuir via transferências)
	allocations := map[string]uint64{
		wallets[0].GetAddress(): 50000,
	}

	genesis := createTestGenesis(t, allocations)
	config := DefaultChainConfig()
	config.BlockTime = 200 * time.Millisecond

	// Cria nós
	for i := 0; i < 3; i++ {
		chain, err := NewChain(genesis, config)
		if err != nil {
			t.Fatalf("Failed to create chain: %v", err)
		}

		mempool := NewMempool()
		validators[i] = NewNode(fmt.Sprintf("validator%d", i+1), wallets[i], chain, mempool)
	}

	// Conecta em malha completa
	connectNodesFullMesh(validators)

	// Cria bloco bootstrap:
	// 1. Coinbase para validator 0
	// 2. Transferências de validator 0 para os outros
	// 3. Stakes de todos os validadores
	stakes := []uint64{1000, 2000, 3000}

	time.Sleep(250 * time.Millisecond)
	coinbase := NewCoinbaseTransaction(wallets[0].GetAddress(), config.BlockReward, 1)
	txs := TransactionSlice{coinbase}

	// Transfere fundos para validators 1 e 2
	transferTx1 := NewTransaction(wallets[0].GetAddress(), wallets[1].GetAddress(), 10000, 1, 0, "")
	_ = transferTx1.Sign(wallets[0])
	txs = append(txs, transferTx1)

	transferTx2 := NewTransaction(wallets[0].GetAddress(), wallets[2].GetAddress(), 10000, 1, 1, "")
	_ = transferTx2.Sign(wallets[0])
	txs = append(txs, transferTx2)

	// Agora cada um faz stake
	// Validator 0 já usou nonces 0 e 1 nas transferências, então usa nonce 2
	// Validators 1 e 2 usam nonce 0 (primeira transação deles)
	stakeNonces := []uint64{2, 0, 0}
	for i := range validators {
		addr := wallets[i].GetAddress()
		stakeData := NewStakeData(stakes[i])
		dataStr, _ := stakeData.Serialize()
		stakeTx := NewTransaction(addr, addr, stakes[i], 1, stakeNonces[i], dataStr)
		_ = stakeTx.Sign(wallets[i])
		txs = append(txs, stakeTx)
	}

	block1 := NewBlock(1, genesis.Hash, txs, wallets[0].GetAddress())
	hash, _ := block1.CalculateHash()
	block1.Hash = hash

	// Adiciona o bloco em todos os nós
	for i := range validators {
		err := validators[i].GetChain().AddBlock(block1)
		if err != nil {
			t.Fatalf("Failed to add block to validator %d: %v", i, err)
		}
	}

	// Verifica que todos têm stake
	for i, node := range validators {
		stake := node.GetStake()
		height := node.GetChain().GetHeight()
		balance := node.GetBalance()
		t.Logf("Validator %d: height=%d, stake=%d, balance=%d", i, height, stake, balance)
		if stake < stakes[i] {
			t.Errorf("Validator %d: expected stake >= %d, got %d", i, stakes[i], stake)
		}
	}

	// Inicia mineração automática em todos
	for _, node := range validators {
		node.StartMining()
	}

	// Aguarda convergência em altura 10
	waitForConvergence(t, validators, 10, 10*time.Second)

	// Para mineração
	for _, node := range validators {
		node.StopMining()
	}

	// Verifica que todos têm a mesma chain
	height := validators[0].GetChain().GetHeight()
	lastHash := validators[0].GetChain().GetLastBlock().Hash

	for i := 1; i < len(validators); i++ {
		if validators[i].GetChain().GetHeight() != height {
			t.Errorf("Validator %d has different height: %d vs %d", i, validators[i].GetChain().GetHeight(), height)
		}

		if validators[i].GetChain().GetLastBlock().Hash != lastHash {
			t.Errorf("Validator %d has different last block hash", i)
		}
	}

	t.Logf("Converged at height %d with hash %s", height, lastHash[:16])
}

// Teste 7: Stake e Unstake durante mineração
func TestStakeUnstakeDuringMining(t *testing.T) {
	// Cria 2 validadores
	w1, err := wallet.NewWallet()
	if err != nil {
		t.Fatalf("Failed to create wallet 1: %v", err)
	}
	w2, err2 := wallet.NewWallet()
	if err2 != nil {
		t.Fatalf("Failed to create wallet 2: %v", err2)
	}

	allocations := map[string]uint64{
		w1.GetAddress(): 30000,
	}

	genesis := createTestGenesis(t, allocations)
	config := DefaultChainConfig()
	config.BlockTime = 200 * time.Millisecond

	// Cria nós
	chain1, _ := NewChain(genesis, config)
	chain2, _ := NewChain(genesis, config)

	mempool1 := NewMempool()
	mempool2 := NewMempool()

	node1 := NewNode("validator1", w1, chain1, mempool1)
	node2 := NewNode("validator2", w2, chain2, mempool2)

	// Conecta
	connectNodesFullMesh([]*Node{node1, node2})

	// Cria bloco bootstrap: transfere para node2 e faz stake de ambos
	time.Sleep(250 * time.Millisecond)
	coinbase := NewCoinbaseTransaction(w1.GetAddress(), config.BlockReward, 1)

	// Transfere fundos para node2
	transferTx := NewTransaction(w1.GetAddress(), w2.GetAddress(), 10000, 1, 0, "")
	_ = transferTx.Sign(w1)

	// Stakes
	stakeData1 := NewStakeData(2000)
	dataStr1, _ := stakeData1.Serialize()
	stakeTx1 := NewTransaction(w1.GetAddress(), w1.GetAddress(), 2000, 1, 1, dataStr1)
	_ = stakeTx1.Sign(w1)

	stakeData2 := NewStakeData(1000)
	dataStr2, _ := stakeData2.Serialize()
	stakeTx2 := NewTransaction(w2.GetAddress(), w2.GetAddress(), 1000, 1, 0, dataStr2)
	_ = stakeTx2.Sign(w2)

	txs := TransactionSlice{coinbase, transferTx, stakeTx1, stakeTx2}
	block1 := NewBlock(1, genesis.Hash, txs, w1.GetAddress())
	hash, _ := block1.CalculateHash()
	block1.Hash = hash
	_ = chain1.AddBlock(block1)
	_ = chain2.AddBlock(block1)

	// Inicia mineração
	node1.StartMining()
	node2.StartMining()

	// Aguarda alguns blocos
	waitForConvergence(t, []*Node{node1, node2}, 5, 5*time.Second)

	// Node1 faz unstake
	_, err = node1.CreateUnstakeTransaction(1000, 1)
	if err != nil {
		t.Fatalf("Failed to create unstake: %v", err)
	}

	// Node2 aumenta stake
	_, err = node2.CreateStakeTransaction(1000, 1)
	if err != nil {
		t.Fatalf("Failed to increase stake: %v", err)
	}

	// Aguarda mais blocos
	waitForConvergence(t, []*Node{node1, node2}, 10, 5*time.Second)

	node1.StopMining()
	node2.StopMining()

	// Verifica stakes finais
	stake1 := node1.GetStake()
	stake2 := node2.GetStake()

	t.Logf("Final stakes: node1=%d, node2=%d", stake1, stake2)

	if stake1 < 1000 {
		t.Errorf("Node1 stake should be ~1000 after unstake, got %d", stake1)
	}

	if stake2 < 2000 {
		t.Errorf("Node2 stake should be ~2000 after second stake, got %d", stake2)
	}

	// Verifica convergência
	if node1.GetChain().GetHeight() != node2.GetChain().GetHeight() {
		t.Error("Nodes did not converge to same height")
	}
}

// Teste 8: Sincronização de nó atrasado
func TestNodeSynchronization(t *testing.T) {
	w, err := wallet.NewWallet()
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}
	allocations := map[string]uint64{
		w.GetAddress(): 10000,
	}

	genesis := createTestGenesis(t, allocations)
	config := DefaultChainConfig()
	config.BlockTime = 200 * time.Millisecond

	// Node1 vai minerar sozinho
	chain1, _ := NewChain(genesis, config)
	mempool1 := NewMempool()
	node1 := NewNode("validator1", w, chain1, mempool1)

	// Cria bloco bootstrap com stake
	stakeData := NewStakeData(1000)
	dataStr, _ := stakeData.Serialize()
	stakeTx := NewTransaction(w.GetAddress(), w.GetAddress(), 1000, 1, 0, dataStr)
	_ = stakeTx.Sign(w)

	time.Sleep(250 * time.Millisecond)
	coinbase := NewCoinbaseTransaction(w.GetAddress(), config.BlockReward, 1)
	txs := TransactionSlice{coinbase, stakeTx}
	block1 := NewBlock(1, genesis.Hash, txs, w.GetAddress())
	hash, _ := block1.CalculateHash()
	block1.Hash = hash
	_ = chain1.AddBlock(block1)

	// Minera mais alguns blocos
	node1.StartMining()
	time.Sleep(1 * time.Second)
	node1.StopMining()

	// Aguarda um pouco para garantir que a mineração parou completamente
	time.Sleep(100 * time.Millisecond)

	height1 := node1.GetChain().GetHeight()
	t.Logf("Node1 mined to height %d", height1)

	// Cria node2 atrasado (só com gênesis)
	chain2, _ := NewChain(genesis, config)
	mempool2 := NewMempool()
	node2 := NewNode("validator2", w, chain2, mempool2)

	if node2.GetChain().GetHeight() != 0 {
		t.Fatal("Node2 should start at height 0")
	}

	// Conecta e sincroniza
	node2.ConnectPeer(node1)
	err = node2.SyncWithPeer(node1)
	if err != nil {
		t.Fatalf("Failed to sync: %v", err)
	}

	// Verifica que sincronizou (permite diferença de 1 devido a possível race)
	height2 := node2.GetChain().GetHeight()
	if height2 < height1 || height2 > height1+1 {
		t.Errorf("Node2 should be at height %d (±1) after sync, got %d", height1, height2)
	}

	// Verifica que têm a mesma chain
	hash1 := node1.GetChain().GetLastBlock().Hash
	hash2 := node2.GetChain().GetLastBlock().Hash

	if hash1 != hash2 {
		t.Error("Nodes have different chain after sync")
	}
}

// Teste 9: Distribuição proporcional de blocos
func TestProportionalBlockDistribution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping proportional distribution test in short mode")
	}

	// Cria 3 validadores com stakes 1:2:3
	validators := make([]*Node, 3)
	wallets := make([]*wallet.Wallet, 3)

	for i := 0; i < 3; i++ {
		w, err := wallet.NewWallet()
		if err != nil {
			t.Fatalf("Failed to create wallet %d: %v", i, err)
		}
		wallets[i] = w
	}

	allocations := map[string]uint64{
		wallets[0].GetAddress(): 60000,
	}

	genesis := createTestGenesis(t, allocations)
	config := DefaultChainConfig()
	config.BlockTime = 100 * time.Millisecond // Mais rápido para este teste

	for i := 0; i < 3; i++ {
		chain, _ := NewChain(genesis, config)
		mempool := NewMempool()
		validators[i] = NewNode(fmt.Sprintf("validator%d", i+1), wallets[i], chain, mempool)
	}

	connectNodesFullMesh(validators)

	// Stakes proporcionais: 1000, 2000, 3000 (total 6000)
	stakes := []uint64{1000, 2000, 3000}

	time.Sleep(150 * time.Millisecond)
	coinbase := NewCoinbaseTransaction(wallets[0].GetAddress(), config.BlockReward, 1)
	txs := TransactionSlice{coinbase}

	// Transfere fundos para validators 1 e 2
	transferTx1 := NewTransaction(wallets[0].GetAddress(), wallets[1].GetAddress(), 20000, 1, 0, "")
	_ = transferTx1.Sign(wallets[0])
	txs = append(txs, transferTx1)

	transferTx2 := NewTransaction(wallets[0].GetAddress(), wallets[2].GetAddress(), 20000, 1, 1, "")
	_ = transferTx2.Sign(wallets[0])
	txs = append(txs, transferTx2)

	// Stakes (nonces corretos após as transferências)
	stakeNonces := []uint64{2, 0, 0}
	for i := range validators {
		addr := wallets[i].GetAddress()
		stakeData := NewStakeData(stakes[i])
		dataStr, _ := stakeData.Serialize()
		stakeTx := NewTransaction(addr, addr, stakes[i], 1, stakeNonces[i], dataStr)
		_ = stakeTx.Sign(wallets[i])
		txs = append(txs, stakeTx)
	}

	block1 := NewBlock(1, genesis.Hash, txs, wallets[0].GetAddress())
	hash, _ := block1.CalculateHash()
	block1.Hash = hash

	for i := range validators {
		_ = validators[i].GetChain().AddBlock(block1)
	}

	// Contador de blocos por validador (thread-safe)
	var blockCountMu sync.Mutex
	blockCount := make(map[string]int)
	for _, node := range validators {
		blockCount[node.GetMiner().GetAddress()] = 0
	}

	// Callback para contar blocos
	for _, node := range validators {
		addr := node.GetMiner().GetAddress()
		node.SetOnBlockReceived(func(block *Block) {
			if block.Header.ValidatorAddr == addr {
				blockCountMu.Lock()
				blockCount[addr]++
				blockCountMu.Unlock()
			}
		})
	}

	// Minera por 3 segundos
	for _, node := range validators {
		node.StartMining()
	}

	time.Sleep(3 * time.Second)

	for _, node := range validators {
		node.StopMining()
	}

	// Conta blocos na chain final
	blocks := validators[0].GetChain().GetAllBlocks()
	for i := 1; i < len(blocks); i++ { // Pula gênesis
		validator := blocks[i].Header.ValidatorAddr
		blockCount[validator]++
	}

	total := 0
	for _, count := range blockCount {
		total += count
	}

	t.Logf("Total blocks mined: %d", total)
	for i, node := range validators {
		addr := node.GetMiner().GetAddress()
		count := blockCount[addr]
		percentage := float64(count) / float64(total) * 100
		expectedPercentage := float64(stakes[i]) / 6000.0 * 100

		t.Logf("Validator %d: %d blocks (%.1f%%, expected %.1f%%)",
			i+1, count, percentage, expectedPercentage)
	}

	// Verifica que houve distribuição (não precisa ser exata)
	for _, count := range blockCount {
		if count == 0 {
			t.Error("At least one validator did not mine any blocks")
		}
	}
}

// Teste 10: Verificação de consistência da chain
func TestChainConsistency(t *testing.T) {
	// Cria 4 nós
	nodes := make([]*Node, 4)
	wallets := make([]*wallet.Wallet, 4)

	for i := 0; i < 4; i++ {
		w, err := wallet.NewWallet()
		if err != nil {
			t.Fatalf("Failed to create wallet %d: %v", i, err)
		}
		wallets[i] = w
	}

	allocations := map[string]uint64{
		wallets[0].GetAddress(): 50000,
	}

	genesis := createTestGenesis(t, allocations)
	config := DefaultChainConfig()
	config.BlockTime = 200 * time.Millisecond

	for i := 0; i < 4; i++ {
		chain, _ := NewChain(genesis, config)
		mempool := NewMempool()
		nodes[i] = NewNode(fmt.Sprintf("node%d", i+1), wallets[i], chain, mempool)
	}

	// Conecta em malha completa
	connectNodesFullMesh(nodes)

	// Cria bloco bootstrap com transferências e stakes de todos
	time.Sleep(250 * time.Millisecond)
	coinbase := NewCoinbaseTransaction(wallets[0].GetAddress(), config.BlockReward, 1)
	txs := TransactionSlice{coinbase}

	// Transfere fundos para os outros nós
	for i := 1; i < len(nodes); i++ {
		transferTx := NewTransaction(wallets[0].GetAddress(), wallets[i].GetAddress(), 10000, 1, uint64(i-1), "")
		_ = transferTx.Sign(wallets[0])
		txs = append(txs, transferTx)
	}

	// Stakes de todos (nonces corretos)
	// Node 0 usou nonces 0,1,2 nas transferências, então usa nonce 3
	// Nodes 1,2,3 usam nonce 0 (primeira transação deles)
	for i := range nodes {
		addr := wallets[i].GetAddress()
		stakeData := NewStakeData(1000)
		dataStr, _ := stakeData.Serialize()
		var nonce uint64
		if i == 0 {
			nonce = 3 // Após 3 transferências
		} else {
			nonce = 0
		}
		stakeTx := NewTransaction(addr, addr, 1000, 1, nonce, dataStr)
		_ = stakeTx.Sign(wallets[i])
		txs = append(txs, stakeTx)
	}

	block1 := NewBlock(1, genesis.Hash, txs, wallets[0].GetAddress())
	hash, _ := block1.CalculateHash()
	block1.Hash = hash

	for i := range nodes {
		_ = nodes[i].GetChain().AddBlock(block1)
	}

	// Inicia mineração
	for _, node := range nodes {
		node.StartMining()
	}

	// Aguarda convergência
	waitForConvergence(t, nodes, 20, 15*time.Second)

	for _, node := range nodes {
		node.StopMining()
	}

	// Verifica que todas as chains são idênticas
	referenceChain := nodes[0].GetChain()
	referenceBlocks := referenceChain.GetAllBlocks()

	for i := 1; i < len(nodes); i++ {
		chain := nodes[i].GetChain()

		// Verifica altura
		if chain.GetHeight() != referenceChain.GetHeight() {
			t.Errorf("Node %d has different height: %d vs %d",
				i, chain.GetHeight(), referenceChain.GetHeight())
			continue
		}

		// Verifica todos os blocos
		blocks := chain.GetAllBlocks()
		for j := 0; j < len(blocks); j++ {
			if blocks[j].Hash != referenceBlocks[j].Hash {
				t.Errorf("Node %d has different block at height %d", i, j)
			}
		}

		// Verifica contexto (saldos e stakes)
		for _, w := range wallets {
			addr := w.GetAddress()

			balance := chain.GetBalance(addr)
			refBalance := referenceChain.GetBalance(addr)
			if balance != refBalance {
				t.Errorf("Node %d has different balance for %s: %d vs %d",
					i, addr[:8], balance, refBalance)
			}

			stake := chain.GetStake(addr)
			refStake := referenceChain.GetStake(addr)
			if stake != refStake {
				t.Errorf("Node %d has different stake for %s: %d vs %d",
					i, addr[:8], stake, refStake)
			}
		}
	}

	// Verifica integridade de cada chain
	for i, node := range nodes {
		if err := node.GetChain().VerifyChain(); err != nil {
			t.Errorf("Node %d chain verification failed: %v", i, err)
		}
	}

	t.Logf("All %d nodes converged to height %d with consistent state",
		len(nodes), referenceChain.GetHeight())
}
