package blockchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/krakovia/blockchain/pkg/wallet"
)

// Transaction representa uma transação na blockchain
type Transaction struct {
	ID        string    `json:"id"`         // Hash da transação
	From      string    `json:"from"`       // Endereço do remetente (hash da chave pública)
	To        string    `json:"to"`         // Endereço do destinatário
	Amount    uint64    `json:"amount"`     // Quantidade transferida
	Fee       uint64    `json:"fee"`        // Taxa da transação
	Timestamp int64     `json:"timestamp"`  // Timestamp Unix
	Signature string    `json:"signature"`  // Assinatura ECDSA
	PublicKey string    `json:"public_key"` // Chave pública do remetente
	Nonce     uint64    `json:"nonce"`      // Nonce para prevenir replay attacks
	Data      string    `json:"data"`       // Dados adicionais (opcional)
}

// NewTransaction cria uma nova transação
func NewTransaction(from, to string, amount, fee, nonce uint64, data string) *Transaction {
	tx := &Transaction{
		From:      from,
		To:        to,
		Amount:    amount,
		Fee:       fee,
		Timestamp: time.Now().Unix(),
		Nonce:     nonce,
		Data:      data,
	}
	return tx
}

// CalculateHash calcula o hash da transação (sem incluir a assinatura)
func (tx *Transaction) CalculateHash() (string, error) {
	// Cria uma cópia da transação sem assinatura e ID para calcular o hash
	txCopy := Transaction{
		From:      tx.From,
		To:        tx.To,
		Amount:    tx.Amount,
		Fee:       tx.Fee,
		Timestamp: tx.Timestamp,
		PublicKey: tx.PublicKey,
		Nonce:     tx.Nonce,
		Data:      tx.Data,
	}

	data, err := json.Marshal(txCopy)
	if err != nil {
		return "", fmt.Errorf("failed to marshal transaction: %w", err)
	}

	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

// GetSignData retorna os dados que devem ser assinados
func (tx *Transaction) GetSignData() ([]byte, error) {
	txCopy := Transaction{
		From:      tx.From,
		To:        tx.To,
		Amount:    tx.Amount,
		Fee:       tx.Fee,
		Timestamp: tx.Timestamp,
		Nonce:     tx.Nonce,
		Data:      tx.Data,
	}

	data, err := json.Marshal(txCopy)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal transaction for signing: %w", err)
	}

	return data, nil
}

// Sign assina a transação usando uma carteira
func (tx *Transaction) Sign(w *wallet.Wallet) error {
	// Valida que o endereço From corresponde à carteira
	if tx.From != w.GetAddress() {
		return fmt.Errorf("wallet address does not match transaction from address")
	}

	// Define a chave pública
	tx.PublicKey = w.GetPublicKeyHex()

	// Obtém os dados para assinar
	signData, err := tx.GetSignData()
	if err != nil {
		return err
	}

	// Assina os dados
	signature, err := w.Sign(signData)
	if err != nil {
		return fmt.Errorf("failed to sign transaction: %w", err)
	}

	tx.Signature = signature

	// Calcula e define o ID da transação
	hash, err := tx.CalculateHash()
	if err != nil {
		return err
	}
	tx.ID = hash

	return nil
}

// Verify verifica a assinatura da transação
func (tx *Transaction) Verify() error {
	// Verifica se todos os campos obrigatórios estão preenchidos
	if tx.ID == "" {
		return fmt.Errorf("transaction ID is empty")
	}
	if tx.From == "" {
		return fmt.Errorf("transaction from address is empty")
	}
	if tx.To == "" {
		return fmt.Errorf("transaction to address is empty")
	}
	if tx.Signature == "" {
		return fmt.Errorf("transaction signature is empty")
	}
	if tx.PublicKey == "" {
		return fmt.Errorf("transaction public key is empty")
	}

	// Verifica se o endereço From corresponde à chave pública
	expectedAddress, err := wallet.AddressFromPublicKey(tx.PublicKey)
	if err != nil {
		return fmt.Errorf("failed to derive address from public key: %w", err)
	}
	if tx.From != expectedAddress {
		return fmt.Errorf("from address does not match public key")
	}

	// Verifica o hash da transação
	calculatedHash, err := tx.CalculateHash()
	if err != nil {
		return err
	}
	if tx.ID != calculatedHash {
		return fmt.Errorf("transaction hash mismatch: expected %s, got %s", calculatedHash, tx.ID)
	}

	// Obtém os dados que foram assinados
	signData, err := tx.GetSignData()
	if err != nil {
		return err
	}

	// Verifica a assinatura
	valid, err := wallet.Verify(tx.PublicKey, signData, tx.Signature)
	if err != nil {
		return fmt.Errorf("failed to verify signature: %w", err)
	}
	if !valid {
		return fmt.Errorf("invalid transaction signature")
	}

	return nil
}

// Validate valida os campos da transação (regras de negócio)
func (tx *Transaction) Validate() error {
	// Verifica a assinatura primeiro
	if err := tx.Verify(); err != nil {
		return err
	}

	// Valida valores
	if tx.Amount == 0 {
		return fmt.Errorf("transaction amount must be greater than 0")
	}

	// Valida timestamp (não pode ser muito no futuro)
	now := time.Now().Unix()
	if tx.Timestamp > now+300 { // 5 minutos de tolerância
		return fmt.Errorf("transaction timestamp is too far in the future")
	}

	// Parse transaction data para verificar se é stake operation
	txData, _ := DeserializeTransactionData(tx.Data)

	// Valida que remetente e destinatário são diferentes (exceto para operações de stake)
	if tx.From == tx.To {
		// Permite From == To apenas para stake/unstake
		if txData == nil || !txData.IsStakeOperation() {
			return fmt.Errorf("sender and receiver cannot be the same")
		}
	}

	return nil
}

// Serialize serializa a transação para JSON
func (tx *Transaction) Serialize() ([]byte, error) {
	return json.Marshal(tx)
}

// DeserializeTransaction desserializa uma transação de JSON
func DeserializeTransaction(data []byte) (*Transaction, error) {
	var tx Transaction
	err := json.Unmarshal(data, &tx)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize transaction: %w", err)
	}
	return &tx, nil
}

// IsCoinbase verifica se a transação é uma transação coinbase (recompensa de mineração)
func (tx *Transaction) IsCoinbase() bool {
	return tx.From == ""
}

// NewCoinbaseTransaction cria uma transação coinbase (recompensa de bloco)
func NewCoinbaseTransaction(to string, amount uint64, blockHeight uint64) *Transaction {
	return NewCoinbaseTransactionWithTimestamp(to, amount, blockHeight, time.Now().Unix())
}

// NewCoinbaseTransactionWithTimestamp cria uma transação coinbase com timestamp específico
func NewCoinbaseTransactionWithTimestamp(to string, amount uint64, blockHeight uint64, timestamp int64) *Transaction {
	tx := &Transaction{
		From:      "", // Transação coinbase não tem remetente
		To:        to,
		Amount:    amount,
		Fee:       0,
		Timestamp: timestamp,
		Nonce:     blockHeight, // Usa a altura do bloco como nonce
		Data:      fmt.Sprintf("Coinbase reward for block %d", blockHeight),
	}

	// Calcula o hash
	hash, _ := tx.CalculateHash()
	tx.ID = hash

	return tx
}

// VerifyCoinbase verifica se uma transação coinbase é válida
func (tx *Transaction) VerifyCoinbase() error {
	if !tx.IsCoinbase() {
		return fmt.Errorf("transaction is not a coinbase transaction")
	}

	if tx.ID == "" {
		return fmt.Errorf("coinbase transaction ID is empty")
	}

	if tx.To == "" {
		return fmt.Errorf("coinbase transaction to address is empty")
	}

	if tx.Amount == 0 {
		return fmt.Errorf("coinbase transaction amount must be greater than 0")
	}

	// Verifica o hash
	calculatedHash, err := tx.CalculateHash()
	if err != nil {
		return err
	}
	if tx.ID != calculatedHash {
		return fmt.Errorf("coinbase transaction hash mismatch")
	}

	return nil
}

// Equal compara se duas transações são iguais
func (tx *Transaction) Equal(other *Transaction) bool {
	if tx == nil || other == nil {
		return tx == other
	}
	return tx.ID == other.ID
}

// Copy cria uma cópia profunda da transação
func (tx *Transaction) Copy() *Transaction {
	return &Transaction{
		ID:        tx.ID,
		From:      tx.From,
		To:        tx.To,
		Amount:    tx.Amount,
		Fee:       tx.Fee,
		Timestamp: tx.Timestamp,
		Signature: tx.Signature,
		PublicKey: tx.PublicKey,
		Nonce:     tx.Nonce,
		Data:      tx.Data,
	}
}

// Hash retorna o hash da transação serializada (usado para Merkle Tree)
func (tx *Transaction) Hash() []byte {
	data, err := tx.Serialize()
	if err != nil {
		return nil
	}
	hash := sha256.Sum256(data)
	return hash[:]
}

// TransactionSlice é um slice de transações com métodos auxiliares
type TransactionSlice []*Transaction

// CalculateMerkleRoot calcula a raiz da árvore de Merkle das transações
func (txs TransactionSlice) CalculateMerkleRoot() string {
	if len(txs) == 0 {
		return ""
	}

	// Coleta os hashes de todas as transações
	var hashes [][]byte
	for _, tx := range txs {
		hashes = append(hashes, tx.Hash())
	}

	// Constrói a árvore de Merkle
	for len(hashes) > 1 {
		// Se o número de hashes for ímpar, duplica o último
		if len(hashes)%2 != 0 {
			hashes = append(hashes, hashes[len(hashes)-1])
		}

		var newLevel [][]byte
		for i := 0; i < len(hashes); i += 2 {
			combined := append(hashes[i], hashes[i+1]...)
			hash := sha256.Sum256(combined)
			newLevel = append(newLevel, hash[:])
		}
		hashes = newLevel
	}

	return hex.EncodeToString(hashes[0])
}

// TotalAmount retorna a soma total de valores das transações
func (txs TransactionSlice) TotalAmount() uint64 {
	var total uint64
	for _, tx := range txs {
		total += tx.Amount
	}
	return total
}

// TotalFees retorna a soma total de taxas das transações
func (txs TransactionSlice) TotalFees() uint64 {
	var total uint64
	for _, tx := range txs {
		total += tx.Fee
	}
	return total
}

// ContainsID verifica se uma transação com o ID especificado existe no slice
func (txs TransactionSlice) ContainsID(id string) bool {
	for _, tx := range txs {
		if tx.ID == id {
			return true
		}
	}
	return false
}

// Filter filtra transações usando uma função predicado
func (txs TransactionSlice) Filter(predicate func(*Transaction) bool) TransactionSlice {
	var result TransactionSlice
	for _, tx := range txs {
		if predicate(tx) {
			result = append(result, tx)
		}
	}
	return result
}

// Validate valida todas as transações do slice
func (txs TransactionSlice) Validate() error {
	for i, tx := range txs {
		if err := tx.Validate(); err != nil {
			return fmt.Errorf("transaction %d is invalid: %w", i, err)
		}
	}
	return nil
}

// HasDuplicates verifica se há transações duplicadas no slice
func (txs TransactionSlice) HasDuplicates() bool {
	seen := make(map[string]bool)
	for _, tx := range txs {
		if seen[tx.ID] {
			return true
		}
		seen[tx.ID] = true
	}
	return false
}

// Sort ordena as transações por timestamp e depois por ID
func (txs TransactionSlice) Sort() {
	// Implementação simples de bubble sort
	for i := 0; i < len(txs); i++ {
		for j := i + 1; j < len(txs); j++ {
			if txs[i].Timestamp > txs[j].Timestamp {
				txs[i], txs[j] = txs[j], txs[i]
			} else if txs[i].Timestamp == txs[j].Timestamp {
				if bytes.Compare([]byte(txs[i].ID), []byte(txs[j].ID)) > 0 {
					txs[i], txs[j] = txs[j], txs[i]
				}
			}
		}
	}
}
