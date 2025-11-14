package blockchain

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/syndtr/goleveldb/leveldb"
)

// createTestAccounts cria contas simuladas para testes
func createTestAccounts() map[string]*AccountState {
	return map[string]*AccountState{
		"addr1": {
			Address: "addr1",
			Balance: 1000,
			Stake:   100,
			Nonce:   5,
		},
		"addr2": {
			Address: "addr2",
			Balance: 2000,
			Stake:   0,
			Nonce:   3,
		},
		"addr3": {
			Address: "addr3",
			Balance: 500,
			Stake:   50,
			Nonce:   1,
		},
	}
}

// TestGenerateCheckpointCSV testa a geração de CSV ordenado
func TestGenerateCheckpointCSV(t *testing.T) {
	accounts := createTestAccounts()
	delimiter := ","

	csv := GenerateCheckpointCSV(accounts, delimiter)

	// Verificar que não está vazio
	if csv == "" {
		t.Fatal("CSV should not be empty")
	}

	// Verificar ordenação (addr1, addr2, addr3)
	expected := "addr1,1000,100,5\naddr2,2000,0,3\naddr3,500,50,1\n"
	if csv != expected {
		t.Errorf("CSV mismatch.\nExpected:\n%s\nGot:\n%s", expected, csv)
	}
}

// TestGenerateCheckpointCSV_Empty testa CSV com mapa vazio
func TestGenerateCheckpointCSV_Empty(t *testing.T) {
	accounts := make(map[string]*AccountState)
	delimiter := ","

	csv := GenerateCheckpointCSV(accounts, delimiter)

	if csv != "" {
		t.Errorf("Expected empty CSV, got: %s", csv)
	}
}

// TestGenerateCheckpointCSV_Deterministic testa que o CSV é sempre o mesmo
func TestGenerateCheckpointCSV_Deterministic(t *testing.T) {
	accounts := createTestAccounts()
	delimiter := ","

	csv1 := GenerateCheckpointCSV(accounts, delimiter)
	csv2 := GenerateCheckpointCSV(accounts, delimiter)

	if csv1 != csv2 {
		t.Error("CSV generation should be deterministic")
	}
}

// TestCalculateCheckpointHash testa o cálculo do hash SHA-256
func TestCalculateCheckpointHash(t *testing.T) {
	csv := "addr1,1000,100,5\naddr2,2000,0,3\naddr3,500,50,1\n"

	hash := CalculateCheckpointHash(csv)

	// Hash deve ter 64 caracteres (SHA-256 em hex)
	if len(hash) != 64 {
		t.Errorf("Hash should be 64 characters, got %d", len(hash))
	}

	// Hash deve ser determinístico
	hash2 := CalculateCheckpointHash(csv)
	if hash != hash2 {
		t.Error("Hash calculation should be deterministic")
	}
}

// TestCalculateCheckpointHash_Different testa que CSVs diferentes geram hashes diferentes
func TestCalculateCheckpointHash_Different(t *testing.T) {
	csv1 := "addr1,1000,100,5\n"
	csv2 := "addr1,1000,100,6\n" // Nonce diferente

	hash1 := CalculateCheckpointHash(csv1)
	hash2 := CalculateCheckpointHash(csv2)

	if hash1 == hash2 {
		t.Error("Different CSVs should produce different hashes")
	}
}

// TestCreateCheckpoint testa a criação de checkpoint
func TestCreateCheckpoint(t *testing.T) {
	accounts := createTestAccounts()
	height := uint64(100)
	timestamp := int64(1234567890)
	delimiter := ","

	checkpoint, err := CreateCheckpoint(height, timestamp, accounts, delimiter)
	if err != nil {
		t.Fatalf("Failed to create checkpoint: %v", err)
	}

	// Verificar campos
	if checkpoint.Height != height {
		t.Errorf("Expected height %d, got %d", height, checkpoint.Height)
	}
	if checkpoint.Timestamp != timestamp {
		t.Errorf("Expected timestamp %d, got %d", timestamp, checkpoint.Timestamp)
	}
	if len(checkpoint.Accounts) != 3 {
		t.Errorf("Expected 3 accounts, got %d", len(checkpoint.Accounts))
	}
	if checkpoint.Hash == "" {
		t.Error("Hash should not be empty")
	}
	if checkpoint.CSV == "" {
		t.Error("CSV should not be empty")
	}
}

// TestCreateCheckpoint_NilAccounts testa erro ao criar checkpoint com contas nil
func TestCreateCheckpoint_NilAccounts(t *testing.T) {
	_, err := CreateCheckpoint(100, 1234567890, nil, ",")
	if err == nil {
		t.Error("Expected error for nil accounts")
	}
}

// TestValidateCheckpointHash testa a validação de hash
func TestValidateCheckpointHash(t *testing.T) {
	accounts := createTestAccounts()
	checkpoint, _ := CreateCheckpoint(100, 1234567890, accounts, ",")

	err := ValidateCheckpointHash(checkpoint, ",")
	if err != nil {
		t.Errorf("Checkpoint should be valid: %v", err)
	}
}

// TestValidateCheckpointHash_Invalid testa detecção de hash inválido
func TestValidateCheckpointHash_Invalid(t *testing.T) {
	accounts := createTestAccounts()
	checkpoint, _ := CreateCheckpoint(100, 1234567890, accounts, ",")

	// Corromper hash
	checkpoint.Hash = "invalid_hash"

	err := ValidateCheckpointHash(checkpoint, ",")
	if err == nil {
		t.Error("Expected error for invalid hash")
	}
}

// TestValidateCheckpointHash_NilCheckpoint testa erro ao validar checkpoint nil
func TestValidateCheckpointHash_NilCheckpoint(t *testing.T) {
	err := ValidateCheckpointHash(nil, ",")
	if err == nil {
		t.Error("Expected error for nil checkpoint")
	}
}

// TestSaveAndLoadCheckpoint testa salvar e carregar checkpoint do LevelDB
func TestSaveAndLoadCheckpoint(t *testing.T) {
	// Criar banco temporário
	tmpDir, err := os.MkdirTemp("", "checkpoint-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	db, err := leveldb.OpenFile(filepath.Join(tmpDir, "test.db"), nil)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer func() {
		_ = db.Close()
	}()

	// Criar checkpoint
	accounts := createTestAccounts()
	checkpoint, _ := CreateCheckpoint(100, 1234567890, accounts, ",")

	// Salvar
	err = SaveCheckpointToDB(db, checkpoint, false)
	if err != nil {
		t.Fatalf("Failed to save checkpoint: %v", err)
	}

	// Carregar
	loaded, err := LoadCheckpointFromDB(db, 100)
	if err != nil {
		t.Fatalf("Failed to load checkpoint: %v", err)
	}

	// Verificar
	if loaded.Height != checkpoint.Height {
		t.Errorf("Height mismatch: expected %d, got %d", checkpoint.Height, loaded.Height)
	}
	if loaded.Hash != checkpoint.Hash {
		t.Errorf("Hash mismatch: expected %s, got %s", checkpoint.Hash, loaded.Hash)
	}
	if len(loaded.Accounts) != len(checkpoint.Accounts) {
		t.Errorf("Accounts count mismatch: expected %d, got %d", len(checkpoint.Accounts), len(loaded.Accounts))
	}
	if loaded.CSV != checkpoint.CSV {
		t.Errorf("CSV mismatch")
	}
}

// TestSaveAndLoadCheckpoint_Compressed testa checkpoint com compressão
func TestSaveAndLoadCheckpoint_Compressed(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "checkpoint-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	db, err := leveldb.OpenFile(filepath.Join(tmpDir, "test.db"), nil)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer func() {
		_ = db.Close()
	}()

	accounts := createTestAccounts()
	checkpoint, _ := CreateCheckpoint(100, 1234567890, accounts, ",")

	// Salvar com compressão
	err = SaveCheckpointToDB(db, checkpoint, true)
	if err != nil {
		t.Fatalf("Failed to save compressed checkpoint: %v", err)
	}

	// Carregar
	loaded, err := LoadCheckpointFromDB(db, 100)
	if err != nil {
		t.Fatalf("Failed to load compressed checkpoint: %v", err)
	}

	// CSV deve ser igual após descompressão
	if loaded.CSV != checkpoint.CSV {
		t.Error("CSV mismatch after decompression")
	}
}

// TestGetLastCheckpointHeight testa obter última altura de checkpoint
func TestGetLastCheckpointHeight(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "checkpoint-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	db, err := leveldb.OpenFile(filepath.Join(tmpDir, "test.db"), nil)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer func() {
		_ = db.Close()
	}()

	// Deve retornar 0 se não há checkpoints
	height, err := GetLastCheckpointHeight(db)
	if err != nil {
		t.Fatalf("Failed to get last checkpoint height: %v", err)
	}
	if height != 0 {
		t.Errorf("Expected height 0, got %d", height)
	}

	// Salvar checkpoint
	accounts := createTestAccounts()
	checkpoint, _ := CreateCheckpoint(100, 1234567890, accounts, ",")
	if err := SaveCheckpointToDB(db, checkpoint, false); err != nil {
		t.Fatalf("Failed to save checkpoint: %v", err)
	}

	// Deve retornar 100
	height, err = GetLastCheckpointHeight(db)
	if err != nil {
		t.Fatalf("Failed to get last checkpoint height: %v", err)
	}
	if height != 100 {
		t.Errorf("Expected height 100, got %d", height)
	}
}

// TestPruneOldCheckpoints testa remoção de checkpoints antigos
func TestPruneOldCheckpoints(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "checkpoint-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	db, err := leveldb.OpenFile(filepath.Join(tmpDir, "test.db"), nil)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer func() {
		_ = db.Close()
	}()

	// Criar 5 checkpoints
	accounts := createTestAccounts()
	for i := 0; i < 5; i++ {
		checkpoint, _ := CreateCheckpoint(uint64(i*100), 1234567890, accounts, ",")
		if err := SaveCheckpointToDB(db, checkpoint, false); err != nil {
			t.Fatalf("Failed to save checkpoint: %v", err)
		}
	}

	// Manter apenas últimos 2
	err = PruneOldCheckpoints(db, 2)
	if err != nil {
		t.Fatalf("Failed to prune checkpoints: %v", err)
	}

	// Verificar que apenas 2 permanecem
	for i := 0; i < 3; i++ {
		_, err := LoadCheckpointFromDB(db, uint64(i*100))
		if err == nil {
			t.Errorf("Checkpoint at height %d should have been deleted", i*100)
		}
	}

	for i := 3; i < 5; i++ {
		_, err := LoadCheckpointFromDB(db, uint64(i*100))
		if err != nil {
			t.Errorf("Checkpoint at height %d should exist: %v", i*100, err)
		}
	}
}

// TestDeleteCheckpoint testa remoção de checkpoint individual
func TestDeleteCheckpoint(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "checkpoint-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	db, err := leveldb.OpenFile(filepath.Join(tmpDir, "test.db"), nil)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer func() {
		_ = db.Close()
	}()

	// Criar checkpoint
	accounts := createTestAccounts()
	checkpoint, _ := CreateCheckpoint(100, 1234567890, accounts, ",")
	if err := SaveCheckpointToDB(db, checkpoint, false); err != nil {
		t.Fatalf("Failed to save checkpoint: %v", err)
	}

	// Deletar
	err = DeleteCheckpoint(db, 100)
	if err != nil {
		t.Fatalf("Failed to delete checkpoint: %v", err)
	}

	// Verificar que foi deletado
	_, err = LoadCheckpointFromDB(db, 100)
	if err == nil {
		t.Error("Checkpoint should have been deleted")
	}
}

// TestSaveAndLoadBlock testa salvar e carregar bloco
func TestSaveAndLoadBlock(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "checkpoint-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	db, err := leveldb.OpenFile(filepath.Join(tmpDir, "test.db"), nil)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer func() {
		_ = db.Close()
	}()

	// Criar bloco simulado
	block := &Block{
		Header: BlockHeader{
			Version:      1,
			Height:       100,
			Timestamp:    1234567890,
			PreviousHash: "prev_hash",
			MerkleRoot:   "merkle_root",
			ValidatorAddr: "validator",
		},
		Hash: "block_hash",
		Transactions: TransactionSlice{},
	}

	// Salvar
	err = SaveBlockToDB(db, block)
	if err != nil {
		t.Fatalf("Failed to save block: %v", err)
	}

	// Carregar
	loaded, err := LoadBlockFromDB(db, 100)
	if err != nil {
		t.Fatalf("Failed to load block: %v", err)
	}

	// Verificar
	if loaded.Header.Height != block.Header.Height {
		t.Errorf("Height mismatch: expected %d, got %d", block.Header.Height, loaded.Header.Height)
	}
	if loaded.Hash != block.Hash {
		t.Errorf("Hash mismatch: expected %s, got %s", block.Hash, loaded.Hash)
	}
}

// TestPruneOldBlocks testa remoção de blocos da memória
func TestPruneOldBlocks(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "checkpoint-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	db, err := leveldb.OpenFile(filepath.Join(tmpDir, "test.db"), nil)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer func() {
		_ = db.Close()
	}()

	// Criar 10 blocos em memória
	blocks := make(BlockSlice, 10)
	for i := 0; i < 10; i++ {
		blocks[i] = &Block{
			Header: BlockHeader{
				Height: uint64(i),
			},
			Hash: "hash_" + string(rune('0'+i)),
		}
	}

	// Fazer pruning, mantendo apenas últimos 5
	err = PruneOldBlocks(db, &blocks, 5)
	if err != nil {
		t.Fatalf("Failed to prune old blocks: %v", err)
	}

	// Verificar que apenas 5 blocos permanecem em memória
	if len(blocks) != 5 {
		t.Errorf("Expected 5 blocks in memory, got %d", len(blocks))
	}

	// Verificar que os 5 primeiros foram salvos no disco
	for i := 0; i < 5; i++ {
		_, err := LoadBlockFromDB(db, uint64(i))
		if err != nil {
			t.Errorf("Block at height %d should be in disk: %v", i, err)
		}
	}

	// Verificar que blocos em memória são os últimos 5
	if blocks[0].Header.Height != 5 {
		t.Errorf("First block in memory should be height 5, got %d", blocks[0].Header.Height)
	}
}

// TestPruneBlocksBeforeCheckpoint testa remoção de blocos antes do checkpoint
func TestPruneBlocksBeforeCheckpoint(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "checkpoint-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	db, err := leveldb.OpenFile(filepath.Join(tmpDir, "test.db"), nil)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer func() {
		_ = db.Close()
	}()

	// Criar blocos no disco
	for i := 0; i < 20; i++ {
		block := &Block{
			Header: BlockHeader{
				Height: uint64(i),
			},
			Hash: "hash",
		}
		if err := SaveBlockToDB(db, block); err != nil {
			t.Fatalf("Failed to save block: %v", err)
		}
	}

	// Criar 2 checkpoints (altura 10 e 15)
	accounts := createTestAccounts()
	checkpoint1, _ := CreateCheckpoint(10, 1234567890, accounts, ",")
	checkpoint2, _ := CreateCheckpoint(15, 1234567891, accounts, ",")
	if err := SaveCheckpointToDB(db, checkpoint1, false); err != nil {
		t.Fatalf("Failed to save checkpoint1: %v", err)
	}
	if err := SaveCheckpointToDB(db, checkpoint2, false); err != nil {
		t.Fatalf("Failed to save checkpoint2: %v", err)
	}

	// Fazer pruning, mantendo apenas último checkpoint
	err = PruneBlocksBeforeCheckpoint(db, 15, 1)
	if err != nil {
		t.Fatalf("Failed to prune blocks before checkpoint: %v", err)
	}

	// Verificar que blocos antes de 15 foram removidos
	for i := 0; i < 15; i++ {
		_, err := LoadBlockFromDB(db, uint64(i))
		if err == nil {
			t.Errorf("Block at height %d should have been deleted", i)
		}
	}

	// Verificar que blocos após 15 ainda existem
	for i := 15; i < 20; i++ {
		_, err := LoadBlockFromDB(db, uint64(i))
		if err != nil {
			t.Errorf("Block at height %d should exist: %v", i, err)
		}
	}
}
