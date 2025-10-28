package blockchain

import (
	"fmt"
	"sync"
	"time"

	"github.com/krakovia/blockchain/pkg/wallet"
)

// Node representa um nó da rede blockchain
type Node struct {
	mu sync.RWMutex

	// Identificação
	id string

	// Componentes
	chain   *Chain
	mempool *Mempool
	miner   *Miner

	// Rede simulada (peers conectados)
	peers []*Node

	// Controle de mineração
	mining   bool
	stopChan chan struct{}

	// Callbacks para testes
	onBlockReceived func(*Block)
	onTxReceived    func(*Transaction)
}

// NewNode cria um novo nó com uma blockchain
func NewNode(id string, w *wallet.Wallet, chain *Chain, mempool *Mempool) *Node {
	miner := NewMiner(w, chain, mempool)

	node := &Node{
		id:      id,
		chain:   chain,
		mempool: mempool,
		miner:   miner,
		peers:   make([]*Node, 0),
	}

	// Configura callbacks do minerador para propagar automaticamente
	miner.SetOnBlockCreated(func(block *Block) {
		node.BroadcastBlock(block)
	})

	miner.SetOnTxCreated(func(tx *Transaction) {
		node.BroadcastTransaction(tx)
	})

	return node
}

// GetID retorna o ID do nó
func (n *Node) GetID() string {
	return n.id
}

// GetChain retorna a blockchain do nó
func (n *Node) GetChain() *Chain {
	return n.chain
}

// GetMempool retorna o mempool do nó
func (n *Node) GetMempool() *Mempool {
	return n.mempool
}

// GetMiner retorna o minerador do nó
func (n *Node) GetMiner() *Miner {
	return n.miner
}

// ConnectPeer conecta este nó a outro nó (simula conexão de rede)
func (n *Node) ConnectPeer(peer *Node) {
	n.mu.Lock()
	defer n.mu.Unlock()

	// Evita duplicatas
	for _, p := range n.peers {
		if p.id == peer.id {
			return
		}
	}

	n.peers = append(n.peers, peer)
}

// DisconnectPeer desconecta um peer
func (n *Node) DisconnectPeer(peerID string) {
	n.mu.Lock()
	defer n.mu.Unlock()

	for i, p := range n.peers {
		if p.id == peerID {
			n.peers = append(n.peers[:i], n.peers[i+1:]...)
			return
		}
	}
}

// GetPeers retorna a lista de peers conectados
func (n *Node) GetPeers() []*Node {
	n.mu.RLock()
	defer n.mu.RUnlock()

	peers := make([]*Node, len(n.peers))
	copy(peers, n.peers)
	return peers
}

// BroadcastBlock envia um bloco para todos os peers conectados
func (n *Node) BroadcastBlock(block *Block) {
	peers := n.GetPeers()

	for _, peer := range peers {
		// Envia em goroutine para não bloquear
		go func(p *Node) {
			_ = p.ReceiveBlock(block)
		}(peer)
	}
}

// BroadcastTransaction envia uma transação para todos os peers conectados
func (n *Node) BroadcastTransaction(tx *Transaction) {
	peers := n.GetPeers()

	for _, peer := range peers {
		// Envia em goroutine para não bloquear
		go func(p *Node) {
			_ = p.ReceiveTransaction(tx)
		}(peer)
	}
}

// ReceiveBlock recebe um bloco de outro nó
func (n *Node) ReceiveBlock(block *Block) error {
	// Callback para testes
	if n.onBlockReceived != nil {
		n.onBlockReceived(block)
	}

	// Verifica se já tem o bloco
	if _, exists := n.chain.GetBlock(block.Hash); exists {
		return nil // Já tem, ignora
	}

	// Tenta adicionar à chain
	if err := n.chain.AddBlock(block); err != nil {
		return fmt.Errorf("failed to add received block: %w", err)
	}

	// Remove transações do mempool que estão no bloco
	txIDs := make([]string, 0, len(block.Transactions))
	for _, tx := range block.Transactions {
		if !tx.IsCoinbase() {
			txIDs = append(txIDs, tx.ID)
		}
	}
	n.mempool.RemoveTransactions(txIDs)

	// Propaga para outros peers (exceto quem enviou)
	// Nota: em uma rede real, teríamos lógica de roteamento mais sofisticada
	n.BroadcastBlock(block)

	return nil
}

// ReceiveTransaction recebe uma transação de outro nó
func (n *Node) ReceiveTransaction(tx *Transaction) error {
	// Callback para testes
	if n.onTxReceived != nil {
		n.onTxReceived(tx)
	}

	// Verifica se já tem a transação
	if _, exists := n.mempool.GetTransaction(tx.ID); exists {
		return nil // Já tem, ignora
	}

	// Tenta adicionar ao mempool
	if err := n.mempool.AddTransaction(tx); err != nil {
		return fmt.Errorf("failed to add received transaction: %w", err)
	}

	// Propaga para outros peers
	n.BroadcastTransaction(tx)

	return nil
}

// StartMining inicia a mineração em background
func (n *Node) StartMining() {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.mining {
		return // Já está minerando
	}

	n.mining = true
	n.stopChan = make(chan struct{})

	go n.miner.MineLoop(n.stopChan)
}

// StopMining para a mineração
func (n *Node) StopMining() {
	n.mu.Lock()
	defer n.mu.Unlock()

	if !n.mining {
		return
	}

	close(n.stopChan)
	n.mining = false
}

// IsMining retorna se o nó está minerando
func (n *Node) IsMining() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()

	return n.mining
}

// CreateTransaction cria uma nova transação e adiciona ao mempool
func (n *Node) CreateTransaction(to string, amount, fee uint64, data string) (*Transaction, error) {
	tx, err := n.miner.CreateTransaction(to, amount, fee, data)
	if err != nil {
		return nil, err
	}

	// Adiciona ao próprio mempool
	if err := n.mempool.AddTransaction(tx); err != nil {
		return nil, fmt.Errorf("failed to add transaction to mempool: %w", err)
	}

	// Broadcast é feito automaticamente pelo callback do minerador

	return tx, nil
}

// CreateStakeTransaction cria uma transação de stake
func (n *Node) CreateStakeTransaction(amount, fee uint64) (*Transaction, error) {
	tx, err := n.miner.CreateStakeTransaction(amount, fee)
	if err != nil {
		return nil, err
	}

	if err := n.mempool.AddTransaction(tx); err != nil {
		return nil, fmt.Errorf("failed to add stake transaction to mempool: %w", err)
	}

	return tx, nil
}

// CreateUnstakeTransaction cria uma transação de unstake
func (n *Node) CreateUnstakeTransaction(amount, fee uint64) (*Transaction, error) {
	tx, err := n.miner.CreateUnstakeTransaction(amount, fee)
	if err != nil {
		return nil, err
	}

	if err := n.mempool.AddTransaction(tx); err != nil {
		return nil, fmt.Errorf("failed to add unstake transaction to mempool: %w", err)
	}

	return tx, nil
}

// GetBalance retorna o saldo do nó
func (n *Node) GetBalance() uint64 {
	return n.miner.GetBalance()
}

// GetStake retorna o stake do nó
func (n *Node) GetStake() uint64 {
	return n.miner.GetStake()
}

// GetNonce retorna o nonce do nó
func (n *Node) GetNonce() uint64 {
	return n.miner.GetNonce()
}

// SetOnBlockReceived define callback para quando um bloco é recebido
func (n *Node) SetOnBlockReceived(callback func(*Block)) {
	n.onBlockReceived = callback
}

// SetOnTxReceived define callback para quando uma transação é recebida
func (n *Node) SetOnTxReceived(callback func(*Transaction)) {
	n.onTxReceived = callback
}

// WaitForHeight aguarda até que a chain atinja determinada altura
// Útil para testes de sincronização
func (n *Node) WaitForHeight(height uint64, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if n.chain.GetHeight() >= height {
			return nil
		}
		time.Sleep(10 * time.Millisecond)
	}

	return fmt.Errorf("timeout waiting for height %d (current: %d)", height, n.chain.GetHeight())
}

// SyncWithPeer tenta sincronizar com um peer (baixa blocos faltantes)
func (n *Node) SyncWithPeer(peer *Node) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	myHeight := n.chain.GetHeight()
	peerHeight := peer.chain.GetHeight()

	if peerHeight <= myHeight {
		return nil // Peer não está à frente
	}

	// Baixa blocos do peer
	for h := myHeight + 1; h <= peerHeight; h++ {
		block, exists := peer.chain.GetBlockByHeight(h)
		if !exists {
			return fmt.Errorf("peer missing block at height %d", h)
		}

		if err := n.chain.AddBlock(block); err != nil {
			return fmt.Errorf("failed to add block %d during sync: %w", h, err)
		}

		// Remove transações do mempool
		txIDs := make([]string, 0, len(block.Transactions))
		for _, tx := range block.Transactions {
			if !tx.IsCoinbase() {
				txIDs = append(txIDs, tx.ID)
			}
		}
		n.mempool.RemoveTransactions(txIDs)
	}

	return nil
}

// GetNodeStats retorna estatísticas do nó
func (n *Node) GetNodeStats() NodeStats {
	return NodeStats{
		ID:            n.id,
		Height:        n.chain.GetHeight(),
		Balance:       n.GetBalance(),
		Stake:         n.GetStake(),
		Nonce:         n.GetNonce(),
		MempoolSize:   n.mempool.Size(),
		PeerCount:     len(n.GetPeers()),
		IsMining:      n.IsMining(),
		ValidatorRank: n.miner.GetRank(),
	}
}

// NodeStats estatísticas do nó
type NodeStats struct {
	ID            string
	Height        uint64
	Balance       uint64
	Stake         uint64
	Nonce         uint64
	MempoolSize   int
	PeerCount     int
	IsMining      bool
	ValidatorRank int
}
