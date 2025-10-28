package blockchain

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

// Mempool representa o pool de transações pendentes
type Mempool struct {
	mu sync.RWMutex

	// Map de ID da transação -> transação
	transactions map[string]*Transaction

	// Map de endereço -> lista de transações do endereço (ordenadas por nonce)
	transactionsByAddress map[string][]*Transaction

	// Configurações
	maxSize         int           // Tamanho máximo do mempool
	maxTxAge        time.Duration // Idade máxima de uma transação
	minFee          uint64        // Taxa mínima aceita
	maxTxPerAddress int           // Máximo de transações por endereço
}

// MempoolConfig configurações do mempool
type MempoolConfig struct {
	MaxSize         int           // Padrão: 10000
	MaxTxAge        time.Duration // Padrão: 1 hora
	MinFee          uint64        // Padrão: 1
	MaxTxPerAddress int           // Padrão: 100
}

// DefaultMempoolConfig retorna configurações padrão
func DefaultMempoolConfig() MempoolConfig {
	return MempoolConfig{
		MaxSize:         10000,
		MaxTxAge:        1 * time.Hour,
		MinFee:          1,
		MaxTxPerAddress: 100,
	}
}

// NewMempool cria um novo mempool com configurações padrão
func NewMempool() *Mempool {
	return NewMempoolWithConfig(DefaultMempoolConfig())
}

// NewMempoolWithConfig cria um novo mempool com configurações customizadas
func NewMempoolWithConfig(config MempoolConfig) *Mempool {
	return &Mempool{
		transactions:          make(map[string]*Transaction),
		transactionsByAddress: make(map[string][]*Transaction),
		maxSize:               config.MaxSize,
		maxTxAge:              config.MaxTxAge,
		minFee:                config.MinFee,
		maxTxPerAddress:       config.MaxTxPerAddress,
	}
}

// AddTransaction adiciona uma transação ao mempool
func (mp *Mempool) AddTransaction(tx *Transaction) error {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	// Valida a transação
	if err := tx.Validate(); err != nil {
		return fmt.Errorf("transaction validation failed: %w", err)
	}

	// Verifica se já existe
	if _, exists := mp.transactions[tx.ID]; exists {
		return fmt.Errorf("transaction already in mempool")
	}

	// Verifica taxa mínima
	if tx.Fee < mp.minFee {
		return fmt.Errorf("transaction fee %d is below minimum %d", tx.Fee, mp.minFee)
	}

	// Verifica tamanho do mempool
	if len(mp.transactions) >= mp.maxSize {
		// Remove transação com menor taxa para dar espaço
		if !mp.removeLowFeeTx(tx.Fee) {
			return fmt.Errorf("mempool is full and transaction fee is too low")
		}
	}

	// Verifica limite de transações por endereço
	addressTxs := mp.transactionsByAddress[tx.From]
	if len(addressTxs) >= mp.maxTxPerAddress {
		return fmt.Errorf("address %s has reached maximum pending transactions (%d)",
			tx.From, mp.maxTxPerAddress)
	}

	// Adiciona ao mempool
	mp.transactions[tx.ID] = tx

	// Adiciona ao índice por endereço
	mp.transactionsByAddress[tx.From] = append(addressTxs, tx)

	// Ordena por nonce
	sort.Slice(mp.transactionsByAddress[tx.From], func(i, j int) bool {
		return mp.transactionsByAddress[tx.From][i].Nonce < mp.transactionsByAddress[tx.From][j].Nonce
	})

	return nil
}

// RemoveTransaction remove uma transação do mempool
func (mp *Mempool) RemoveTransaction(txID string) bool {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	tx, exists := mp.transactions[txID]
	if !exists {
		return false
	}

	// Remove do mapa principal
	delete(mp.transactions, txID)

	// Remove do índice por endereço
	addressTxs := mp.transactionsByAddress[tx.From]
	for i, addrTx := range addressTxs {
		if addrTx.ID == txID {
			mp.transactionsByAddress[tx.From] = append(addressTxs[:i], addressTxs[i+1:]...)
			break
		}
	}

	// Remove entrada vazia do mapa
	if len(mp.transactionsByAddress[tx.From]) == 0 {
		delete(mp.transactionsByAddress, tx.From)
	}

	return true
}

// GetTransaction retorna uma transação pelo ID
func (mp *Mempool) GetTransaction(txID string) (*Transaction, bool) {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	tx, exists := mp.transactions[txID]
	return tx, exists
}

// GetTransactions retorna todas as transações do mempool
func (mp *Mempool) GetTransactions() []*Transaction {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	txs := make([]*Transaction, 0, len(mp.transactions))
	for _, tx := range mp.transactions {
		txs = append(txs, tx)
	}
	return txs
}

// GetTransactionsByAddress retorna transações de um endereço específico
func (mp *Mempool) GetTransactionsByAddress(address string) []*Transaction {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	txs := mp.transactionsByAddress[address]
	result := make([]*Transaction, len(txs))
	copy(result, txs)
	return result
}

// GetPendingTransactions retorna transações ordenadas por fee (maior primeiro)
// Útil para mineração
func (mp *Mempool) GetPendingTransactions(maxCount int) []*Transaction {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	txs := make([]*Transaction, 0, len(mp.transactions))
	for _, tx := range mp.transactions {
		txs = append(txs, tx)
	}

	// Ordena por fee (maior primeiro), depois por timestamp (mais antigo primeiro)
	sort.Slice(txs, func(i, j int) bool {
		if txs[i].Fee != txs[j].Fee {
			return txs[i].Fee > txs[j].Fee
		}
		return txs[i].Timestamp < txs[j].Timestamp
	})

	if maxCount > 0 && len(txs) > maxCount {
		return txs[:maxCount]
	}

	return txs
}

// GetValidTransactions retorna transações válidas para inclusão em um bloco
// Filtra por nonce correto e valida no contexto atual
func (mp *Mempool) GetValidTransactions(ctx *Context, maxCount int) []*Transaction {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	valid := make([]*Transaction, 0)

	// Agrupa transações por endereço
	for address, addressTxs := range mp.transactionsByAddress {
		// Pega o nonce atual do endereço
		currentNonce := ctx.GetNonce(address)

		// Procura transações com nonce sequencial
		for _, tx := range addressTxs {
			if tx.Nonce == currentNonce {
				// Simula execução
				_, err := ctx.ExecuteTransaction(tx)
				if err == nil {
					valid = append(valid, tx)
					currentNonce++

					if maxCount > 0 && len(valid) >= maxCount {
						break
					}
				}
			} else if tx.Nonce > currentNonce {
				// Nonces futuros, pula
				break
			}
			// Nonces antigos são ignorados
		}

		if maxCount > 0 && len(valid) >= maxCount {
			break
		}
	}

	// Ordena por fee
	sort.Slice(valid, func(i, j int) bool {
		return valid[i].Fee > valid[j].Fee
	})

	return valid
}

// RemoveTransactions remove múltiplas transações (útil após criar um bloco)
func (mp *Mempool) RemoveTransactions(txIDs []string) int {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	count := 0
	for _, txID := range txIDs {
		tx, exists := mp.transactions[txID]
		if !exists {
			continue
		}

		// Remove do mapa principal
		delete(mp.transactions, txID)

		// Remove do índice por endereço
		addressTxs := mp.transactionsByAddress[tx.From]
		for i, addrTx := range addressTxs {
			if addrTx.ID == txID {
				mp.transactionsByAddress[tx.From] = append(addressTxs[:i], addressTxs[i+1:]...)
				break
			}
		}

		// Remove entrada vazia
		if len(mp.transactionsByAddress[tx.From]) == 0 {
			delete(mp.transactionsByAddress, tx.From)
		}

		count++
	}

	return count
}

// Clear limpa todas as transações do mempool
func (mp *Mempool) Clear() {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	mp.transactions = make(map[string]*Transaction)
	mp.transactionsByAddress = make(map[string][]*Transaction)
}

// Size retorna o número de transações no mempool
func (mp *Mempool) Size() int {
	mp.mu.RLock()
	defer mp.mu.RUnlock()
	return len(mp.transactions)
}

// PruneExpired remove transações expiradas
func (mp *Mempool) PruneExpired() int {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	now := time.Now().Unix()
	expired := make([]string, 0)

	for txID, tx := range mp.transactions {
		age := time.Duration(now-tx.Timestamp) * time.Second
		if age > mp.maxTxAge {
			expired = append(expired, txID)
		}
	}

	// Remove transações expiradas
	for _, txID := range expired {
		tx := mp.transactions[txID]
		delete(mp.transactions, txID)

		// Remove do índice por endereço
		addressTxs := mp.transactionsByAddress[tx.From]
		for i, addrTx := range addressTxs {
			if addrTx.ID == txID {
				mp.transactionsByAddress[tx.From] = append(addressTxs[:i], addressTxs[i+1:]...)
				break
			}
		}

		if len(mp.transactionsByAddress[tx.From]) == 0 {
			delete(mp.transactionsByAddress, tx.From)
		}
	}

	return len(expired)
}

// removeLowFeeTx remove a transação com menor taxa (não thread-safe)
// Retorna true se conseguiu remover uma transação com taxa menor que minFee
func (mp *Mempool) removeLowFeeTx(minFee uint64) bool {
	var lowestFeeTx *Transaction
	lowestFee := uint64(^uint64(0)) // MaxUint64

	for _, tx := range mp.transactions {
		if tx.Fee < lowestFee {
			lowestFee = tx.Fee
			lowestFeeTx = tx
		}
	}

	if lowestFeeTx != nil && lowestFeeTx.Fee < minFee {
		delete(mp.transactions, lowestFeeTx.ID)

		// Remove do índice
		addressTxs := mp.transactionsByAddress[lowestFeeTx.From]
		for i, tx := range addressTxs {
			if tx.ID == lowestFeeTx.ID {
				mp.transactionsByAddress[lowestFeeTx.From] = append(addressTxs[:i], addressTxs[i+1:]...)
				break
			}
		}

		return true
	}

	return false
}

// GetStats retorna estatísticas do mempool
func (mp *Mempool) GetStats() MempoolStats {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	stats := MempoolStats{
		TotalTransactions: len(mp.transactions),
		UniqueAddresses:   len(mp.transactionsByAddress),
	}

	// Calcula taxa média e total de fees
	var totalFees uint64
	var minFee, maxFee uint64 = ^uint64(0), 0

	for _, tx := range mp.transactions {
		totalFees += tx.Fee
		if tx.Fee < minFee {
			minFee = tx.Fee
		}
		if tx.Fee > maxFee {
			maxFee = tx.Fee
		}
	}

	stats.TotalFees = totalFees
	stats.MinFee = minFee
	stats.MaxFee = maxFee

	if len(mp.transactions) > 0 {
		stats.AverageFee = totalFees / uint64(len(mp.transactions))
	}

	return stats
}

// MempoolStats estatísticas do mempool
type MempoolStats struct {
	TotalTransactions int    // Total de transações
	UniqueAddresses   int    // Número de endereços únicos
	TotalFees         uint64 // Soma de todas as taxas
	AverageFee        uint64 // Taxa média
	MinFee            uint64 // Taxa mínima
	MaxFee            uint64 // Taxa máxima
}
