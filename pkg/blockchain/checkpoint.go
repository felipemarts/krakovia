package blockchain

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/syndtr/goleveldb/leveldb"
)

// Checkpoint representa um snapshot do estado da blockchain em uma determinada altura
type Checkpoint struct {
	Height    uint64            `json:"height"`    // Altura do bloco do checkpoint
	Timestamp int64             `json:"timestamp"` // Timestamp do checkpoint
	Accounts  map[string]*AccountState `json:"accounts"`  // Estado de todas as contas
	Hash      string            `json:"hash"`      // Hash SHA-256 do CSV
	CSV       string            `json:"-"`         // CSV gerado (não serializado em JSON)
}

// AccountState representa o estado de uma conta em um checkpoint
type AccountState struct {
	Address string `json:"address"` // Endereço da conta
	Balance uint64 `json:"balance"` // Saldo da conta
	Stake   uint64 `json:"stake"`   // Stake da conta
	Nonce   uint64 `json:"nonce"`   // Nonce da conta
}

// CheckpointMetadata contém metadados sobre um checkpoint
type CheckpointMetadata struct {
	Height       uint64 `json:"height"`
	Timestamp    int64  `json:"timestamp"`
	Hash         string `json:"hash"`
	TotalAccounts int   `json:"total_accounts"`
	Compressed   bool   `json:"compressed"`
}

// GenerateCheckpointCSV gera um CSV ordenado com o estado de todas as contas
// Formato: address,balance,stake,nonce
// Ordenado alfabeticamente por address para garantir determinismo
func GenerateCheckpointCSV(accounts map[string]*AccountState, delimiter string) string {
	if len(accounts) == 0 {
		return ""
	}

	// Coletar endereços e ordenar
	addresses := make([]string, 0, len(accounts))
	for addr := range accounts {
		addresses = append(addresses, addr)
	}
	sort.Strings(addresses)

	// Gerar CSV
	var csv strings.Builder
	for _, addr := range addresses {
		account := accounts[addr]
		csv.WriteString(fmt.Sprintf("%s%s%d%s%d%s%d\n",
			account.Address, delimiter,
			account.Balance, delimiter,
			account.Stake, delimiter,
			account.Nonce))
	}

	return csv.String()
}

// CalculateCheckpointHash calcula o hash SHA-256 de um CSV
func CalculateCheckpointHash(csv string) string {
	hash := sha256.Sum256([]byte(csv))
	return hex.EncodeToString(hash[:])
}

// CreateCheckpoint cria um checkpoint a partir do estado atual
func CreateCheckpoint(height uint64, timestamp int64, accounts map[string]*AccountState, delimiter string) (*Checkpoint, error) {
	if accounts == nil {
		return nil, fmt.Errorf("accounts map cannot be nil")
	}

	csv := GenerateCheckpointCSV(accounts, delimiter)
	hash := CalculateCheckpointHash(csv)

	checkpoint := &Checkpoint{
		Height:    height,
		Timestamp: timestamp,
		Accounts:  accounts,
		Hash:      hash,
		CSV:       csv,
	}

	return checkpoint, nil
}

// ValidateCheckpointHash valida se o hash de um checkpoint está correto
func ValidateCheckpointHash(checkpoint *Checkpoint, delimiter string) error {
	if checkpoint == nil {
		return fmt.Errorf("checkpoint cannot be nil")
	}

	// Regenerar CSV
	csv := GenerateCheckpointCSV(checkpoint.Accounts, delimiter)

	// Calcular hash
	calculatedHash := CalculateCheckpointHash(csv)

	// Comparar
	if calculatedHash != checkpoint.Hash {
		return fmt.Errorf("checkpoint hash mismatch: expected %s, got %s", checkpoint.Hash, calculatedHash)
	}

	return nil
}

// SaveCheckpointToDB salva um checkpoint no LevelDB
func SaveCheckpointToDB(db *leveldb.DB, checkpoint *Checkpoint, compress bool) error {
	if db == nil {
		return fmt.Errorf("database cannot be nil")
	}
	if checkpoint == nil {
		return fmt.Errorf("checkpoint cannot be nil")
	}

	// Salvar estado serializado
	stateData, err := json.Marshal(checkpoint)
	if err != nil {
		return fmt.Errorf("failed to marshal checkpoint: %w", err)
	}

	stateKey := fmt.Sprintf("checkpoint-%d-state", checkpoint.Height)
	if err := db.Put([]byte(stateKey), stateData, nil); err != nil {
		return fmt.Errorf("failed to save checkpoint state: %w", err)
	}

	// Salvar CSV (com ou sem compressão)
	csvData := []byte(checkpoint.CSV)
	if compress {
		var buf bytes.Buffer
		gzWriter := gzip.NewWriter(&buf)
		if _, err := gzWriter.Write(csvData); err != nil {
			return fmt.Errorf("failed to compress CSV: %w", err)
		}
		if err := gzWriter.Close(); err != nil {
			return fmt.Errorf("failed to close gzip writer: %w", err)
		}
		csvData = buf.Bytes()
	}

	csvKey := fmt.Sprintf("checkpoint-%d-csv", checkpoint.Height)
	if err := db.Put([]byte(csvKey), csvData, nil); err != nil {
		return fmt.Errorf("failed to save checkpoint CSV: %w", err)
	}

	// Salvar hash
	hashKey := fmt.Sprintf("checkpoint-%d-hash", checkpoint.Height)
	if err := db.Put([]byte(hashKey), []byte(checkpoint.Hash), nil); err != nil {
		return fmt.Errorf("failed to save checkpoint hash: %w", err)
	}

	// Salvar metadata
	metadata := CheckpointMetadata{
		Height:       checkpoint.Height,
		Timestamp:    checkpoint.Timestamp,
		Hash:         checkpoint.Hash,
		TotalAccounts: len(checkpoint.Accounts),
		Compressed:   compress,
	}
	metadataData, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal checkpoint metadata: %w", err)
	}

	metadataKey := fmt.Sprintf("checkpoint-%d-metadata", checkpoint.Height)
	if err := db.Put([]byte(metadataKey), metadataData, nil); err != nil {
		return fmt.Errorf("failed to save checkpoint metadata: %w", err)
	}

	// Atualizar último checkpoint
	if err := db.Put([]byte("metadata-last-checkpoint"), []byte(fmt.Sprintf("%d", checkpoint.Height)), nil); err != nil {
		return fmt.Errorf("failed to update last checkpoint: %w", err)
	}

	return nil
}

// LoadCheckpointFromDB carrega um checkpoint do LevelDB
func LoadCheckpointFromDB(db *leveldb.DB, height uint64) (*Checkpoint, error) {
	if db == nil {
		return nil, fmt.Errorf("database cannot be nil")
	}

	// Carregar metadata primeiro para saber se está comprimido
	metadataKey := fmt.Sprintf("checkpoint-%d-metadata", height)
	metadataData, err := db.Get([]byte(metadataKey), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to load checkpoint metadata: %w", err)
	}

	var metadata CheckpointMetadata
	if err := json.Unmarshal(metadataData, &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal checkpoint metadata: %w", err)
	}

	// Carregar estado
	stateKey := fmt.Sprintf("checkpoint-%d-state", height)
	stateData, err := db.Get([]byte(stateKey), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to load checkpoint state: %w", err)
	}

	var checkpoint Checkpoint
	if err := json.Unmarshal(stateData, &checkpoint); err != nil {
		return nil, fmt.Errorf("failed to unmarshal checkpoint state: %w", err)
	}

	// Carregar CSV
	csvKey := fmt.Sprintf("checkpoint-%d-csv", height)
	csvData, err := db.Get([]byte(csvKey), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to load checkpoint CSV: %w", err)
	}

	// Descomprimir se necessário
	if metadata.Compressed {
		gzReader, err := gzip.NewReader(bytes.NewReader(csvData))
		if err != nil {
			return nil, fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer gzReader.Close()

		decompressed, err := io.ReadAll(gzReader)
		if err != nil {
			return nil, fmt.Errorf("failed to decompress CSV: %w", err)
		}
		csvData = decompressed
	}

	checkpoint.CSV = string(csvData)

	return &checkpoint, nil
}

// GetLastCheckpointHeight retorna a altura do último checkpoint salvo
func GetLastCheckpointHeight(db *leveldb.DB) (uint64, error) {
	if db == nil {
		return 0, fmt.Errorf("database cannot be nil")
	}

	data, err := db.Get([]byte("metadata-last-checkpoint"), nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return 0, nil // Nenhum checkpoint ainda
		}
		return 0, fmt.Errorf("failed to get last checkpoint height: %w", err)
	}

	var height uint64
	if _, err := fmt.Sscanf(string(data), "%d", &height); err != nil {
		return 0, fmt.Errorf("failed to parse checkpoint height: %w", err)
	}

	return height, nil
}

// PruneOldCheckpoints remove checkpoints antigos, mantendo apenas os últimos N
func PruneOldCheckpoints(db *leveldb.DB, keepLast int) error {
	if db == nil {
		return fmt.Errorf("database cannot be nil")
	}
	if keepLast < 1 {
		return fmt.Errorf("must keep at least 1 checkpoint")
	}

	lastHeight, err := GetLastCheckpointHeight(db)
	if err != nil {
		return fmt.Errorf("failed to get last checkpoint height: %w", err)
	}

	if lastHeight == 0 {
		return nil // Nenhum checkpoint para fazer pruning
	}

	// Coletar todos os checkpoints
	iter := db.NewIterator(nil, nil)
	defer iter.Release()

	checkpointHeights := make([]uint64, 0)
	for iter.Next() {
		key := string(iter.Key())
		if strings.HasPrefix(key, "checkpoint-") && strings.HasSuffix(key, "-metadata") {
			var metadata CheckpointMetadata
			if err := json.Unmarshal(iter.Value(), &metadata); err != nil {
				continue
			}
			checkpointHeights = append(checkpointHeights, metadata.Height)
		}
	}

	if err := iter.Error(); err != nil {
		return fmt.Errorf("failed to iterate checkpoints: %w", err)
	}

	// Ordenar alturas
	sort.Slice(checkpointHeights, func(i, j int) bool {
		return checkpointHeights[i] < checkpointHeights[j]
	})

	// Remover checkpoints antigos
	if len(checkpointHeights) > keepLast {
		toRemove := checkpointHeights[:len(checkpointHeights)-keepLast]
		for _, height := range toRemove {
			if err := DeleteCheckpoint(db, height); err != nil {
				return fmt.Errorf("failed to delete checkpoint at height %d: %w", height, err)
			}
		}
	}

	return nil
}

// DeleteCheckpoint remove um checkpoint do LevelDB
func DeleteCheckpoint(db *leveldb.DB, height uint64) error {
	if db == nil {
		return fmt.Errorf("database cannot be nil")
	}

	keys := []string{
		fmt.Sprintf("checkpoint-%d-state", height),
		fmt.Sprintf("checkpoint-%d-csv", height),
		fmt.Sprintf("checkpoint-%d-hash", height),
		fmt.Sprintf("checkpoint-%d-metadata", height),
	}

	for _, key := range keys {
		if err := db.Delete([]byte(key), nil); err != nil {
			return fmt.Errorf("failed to delete key %s: %w", key, err)
		}
	}

	return nil
}

// SaveBlockToDB salva um bloco no LevelDB
func SaveBlockToDB(db *leveldb.DB, block *Block) error {
	if db == nil {
		return fmt.Errorf("database cannot be nil")
	}
	if block == nil {
		return fmt.Errorf("block cannot be nil")
	}

	// Serializar bloco
	blockData, err := json.Marshal(block)
	if err != nil {
		return fmt.Errorf("failed to marshal block: %w", err)
	}

	// Salvar por altura
	heightKey := fmt.Sprintf("block-%d", block.Header.Height)
	if err := db.Put([]byte(heightKey), blockData, nil); err != nil {
		return fmt.Errorf("failed to save block by height: %w", err)
	}

	// Salvar índice por hash
	hashKey := fmt.Sprintf("block-hash-%s", block.Hash)
	heightBytes := []byte(fmt.Sprintf("%d", block.Header.Height))
	if err := db.Put([]byte(hashKey), heightBytes, nil); err != nil {
		return fmt.Errorf("failed to save block hash index: %w", err)
	}

	// Atualizar altura da chain
	if err := db.Put([]byte("metadata-chain-height"), heightBytes, nil); err != nil {
		return fmt.Errorf("failed to update chain height: %w", err)
	}

	return nil
}

// LoadBlockFromDB carrega um bloco do LevelDB pela altura
func LoadBlockFromDB(db *leveldb.DB, height uint64) (*Block, error) {
	if db == nil {
		return nil, fmt.Errorf("database cannot be nil")
	}

	heightKey := fmt.Sprintf("block-%d", height)
	blockData, err := db.Get([]byte(heightKey), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to load block at height %d: %w", height, err)
	}

	var block Block
	if err := json.Unmarshal(blockData, &block); err != nil {
		return nil, fmt.Errorf("failed to unmarshal block: %w", err)
	}

	return &block, nil
}

// DeleteBlockFromDB remove um bloco do LevelDB
func DeleteBlockFromDB(db *leveldb.DB, block *Block) error {
	if db == nil {
		return fmt.Errorf("database cannot be nil")
	}
	if block == nil {
		return fmt.Errorf("block cannot be nil")
	}

	// Remover por altura
	heightKey := fmt.Sprintf("block-%d", block.Header.Height)
	if err := db.Delete([]byte(heightKey), nil); err != nil {
		return fmt.Errorf("failed to delete block by height: %w", err)
	}

	// Remover índice por hash
	hashKey := fmt.Sprintf("block-hash-%s", block.Hash)
	if err := db.Delete([]byte(hashKey), nil); err != nil {
		return fmt.Errorf("failed to delete block hash index: %w", err)
	}

	return nil
}

// PruneOldBlocks remove blocos antigos da memória e disco, mantendo apenas os blocos recentes
// keepInMemory: número de blocos para manter em memória
// Blocos mais antigos são salvos no disco antes de serem removidos da memória
func PruneOldBlocks(db *leveldb.DB, blocks *BlockSlice, keepInMemory int) error {
	if db == nil {
		return fmt.Errorf("database cannot be nil")
	}
	if blocks == nil {
		return fmt.Errorf("blocks cannot be nil")
	}
	if keepInMemory < 1 {
		return fmt.Errorf("must keep at least 1 block in memory")
	}

	if len(*blocks) <= keepInMemory {
		return nil // Nada para fazer pruning
	}

	// Blocos para remover da memória
	toRemove := len(*blocks) - keepInMemory
	blocksToRemove := (*blocks)[:toRemove]

	// Salvar blocos no disco antes de remover da memória
	for _, block := range blocksToRemove {
		if err := SaveBlockToDB(db, block); err != nil {
			return fmt.Errorf("failed to save block %d to disk: %w", block.Header.Height, err)
		}
	}

	// Remover blocos da memória
	*blocks = (*blocks)[toRemove:]

	return nil
}

// PruneBlocksBeforeCheckpoint remove blocos do disco que são anteriores ao checkpoint
// Mantém apenas blocos após o checkpoint na memória e disco
func PruneBlocksBeforeCheckpoint(db *leveldb.DB, checkpointHeight uint64, keepCheckpoints int) error {
	if db == nil {
		return fmt.Errorf("database cannot be nil")
	}

	// Coletar todos os checkpoints
	iter := db.NewIterator(nil, nil)
	defer iter.Release()

	checkpointHeights := make([]uint64, 0)
	for iter.Next() {
		key := string(iter.Key())
		if strings.HasPrefix(key, "checkpoint-") && strings.HasSuffix(key, "-metadata") {
			var metadata CheckpointMetadata
			if err := json.Unmarshal(iter.Value(), &metadata); err != nil {
				continue
			}
			checkpointHeights = append(checkpointHeights, metadata.Height)
		}
	}

	if err := iter.Error(); err != nil {
		return fmt.Errorf("failed to iterate checkpoints: %w", err)
	}

	// Ordenar alturas
	sort.Slice(checkpointHeights, func(i, j int) bool {
		return checkpointHeights[i] < checkpointHeights[j]
	})

	// Determinar altura mínima a manter (checkpoint mais antigo que queremos manter)
	if len(checkpointHeights) < keepCheckpoints {
		return nil // Não há checkpoints suficientes para fazer pruning
	}

	minHeightToKeep := checkpointHeights[len(checkpointHeights)-keepCheckpoints]

	// Remover blocos antes do checkpoint mínimo
	iter2 := db.NewIterator(nil, nil)
	defer iter2.Release()

	for iter2.Next() {
		key := string(iter2.Key())
		if strings.HasPrefix(key, "block-") && !strings.Contains(key, "hash") {
			var height uint64
			if _, err := fmt.Sscanf(key, "block-%d", &height); err != nil {
				continue
			}

			// Remover blocos antes do checkpoint mínimo
			if height < minHeightToKeep {
				// Carregar bloco para obter o hash
				block, err := LoadBlockFromDB(db, height)
				if err != nil {
					continue
				}

				// Deletar bloco
				if err := DeleteBlockFromDB(db, block); err != nil {
					return fmt.Errorf("failed to delete block at height %d: %w", height, err)
				}
			}
		}
	}

	if err := iter2.Error(); err != nil {
		return fmt.Errorf("failed to iterate blocks: %w", err)
	}

	return nil
}
