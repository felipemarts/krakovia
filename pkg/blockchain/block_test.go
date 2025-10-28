package blockchain

import (
	"testing"
	"time"

	"github.com/krakovia/blockchain/pkg/wallet"
)

func TestNewBlock(t *testing.T) {
	coinbase := NewCoinbaseTransaction("validator_addr", 50, 1)
	txs := TransactionSlice{coinbase}

	block := NewBlock(1, "prev_hash", txs, "validator_addr")

	if block.Header.Height != 1 {
		t.Errorf("Expected height 1, got %d", block.Header.Height)
	}

	if block.Header.PreviousHash != "prev_hash" {
		t.Errorf("Expected previous hash 'prev_hash', got '%s'", block.Header.PreviousHash)
	}

	if block.Header.ValidatorAddr != "validator_addr" {
		t.Errorf("Expected validator 'validator_addr', got '%s'", block.Header.ValidatorAddr)
	}

	if len(block.Transactions) != 1 {
		t.Errorf("Expected 1 transaction, got %d", len(block.Transactions))
	}
}

func TestBlockCalculateHash(t *testing.T) {
	coinbase := NewCoinbaseTransaction("validator_addr", 50, 1)
	txs := TransactionSlice{coinbase}

	block := NewBlock(1, "prev_hash", txs, "validator_addr")

	hash1, err := block.CalculateHash()
	if err != nil {
		t.Fatalf("Failed to calculate hash: %v", err)
	}

	hash2, err := block.CalculateHash()
	if err != nil {
		t.Fatalf("Failed to calculate hash: %v", err)
	}

	// Hash deve ser determinístico
	if hash1 != hash2 {
		t.Error("Hash should be deterministic")
	}

	if hash1 == "" {
		t.Error("Hash is empty")
	}
}

func TestBlockVerifyHash(t *testing.T) {
	coinbase := NewCoinbaseTransaction("validator_addr", 50, 1)
	txs := TransactionSlice{coinbase}

	block := NewBlock(1, "prev_hash", txs, "validator_addr")

	// Define o hash correto
	hash, _ := block.CalculateHash()
	block.Hash = hash

	// Verificação deve passar
	err := block.VerifyHash()
	if err != nil {
		t.Errorf("Hash verification failed: %v", err)
	}

	// Altera o hash
	block.Hash = "wrong_hash"

	// Verificação deve falhar
	err = block.VerifyHash()
	if err == nil {
		t.Error("Expected hash verification to fail")
	}
}

func TestBlockVerifyMerkleRoot(t *testing.T) {
	coinbase := NewCoinbaseTransaction("validator_addr", 50, 1)
	txs := TransactionSlice{coinbase}

	block := NewBlock(1, "prev_hash", txs, "validator_addr")

	// Verificação deve passar (NewBlock já calcula a raiz correta)
	err := block.VerifyMerkleRoot()
	if err != nil {
		t.Errorf("Merkle root verification failed: %v", err)
	}

	// Altera a raiz de Merkle
	block.Header.MerkleRoot = "wrong_root"

	// Verificação deve falhar
	err = block.VerifyMerkleRoot()
	if err == nil {
		t.Error("Expected merkle root verification to fail")
	}
}

func TestBlockVerifyTransactions(t *testing.T) {
	w, _ := wallet.NewWallet()

	coinbase := NewCoinbaseTransaction(w.GetAddress(), 50, 1)

	tx1 := NewTransaction(w.GetAddress(), "addr1", 100, 1, 0, "tx1")
	tx1.Sign(w)

	txs := TransactionSlice{coinbase, tx1}

	block := NewBlock(1, "prev_hash", txs, w.GetAddress())

	// Verificação deve passar
	err := block.VerifyTransactions()
	if err != nil {
		t.Errorf("Transaction verification failed: %v", err)
	}
}

func TestBlockVerifyTransactionsNoCoinbase(t *testing.T) {
	w, _ := wallet.NewWallet()

	tx1 := NewTransaction(w.GetAddress(), "addr1", 100, 1, 0, "tx1")
	tx1.Sign(w)

	txs := TransactionSlice{tx1}

	block := NewBlock(1, "prev_hash", txs, w.GetAddress())

	// Verificação deve falhar (primeira transação não é coinbase)
	err := block.VerifyTransactions()
	if err == nil {
		t.Error("Expected transaction verification to fail without coinbase")
	}
}

func TestBlockVerifyTransactionsMultipleCoinbase(t *testing.T) {
	coinbase1 := NewCoinbaseTransaction("addr1", 50, 1)
	coinbase2 := NewCoinbaseTransaction("addr2", 50, 2)

	txs := TransactionSlice{coinbase1, coinbase2}

	block := NewBlock(1, "prev_hash", txs, "validator_addr")

	// Verificação deve falhar (múltiplas transações coinbase)
	err := block.VerifyTransactions()
	if err == nil {
		t.Error("Expected transaction verification to fail with multiple coinbase")
	}
}

func TestBlockVerifyTransactionsDuplicates(t *testing.T) {
	w, _ := wallet.NewWallet()

	coinbase := NewCoinbaseTransaction(w.GetAddress(), 50, 1)

	tx1 := NewTransaction(w.GetAddress(), "addr1", 100, 1, 0, "tx1")
	tx1.Sign(w)

	txs := TransactionSlice{coinbase, tx1, tx1} // Duplicata

	block := NewBlock(1, "prev_hash", txs, w.GetAddress())

	// Verificação deve falhar (transações duplicadas)
	err := block.VerifyTransactions()
	if err == nil {
		t.Error("Expected transaction verification to fail with duplicates")
	}
}

func TestBlockValidate(t *testing.T) {
	w, _ := wallet.NewWallet()

	coinbase := NewCoinbaseTransaction(w.GetAddress(), 50, 1)

	tx1 := NewTransaction(w.GetAddress(), "addr1", 100, 1, 0, "tx1")
	tx1.Sign(w)

	txs := TransactionSlice{coinbase, tx1}

	block := NewBlock(1, "prev_hash", txs, w.GetAddress())

	// Define o hash
	hash, _ := block.CalculateHash()
	block.Hash = hash

	// Validação deve passar
	err := block.Validate()
	if err != nil {
		t.Errorf("Block validation failed: %v", err)
	}
}

func TestBlockValidateFutureTimestamp(t *testing.T) {
	coinbase := NewCoinbaseTransaction("validator_addr", 50, 1)
	txs := TransactionSlice{coinbase}

	block := NewBlock(1, "prev_hash", txs, "validator_addr")
	block.Header.Timestamp = time.Now().Unix() + 1000

	hash, _ := block.CalculateHash()
	block.Hash = hash

	// Validação deve falhar (timestamp no futuro)
	err := block.Validate()
	if err == nil {
		t.Error("Expected validation to fail for future timestamp")
	}
}

func TestBlockSerializeDeserialize(t *testing.T) {
	coinbase := NewCoinbaseTransaction("validator_addr", 50, 1)
	txs := TransactionSlice{coinbase}

	block := NewBlock(1, "prev_hash", txs, "validator_addr")
	hash, _ := block.CalculateHash()
	block.Hash = hash

	// Serializa
	data, err := block.Serialize()
	if err != nil {
		t.Fatalf("Failed to serialize block: %v", err)
	}

	// Desserializa
	block2, err := DeserializeBlock(data)
	if err != nil {
		t.Fatalf("Failed to deserialize block: %v", err)
	}

	// Compara campos
	if block.Hash != block2.Hash {
		t.Error("Block hashes do not match")
	}
	if block.Header.Height != block2.Header.Height {
		t.Error("Block heights do not match")
	}
	if block.Header.PreviousHash != block2.Header.PreviousHash {
		t.Error("Previous hashes do not match")
	}
	if len(block.Transactions) != len(block2.Transactions) {
		t.Error("Transaction counts do not match")
	}
}

func TestBlockIsGenesis(t *testing.T) {
	coinbase := NewCoinbaseTransaction("validator_addr", 50, 0)
	txs := TransactionSlice{coinbase}

	genesisBlock := NewBlock(0, "", txs, "validator_addr")

	if !genesisBlock.IsGenesis() {
		t.Error("Block should be genesis")
	}

	regularBlock := NewBlock(1, "prev_hash", txs, "validator_addr")

	if regularBlock.IsGenesis() {
		t.Error("Block should not be genesis")
	}
}

func TestGenesisBlock(t *testing.T) {
	coinbase := NewCoinbaseTransaction("initial_addr", 1000000, 0)

	genesis := GenesisBlock(coinbase)

	if genesis == nil {
		t.Fatal("Genesis block is nil")
	}

	if !genesis.IsGenesis() {
		t.Error("Block should be genesis")
	}

	if genesis.Header.Height != 0 {
		t.Errorf("Genesis block height should be 0, got %d", genesis.Header.Height)
	}

	if genesis.Header.PreviousHash != "" {
		t.Error("Genesis block should have empty previous hash")
	}

	if genesis.Hash == "" {
		t.Error("Genesis block should have hash")
	}
}

func TestValidateGenesisBlock(t *testing.T) {
	coinbase := NewCoinbaseTransaction("initial_addr", 1000000, 0)
	genesis := GenesisBlock(coinbase)

	err := ValidateGenesisBlock(genesis, genesis.Hash)
	if err != nil {
		t.Errorf("Genesis block validation failed: %v", err)
	}

	// Testa com hash errado
	err = ValidateGenesisBlock(genesis, "wrong_hash")
	if err == nil {
		t.Error("Expected genesis validation to fail with wrong hash")
	}
}

func TestBlockGetCoinbaseTransaction(t *testing.T) {
	coinbase := NewCoinbaseTransaction("validator_addr", 50, 1)
	txs := TransactionSlice{coinbase}

	block := NewBlock(1, "prev_hash", txs, "validator_addr")

	cb := block.GetCoinbaseTransaction()
	if cb == nil {
		t.Error("Coinbase transaction is nil")
	}

	if !cb.IsCoinbase() {
		t.Error("Returned transaction is not coinbase")
	}
}

func TestBlockGetRegularTransactions(t *testing.T) {
	w, _ := wallet.NewWallet()

	coinbase := NewCoinbaseTransaction(w.GetAddress(), 50, 1)

	tx1 := NewTransaction(w.GetAddress(), "addr1", 100, 1, 0, "tx1")
	tx1.Sign(w)

	tx2 := NewTransaction(w.GetAddress(), "addr2", 200, 2, 1, "tx2")
	tx2.Sign(w)

	txs := TransactionSlice{coinbase, tx1, tx2}

	block := NewBlock(1, "prev_hash", txs, w.GetAddress())

	regular := block.GetRegularTransactions()

	if len(regular) != 2 {
		t.Errorf("Expected 2 regular transactions, got %d", len(regular))
	}

	for _, tx := range regular {
		if tx.IsCoinbase() {
			t.Error("Regular transactions should not include coinbase")
		}
	}
}

func TestBlockTotalFees(t *testing.T) {
	w, _ := wallet.NewWallet()

	coinbase := NewCoinbaseTransaction(w.GetAddress(), 50, 1)

	tx1 := NewTransaction(w.GetAddress(), "addr1", 100, 1, 0, "tx1")
	tx1.Sign(w)

	tx2 := NewTransaction(w.GetAddress(), "addr2", 200, 2, 1, "tx2")
	tx2.Sign(w)

	txs := TransactionSlice{coinbase, tx1, tx2}

	block := NewBlock(1, "prev_hash", txs, w.GetAddress())

	totalFees := block.TotalFees()
	if totalFees != 3 {
		t.Errorf("Expected total fees 3, got %d", totalFees)
	}
}

func TestBlockCopy(t *testing.T) {
	coinbase := NewCoinbaseTransaction("validator_addr", 50, 1)
	txs := TransactionSlice{coinbase}

	block := NewBlock(1, "prev_hash", txs, "validator_addr")
	hash, _ := block.CalculateHash()
	block.Hash = hash

	blockCopy := block.Copy()

	// Modifica a cópia
	blockCopy.Header.Height = 999

	// Original não deve ser afetado
	if block.Header.Height == 999 {
		t.Error("Original block was modified")
	}
}

func TestBlockEqual(t *testing.T) {
	coinbase := NewCoinbaseTransaction("validator_addr", 50, 1)
	txs := TransactionSlice{coinbase}

	block1 := NewBlock(1, "prev_hash", txs, "validator_addr")
	hash, _ := block1.CalculateHash()
	block1.Hash = hash

	block2 := block1.Copy()

	if !block1.Equal(block2) {
		t.Error("Blocks should be equal")
	}

	block3 := NewBlock(2, "other_hash", txs, "validator_addr")
	hash3, _ := block3.CalculateHash()
	block3.Hash = hash3

	if block1.Equal(block3) {
		t.Error("Blocks should not be equal")
	}
}

func TestBlockSliceGetByHeight(t *testing.T) {
	coinbase1 := NewCoinbaseTransaction("addr", 50, 1)
	block1 := NewBlock(1, "hash0", TransactionSlice{coinbase1}, "addr")
	block1.Hash = "hash1"

	coinbase2 := NewCoinbaseTransaction("addr", 50, 2)
	block2 := NewBlock(2, "hash1", TransactionSlice{coinbase2}, "addr")
	block2.Hash = "hash2"

	blocks := BlockSlice{block1, block2}

	found := blocks.GetByHeight(1)
	if found == nil {
		t.Error("Block with height 1 not found")
	}

	if found.Header.Height != 1 {
		t.Error("Found wrong block")
	}
}

func TestBlockSliceGetByHash(t *testing.T) {
	coinbase1 := NewCoinbaseTransaction("addr", 50, 1)
	block1 := NewBlock(1, "hash0", TransactionSlice{coinbase1}, "addr")
	block1.Hash = "hash1"

	coinbase2 := NewCoinbaseTransaction("addr", 50, 2)
	block2 := NewBlock(2, "hash1", TransactionSlice{coinbase2}, "addr")
	block2.Hash = "hash2"

	blocks := BlockSlice{block1, block2}

	found := blocks.GetByHash("hash1")
	if found == nil {
		t.Error("Block with hash 'hash1' not found")
	}

	if found.Hash != "hash1" {
		t.Error("Found wrong block")
	}
}

func TestBlockSliceValidateChain(t *testing.T) {
	w, _ := wallet.NewWallet()

	// Bloco gênesis
	genesisCoinbase := NewCoinbaseTransaction(w.GetAddress(), 1000000, 0)
	genesis := GenesisBlock(genesisCoinbase)

	// Bloco 1
	coinbase1 := NewCoinbaseTransaction(w.GetAddress(), 50, 1)
	tx1 := NewTransaction(w.GetAddress(), "addr1", 100, 1, 0, "tx1")
	tx1.Sign(w)
	block1 := NewBlock(1, genesis.Hash, TransactionSlice{coinbase1, tx1}, w.GetAddress())
	hash1, _ := block1.CalculateHash()
	block1.Hash = hash1

	// Bloco 2
	coinbase2 := NewCoinbaseTransaction(w.GetAddress(), 50, 2)
	tx2 := NewTransaction(w.GetAddress(), "addr2", 200, 2, 1, "tx2")
	tx2.Sign(w)
	block2 := NewBlock(2, block1.Hash, TransactionSlice{coinbase2, tx2}, w.GetAddress())
	hash2, _ := block2.CalculateHash()
	block2.Hash = hash2

	chain := BlockSlice{genesis, block1, block2}

	err := chain.ValidateChain()
	if err != nil {
		t.Errorf("Chain validation failed: %v", err)
	}
}

func TestBlockSliceValidateChainBrokenLink(t *testing.T) {
	w, _ := wallet.NewWallet()

	genesisCoinbase := NewCoinbaseTransaction(w.GetAddress(), 1000000, 0)
	genesis := GenesisBlock(genesisCoinbase)

	coinbase1 := NewCoinbaseTransaction(w.GetAddress(), 50, 1)
	block1 := NewBlock(1, genesis.Hash, TransactionSlice{coinbase1}, w.GetAddress())
	hash1, _ := block1.CalculateHash()
	block1.Hash = hash1

	coinbase2 := NewCoinbaseTransaction(w.GetAddress(), 50, 2)
	block2 := NewBlock(2, "wrong_hash", TransactionSlice{coinbase2}, w.GetAddress()) // Hash errado
	hash2, _ := block2.CalculateHash()
	block2.Hash = hash2

	chain := BlockSlice{genesis, block1, block2}

	err := chain.ValidateChain()
	if err == nil {
		t.Error("Expected chain validation to fail with broken link")
	}
}

func TestBlockSliceLastBlock(t *testing.T) {
	coinbase1 := NewCoinbaseTransaction("addr", 50, 1)
	block1 := NewBlock(1, "hash0", TransactionSlice{coinbase1}, "addr")

	coinbase2 := NewCoinbaseTransaction("addr", 50, 2)
	block2 := NewBlock(2, "hash1", TransactionSlice{coinbase2}, "addr")

	blocks := BlockSlice{block1, block2}

	last := blocks.LastBlock()
	if last == nil {
		t.Error("Last block is nil")
	}

	if last.Header.Height != 2 {
		t.Errorf("Expected last block height 2, got %d", last.Header.Height)
	}
}

func TestBlockSliceHeight(t *testing.T) {
	coinbase1 := NewCoinbaseTransaction("addr", 50, 1)
	block1 := NewBlock(1, "hash0", TransactionSlice{coinbase1}, "addr")

	coinbase2 := NewCoinbaseTransaction("addr", 50, 5)
	block2 := NewBlock(5, "hash1", TransactionSlice{coinbase2}, "addr")

	blocks := BlockSlice{block1, block2}

	height := blocks.Height()
	if height != 5 {
		t.Errorf("Expected chain height 5, got %d", height)
	}
}

func BenchmarkBlockCalculateHash(b *testing.B) {
	coinbase := NewCoinbaseTransaction("validator_addr", 50, 1)
	txs := TransactionSlice{coinbase}
	block := NewBlock(1, "prev_hash", txs, "validator_addr")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		block.CalculateHash()
	}
}

func BenchmarkBlockVerify(b *testing.B) {
	w, _ := wallet.NewWallet()
	coinbase := NewCoinbaseTransaction(w.GetAddress(), 50, 1)
	tx := NewTransaction(w.GetAddress(), "addr", 100, 1, 0, "tx")
	tx.Sign(w)
	txs := TransactionSlice{coinbase, tx}

	block := NewBlock(1, "prev_hash", txs, w.GetAddress())
	hash, _ := block.CalculateHash()
	block.Hash = hash

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		block.Validate()
	}
}

func BenchmarkBlockWithManyTransactions(b *testing.B) {
	w, _ := wallet.NewWallet()

	var txs TransactionSlice
	coinbase := NewCoinbaseTransaction(w.GetAddress(), 50, 1)
	txs = append(txs, coinbase)

	for i := 0; i < 1000; i++ {
		tx := NewTransaction(w.GetAddress(), "addr", 100, 1, uint64(i), "tx")
		tx.Sign(w)
		txs = append(txs, tx)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		block := NewBlock(1, "prev_hash", txs, w.GetAddress())
		block.CalculateHash()
	}
}
