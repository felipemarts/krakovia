package blockchain

import (
	"testing"

	"github.com/krakovia/blockchain/pkg/wallet"
)

func TestNewContext(t *testing.T) {
	ctx := NewContext()

	if ctx == nil {
		t.Fatal("Context is nil")
	}

	if ctx.GetBlockCount() != 0 {
		t.Errorf("Expected 0 blocks, got %d", ctx.GetBlockCount())
	}

	if ctx.GetLastBlockHash() != "" {
		t.Error("Expected empty last block hash")
	}
}

func TestContextWithGenesis(t *testing.T) {
	// Cria bloco gênesis
	coinbase := NewCoinbaseTransaction("genesis_addr", 1000000, 0)
	genesis := GenesisBlock(coinbase)

	// Cria contexto com gênesis
	ctx, err := NewContextWithGenesis(genesis)
	if err != nil {
		t.Fatalf("Failed to create context with genesis: %v", err)
	}

	// Verifica que o bloco foi adicionado
	if ctx.GetBlockCount() != 1 {
		t.Errorf("Expected 1 block, got %d", ctx.GetBlockCount())
	}

	if ctx.GetLastBlockHash() != genesis.Hash {
		t.Error("Last block hash doesn't match genesis hash")
	}

	// Verifica saldo do endereço genesis
	balance := ctx.GetBalance("genesis_addr")
	if balance != 1000000 {
		t.Errorf("Expected balance 1000000, got %d", balance)
	}
}

func TestContextTransferExecution(t *testing.T) {
	w1, _ := wallet.NewWallet()
	w2, _ := wallet.NewWallet()

	// Cria bloco gênesis com saldo inicial
	coinbase := NewCoinbaseTransaction(w1.GetAddress(), 1000, 0)
	genesis := GenesisBlock(coinbase)

	ctx, _ := NewContextWithGenesis(genesis)

	// Verifica saldo inicial
	balance1 := ctx.GetBalance(w1.GetAddress())
	if balance1 != 1000 {
		t.Errorf("Expected initial balance 1000, got %d", balance1)
	}

	// Cria transação de transferência
	tx := NewTransaction(w1.GetAddress(), w2.GetAddress(), 100, 1, 0, "")
	_ = tx.Sign(w1)

	// Simula execução
	modifications, err := ctx.ExecuteTransaction(tx)
	if err != nil {
		t.Fatalf("Failed to execute transaction: %v", err)
	}

	// Verifica modificações
	if len(modifications) == 0 {
		t.Error("Expected modifications, got none")
	}

	t.Logf("Modifications: %+v", modifications)

	// Cria bloco com a transação
	block1 := NewBlock(1, genesis.Hash, TransactionSlice{tx}, w1.GetAddress())
	hash1, _ := block1.CalculateHash()
	block1.Hash = hash1

	// Adiciona bloco ao contexto
	err = ctx.AddBlock(block1)
	if err != nil {
		t.Fatalf("Failed to add block: %v", err)
	}

	// Verifica saldos finais
	finalBalance1 := ctx.GetBalance(w1.GetAddress())
	finalBalance2 := ctx.GetBalance(w2.GetAddress())

	expectedBalance1 := uint64(1000 - 100 - 1) // inicial - amount - fee
	if finalBalance1 != expectedBalance1 {
		t.Errorf("Expected w1 balance %d, got %d", expectedBalance1, finalBalance1)
	}

	if finalBalance2 != 100 {
		t.Errorf("Expected w2 balance 100, got %d", finalBalance2)
	}
}

func TestContextStakeExecution(t *testing.T) {
	w, _ := wallet.NewWallet()

	// Cria bloco gênesis com saldo inicial
	coinbase := NewCoinbaseTransaction(w.GetAddress(), 1000, 0)
	genesis := GenesisBlock(coinbase)

	ctx, _ := NewContextWithGenesis(genesis)

	// Cria transação de stake
	stakeData := NewStakeData(500)
	dataStr, _ := stakeData.Serialize()

	tx := NewTransaction(w.GetAddress(), w.GetAddress(), 500, 1, 0, dataStr)
	_ = tx.Sign(w)

	// Cria bloco com a transação
	block1 := NewBlock(1, genesis.Hash, TransactionSlice{tx}, w.GetAddress())
	hash1, _ := block1.CalculateHash()
	block1.Hash = hash1

	// Adiciona bloco
	err := ctx.AddBlock(block1)
	if err != nil {
		t.Fatalf("Failed to add block with stake: %v", err)
	}

	// Verifica saldo e stake
	balance := ctx.GetBalance(w.GetAddress())
	stake := ctx.GetStake(w.GetAddress())

	expectedBalance := uint64(1000 - 500 - 1) // inicial - stake - fee
	if balance != expectedBalance {
		t.Errorf("Expected balance %d, got %d", expectedBalance, balance)
	}

	if stake != 500 {
		t.Errorf("Expected stake 500, got %d", stake)
	}
}

func TestContextUnstakeExecution(t *testing.T) {
	w, _ := wallet.NewWallet()

	// Cria bloco gênesis
	coinbase := NewCoinbaseTransaction(w.GetAddress(), 1000, 0)
	genesis := GenesisBlock(coinbase)

	ctx, _ := NewContextWithGenesis(genesis)

	// Primeiro faz stake
	stakeData := NewStakeData(500)
	dataStr1, _ := stakeData.Serialize()
	tx1 := NewTransaction(w.GetAddress(), w.GetAddress(), 500, 1, 0, dataStr1)
	_ = tx1.Sign(w)

	block1 := NewBlock(1, genesis.Hash, TransactionSlice{tx1}, w.GetAddress())
	hash1, _ := block1.CalculateHash()
	block1.Hash = hash1
	_ = ctx.AddBlock(block1)

	// Depois faz unstake
	unstakeData := NewUnstakeData(200)
	dataStr2, _ := unstakeData.Serialize()
	tx2 := NewTransaction(w.GetAddress(), w.GetAddress(), 200, 1, 1, dataStr2)
	_ = tx2.Sign(w)

	block2 := NewBlock(2, block1.Hash, TransactionSlice{tx2}, w.GetAddress())
	hash2, _ := block2.CalculateHash()
	block2.Hash = hash2
	_ = ctx.AddBlock(block2)

	// Verifica saldo e stake finais
	balance := ctx.GetBalance(w.GetAddress())
	stake := ctx.GetStake(w.GetAddress())

	// inicial: 1000
	// após stake: 1000 - 500 - 1 = 499
	// após unstake: 499 + 200 - 1 = 698
	expectedBalance := uint64(698)
	if balance != expectedBalance {
		t.Errorf("Expected balance %d, got %d", expectedBalance, balance)
	}

	// stake inicial: 500
	// após unstake: 500 - 200 = 300
	expectedStake := uint64(300)
	if stake != expectedStake {
		t.Errorf("Expected stake %d, got %d", expectedStake, stake)
	}
}

func TestContextInsufficientBalance(t *testing.T) {
	w1, _ := wallet.NewWallet()
	w2, _ := wallet.NewWallet()

	// Cria bloco gênesis com saldo pequeno
	coinbase := NewCoinbaseTransaction(w1.GetAddress(), 10, 0)
	genesis := GenesisBlock(coinbase)

	ctx, _ := NewContextWithGenesis(genesis)

	// Tenta transferir mais do que tem
	tx := NewTransaction(w1.GetAddress(), w2.GetAddress(), 100, 1, 0, "")
	_ = tx.Sign(w1)

	// Deve falhar ao executar
	_, err := ctx.ExecuteTransaction(tx)
	if err == nil {
		t.Error("Expected error for insufficient balance")
	}
}

func TestContextInvalidNonce(t *testing.T) {
	w1, _ := wallet.NewWallet()
	w2, _ := wallet.NewWallet()

	// Cria bloco gênesis
	coinbase := NewCoinbaseTransaction(w1.GetAddress(), 1000, 0)
	genesis := GenesisBlock(coinbase)

	ctx, _ := NewContextWithGenesis(genesis)

	// Cria transação com nonce errado
	tx := NewTransaction(w1.GetAddress(), w2.GetAddress(), 100, 1, 5, "") // nonce deveria ser 0
	_ = tx.Sign(w1)

	// Deve falhar
	_, err := ctx.ExecuteTransaction(tx)
	if err == nil {
		t.Error("Expected error for invalid nonce")
	}
}

func TestContextGetValidators(t *testing.T) {
	w1, _ := wallet.NewWallet()
	w2, _ := wallet.NewWallet()

	// Cria bloco gênesis
	coinbase := NewCoinbaseTransaction(w1.GetAddress(), 2000, 0)
	genesis := GenesisBlock(coinbase)

	ctx, _ := NewContextWithGenesis(genesis)

	// w1 faz stake
	stakeData1 := NewStakeData(1000)
	data1, _ := stakeData1.Serialize()
	tx1 := NewTransaction(w1.GetAddress(), w1.GetAddress(), 1000, 1, 0, data1)
	_ = tx1.Sign(w1)

	block1 := NewBlock(1, genesis.Hash, TransactionSlice{tx1}, w1.GetAddress())
	hash1, _ := block1.CalculateHash()
	block1.Hash = hash1
	_ = ctx.AddBlock(block1)

	// Transfere para w2
	tx2 := NewTransaction(w1.GetAddress(), w2.GetAddress(), 500, 1, 1, "")
	_ = tx2.Sign(w1)

	block2 := NewBlock(2, block1.Hash, TransactionSlice{tx2}, w1.GetAddress())
	hash2, _ := block2.CalculateHash()
	block2.Hash = hash2
	_ = ctx.AddBlock(block2)

	// w2 faz stake
	stakeData2 := NewStakeData(300)
	data2, _ := stakeData2.Serialize()
	tx3 := NewTransaction(w2.GetAddress(), w2.GetAddress(), 300, 1, 0, data2)
	_ = tx3.Sign(w2)

	block3 := NewBlock(3, block2.Hash, TransactionSlice{tx3}, w1.GetAddress())
	hash3, _ := block3.CalculateHash()
	block3.Hash = hash3
	_ = ctx.AddBlock(block3)

	// Pega validadores
	validators := ctx.GetValidators()

	if len(validators) != 2 {
		t.Errorf("Expected 2 validators, got %d", len(validators))
	}

	// Verifica stakes
	stakes := ctx.GetAllStakes()
	if stakes[w1.GetAddress()] != 1000 {
		t.Errorf("Expected w1 stake 1000, got %d", stakes[w1.GetAddress()])
	}
	if stakes[w2.GetAddress()] != 300 {
		t.Errorf("Expected w2 stake 300, got %d", stakes[w2.GetAddress()])
	}
}

func TestContextReset(t *testing.T) {
	coinbase := NewCoinbaseTransaction("test", 1000, 0)
	genesis := GenesisBlock(coinbase)

	ctx, _ := NewContextWithGenesis(genesis)

	if ctx.GetBlockCount() != 1 {
		t.Error("Expected 1 block before reset")
	}

	ctx.Reset()

	if ctx.GetBlockCount() != 0 {
		t.Error("Expected 0 blocks after reset")
	}

	if ctx.GetBalance("test") != 0 {
		t.Error("Expected 0 balance after reset")
	}
}
