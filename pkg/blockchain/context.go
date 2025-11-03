package blockchain

import (
	"fmt"
	"strings"
	"sync"
)

// StateKey representa uma chave no estado da blockchain
type StateKey string

// Prefixos para diferentes tipos de dados no estado
const (
	PrefixBalance = "wallet"  // wallet-<address> = saldo
	PrefixStake   = "stake"   // stake-<address> = stake amount
	PrefixNonce   = "nonce"   // nonce-<address> = nonce
	PrefixCustom  = "custom"  // custom-<key> = valor customizado
)

// StateModifications representa as modificações de estado em um bloco
type StateModifications map[StateKey]uint64

// BlockContext representa o contexto de execução de um bloco específico
type BlockContext struct {
	BlockHash     string             // Hash do bloco
	PreviousHash  string             // Hash do bloco anterior
	Height        uint64             // Altura do bloco
	Transactions  TransactionSlice   // Transações do bloco
	Modifications StateModifications // Modificações de estado causadas por este bloco
}

// Context representa o banco de dados em memória da blockchain
type Context struct {
	mu sync.RWMutex

	// Map de hash do bloco -> contexto do bloco
	blocks map[string]*BlockContext

	// Hash do último bloco processado
	lastBlockHash string

	// Altura do último bloco
	lastBlockHeight uint64

	// Estado atual acumulado (cache para performance)
	currentState StateModifications
}

// NewContext cria um novo contexto vazio
func NewContext() *Context {
	return &Context{
		blocks:       make(map[string]*BlockContext),
		currentState: make(StateModifications),
	}
}

// NewContextWithGenesis cria um novo contexto com bloco gênesis
func NewContextWithGenesis(genesisBlock *Block) (*Context, error) {
	ctx := NewContext()

	// Processa o bloco gênesis
	err := ctx.AddBlock(genesisBlock)
	if err != nil {
		return nil, fmt.Errorf("failed to add genesis block: %w", err)
	}

	return ctx, nil
}

// GetBlock retorna o contexto de um bloco pelo hash
func (c *Context) GetBlock(blockHash string) (*BlockContext, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	blockCtx, ok := c.blocks[blockHash]
	return blockCtx, ok
}

// GetLastBlockHash retorna o hash do último bloco
func (c *Context) GetLastBlockHash() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastBlockHash
}

// GetLastBlockHeight retorna a altura do último bloco
func (c *Context) GetLastBlockHeight() uint64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastBlockHeight
}

// GetBalance retorna o saldo de um endereço
func (c *Context) GetBalance(address string) uint64 {
	key := MakeBalanceKey(address)
	return c.GetState(key)
}

// GetStake retorna o stake de um endereço
func (c *Context) GetStake(address string) uint64 {
	key := MakeStakeKey(address)
	return c.GetState(key)
}

// GetNonce retorna o nonce de um endereço
func (c *Context) GetNonce(address string) uint64 {
	key := MakeNonceKey(address)
	return c.GetState(key)
}

// GetState retorna um valor do estado, percorrendo a cadeia de blocos se necessário
func (c *Context) GetState(key StateKey) uint64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Primeiro verifica no estado atual (cache)
	if value, ok := c.currentState[key]; ok {
		return value
	}

	// Se não está no cache, procura nos blocos anteriores
	return c.getStateFromChain(key, c.lastBlockHash)
}

// getStateFromChain busca um valor percorrendo a cadeia de blocos (não thread-safe, deve ser chamado com lock)
func (c *Context) getStateFromChain(key StateKey, blockHash string) uint64 {
	if blockHash == "" {
		return 0 // Valor padrão
	}

	blockCtx, ok := c.blocks[blockHash]
	if !ok {
		return 0
	}

	// Verifica se a chave foi modificada neste bloco
	if value, ok := blockCtx.Modifications[key]; ok {
		return value
	}

	// Recursivamente busca no bloco anterior
	return c.getStateFromChain(key, blockCtx.PreviousHash)
}

// AddBlock adiciona um novo bloco ao contexto
func (c *Context) AddBlock(block *Block) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Valida que o bloco conecta corretamente
	if block.Header.Height > 0 {
		if block.Header.PreviousHash != c.lastBlockHash {
			return fmt.Errorf("block does not connect to last block: expected previous hash %s, got %s",
				c.lastBlockHash, block.Header.PreviousHash)
		}
		if block.Header.Height != c.lastBlockHeight+1 {
			return fmt.Errorf("block height mismatch: expected %d, got %d",
				c.lastBlockHeight+1, block.Header.Height)
		}
	}

	// Cria contexto temporário para executar o bloco
	tempModifications := make(StateModifications)

	// Copia o estado atual para o temporário
	for k, v := range c.currentState {
		tempModifications[k] = v
	}

	// Executa todas as transações do bloco
	for i, tx := range block.Transactions {
		modifications, err := c.executeTransactionInternal(tx, tempModifications, block.Header.Height)
		if err != nil {
			return fmt.Errorf("failed to execute transaction %d (%s): %w", i, tx.ID, err)
		}

		// Aplica as modificações ao estado temporário
		for key, value := range modifications {
			tempModifications[key] = value
		}
	}

	// Calcula apenas as modificações deste bloco (diferença do estado anterior)
	blockModifications := make(StateModifications)
	for key, newValue := range tempModifications {
		oldValue := c.currentState[key]
		if newValue != oldValue {
			blockModifications[key] = newValue
		}
	}

	// Cria o contexto do bloco
	blockCtx := &BlockContext{
		BlockHash:     block.Hash,
		PreviousHash:  block.Header.PreviousHash,
		Height:        block.Header.Height,
		Transactions:  block.Transactions,
		Modifications: blockModifications,
	}

	// Adiciona ao mapa de blocos
	c.blocks[block.Hash] = blockCtx

	// Atualiza o estado atual
	c.currentState = tempModifications

	// Atualiza referências do último bloco
	c.lastBlockHash = block.Hash
	c.lastBlockHeight = block.Header.Height

	return nil
}

// executeTransactionInternal executa uma transação e retorna as modificações (não thread-safe)
func (c *Context) executeTransactionInternal(tx *Transaction, currentState StateModifications, blockHeight uint64) (StateModifications, error) {
	modifications := make(StateModifications)

	// Valida a transação
	if !tx.IsCoinbase() {
		if err := tx.Validate(); err != nil {
			return nil, fmt.Errorf("transaction validation failed: %w", err)
		}

		// Verifica nonce
		expectedNonce := currentState[MakeNonceKey(tx.From)]
		if tx.Nonce != expectedNonce {
			return nil, fmt.Errorf("invalid nonce: expected %d, got %d", expectedNonce, tx.Nonce)
		}

		// Incrementa nonce
		modifications[MakeNonceKey(tx.From)] = expectedNonce + 1

		// Verifica saldo suficiente (amount + fee)
		balance := currentState[MakeBalanceKey(tx.From)]
		totalCost := tx.Amount + tx.Fee

		if balance < totalCost {
			return nil, fmt.Errorf("insufficient balance: have %d, need %d", balance, totalCost)
		}
	}

	// Parse transaction data
	txData, err := DeserializeTransactionData(tx.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse transaction data: %w", err)
	}

	// Valida transaction data
	if txData != nil {
		if err := txData.Validate(); err != nil {
			return nil, fmt.Errorf("transaction data validation failed: %w", err)
		}
	}

	// Executa baseado no tipo
	if tx.IsCoinbase() {
		// Coinbase: adiciona recompensa ao validador
		toBalance := currentState[MakeBalanceKey(tx.To)]
		modifications[MakeBalanceKey(tx.To)] = toBalance + tx.Amount
	} else if txData != nil && txData.Type == TransactionTypeStake {
		// Stake: move saldo para stake
		stakeAmount, err := txData.GetStakeAmount()
		if err != nil {
			return nil, err
		}

		if stakeAmount != tx.Amount {
			return nil, fmt.Errorf("stake amount mismatch: payload=%d, tx.Amount=%d", stakeAmount, tx.Amount)
		}

		fromBalance := currentState[MakeBalanceKey(tx.From)]
		fromStake := currentState[MakeStakeKey(tx.From)]

		modifications[MakeBalanceKey(tx.From)] = fromBalance - tx.Amount - tx.Fee
		modifications[MakeStakeKey(tx.From)] = fromStake + tx.Amount
	} else if txData != nil && txData.Type == TransactionTypeUnstake {
		// Unstake: move stake para saldo
		unstakeAmount, err := txData.GetStakeAmount()
		if err != nil {
			return nil, err
		}

		if unstakeAmount != tx.Amount {
			return nil, fmt.Errorf("unstake amount mismatch: payload=%d, tx.Amount=%d", unstakeAmount, tx.Amount)
		}

		fromBalance := currentState[MakeBalanceKey(tx.From)]
		fromStake := currentState[MakeStakeKey(tx.From)]

		if fromStake < unstakeAmount {
			return nil, fmt.Errorf("insufficient stake: have %d, need %d", fromStake, unstakeAmount)
		}

		modifications[MakeBalanceKey(tx.From)] = fromBalance - tx.Fee + tx.Amount
		modifications[MakeStakeKey(tx.From)] = fromStake - tx.Amount
	} else {
		// Transfer: transferência normal
		fromBalance := currentState[MakeBalanceKey(tx.From)]
		toBalance := currentState[MakeBalanceKey(tx.To)]

		modifications[MakeBalanceKey(tx.From)] = fromBalance - tx.Amount - tx.Fee
		modifications[MakeBalanceKey(tx.To)] = toBalance + tx.Amount
	}

	return modifications, nil
}

// ExecuteTransaction simula a execução de uma transação sem modificar o contexto
// Retorna as modificações que seriam aplicadas e um erro se a transação falhar
func (c *Context) ExecuteTransaction(tx *Transaction) (StateModifications, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Cria uma cópia do estado atual
	tempState := make(StateModifications)
	for k, v := range c.currentState {
		tempState[k] = v
	}

	// Executa a transação
	return c.executeTransactionInternal(tx, tempState, c.lastBlockHeight+1)
}

// MakeBalanceKey cria uma chave para saldo
func MakeBalanceKey(address string) StateKey {
	return StateKey(fmt.Sprintf("%s-%s", PrefixBalance, address))
}

// MakeStakeKey cria uma chave para stake
func MakeStakeKey(address string) StateKey {
	return StateKey(fmt.Sprintf("%s-%s", PrefixStake, address))
}

// MakeNonceKey cria uma chave para nonce
func MakeNonceKey(address string) StateKey {
	return StateKey(fmt.Sprintf("%s-%s", PrefixNonce, address))
}

// MakeCustomKey cria uma chave customizada
func MakeCustomKey(key string) StateKey {
	return StateKey(fmt.Sprintf("%s-%s", PrefixCustom, key))
}

// ParseStateKey extrai o prefixo e o valor de uma chave
func ParseStateKey(key StateKey) (prefix, value string) {
	parts := strings.SplitN(string(key), "-", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return string(key), ""
}

// GetAllBalances retorna todos os saldos no estado atual
func (c *Context) GetAllBalances() map[string]uint64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	balances := make(map[string]uint64)
	for key, value := range c.currentState {
		prefix, address := ParseStateKey(key)
		if prefix == PrefixBalance && value > 0 {
			balances[address] = value
		}
	}
	return balances
}

// GetAllStakes retorna todos os stakes no estado atual
func (c *Context) GetAllStakes() map[string]uint64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stakes := make(map[string]uint64)
	for key, value := range c.currentState {
		prefix, address := ParseStateKey(key)
		if prefix == PrefixStake && value > 0 {
			stakes[address] = value
		}
	}
	return stakes
}

// GetAllNonces retorna todos os nonces no estado atual
func (c *Context) GetAllNonces() map[string]uint64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	nonces := make(map[string]uint64)
	for key, value := range c.currentState {
		prefix, address := ParseStateKey(key)
		if prefix == PrefixNonce && value > 0 {
			nonces[address] = value
		}
	}
	return nonces
}

// GetValidators retorna a lista de validadores ativos (endereços com stake > 0)
func (c *Context) GetValidators() ValidatorList {
	stakes := c.GetAllStakes()

	validators := make(ValidatorList, 0, len(stakes))
	for address, stake := range stakes {
		if stake > 0 {
			validators = append(validators, Validator{
				Address: address,
				Stake:   stake,
			})
		}
	}

	return validators
}

// GetBlockCount retorna o número de blocos no contexto
func (c *Context) GetBlockCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.blocks)
}

// Reset limpa o contexto
func (c *Context) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.blocks = make(map[string]*BlockContext)
	c.currentState = make(StateModifications)
	c.lastBlockHash = ""
	c.lastBlockHeight = 0
}
