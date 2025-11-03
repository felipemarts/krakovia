package node

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/krakovia/blockchain/pkg/blockchain"
	"github.com/krakovia/blockchain/pkg/network"
	"github.com/krakovia/blockchain/pkg/wallet"
	"github.com/syndtr/goleveldb/leveldb"
)

// Node representa um nó na blockchain
type Node struct {
	ID                string
	Address           string
	db                *leveldb.DB
	webRTC            *network.WebRTCClient
	peers             map[string]*network.Peer
	peersMutex        sync.RWMutex
	discovery         *network.PeerDiscovery
	ctx               context.Context
	cancel            context.CancelFunc
	discoveryInterval time.Duration

	// Componentes blockchain
	wallet  *wallet.Wallet
	chain   *blockchain.Chain
	mempool *blockchain.Mempool
	miner   *blockchain.Miner

	// Controle de mineração
	mining   bool
	stopMine chan struct{}
}

// Config contém as configurações para criar um nó
type Config struct {
	ID                string
	Address           string
	DBPath            string
	SignalingServer   string
	MaxPeers          int
	MinPeers          int
	DiscoveryInterval int // em segundos

	// Configurações blockchain
	Wallet       *wallet.Wallet
	GenesisBlock *blockchain.Block
	ChainConfig  blockchain.ChainConfig
}

// NewNode cria uma nova instância de nó
func NewNode(config Config) (*Node, error) {
	// Validar configurações blockchain
	if config.Wallet == nil {
		return nil, fmt.Errorf("wallet is required")
	}
	if config.GenesisBlock == nil {
		return nil, fmt.Errorf("genesis block is required")
	}

	// Abrir banco de dados LevelDB
	db, err := leveldb.OpenFile(config.DBPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Valores padrão
	if config.MaxPeers == 0 {
		config.MaxPeers = 50
	}
	if config.MinPeers == 0 {
		config.MinPeers = 5
	}
	if config.DiscoveryInterval == 0 {
		config.DiscoveryInterval = 30
	}

	// Configuração padrão da chain se não fornecida
	chainConfig := config.ChainConfig
	if chainConfig.BlockTime == 0 {
		chainConfig = blockchain.DefaultChainConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Criar sistema de descoberta de peers
	discovery := network.NewPeerDiscovery(config.ID, config.MaxPeers, config.MinPeers)

	// Inicializar blockchain
	chain, err := blockchain.NewChain(config.GenesisBlock, chainConfig)
	if err != nil {
		if closeErr := db.Close(); closeErr != nil {
			fmt.Printf("Warning: failed to close DB: %v\n", closeErr)
		}
		cancel()
		return nil, fmt.Errorf("failed to create chain: %w", err)
	}

	// Criar mempool
	mempool := blockchain.NewMempool()

	// Criar minerador
	miner := blockchain.NewMiner(config.Wallet, chain, mempool)

	node := &Node{
		ID:                config.ID,
		Address:           config.Address,
		db:                db,
		peers:             make(map[string]*network.Peer),
		discovery:         discovery,
		ctx:               ctx,
		cancel:            cancel,
		discoveryInterval: time.Duration(config.DiscoveryInterval) * time.Second,
		wallet:            config.Wallet,
		chain:             chain,
		mempool:           mempool,
		miner:             miner,
	}

	// Configurar callbacks do minerador para broadcast via rede
	miner.SetOnBlockCreated(func(block *blockchain.Block) {
		node.broadcastBlock(block)
	})

	miner.SetOnTxCreated(func(tx *blockchain.Transaction) {
		node.broadcastTransaction(tx)
	})

	// Inicializar cliente WebRTC com sistema de descoberta
	webRTCClient, err := network.NewWebRTCClientWithDiscovery(config.ID, config.SignalingServer, node, discovery)
	if err != nil {
		if closeErr := db.Close(); closeErr != nil {
			fmt.Printf("Warning: failed to close DB: %v\n", closeErr)
		}
		cancel()
		return nil, fmt.Errorf("failed to create WebRTC client: %w", err)
	}

	node.webRTC = webRTCClient

	// Registrar handlers de mensagens
	node.registerMessageHandlers()

	return node, nil
}

// Start inicia o nó
func (n *Node) Start() error {
	fmt.Printf("Starting node %s at %s\n", n.ID, n.Address)

	// Conectar ao servidor de signaling
	if err := n.webRTC.Connect(); err != nil {
		return fmt.Errorf("failed to connect to signaling server: %w", err)
	}

	// Iniciar goroutine de descoberta periódica
	go n.discoveryLoop()

	return nil
}

// discoveryLoop executa descoberta periódica de peers
func (n *Node) discoveryLoop() {
	ticker := time.NewTicker(n.discoveryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-n.ctx.Done():
			return
		case <-ticker.C:
			n.runDiscovery()
		}
	}
}

// runDiscovery executa uma rodada de descoberta
func (n *Node) runDiscovery() {
	// Verificar se precisa de mais peers
	if n.discovery.NeedsMorePeers() {
		fmt.Printf("[%s] Need more peers, requesting peer list\n", n.ID)
		n.webRTC.RequestPeerList()
	}

	// Verificar se tem peers demais e desconectar alguns
	if !n.discovery.ShouldAcceptNewPeer() {
		peers := n.GetPeers()
		peerIDs := make([]string, len(peers))
		for i, p := range peers {
			peerIDs[i] = p.ID
		}

		toDisconnect := n.discovery.SelectPeersToDisconnect(peerIDs)
		for _, peerID := range toDisconnect {
			fmt.Printf("[%s] Disconnecting peer %s (over limit)\n", n.ID, peerID)
			if err := n.webRTC.DisconnectPeer(peerID); err != nil {
				fmt.Printf("[%s] Failed to disconnect peer %s: %v\n", n.ID, peerID, err)
			}
		}
	}

	// Imprimir estatísticas
	n.discovery.PrintStats()
}

// Stop para o nó e limpa recursos
func (n *Node) Stop() error {
	fmt.Printf("Stopping node %s\n", n.ID)

	// Para mineração se estiver ativa
	n.StopMining()

	n.cancel()

	if n.webRTC != nil {
		n.webRTC.Close()
	}

	if n.db != nil {
		if err := n.db.Close(); err != nil {
			return fmt.Errorf("failed to close database: %w", err)
		}
	}

	return nil
}

// AddPeer adiciona um peer à lista de peers conectados
func (n *Node) AddPeer(peer *network.Peer) {
	n.peersMutex.Lock()
	defer n.peersMutex.Unlock()
	n.peers[peer.ID] = peer
	n.discovery.MarkPeerConnected(peer.ID)

	// Configura handler para mensagens recebidas deste peer
	peer.OnMessage = func(msgType string, data []byte) {
		n.HandlePeerMessage(peer.ID, msgType, data)
	}

	fmt.Printf("Peer %s connected to node %s\n", peer.ID, n.ID)

	// Solicita sincronização com o peer
	go n.requestSync(peer.ID)
}

// RemovePeer remove um peer da lista
func (n *Node) RemovePeer(peerID string) {
	n.peersMutex.Lock()
	defer n.peersMutex.Unlock()
	delete(n.peers, peerID)
	n.discovery.MarkPeerDisconnected(peerID)
	fmt.Printf("Peer %s disconnected from node %s\n", peerID, n.ID)
}

// GetPeers retorna a lista de peers conectados
func (n *Node) GetPeers() []*network.Peer {
	n.peersMutex.RLock()
	defer n.peersMutex.RUnlock()

	peers := make([]*network.Peer, 0, len(n.peers))
	for _, peer := range n.peers {
		peers = append(peers, peer)
	}
	return peers
}

// BroadcastMessage envia uma mensagem para todos os peers
func (n *Node) BroadcastMessage(msgType string, data []byte) {
	n.peersMutex.RLock()
	defer n.peersMutex.RUnlock()

	for _, peer := range n.peers {
		if err := peer.SendMessage(msgType, data); err != nil {
			fmt.Printf("Failed to send message to peer %s: %v\n", peer.ID, err)
		}
	}
}

// GetWebRTC retorna o cliente WebRTC do nó
func (n *Node) GetWebRTC() *network.WebRTCClient {
	return n.webRTC
}

// registerMessageHandlers registra handlers para mensagens recebidas da rede
func (n *Node) registerMessageHandlers() {
	// Não há método RegisterHandler no WebRTCClient
	// Os handlers são configurados quando os peers são adicionados
	// via SetDataChannel que já configura OnMessage callback
}

// HandlePeerMessage processa mensagens recebidas de peers (chamado pelo Peer.OnMessage)
func (n *Node) HandlePeerMessage(peerID string, msgType string, data []byte) {
	switch msgType {
	case "block":
		n.handleBlockMessage(peerID, data)
	case "transaction":
		n.handleTransactionMessage(peerID, data)
	case "sync_request":
		n.handleSyncRequest(peerID, data)
	case "sync_response":
		n.handleSyncResponse(peerID, data)
	default:
		fmt.Printf("[%s] Unknown message type '%s' from peer %s\n", n.ID, msgType, peerID)
	}
}

// handleBlockMessage processa um bloco recebido da rede
func (n *Node) handleBlockMessage(peerID string, data []byte) {
	block, err := blockchain.DeserializeBlock(data)
	if err != nil {
		fmt.Printf("[%s] Failed to deserialize block from %s: %v\n", n.ID, peerID, err)
		return
	}

	fmt.Printf("[%s] Received block %d (hash: %s) from %s\n", n.ID, block.Header.Height, block.Hash[:8], peerID)

	// Verifica se já tem o bloco
	if _, exists := n.chain.GetBlock(block.Hash); exists {
		return // Já tem, ignora
	}

	// Tenta adicionar à chain
	if err := n.chain.AddBlock(block); err != nil {
		fmt.Printf("[%s] Failed to add block: %v\n", n.ID, err)
		return
	}

	fmt.Printf("[%s] Block %d added to chain successfully\n", n.ID, block.Header.Height)

	// Remove transações do mempool que estão no bloco
	txIDs := make([]string, 0, len(block.Transactions))
	for _, tx := range block.Transactions {
		if !tx.IsCoinbase() {
			txIDs = append(txIDs, tx.ID)
		}
	}
	removed := n.mempool.RemoveTransactions(txIDs)
	if removed > 0 {
		fmt.Printf("[%s] Removed %d transactions from mempool\n", n.ID, removed)
	}

	// Propaga para outros peers (exceto quem enviou)
	n.broadcastBlockExcept(block, peerID)
}

// handleTransactionMessage processa uma transação recebida da rede
func (n *Node) handleTransactionMessage(peerID string, data []byte) {
	tx, err := blockchain.DeserializeTransaction(data)
	if err != nil {
		fmt.Printf("[%s] Failed to deserialize transaction from %s: %v\n", n.ID, peerID, err)
		return
	}

	fmt.Printf("[%s] Received transaction %s from %s\n", n.ID, tx.ID[:8], peerID)

	// Verifica se já tem a transação
	if _, exists := n.mempool.GetTransaction(tx.ID); exists {
		return // Já tem, ignora
	}

	// Tenta adicionar ao mempool
	if err := n.mempool.AddTransaction(tx); err != nil {
		fmt.Printf("[%s] Failed to add transaction to mempool: %v\n", n.ID, err)
		return
	}

	fmt.Printf("[%s] Transaction %s added to mempool\n", n.ID, tx.ID[:8])

	// Propaga para outros peers (exceto quem enviou)
	n.broadcastTransactionExcept(tx, peerID)
}

// SyncRequest mensagem de requisição de sincronização
type SyncRequest struct {
	FromHeight uint64 `json:"from_height"`
}

// SyncResponse mensagem de resposta de sincronização
type SyncResponse struct {
	Blocks []*blockchain.Block `json:"blocks"`
}

// handleSyncRequest processa uma requisição de sincronização
func (n *Node) handleSyncRequest(peerID string, data []byte) {
	var req SyncRequest
	if err := json.Unmarshal(data, &req); err != nil {
		fmt.Printf("[%s] Failed to parse sync request from %s: %v\n", n.ID, peerID, err)
		return
	}

	fmt.Printf("[%s] Received sync request from %s (from height %d)\n", n.ID, peerID, req.FromHeight)

	// Pega blocos a partir da altura solicitada
	currentHeight := n.chain.GetHeight()
	if req.FromHeight > currentHeight {
		fmt.Printf("[%s] Peer %s is ahead, nothing to send\n", n.ID, peerID)
		return
	}

	// Limita a quantidade de blocos por vez
	maxBlocks := uint64(100)
	toHeight := req.FromHeight + maxBlocks
	if toHeight > currentHeight {
		toHeight = currentHeight
	}

	blocks := n.chain.GetBlockRange(req.FromHeight, toHeight)

	// Envia resposta
	response := SyncResponse{
		Blocks: blocks,
	}

	responseData, err := json.Marshal(response)
	if err != nil {
		fmt.Printf("[%s] Failed to marshal sync response: %v\n", n.ID, err)
		return
	}

	// Envia para o peer
	n.peersMutex.RLock()
	peer := n.peers[peerID]
	n.peersMutex.RUnlock()

	if peer != nil {
		if err := peer.SendMessage("sync_response", responseData); err != nil {
			fmt.Printf("[%s] Failed to send sync response to %s: %v\n", n.ID, peerID, err)
		} else {
			fmt.Printf("[%s] Sent %d blocks to %s (height %d-%d)\n", n.ID, len(blocks), peerID, req.FromHeight, toHeight)
		}
	}
}

// handleSyncResponse processa uma resposta de sincronização
func (n *Node) handleSyncResponse(peerID string, data []byte) {
	var resp SyncResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		fmt.Printf("[%s] Failed to parse sync response from %s: %v\n", n.ID, peerID, err)
		return
	}

	fmt.Printf("[%s] Received sync response from %s with %d blocks\n", n.ID, peerID, len(resp.Blocks))

	// Adiciona blocos à chain
	added := 0
	for _, block := range resp.Blocks {
		// Verifica se já tem o bloco
		if _, exists := n.chain.GetBlock(block.Hash); exists {
			continue
		}

		// Adiciona à chain
		if err := n.chain.AddBlock(block); err != nil {
			fmt.Printf("[%s] Failed to add synced block %d: %v\n", n.ID, block.Header.Height, err)
			break
		}

		// Remove transações do mempool
		txIDs := make([]string, 0, len(block.Transactions))
		for _, tx := range block.Transactions {
			if !tx.IsCoinbase() {
				txIDs = append(txIDs, tx.ID)
			}
		}
		n.mempool.RemoveTransactions(txIDs)
		added++
	}

	if added > 0 {
		fmt.Printf("[%s] Successfully synced %d blocks, current height: %d\n", n.ID, added, n.chain.GetHeight())
	}
}

// broadcastBlock envia um bloco para todos os peers
func (n *Node) broadcastBlock(block *blockchain.Block) {
	data, err := block.Serialize()
	if err != nil {
		fmt.Printf("[%s] Failed to serialize block: %v\n", n.ID, err)
		return
	}

	fmt.Printf("[%s] Broadcasting block %d to all peers\n", n.ID, block.Header.Height)
	n.BroadcastMessage("block", data)
}

// broadcastBlockExcept envia um bloco para todos os peers exceto um
func (n *Node) broadcastBlockExcept(block *blockchain.Block, exceptPeerID string) {
	data, err := block.Serialize()
	if err != nil {
		fmt.Printf("[%s] Failed to serialize block: %v\n", n.ID, err)
		return
	}

	n.peersMutex.RLock()
	defer n.peersMutex.RUnlock()

	for _, peer := range n.peers {
		if peer.ID != exceptPeerID {
			if err := peer.SendMessage("block", data); err != nil {
				fmt.Printf("[%s] Failed to send block to peer %s: %v\n", n.ID, peer.ID, err)
			}
		}
	}
}

// broadcastTransaction envia uma transação para todos os peers
func (n *Node) broadcastTransaction(tx *blockchain.Transaction) {
	data, err := tx.Serialize()
	if err != nil {
		fmt.Printf("[%s] Failed to serialize transaction: %v\n", n.ID, err)
		return
	}

	fmt.Printf("[%s] Broadcasting transaction %s to all peers\n", n.ID, tx.ID[:8])
	n.BroadcastMessage("transaction", data)
}

// broadcastTransactionExcept envia uma transação para todos os peers exceto um
func (n *Node) broadcastTransactionExcept(tx *blockchain.Transaction, exceptPeerID string) {
	data, err := tx.Serialize()
	if err != nil {
		fmt.Printf("[%s] Failed to serialize transaction: %v\n", n.ID, err)
		return
	}

	n.peersMutex.RLock()
	defer n.peersMutex.RUnlock()

	for _, peer := range n.peers {
		if peer.ID != exceptPeerID {
			if err := peer.SendMessage("transaction", data); err != nil {
				fmt.Printf("[%s] Failed to send transaction to peer %s: %v\n", n.ID, peer.ID, err)
			}
		}
	}
}

// Blockchain API methods

// StartMining inicia a mineração em background
func (n *Node) StartMining() error {
	if n.mining {
		return fmt.Errorf("already mining")
	}

	n.mining = true
	n.stopMine = make(chan struct{})

	go n.miner.MineLoop(n.stopMine)

	fmt.Printf("[%s] Mining started\n", n.ID)
	return nil
}

// StopMining para a mineração
func (n *Node) StopMining() {
	if !n.mining {
		return
	}

	close(n.stopMine)
	n.mining = false

	fmt.Printf("[%s] Mining stopped\n", n.ID)
}

// IsMining retorna se o nó está minerando
func (n *Node) IsMining() bool {
	return n.mining
}

// CreateTransaction cria uma nova transação e adiciona ao mempool
func (n *Node) CreateTransaction(to string, amount, fee uint64, data string) (*blockchain.Transaction, error) {
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
func (n *Node) CreateStakeTransaction(amount, fee uint64) (*blockchain.Transaction, error) {
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
func (n *Node) CreateUnstakeTransaction(amount, fee uint64) (*blockchain.Transaction, error) {
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
	return n.chain.GetBalance(n.wallet.GetAddress())
}

// GetStake retorna o stake do nó
func (n *Node) GetStake() uint64 {
	return n.chain.GetStake(n.wallet.GetAddress())
}

// GetNonce retorna o nonce do nó
func (n *Node) GetNonce() uint64 {
	return n.chain.GetNonce(n.wallet.GetAddress())
}

// GetChainHeight retorna a altura atual da blockchain
func (n *Node) GetChainHeight() uint64 {
	return n.chain.GetHeight()
}

// GetMempoolSize retorna o número de transações no mempool
func (n *Node) GetMempoolSize() int {
	return n.mempool.Size()
}

// GetBlockchainStats retorna estatísticas da blockchain
func (n *Node) GetBlockchainStats() blockchain.ChainStats {
	return n.chain.GetChainStats()
}

// PrintStats imprime estatísticas do nó
func (n *Node) PrintStats() {
	fmt.Printf("\n=== Node %s Stats ===\n", n.ID)
	fmt.Printf("Address: %s\n", n.wallet.GetAddress())
	fmt.Printf("Balance: %d\n", n.GetBalance())
	fmt.Printf("Stake: %d\n", n.GetStake())
	fmt.Printf("Nonce: %d\n", n.GetNonce())
	fmt.Printf("Chain Height: %d\n", n.GetChainHeight())
	fmt.Printf("Mempool Size: %d\n", n.GetMempoolSize())
	fmt.Printf("Connected Peers: %d\n", len(n.GetPeers()))
	fmt.Printf("Mining: %v\n", n.IsMining())
	fmt.Printf("====================\n\n")
}

// requestSync solicita sincronização de blockchain com um peer
func (n *Node) requestSync(peerID string) {
	// Espera um pouco para garantir que a conexão está estabelecida
	time.Sleep(500 * time.Millisecond)

	currentHeight := n.chain.GetHeight()

	// Solicita blocos a partir da próxima altura
	req := SyncRequest{
		FromHeight: currentHeight + 1,
	}

	data, err := json.Marshal(req)
	if err != nil {
		fmt.Printf("[%s] Failed to marshal sync request: %v\n", n.ID, err)
		return
	}

	n.peersMutex.RLock()
	peer := n.peers[peerID]
	n.peersMutex.RUnlock()

	if peer != nil {
		if err := peer.SendMessage("sync_request", data); err != nil {
			fmt.Printf("[%s] Failed to send sync request to %s: %v\n", n.ID, peerID, err)
		} else {
			fmt.Printf("[%s] Requested sync from %s (from height %d)\n", n.ID, peerID, req.FromHeight)
		}
	}
}
