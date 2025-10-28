package blockchain

import (
	"testing"
	"time"

	"github.com/krakovia/blockchain/pkg/wallet"
)

func TestNewTransaction(t *testing.T) {
	tx := NewTransaction("addr1", "addr2", 100, 1, 0, "test")

	if tx.From != "addr1" {
		t.Errorf("Expected from 'addr1', got '%s'", tx.From)
	}
	if tx.To != "addr2" {
		t.Errorf("Expected to 'addr2', got '%s'", tx.To)
	}
	if tx.Amount != 100 {
		t.Errorf("Expected amount 100, got %d", tx.Amount)
	}
	if tx.Fee != 1 {
		t.Errorf("Expected fee 1, got %d", tx.Fee)
	}
	if tx.Nonce != 0 {
		t.Errorf("Expected nonce 0, got %d", tx.Nonce)
	}
	if tx.Data != "test" {
		t.Errorf("Expected data 'test', got '%s'", tx.Data)
	}
}

func TestTransactionSignAndVerify(t *testing.T) {
	w, err := wallet.NewWallet()
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	tx := NewTransaction(w.GetAddress(), "recipient_addr", 100, 1, 0, "payment")

	// Assina a transação
	err = tx.Sign(w)
	if err != nil {
		t.Fatalf("Failed to sign transaction: %v", err)
	}

	// Verifica que os campos foram preenchidos
	if tx.ID == "" {
		t.Error("Transaction ID is empty after signing")
	}
	if tx.Signature == "" {
		t.Error("Transaction signature is empty after signing")
	}
	if tx.PublicKey == "" {
		t.Error("Transaction public key is empty after signing")
	}

	// Verifica a assinatura
	err = tx.Verify()
	if err != nil {
		t.Errorf("Transaction verification failed: %v", err)
	}
}

func TestTransactionSignWithWrongWallet(t *testing.T) {
	w1, _ := wallet.NewWallet()
	w2, _ := wallet.NewWallet()

	tx := NewTransaction(w1.GetAddress(), "recipient_addr", 100, 1, 0, "payment")

	// Tenta assinar com carteira diferente
	err := tx.Sign(w2)
	if err == nil {
		t.Error("Expected error when signing with wrong wallet")
	}
}

func TestTransactionVerifyTamperedData(t *testing.T) {
	w, _ := wallet.NewWallet()
	tx := NewTransaction(w.GetAddress(), "recipient_addr", 100, 1, 0, "payment")

	err := tx.Sign(w)
	if err != nil {
		t.Fatalf("Failed to sign transaction: %v", err)
	}

	// Altera o valor da transação
	tx.Amount = 200

	// A verificação deve falhar
	err = tx.Verify()
	if err == nil {
		t.Error("Expected verification to fail for tampered transaction")
	}
}

func TestTransactionCalculateHash(t *testing.T) {
	w, _ := wallet.NewWallet()
	tx := NewTransaction(w.GetAddress(), "recipient_addr", 100, 1, 0, "payment")

	err := tx.Sign(w)
	if err != nil {
		t.Fatalf("Failed to sign transaction: %v", err)
	}

	hash1, err := tx.CalculateHash()
	if err != nil {
		t.Fatalf("Failed to calculate hash: %v", err)
	}

	hash2, err := tx.CalculateHash()
	if err != nil {
		t.Fatalf("Failed to calculate hash: %v", err)
	}

	// Hash deve ser determinístico
	if hash1 != hash2 {
		t.Error("Hash should be deterministic")
	}

	// Hash deve ser igual ao ID
	if tx.ID != hash1 {
		t.Error("Transaction ID should match calculated hash")
	}
}

func TestTransactionValidate(t *testing.T) {
	w, _ := wallet.NewWallet()
	tx := NewTransaction(w.GetAddress(), "recipient_addr", 100, 1, 0, "payment")

	err := tx.Sign(w)
	if err != nil {
		t.Fatalf("Failed to sign transaction: %v", err)
	}

	// Validação deve passar
	err = tx.Validate()
	if err != nil {
		t.Errorf("Transaction validation failed: %v", err)
	}
}

func TestTransactionValidateZeroAmount(t *testing.T) {
	w, _ := wallet.NewWallet()
	tx := NewTransaction(w.GetAddress(), "recipient_addr", 0, 1, 0, "payment")

	err := tx.Sign(w)
	if err != nil {
		t.Fatalf("Failed to sign transaction: %v", err)
	}

	// Validação deve falhar (amount = 0)
	err = tx.Validate()
	if err == nil {
		t.Error("Expected validation to fail for zero amount")
	}
}

func TestTransactionValidateSameAddresses(t *testing.T) {
	w, _ := wallet.NewWallet()
	addr := w.GetAddress()
	tx := NewTransaction(addr, addr, 100, 1, 0, "payment")

	err := tx.Sign(w)
	if err != nil {
		t.Fatalf("Failed to sign transaction: %v", err)
	}

	// Validação deve falhar (from == to)
	err = tx.Validate()
	if err == nil {
		t.Error("Expected validation to fail when sender equals receiver")
	}
}

func TestTransactionValidateFutureTimestamp(t *testing.T) {
	w, _ := wallet.NewWallet()
	tx := NewTransaction(w.GetAddress(), "recipient_addr", 100, 1, 0, "payment")

	// Define timestamp muito no futuro
	tx.Timestamp = time.Now().Unix() + 1000

	err := tx.Sign(w)
	if err != nil {
		t.Fatalf("Failed to sign transaction: %v", err)
	}

	// Validação deve falhar (timestamp no futuro)
	err = tx.Validate()
	if err == nil {
		t.Error("Expected validation to fail for future timestamp")
	}
}

func TestNewCoinbaseTransaction(t *testing.T) {
	tx := NewCoinbaseTransaction("miner_addr", 50, 1)

	if !tx.IsCoinbase() {
		t.Error("Transaction should be coinbase")
	}

	if tx.From != "" {
		t.Error("Coinbase transaction should have empty from address")
	}

	if tx.To != "miner_addr" {
		t.Errorf("Expected to 'miner_addr', got '%s'", tx.To)
	}

	if tx.Amount != 50 {
		t.Errorf("Expected amount 50, got %d", tx.Amount)
	}

	if tx.Fee != 0 {
		t.Error("Coinbase transaction should have zero fee")
	}

	if tx.ID == "" {
		t.Error("Coinbase transaction should have ID")
	}
}

func TestVerifyCoinbase(t *testing.T) {
	tx := NewCoinbaseTransaction("miner_addr", 50, 1)

	err := tx.VerifyCoinbase()
	if err != nil {
		t.Errorf("Coinbase verification failed: %v", err)
	}
}

func TestVerifyCoinbaseInvalidAmount(t *testing.T) {
	tx := NewCoinbaseTransaction("miner_addr", 0, 1)

	err := tx.VerifyCoinbase()
	if err == nil {
		t.Error("Expected coinbase verification to fail for zero amount")
	}
}

func TestTransactionSerializeDeserialize(t *testing.T) {
	w, _ := wallet.NewWallet()
	tx := NewTransaction(w.GetAddress(), "recipient_addr", 100, 1, 0, "payment")

	err := tx.Sign(w)
	if err != nil {
		t.Fatalf("Failed to sign transaction: %v", err)
	}

	// Serializa
	data, err := tx.Serialize()
	if err != nil {
		t.Fatalf("Failed to serialize transaction: %v", err)
	}

	// Desserializa
	tx2, err := DeserializeTransaction(data)
	if err != nil {
		t.Fatalf("Failed to deserialize transaction: %v", err)
	}

	// Compara campos
	if tx.ID != tx2.ID {
		t.Error("Transaction IDs do not match")
	}
	if tx.From != tx2.From {
		t.Error("From addresses do not match")
	}
	if tx.To != tx2.To {
		t.Error("To addresses do not match")
	}
	if tx.Amount != tx2.Amount {
		t.Error("Amounts do not match")
	}
	if tx.Signature != tx2.Signature {
		t.Error("Signatures do not match")
	}
}

func TestTransactionEqual(t *testing.T) {
	w, _ := wallet.NewWallet()
	tx1 := NewTransaction(w.GetAddress(), "recipient_addr", 100, 1, 0, "payment")
	_ = tx1.Sign(w)

	tx2 := tx1.Copy()

	if !tx1.Equal(tx2) {
		t.Error("Transactions should be equal")
	}

	tx3 := NewTransaction(w.GetAddress(), "other_addr", 100, 1, 0, "payment")
	_ = tx3.Sign(w)

	if tx1.Equal(tx3) {
		t.Error("Transactions should not be equal")
	}
}

func TestTransactionCopy(t *testing.T) {
	w, _ := wallet.NewWallet()
	tx := NewTransaction(w.GetAddress(), "recipient_addr", 100, 1, 0, "payment")
	_ = tx.Sign(w)

	txCopy := tx.Copy()

	// Modifica a cópia
	txCopy.Amount = 200

	// Original não deve ser afetado
	if tx.Amount == 200 {
		t.Error("Original transaction was modified")
	}
}

func TestTransactionSliceMerkleRoot(t *testing.T) {
	w, _ := wallet.NewWallet()

	tx1 := NewTransaction(w.GetAddress(), "addr1", 100, 1, 0, "tx1")
	_ = tx1.Sign(w)

	tx2 := NewTransaction(w.GetAddress(), "addr2", 200, 1, 1, "tx2")
	_ = tx2.Sign(w)

	txs := TransactionSlice{tx1, tx2}

	root := txs.CalculateMerkleRoot()
	if root == "" {
		t.Error("Merkle root is empty")
	}

	// Mesmas transações devem gerar mesma raiz
	root2 := txs.CalculateMerkleRoot()
	if root != root2 {
		t.Error("Merkle root should be deterministic")
	}
}

func TestTransactionSliceTotalAmount(t *testing.T) {
	w, _ := wallet.NewWallet()

	tx1 := NewTransaction(w.GetAddress(), "addr1", 100, 1, 0, "tx1")
	tx2 := NewTransaction(w.GetAddress(), "addr2", 200, 2, 1, "tx2")

	txs := TransactionSlice{tx1, tx2}

	total := txs.TotalAmount()
	if total != 300 {
		t.Errorf("Expected total amount 300, got %d", total)
	}
}

func TestTransactionSliceTotalFees(t *testing.T) {
	w, _ := wallet.NewWallet()

	tx1 := NewTransaction(w.GetAddress(), "addr1", 100, 1, 0, "tx1")
	tx2 := NewTransaction(w.GetAddress(), "addr2", 200, 2, 1, "tx2")

	txs := TransactionSlice{tx1, tx2}

	total := txs.TotalFees()
	if total != 3 {
		t.Errorf("Expected total fees 3, got %d", total)
	}
}

func TestTransactionSliceContainsID(t *testing.T) {
	w, _ := wallet.NewWallet()

	tx1 := NewTransaction(w.GetAddress(), "addr1", 100, 1, 0, "tx1")
	_ = tx1.Sign(w)

	tx2 := NewTransaction(w.GetAddress(), "addr2", 200, 2, 1, "tx2")
	_ = tx2.Sign(w)

	txs := TransactionSlice{tx1, tx2}

	if !txs.ContainsID(tx1.ID) {
		t.Error("Should contain tx1 ID")
	}

	if !txs.ContainsID(tx2.ID) {
		t.Error("Should contain tx2 ID")
	}

	if txs.ContainsID("non_existent_id") {
		t.Error("Should not contain non-existent ID")
	}
}

func TestTransactionSliceHasDuplicates(t *testing.T) {
	w, _ := wallet.NewWallet()

	tx1 := NewTransaction(w.GetAddress(), "addr1", 100, 1, 0, "tx1")
	_ = tx1.Sign(w)

	tx2 := NewTransaction(w.GetAddress(), "addr2", 200, 2, 1, "tx2")
	_ = tx2.Sign(w)

	// Sem duplicatas
	txs := TransactionSlice{tx1, tx2}
	if txs.HasDuplicates() {
		t.Error("Should not have duplicates")
	}

	// Com duplicatas
	txsWithDups := TransactionSlice{tx1, tx2, tx1}
	if !txsWithDups.HasDuplicates() {
		t.Error("Should have duplicates")
	}
}

func TestTransactionSliceValidate(t *testing.T) {
	w, _ := wallet.NewWallet()

	tx1 := NewTransaction(w.GetAddress(), "addr1", 100, 1, 0, "tx1")
	_ = tx1.Sign(w)

	tx2 := NewTransaction(w.GetAddress(), "addr2", 200, 2, 1, "tx2")
	_ = tx2.Sign(w)

	txs := TransactionSlice{tx1, tx2}

	err := txs.Validate()
	if err != nil {
		t.Errorf("Transaction slice validation failed: %v", err)
	}

	// Adiciona transação inválida
	invalidTx := NewTransaction(w.GetAddress(), "addr3", 0, 1, 2, "invalid")
	_ = invalidTx.Sign(w)

	invalidTxs := TransactionSlice{tx1, invalidTx}
	err = invalidTxs.Validate()
	if err == nil {
		t.Error("Expected validation to fail for invalid transaction")
	}
}

func BenchmarkTransactionSign(b *testing.B) {
	w, _ := wallet.NewWallet()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tx := NewTransaction(w.GetAddress(), "recipient", 100, 1, uint64(i), "payment")
		_ = tx.Sign(w)
	}
}

func BenchmarkTransactionVerify(b *testing.B) {
	w, _ := wallet.NewWallet()
	tx := NewTransaction(w.GetAddress(), "recipient", 100, 1, 0, "payment")
	_ = tx.Sign(w)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tx.Verify()
	}
}

func BenchmarkTransactionCalculateHash(b *testing.B) {
	w, _ := wallet.NewWallet()
	tx := NewTransaction(w.GetAddress(), "recipient", 100, 1, 0, "payment")
	_ = tx.Sign(w)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = tx.CalculateHash()
	}
}

func BenchmarkMerkleRoot(b *testing.B) {
	w, _ := wallet.NewWallet()

	var txs TransactionSlice
	for i := 0; i < 100; i++ {
		tx := NewTransaction(w.GetAddress(), "recipient", 100, 1, uint64(i), "payment")
		_ = tx.Sign(w)
		txs = append(txs, tx)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		txs.CalculateMerkleRoot()
	}
}
