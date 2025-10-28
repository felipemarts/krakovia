package blockchain

import (
	"fmt"
	"time"

	"github.com/krakovia/blockchain/pkg/wallet"
)

// Miner representa um minerador/validador
type Miner struct {
	// Identificação
	address string
	wallet  *wallet.Wallet

	// Referências
	chain   *Chain
	mempool *Mempool

	// Callbacks para propagação
	onBlockCreated func(*Block)
	onTxCreated    func(*Transaction)

	// Controle
	mining    bool
	lastMined time.Time
}

// NewMiner cria um novo minerador
func NewMiner(w *wallet.Wallet, chain *Chain, mempool *Mempool) *Miner {
	return &Miner{
		address: w.GetAddress(),
		wallet:  w,
		chain:   chain,
		mempool: mempool,
	}
}

// SetOnBlockCreated define callback para quando um bloco é criado
func (m *Miner) SetOnBlockCreated(callback func(*Block)) {
	m.onBlockCreated = callback
}

// SetOnTxCreated define callback para quando uma transação é criada
func (m *Miner) SetOnTxCreated(callback func(*Transaction)) {
	m.onTxCreated = callback
}

// GetAddress retorna o endereço do minerador
func (m *Miner) GetAddress() string {
	return m.address
}

// GetWallet retorna a carteira do minerador
func (m *Miner) GetWallet() *wallet.Wallet {
	return m.wallet
}

// CanMine verifica se o minerador pode minerar (tem stake suficiente)
func (m *Miner) CanMine() bool {
	stake := m.chain.GetStake(m.address)
	minStake := m.chain.GetConfig().MinValidatorStake
	return stake >= minStake
}

// IsMyTurn verifica se é a vez deste minerador criar o bloco
func (m *Miner) IsMyTurn() bool {
	validators := m.chain.GetValidators()
	if len(validators) == 0 {
		return false
	}

	// Usa o hash do último bloco para determinar prioridade
	lastBlock := m.chain.GetLastBlock()
	if lastBlock == nil {
		return false
	}

	// Calcula prioridade dos validadores
	pq, err := CalculateValidatorPriority(lastBlock.Hash, validators)
	if err != nil {
		return false
	}

	// Verifica se somos o top validator
	return pq.IsTopValidator(m.address)
}

// TryMineBlock tenta criar um bloco se for a vez do minerador
func (m *Miner) TryMineBlock() (*Block, error) {
	// Verifica se pode minerar
	if !m.CanMine() {
		return nil, fmt.Errorf("insufficient stake to mine")
	}

	// Verifica se é sua vez
	if !m.IsMyTurn() {
		return nil, fmt.Errorf("not this miner's turn")
	}

	// Verifica tempo desde último bloco
	lastBlock := m.chain.GetLastBlock()
	config := m.chain.GetConfig()

	timeSinceLastBlock := time.Since(time.Unix(lastBlock.Header.Timestamp, 0))
	if timeSinceLastBlock < config.BlockTime {
		return nil, fmt.Errorf("too soon to mine (need to wait %v)", config.BlockTime-timeSinceLastBlock)
	}

	// Cria o bloco
	block, err := m.CreateBlock()
	if err != nil {
		return nil, fmt.Errorf("failed to create block: %w", err)
	}

	// Marca tempo de mineração
	m.lastMined = time.Now()

	// Callback
	if m.onBlockCreated != nil {
		m.onBlockCreated(block)
	}

	return block, nil
}

// CreateBlock cria um novo bloco com transações do mempool
func (m *Miner) CreateBlock() (*Block, error) {
	lastBlock := m.chain.GetLastBlock()
	if lastBlock == nil {
		return nil, fmt.Errorf("no last block")
	}

	config := m.chain.GetConfig()

	// Cria transação coinbase (recompensa)
	coinbase := NewCoinbaseTransaction(
		m.address,
		config.BlockReward,
		lastBlock.Header.Height+1,
	)

	// Pega transações válidas do mempool
	validTxs := m.mempool.GetValidTransactions(m.chain.context, config.MaxBlockSize-1)

	// Monta lista de transações (coinbase primeiro)
	transactions := make(TransactionSlice, 0, len(validTxs)+1)
	transactions = append(transactions, coinbase)
	transactions = append(transactions, validTxs...)

	// Cria o bloco
	block := NewBlock(
		lastBlock.Header.Height+1,
		lastBlock.Hash,
		transactions,
		m.address,
	)

	// Calcula hash
	hash, err := block.CalculateHash()
	if err != nil {
		return nil, fmt.Errorf("failed to calculate block hash: %w", err)
	}
	block.Hash = hash

	// Valida bloco
	if err := block.Validate(); err != nil {
		return nil, fmt.Errorf("created invalid block: %w", err)
	}

	return block, nil
}

// CreateTransaction cria uma nova transação assinada
func (m *Miner) CreateTransaction(to string, amount, fee uint64, data string) (*Transaction, error) {
	nonce := m.chain.GetNonce(m.address)

	tx := NewTransaction(m.address, to, amount, fee, nonce, data)

	if err := tx.Sign(m.wallet); err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Callback
	if m.onTxCreated != nil {
		m.onTxCreated(tx)
	}

	return tx, nil
}

// CreateStakeTransaction cria uma transação para fazer stake
func (m *Miner) CreateStakeTransaction(amount, fee uint64) (*Transaction, error) {
	stakeData := NewStakeData(amount)
	dataStr, err := stakeData.Serialize()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize stake data: %w", err)
	}

	return m.CreateTransaction(m.address, amount, fee, dataStr)
}

// CreateUnstakeTransaction cria uma transação para fazer unstake
func (m *Miner) CreateUnstakeTransaction(amount, fee uint64) (*Transaction, error) {
	unstakeData := NewUnstakeData(amount)
	dataStr, err := unstakeData.Serialize()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize unstake data: %w", err)
	}

	return m.CreateTransaction(m.address, amount, fee, dataStr)
}

// MineLoop inicia loop de mineração (para testes)
// Retorna quando stopChan recebe sinal
func (m *Miner) MineLoop(stopChan <-chan struct{}) {
	m.mining = true
	defer func() { m.mining = false }()

	config := m.chain.GetConfig()
	ticker := time.NewTicker(config.BlockTime / 4) // Verifica 4x por período de bloco
	defer ticker.Stop()

	for {
		select {
		case <-stopChan:
			return

		case <-ticker.C:
			// Tenta minerar
			block, err := m.TryMineBlock()
			if err != nil {
				// Silenciosamente ignora erros (é normal não ser a vez)
				continue
			}

			// Se conseguiu criar bloco, adiciona à chain
			if err := m.chain.AddBlock(block); err != nil {
				// Erro ao adicionar bloco
				continue
			}

			// Remove transações do mempool
			txIDs := make([]string, 0, len(block.Transactions)-1)
			for i := 1; i < len(block.Transactions); i++ { // Pula coinbase
				txIDs = append(txIDs, block.Transactions[i].ID)
			}
			m.mempool.RemoveTransactions(txIDs)
		}
	}
}

// IsMining retorna se o minerador está ativamente minerando
func (m *Miner) IsMining() bool {
	return m.mining
}

// GetLastMinedTime retorna o tempo desde a última mineração
func (m *Miner) GetLastMinedTime() time.Time {
	return m.lastMined
}

// GetBalance retorna o saldo do minerador
func (m *Miner) GetBalance() uint64 {
	return m.chain.GetBalance(m.address)
}

// GetStake retorna o stake do minerador
func (m *Miner) GetStake() uint64 {
	return m.chain.GetStake(m.address)
}

// GetNonce retorna o nonce do minerador
func (m *Miner) GetNonce() uint64 {
	return m.chain.GetNonce(m.address)
}

// GetRank retorna a posição do minerador no ranking de validadores
func (m *Miner) GetRank() int {
	validators := m.chain.GetValidators()
	if len(validators) == 0 {
		return -1
	}

	lastBlock := m.chain.GetLastBlock()
	if lastBlock == nil {
		return -1
	}

	pq, err := CalculateValidatorPriority(lastBlock.Hash, validators)
	if err != nil {
		return -1
	}

	return pq.GetValidatorRank(m.address)
}
