package blockchain

import (
	"fmt"
	"sync"
	"time"
)

// ChainConfig configurações da blockchain
type ChainConfig struct {
	BlockTime         time.Duration // Tempo entre blocos (200-300ms para testes)
	MaxBlockSize      int           // Máximo de transações por bloco
	BlockReward       uint64        // Recompensa por bloco
	MinValidatorStake uint64        // Stake mínimo para ser validador
}

// DefaultChainConfig retorna configurações padrão para testes
func DefaultChainConfig() ChainConfig {
	return ChainConfig{
		BlockTime:         200 * time.Millisecond, // 200ms entre blocos (otimizado para testes rápidos)
		MaxBlockSize:      1000,
		BlockReward:       50,
		MinValidatorStake: 100,
	}
}

// Chain representa a blockchain completa
type Chain struct {
	mu sync.RWMutex

	// Configuração
	config ChainConfig

	// Blocos da chain (ordenados por altura)
	blocks BlockSlice

	// Contexto de execução (estado)
	context *Context

	// Map de hash -> bloco para lookup rápido
	blocksByHash map[string]*Block

	// Bloco gênesis
	genesis *Block
}

// NewChain cria uma nova blockchain com bloco gênesis
func NewChain(genesisBlock *Block, config ChainConfig) (*Chain, error) {
	if genesisBlock == nil {
		return nil, fmt.Errorf("genesis block is required")
	}

	if !genesisBlock.IsGenesis() {
		return nil, fmt.Errorf("provided block is not a genesis block")
	}

	// Valida bloco gênesis
	if err := genesisBlock.Validate(); err != nil {
		return nil, fmt.Errorf("invalid genesis block: %w", err)
	}

	// Cria contexto com gênesis
	ctx, err := NewContextWithGenesis(genesisBlock)
	if err != nil {
		return nil, fmt.Errorf("failed to create context: %w", err)
	}

	chain := &Chain{
		config:       config,
		blocks:       BlockSlice{genesisBlock},
		context:      ctx,
		blocksByHash: make(map[string]*Block),
		genesis:      genesisBlock,
	}

	chain.blocksByHash[genesisBlock.Hash] = genesisBlock

	return chain, nil
}

// AddBlock adiciona um novo bloco à chain
func (c *Chain) AddBlock(block *Block) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Valida o bloco
	if err := block.Validate(); err != nil {
		return fmt.Errorf("block validation failed: %w", err)
	}

	// Verifica se já existe
	if _, exists := c.blocksByHash[block.Hash]; exists {
		return fmt.Errorf("block already exists in chain")
	}

	// Verifica conexão com a chain
	lastBlock := c.blocks[len(c.blocks)-1]
	if block.Header.PreviousHash != lastBlock.Hash {
		return fmt.Errorf("block does not connect to chain: expected previous hash %s, got %s",
			lastBlock.Hash, block.Header.PreviousHash)
	}

	if block.Header.Height != lastBlock.Header.Height+1 {
		return fmt.Errorf("invalid block height: expected %d, got %d",
			lastBlock.Header.Height+1, block.Header.Height)
	}

	// Adiciona ao contexto (executa transações)
	if err := c.context.AddBlock(block); err != nil {
		return fmt.Errorf("failed to add block to context: %w", err)
	}

	// Adiciona à chain
	c.blocks = append(c.blocks, block)
	c.blocksByHash[block.Hash] = block

	return nil
}

// GetBlock retorna um bloco pelo hash
func (c *Chain) GetBlock(hash string) (*Block, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	block, exists := c.blocksByHash[hash]
	return block, exists
}

// GetBlockByHeight retorna um bloco pela altura
func (c *Chain) GetBlockByHeight(height uint64) (*Block, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if height >= uint64(len(c.blocks)) {
		return nil, false
	}

	return c.blocks[height], true
}

// GetLastBlock retorna o último bloco da chain
func (c *Chain) GetLastBlock() *Block {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.blocks) == 0 {
		return nil
	}

	return c.blocks[len(c.blocks)-1]
}

// GetHeight retorna a altura atual da chain
func (c *Chain) GetHeight() uint64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return uint64(len(c.blocks) - 1)
}

// GetBalance retorna o saldo de um endereço
func (c *Chain) GetBalance(address string) uint64 {
	return c.context.GetBalance(address)
}

// GetStake retorna o stake de um endereço
func (c *Chain) GetStake(address string) uint64 {
	return c.context.GetStake(address)
}

// GetNonce retorna o nonce de um endereço
func (c *Chain) GetNonce(address string) uint64 {
	return c.context.GetNonce(address)
}

// GetValidators retorna os validadores ativos
func (c *Chain) GetValidators() ValidatorList {
	validators := c.context.GetValidators()

	// Filtra por stake mínimo
	filtered := make(ValidatorList, 0)
	for _, v := range validators {
		if v.Stake >= c.config.MinValidatorStake {
			filtered = append(filtered, v)
		}
	}

	return filtered
}

// ValidateTransaction valida uma transação no contexto atual
func (c *Chain) ValidateTransaction(tx *Transaction) error {
	_, err := c.context.ExecuteTransaction(tx)
	return err
}

// GetConfig retorna a configuração da chain
func (c *Chain) GetConfig() ChainConfig {
	return c.config
}

// GetGenesis retorna o bloco gênesis
func (c *Chain) GetGenesis() *Block {
	return c.genesis
}

// GetAllBlocks retorna todos os blocos (cópia)
func (c *Chain) GetAllBlocks() []*Block {
	c.mu.RLock()
	defer c.mu.RUnlock()

	blocks := make([]*Block, len(c.blocks))
	copy(blocks, c.blocks)
	return blocks
}

// GetBlockRange retorna blocos em um intervalo de altura
func (c *Chain) GetBlockRange(start, end uint64) []*Block {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if start > end || start >= uint64(len(c.blocks)) {
		return []*Block{}
	}

	if end >= uint64(len(c.blocks)) {
		end = uint64(len(c.blocks) - 1)
	}

	blocks := make([]*Block, end-start+1)
	for i := start; i <= end; i++ {
		blocks[i-start] = c.blocks[i]
	}

	return blocks
}

// VerifyChain verifica a integridade de toda a chain
func (c *Chain) VerifyChain() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.blocks.ValidateChainWithConfig(&c.config)
}

// Clone cria uma cópia da chain (para simulações)
func (c *Chain) Clone() (*Chain, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Cria nova chain com mesmo gênesis
	newChain, err := NewChain(c.genesis, c.config)
	if err != nil {
		return nil, err
	}

	// Adiciona todos os blocos exceto gênesis
	for i := 1; i < len(c.blocks); i++ {
		if err := newChain.AddBlock(c.blocks[i]); err != nil {
			return nil, fmt.Errorf("failed to clone block %d: %w", i, err)
		}
	}

	return newChain, nil
}

// GetChainStats retorna estatísticas da chain
func (c *Chain) GetChainStats() ChainStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := ChainStats{
		Height:       uint64(len(c.blocks) - 1),
		TotalBlocks:  len(c.blocks),
		GenesisHash:  c.genesis.Hash,
		LastBlock:    c.blocks[len(c.blocks)-1].Hash,
		Validators:   len(c.GetValidators()),
	}

	// Calcula total de transações
	for _, block := range c.blocks {
		stats.TotalTransactions += len(block.Transactions)
	}

	// Calcula tempo médio entre blocos
	if len(c.blocks) > 1 {
		totalTime := c.blocks[len(c.blocks)-1].Header.Timestamp - c.blocks[0].Header.Timestamp
		stats.AverageBlockTime = time.Duration(totalTime/(int64(len(c.blocks))-1)) * time.Second
	}

	return stats
}

// ChainStats estatísticas da blockchain
type ChainStats struct {
	Height             uint64        // Altura atual
	TotalBlocks        int           // Total de blocos
	TotalTransactions  int           // Total de transações
	GenesisHash        string        // Hash do gênesis
	LastBlock          string        // Hash do último bloco
	Validators         int           // Número de validadores ativos
	AverageBlockTime   time.Duration // Tempo médio entre blocos
}
