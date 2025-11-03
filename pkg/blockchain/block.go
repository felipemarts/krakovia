package blockchain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// BlockHeader contém os metadados do bloco
type BlockHeader struct {
	Version          uint32 `json:"version"`                    // Versão do protocolo
	Height           uint64 `json:"height"`                     // Altura do bloco na chain
	Timestamp        int64  `json:"timestamp"`                  // Timestamp Unix
	PreviousHash     string `json:"previous_hash"`              // Hash do bloco anterior
	MerkleRoot       string `json:"merkle_root"`                // Raiz da árvore de Merkle das transações
	ValidatorAddr    string `json:"validator_addr"`             // Endereço do validador que criou o bloco
	Signature        string `json:"signature"`                  // Assinatura do validador
	PublicKey        string `json:"public_key"`                 // Chave pública do validador
	Nonce            uint64 `json:"nonce"`                      // Nonce (pode ser usado para desempate ou ordenação)
	CheckpointHash   string `json:"checkpoint_hash,omitempty"`  // Hash do checkpoint (se este bloco marca um checkpoint)
	CheckpointHeight uint64 `json:"checkpoint_height,omitempty"` // Altura do bloco referente ao checkpoint
}

// Block representa um bloco na blockchain
type Block struct {
	Header       BlockHeader       `json:"header"`
	Transactions TransactionSlice  `json:"transactions"`
	Hash         string            `json:"hash"`
}

// NewBlock cria um novo bloco
func NewBlock(height uint64, previousHash string, transactions TransactionSlice, validatorAddr string) *Block {
	merkleRoot := transactions.CalculateMerkleRoot()

	block := &Block{
		Header: BlockHeader{
			Version:       1,
			Height:        height,
			Timestamp:     time.Now().Unix(),
			PreviousHash:  previousHash,
			MerkleRoot:    merkleRoot,
			ValidatorAddr: validatorAddr,
			Nonce:         0,
		},
		Transactions: transactions,
	}

	return block
}

// CalculateHash calcula o hash do bloco (sem incluir a assinatura)
func (b *Block) CalculateHash() (string, error) {
	// Cria uma cópia do header sem assinatura para calcular o hash
	headerCopy := BlockHeader{
		Version:       b.Header.Version,
		Height:        b.Header.Height,
		Timestamp:     b.Header.Timestamp,
		PreviousHash:  b.Header.PreviousHash,
		MerkleRoot:    b.Header.MerkleRoot,
		ValidatorAddr: b.Header.ValidatorAddr,
		PublicKey:     b.Header.PublicKey,
		Nonce:         b.Header.Nonce,
	}

	data, err := json.Marshal(headerCopy)
	if err != nil {
		return "", fmt.Errorf("failed to marshal block header: %w", err)
	}

	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

// GetSignData retorna os dados que devem ser assinados pelo validador
func (b *Block) GetSignData() ([]byte, error) {
	headerCopy := BlockHeader{
		Version:       b.Header.Version,
		Height:        b.Header.Height,
		Timestamp:     b.Header.Timestamp,
		PreviousHash:  b.Header.PreviousHash,
		MerkleRoot:    b.Header.MerkleRoot,
		ValidatorAddr: b.Header.ValidatorAddr,
		Nonce:         b.Header.Nonce,
	}

	data, err := json.Marshal(headerCopy)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal block header for signing: %w", err)
	}

	return data, nil
}

// VerifyHash verifica se o hash do bloco está correto
func (b *Block) VerifyHash() error {
	if b.Hash == "" {
		return fmt.Errorf("block hash is empty")
	}

	calculatedHash, err := b.CalculateHash()
	if err != nil {
		return err
	}

	if b.Hash != calculatedHash {
		return fmt.Errorf("block hash mismatch: expected %s, got %s", calculatedHash, b.Hash)
	}

	return nil
}

// VerifyMerkleRoot verifica se a raiz de Merkle está correta
func (b *Block) VerifyMerkleRoot() error {
	calculatedRoot := b.Transactions.CalculateMerkleRoot()
	if b.Header.MerkleRoot != calculatedRoot {
		return fmt.Errorf("merkle root mismatch: expected %s, got %s", calculatedRoot, b.Header.MerkleRoot)
	}
	return nil
}

// VerifyTransactions verifica todas as transações do bloco
func (b *Block) VerifyTransactions() error {
	if len(b.Transactions) == 0 {
		return fmt.Errorf("block has no transactions")
	}

	// Verifica se há transações duplicadas
	if b.Transactions.HasDuplicates() {
		return fmt.Errorf("block contains duplicate transactions")
	}

	// Valida cada transação
	for i, tx := range b.Transactions {
		// Primeira transação deve ser coinbase
		if i == 0 {
			if !tx.IsCoinbase() {
				return fmt.Errorf("first transaction must be coinbase")
			}
			if err := tx.VerifyCoinbase(); err != nil {
				return fmt.Errorf("invalid coinbase transaction: %w", err)
			}
		} else {
			// Outras transações não devem ser coinbase
			if tx.IsCoinbase() {
				return fmt.Errorf("only first transaction can be coinbase")
			}
			if err := tx.Validate(); err != nil {
				return fmt.Errorf("invalid transaction at index %d: %w", i, err)
			}
		}
	}

	return nil
}

// Validate valida o bloco completamente
func (b *Block) Validate() error {
	// Valida campos obrigatórios
	if b.Header.Height == 0 && b.Header.PreviousHash != "" {
		return fmt.Errorf("genesis block must have empty previous hash")
	}

	if b.Header.Height > 0 && b.Header.PreviousHash == "" {
		return fmt.Errorf("non-genesis block must have previous hash")
	}

	if b.Header.ValidatorAddr == "" {
		return fmt.Errorf("validator address is empty")
	}

	// Valida timestamp (não pode ser muito no futuro)
	now := time.Now().Unix()
	if b.Header.Timestamp > now+300 { // 5 minutos de tolerância
		return fmt.Errorf("block timestamp is too far in the future")
	}

	// Verifica o hash do bloco
	if err := b.VerifyHash(); err != nil {
		return fmt.Errorf("block hash verification failed: %w", err)
	}

	// Verifica a raiz de Merkle
	if err := b.VerifyMerkleRoot(); err != nil {
		return fmt.Errorf("merkle root verification failed: %w", err)
	}

	// Verifica todas as transações
	if err := b.VerifyTransactions(); err != nil {
		return fmt.Errorf("transaction verification failed: %w", err)
	}

	return nil
}

// Serialize serializa o bloco para JSON
func (b *Block) Serialize() ([]byte, error) {
	return json.Marshal(b)
}

// DeserializeBlock desserializa um bloco de JSON
func DeserializeBlock(data []byte) (*Block, error) {
	var block Block
	err := json.Unmarshal(data, &block)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize block: %w", err)
	}
	return &block, nil
}

// IsGenesis verifica se o bloco é o bloco gênesis
func (b *Block) IsGenesis() bool {
	return b.Header.Height == 0 && b.Header.PreviousHash == ""
}

// GetCoinbaseTransaction retorna a transação coinbase do bloco
func (b *Block) GetCoinbaseTransaction() *Transaction {
	if len(b.Transactions) == 0 {
		return nil
	}
	if b.Transactions[0].IsCoinbase() {
		return b.Transactions[0]
	}
	return nil
}

// GetRegularTransactions retorna todas as transações exceto a coinbase
func (b *Block) GetRegularTransactions() TransactionSlice {
	if len(b.Transactions) <= 1 {
		return TransactionSlice{}
	}
	return b.Transactions[1:]
}

// TotalFees calcula o total de taxas das transações no bloco
func (b *Block) TotalFees() uint64 {
	return b.GetRegularTransactions().TotalFees()
}

// Copy cria uma cópia profunda do bloco
func (b *Block) Copy() *Block {
	// Copia as transações
	txsCopy := make(TransactionSlice, len(b.Transactions))
	for i, tx := range b.Transactions {
		txsCopy[i] = tx.Copy()
	}

	return &Block{
		Header: BlockHeader{
			Version:       b.Header.Version,
			Height:        b.Header.Height,
			Timestamp:     b.Header.Timestamp,
			PreviousHash:  b.Header.PreviousHash,
			MerkleRoot:    b.Header.MerkleRoot,
			ValidatorAddr: b.Header.ValidatorAddr,
			Signature:     b.Header.Signature,
			PublicKey:     b.Header.PublicKey,
			Nonce:         b.Header.Nonce,
		},
		Transactions: txsCopy,
		Hash:         b.Hash,
	}
}

// Equal compara se dois blocos são iguais
func (b *Block) Equal(other *Block) bool {
	if b == nil || other == nil {
		return b == other
	}
	return b.Hash == other.Hash
}

// String retorna uma representação em string do bloco
func (b *Block) String() string {
	return fmt.Sprintf("Block{Height: %d, Hash: %s, PrevHash: %s, Validator: %s, Txs: %d}",
		b.Header.Height,
		b.Hash[:16]+"...",
		b.Header.PreviousHash[:16]+"...",
		b.Header.ValidatorAddr[:16]+"...",
		len(b.Transactions),
	)
}

// GenesisBlock cria o bloco gênesis
func GenesisBlock(genesisTransaction *Transaction) *Block {
	if genesisTransaction == nil {
		return nil
	}

	transactions := TransactionSlice{genesisTransaction}
	merkleRoot := transactions.CalculateMerkleRoot()

	block := &Block{
		Header: BlockHeader{
			Version:       1,
			Height:        0,
			Timestamp:     time.Now().Unix(),
			PreviousHash:  "",
			MerkleRoot:    merkleRoot,
			ValidatorAddr: genesisTransaction.To, // O endereço do destinatário é o validador inicial
			Nonce:         0,
		},
		Transactions: transactions,
	}

	// Calcula o hash do bloco gênesis
	hash, _ := block.CalculateHash()
	block.Hash = hash

	return block
}

// ValidateGenesisBlock valida o bloco gênesis
func ValidateGenesisBlock(block *Block, expectedGenesisHash string) error {
	if !block.IsGenesis() {
		return fmt.Errorf("block is not a genesis block")
	}

	if block.Hash != expectedGenesisHash {
		return fmt.Errorf("genesis block hash mismatch: expected %s, got %s", expectedGenesisHash, block.Hash)
	}

	// Valida que tem exatamente uma transação coinbase
	if len(block.Transactions) != 1 {
		return fmt.Errorf("genesis block must have exactly one transaction")
	}

	if !block.Transactions[0].IsCoinbase() {
		return fmt.Errorf("genesis block transaction must be coinbase")
	}

	// Verifica o hash e merkle root
	if err := block.VerifyHash(); err != nil {
		return err
	}

	if err := block.VerifyMerkleRoot(); err != nil {
		return err
	}

	return nil
}

// BlockSlice é um slice de blocos com métodos auxiliares
type BlockSlice []*Block

// GetByHeight retorna um bloco pela altura
func (blocks BlockSlice) GetByHeight(height uint64) *Block {
	for _, block := range blocks {
		if block.Header.Height == height {
			return block
		}
	}
	return nil
}

// GetByHash retorna um bloco pelo hash
func (blocks BlockSlice) GetByHash(hash string) *Block {
	for _, block := range blocks {
		if block.Hash == hash {
			return block
		}
	}
	return nil
}

// ContainsHash verifica se um bloco com o hash especificado existe
func (blocks BlockSlice) ContainsHash(hash string) bool {
	return blocks.GetByHash(hash) != nil
}

// Sort ordena os blocos por altura
func (blocks BlockSlice) Sort() {
	// Bubble sort simples
	for i := 0; i < len(blocks); i++ {
		for j := i + 1; j < len(blocks); j++ {
			if blocks[i].Header.Height > blocks[j].Header.Height {
				blocks[i], blocks[j] = blocks[j], blocks[i]
			}
		}
	}
}

// ValidateChain valida uma sequência de blocos (sem validação de tempo mínimo)
func (blocks BlockSlice) ValidateChain() error {
	return blocks.ValidateChainWithConfig(nil)
}

// ValidateChainWithConfig valida uma sequência de blocos com configuração
func (blocks BlockSlice) ValidateChainWithConfig(config *ChainConfig) error {
	if len(blocks) == 0 {
		return fmt.Errorf("empty chain")
	}

	// Ordena os blocos
	blocks.Sort()

	// Primeiro bloco deve ser gênesis
	if !blocks[0].IsGenesis() {
		return fmt.Errorf("first block must be genesis")
	}

	// Valida cada bloco e a ligação com o anterior
	for i := 0; i < len(blocks); i++ {
		// Valida o bloco
		if err := blocks[i].Validate(); err != nil {
			return fmt.Errorf("block %d validation failed: %w", i, err)
		}

		// Verifica ligação com bloco anterior (exceto gênesis)
		if i > 0 {
			if blocks[i].Header.PreviousHash != blocks[i-1].Hash {
				return fmt.Errorf("block %d previous hash does not match block %d hash", i, i-1)
			}

			if blocks[i].Header.Height != blocks[i-1].Header.Height+1 {
				return fmt.Errorf("block %d height is not sequential", i)
			}

			if blocks[i].Header.Timestamp < blocks[i-1].Header.Timestamp {
				return fmt.Errorf("block %d timestamp is before previous block", i)
			}

			// Verifica tempo mínimo entre blocos (80% do BlockTime configurado)
			if config != nil {
				minBlockTime := int64(config.BlockTime.Seconds() * 0.8)
				timeDiff := blocks[i].Header.Timestamp - blocks[i-1].Header.Timestamp
				if timeDiff < minBlockTime {
					return fmt.Errorf("block %d timestamp difference (%d seconds) is less than minimum block time (%d seconds, 80%% of %v)",
						i, timeDiff, minBlockTime, config.BlockTime)
				}
			}
		}
	}

	return nil
}

// LastBlock retorna o último bloco da chain
func (blocks BlockSlice) LastBlock() *Block {
	if len(blocks) == 0 {
		return nil
	}
	// Assume que está ordenado
	return blocks[len(blocks)-1]
}

// Height retorna a altura da chain (altura do último bloco)
func (blocks BlockSlice) Height() uint64 {
	last := blocks.LastBlock()
	if last == nil {
		return 0
	}
	return last.Header.Height
}
