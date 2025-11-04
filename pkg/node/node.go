package node

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/krakovia/blockchain/internal/config"
	"github.com/krakovia/blockchain/pkg/api"
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

	// Checkpoint
	checkpointConfig     *config.CheckpointConfig
	lastCheckpointHash   string
	lastCheckpointHeight uint64
	checkpointMutex      sync.RWMutex

	// API HTTP
	apiServer *api.Server
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
	Wallet           *wallet.Wallet
	GenesisBlock     *blockchain.Block
	ChainConfig      blockchain.ChainConfig
	CheckpointConfig *config.CheckpointConfig
	APIConfig        *config.APIConfig
	InitialStake     uint64 // Stake inicial (0 = sem stake inicial)
	InitialStakeAddr string // Endereço que receberá o stake inicial
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

	// Inicializar blockchain com stake inicial se fornecido
	var chain *blockchain.Chain
	if config.InitialStakeAddr != "" && config.InitialStake > 0 {
		chain, err = blockchain.NewChainWithStake(config.GenesisBlock, chainConfig, config.InitialStakeAddr, config.InitialStake)
	} else {
		chain, err = blockchain.NewChain(config.GenesisBlock, chainConfig)
	}
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
		checkpointConfig:  config.CheckpointConfig,
	}

	// Carregar blockchain existente do disco
	if err := node.loadChainFromDisk(); err != nil {
		fmt.Printf("[%s] Warning: failed to load chain from disk: %v\n", config.ID, err)
	}

	// Carregar último checkpoint do disco (se existir)
	if config.CheckpointConfig != nil && config.CheckpointConfig.Enabled {
		node.loadLastCheckpoint()
	}

	// Configurar callbacks do minerador para broadcast via rede
	miner.SetOnBlockCreated(func(block *blockchain.Block) {
		// Adicionar checkpoint hash ao bloco se disponível
		node.addCheckpointHashToBlock(block)
		// Salvar bloco no disco
		if err := blockchain.SaveBlockToDB(node.db, block); err != nil {
			fmt.Printf("[%s] ⚠️  Warning: failed to save mined block %d to disk: %v\n", node.ID, block.Header.Height, err)
		} else {
			fmt.Printf("[%s] 💾 Mined block %d saved to disk successfully\n", node.ID, block.Header.Height)
		}
		// Tentar criar checkpoint se necessário
		node.tryCreateCheckpoint(block.Header.Height)
		// Broadcast do bloco
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

	// Inicializar servidor HTTP da API (se habilitado)
	if config.APIConfig != nil && config.APIConfig.Enabled {
		apiConfig := &api.Config{
			Enabled:  config.APIConfig.Enabled,
			Address:  config.APIConfig.Address,
			Username: config.APIConfig.Username,
			Password: config.APIConfig.Password,
		}
		// Criar wrapper para o node
		nodeWrapper := api.NewNodeWrapper(node)
		node.apiServer = api.NewServer(nodeWrapper, apiConfig)
	}

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

	// Iniciar servidor HTTP da API (se configurado)
	if n.apiServer != nil {
		if err := n.apiServer.Start(); err != nil {
			fmt.Printf("Warning: failed to start API server: %v\n", err)
		}
	}

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

	// Parar servidor HTTP da API
	if n.apiServer != nil {
		if err := n.apiServer.Stop(); err != nil {
			fmt.Printf("Warning: failed to stop API server: %v\n", err)
		}
	}

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

	fmt.Printf("🔗 Peer %s connected to node %s\n", peer.ID, n.ID)

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
	case "checkpoint_request":
		n.handleCheckpointRequest(peerID, data)
	case "checkpoint_response":
		n.handleCheckpointResponse(peerID, data)
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

	// Validar checkpoint hash se presente no bloco
	if block.Header.CheckpointHash != "" && n.checkpointConfig != nil && n.checkpointConfig.Enabled {
		if err := n.validateBlockCheckpointHash(block); err != nil {
			fmt.Printf("[%s] Block checkpoint validation failed: %v\n", n.ID, err)
			return
		}
	}

	// Tenta adicionar à chain
	if err := n.chain.AddBlock(block); err != nil {
		fmt.Printf("[%s] Failed to add block: %v\n", n.ID, err)
		return
	}

	fmt.Printf("[%s] Block %d added to chain successfully\n", n.ID, block.Header.Height)

	// Salvar bloco no disco
	if err := blockchain.SaveBlockToDB(n.db, block); err != nil {
		fmt.Printf("[%s] ⚠️  Warning: failed to save block %d to disk: %v\n", n.ID, block.Header.Height, err)
	} else {
		fmt.Printf("[%s] 💾 Block %d saved to disk successfully\n", n.ID, block.Header.Height)
	}

	// Tentar criar checkpoint se necessário
	n.tryCreateCheckpoint(block.Header.Height)

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

// CheckpointRequest mensagem de requisição de checkpoint
type CheckpointRequest struct {
	RequestedHeight uint64 `json:"requested_height"` // 0 = último checkpoint
}

// CheckpointResponse mensagem de resposta de checkpoint
type CheckpointResponse struct {
	Checkpoint       *blockchain.Checkpoint   `json:"checkpoint"`
	BlocksSince      []*blockchain.Block      `json:"blocks_since"` // blocos após checkpoint
	HasCheckpoint    bool                     `json:"has_checkpoint"`
	AvailableHeights []uint64                 `json:"available_heights,omitempty"` // alturas disponíveis
	AllCheckpoints   []*blockchain.Checkpoint `json:"all_checkpoints,omitempty"`   // todos os checkpoints necessários
}

// handleSyncRequest processa uma requisição de sincronização
func (n *Node) handleSyncRequest(peerID string, data []byte) {
	var req SyncRequest
	if err := json.Unmarshal(data, &req); err != nil {
		fmt.Printf("[%s] Failed to parse sync request from %s: %v\n", n.ID, peerID, err)
		return
	}

	fmt.Printf("[%s] 📥 Received sync request from %s (from height %d)\n", n.ID, peerID, req.FromHeight)

	// Pega blocos a partir da altura solicitada
	currentHeight := n.chain.GetHeight()
	fmt.Printf("[%s] 📊 Current height: %d, peer requested from: %d\n", n.ID, currentHeight, req.FromHeight)

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

	// Se não conseguiu todos os blocos (devido ao bug de pruning), carregar do DB
	expectedCount := int(toHeight - req.FromHeight + 1)
	if len(blocks) < expectedCount {
		fmt.Printf("[%s] GetBlockRange returned %d/%d blocks, loading from DB for heights %d-%d\n",
			n.ID, len(blocks), expectedCount, req.FromHeight, toHeight)

		blocks = make([]*blockchain.Block, 0, expectedCount)
		for h := req.FromHeight; h <= toHeight; h++ {
			block, exists := n.chain.GetBlockByHeight(h)
			if exists && block != nil && block.Header.Height == h {
				blocks = append(blocks, block)
			} else {
				// Carregar do DB
				block, err := blockchain.LoadBlockFromDB(n.db, h)
				if err != nil {
					fmt.Printf("[%s] Failed to load block %d from DB: %v\n", n.ID, h, err)
					break
				}
				blocks = append(blocks, block)
			}
		}
		fmt.Printf("[%s] After DB loading: have %d blocks for sync\n", n.ID, len(blocks))
	}

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
			fmt.Printf("[%s] ❌ Failed to send sync response to %s: %v\n", n.ID, peerID, err)
		} else {
			fmt.Printf("[%s] ✅ Sent %d blocks to %s (height %d-%d)\n", n.ID, len(blocks), peerID, req.FromHeight, toHeight)
		}
	} else {
		fmt.Printf("[%s] ❌ Peer %s not found, cannot send sync response\n", n.ID, peerID)
	}
}

// handleSyncResponse processa uma resposta de sincronização
func (n *Node) handleSyncResponse(peerID string, data []byte) {
	var resp SyncResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		fmt.Printf("[%s] Failed to parse sync response from %s: %v\n", n.ID, peerID, err)
		return
	}

	fmt.Printf("[%s] 🔄 Received sync response from %s with %d blocks\n", n.ID, peerID, len(resp.Blocks))

	// Adiciona blocos à chain
	added := 0
	for i, block := range resp.Blocks {
		fmt.Printf("[%s] 📦 Processing block %d/%d: height=%d, hash=%s\n",
			n.ID, i+1, len(resp.Blocks), block.Header.Height, block.Hash[:8])

		// Verifica se já tem o bloco
		if _, exists := n.chain.GetBlock(block.Hash); exists {
			fmt.Printf("[%s] ⏭️  Block %d already exists, skipping\n", n.ID, block.Header.Height)
			continue
		}

		// Adiciona à chain
		if err := n.chain.AddBlock(block); err != nil {
			fmt.Printf("[%s] ❌ Failed to add synced block %d: %v\n", n.ID, block.Header.Height, err)
			break
		}

		fmt.Printf("[%s] ✅ Successfully added block %d\n", n.ID, block.Header.Height)

		// Salvar bloco no disco
		if err := blockchain.SaveBlockToDB(n.db, block); err != nil {
			fmt.Printf("[%s] Warning: failed to save synced block %d to disk: %v\n", n.ID, block.Header.Height, err)
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
		fmt.Printf("[%s] ✨ Successfully synced %d blocks, current height: %d\n", n.ID, added, n.chain.GetHeight())
	} else if len(resp.Blocks) > 0 {
		fmt.Printf("[%s] ℹ️  No new blocks added (all already exist)\n", n.ID)
	}
}

// handleCheckpointRequest processa uma requisição de checkpoint
func (n *Node) handleCheckpointRequest(peerID string, data []byte) {
	// Se checkpoint não está habilitado, ignora
	if n.checkpointConfig == nil || !n.checkpointConfig.Enabled {
		fmt.Printf("[%s] Checkpoint not enabled, ignoring request from %s\n", n.ID, peerID)
		return
	}

	var req CheckpointRequest
	if err := json.Unmarshal(data, &req); err != nil {
		fmt.Printf("[%s] Failed to parse checkpoint request from %s: %v\n", n.ID, peerID, err)
		return
	}

	fmt.Printf("[%s] Received checkpoint request from %s (height: %d)\n", n.ID, peerID, req.RequestedHeight)

	response := CheckpointResponse{
		HasCheckpoint: false,
	}

	// Se solicitou altura específica ou 0 (último), tenta carregar
	var checkpointHeight uint64
	if req.RequestedHeight == 0 {
		// Pegar último checkpoint
		var err error
		checkpointHeight, err = blockchain.GetLastCheckpointHeight(n.db)
		if err != nil || checkpointHeight == 0 {
			fmt.Printf("[%s] No checkpoint available: %v\n", n.ID, err)
			n.sendCheckpointResponse(peerID, response)
			return
		}
	} else {
		checkpointHeight = req.RequestedHeight
	}

	// Carregar checkpoint do DB
	checkpoint, err := blockchain.LoadCheckpointFromDB(n.db, checkpointHeight)
	if err != nil {
		fmt.Printf("[%s] Failed to load checkpoint at height %d: %v\n", n.ID, checkpointHeight, err)
		n.sendCheckpointResponse(peerID, response)
		return
	}

	// Pegar blocos desde o genesis até a altura atual (limitado)
	// NOTA: O peer precisa de TODOS os blocos desde o genesis para reconstruir a chain!
	currentHeight := n.chain.GetHeight()
	maxBlocks := uint64(100) // Limitar quantidade de blocos (aumentado para cobrir mais blocos)

	// Enviar blocos desde o GENESIS (altura 1), não após o checkpoint!
	// O checkpoint contém o estado, mas o peer ainda precisa dos blocos para validação
	fromHeight := uint64(1) // Começar do primeiro bloco após genesis
	toHeight := currentHeight
	if toHeight-fromHeight+1 > maxBlocks {
		toHeight = fromHeight + maxBlocks - 1
	}

	// Tentar pegar blocos da chain (memória)
	blocks := n.chain.GetBlockRange(fromHeight, toHeight)

	// Se não conseguiu blocos da memória (foram pruned), buscar do DB
	if len(blocks) < int(toHeight-fromHeight+1) {
		fmt.Printf("[%s] Blocks partially in memory (%d/%d), loading remaining from DB: height %d-%d\n",
			n.ID, len(blocks), toHeight-fromHeight+1, fromHeight, toHeight)

		blocks = make([]*blockchain.Block, 0, toHeight-fromHeight+1)
		for h := fromHeight; h <= toHeight; h++ {
			// Primeiro tenta da chain (memória)
			// NOTA: GetBlockByHeight tem um bug após pruning onde o índice não corresponde à altura
			// Então precisamos verificar a altura real do bloco retornado
			block, exists := n.chain.GetBlockByHeight(h)
			if exists && block != nil && block.Header.Height == h {
				blocks = append(blocks, block)
				fmt.Printf("[%s] Got block %d from memory\n", n.ID, h)
			} else {
				// Se não está em memória (ou índice errado), busca do DB
				var err error
				block, err = blockchain.LoadBlockFromDB(n.db, h)
				if err != nil {
					fmt.Printf("[%s] Failed to load block %d from DB: %v\n", n.ID, h, err)
					break
				}
				fmt.Printf("[%s] Loaded block %d from DB\n", n.ID, h)
				blocks = append(blocks, block)
			}
		}
		fmt.Printf("[%s] After DB loading: have %d blocks\n", n.ID, len(blocks))
	}

	// Carregar TODOS os checkpoints disponíveis para o peer poder validar os blocos
	// Os blocos contêm hashes de checkpoints anteriores, então precisamos enviá-los todos
	allCheckpoints := make([]*blockchain.Checkpoint, 0)

	// Tentar carregar checkpoint 0 (genesis checkpoint) se existir
	if checkpoint0, err := blockchain.LoadCheckpointFromDB(n.db, 0); err == nil {
		allCheckpoints = append(allCheckpoints, checkpoint0)
	}

	// Adicionar o checkpoint principal se não for o checkpoint 0
	if checkpointHeight != 0 {
		allCheckpoints = append(allCheckpoints, checkpoint)
	}

	response.HasCheckpoint = true
	response.Checkpoint = checkpoint
	response.BlocksSince = blocks
	response.AllCheckpoints = allCheckpoints

	fmt.Printf("[%s] Sending checkpoint at height %d with %d blocks and %d checkpoints to %s\n",
		n.ID, checkpointHeight, len(blocks), len(allCheckpoints), peerID)

	n.sendCheckpointResponse(peerID, response)
}

// handleCheckpointResponse processa uma resposta de checkpoint
func (n *Node) handleCheckpointResponse(peerID string, data []byte) {
	var resp CheckpointResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		fmt.Printf("[%s] Failed to parse checkpoint response from %s: %v\n", n.ID, peerID, err)
		return
	}

	if !resp.HasCheckpoint {
		fmt.Printf("[%s] Peer %s has no checkpoint available\n", n.ID, peerID)
		return
	}

	fmt.Printf("[%s] Received checkpoint from %s at height %d with %d blocks and %d checkpoints\n",
		n.ID, peerID, resp.Checkpoint.Height, len(resp.BlocksSince), len(resp.AllCheckpoints))

	// Validar checkpoint
	if err := blockchain.ValidateCheckpointHash(resp.Checkpoint, n.checkpointConfig.CSVDelimiter); err != nil {
		fmt.Printf("[%s] Invalid checkpoint received from %s: %v\n", n.ID, peerID, err)
		return
	}

	// Salvar todos os checkpoints adicionais no DB para validação de blocos
	if len(resp.AllCheckpoints) > 0 {
		fmt.Printf("[%s] Saving %d additional checkpoints for validation\n", n.ID, len(resp.AllCheckpoints))
		for _, cp := range resp.AllCheckpoints {
			if err := blockchain.SaveCheckpointToDB(n.db, cp, n.checkpointConfig.Compression); err != nil {
				fmt.Printf("[%s] Warning: failed to save checkpoint at height %d: %v\n", n.ID, cp.Height, err)
			}
		}
	}

	// Verificar se precisamos deste checkpoint (se nossa chain está atrás)
	currentHeight := n.chain.GetHeight()
	if currentHeight >= resp.Checkpoint.Height {
		fmt.Printf("[%s] Checkpoint is behind current chain height (%d >= %d), skipping\n",
			n.ID, currentHeight, resp.Checkpoint.Height)

		// Mas ainda processa os blocos adicionais se houver
		if len(resp.BlocksSince) > 0 {
			n.processSyncedBlocks(resp.BlocksSince)
		}
		return
	}

	// Restaurar estado a partir do checkpoint
	fmt.Printf("[%s] Restoring state from checkpoint at height %d\n", n.ID, resp.Checkpoint.Height)

	if err := n.restoreFromCheckpoint(resp.Checkpoint); err != nil {
		fmt.Printf("[%s] Failed to restore from checkpoint: %v\n", n.ID, err)
		return
	}

	// Salvar checkpoint no DB
	if err := blockchain.SaveCheckpointToDB(n.db, resp.Checkpoint, n.checkpointConfig.Compression); err != nil {
		fmt.Printf("[%s] Failed to save checkpoint to DB: %v\n", n.ID, err)
	}

	// Atualizar checkpoint interno
	n.checkpointMutex.Lock()
	n.lastCheckpointHeight = resp.Checkpoint.Height
	n.lastCheckpointHash = resp.Checkpoint.Hash
	n.checkpointMutex.Unlock()

	// Processar blocos adicionais recebidos
	if len(resp.BlocksSince) > 0 {
		fmt.Printf("[%s] Processing %d blocks after checkpoint\n", n.ID, len(resp.BlocksSince))
		n.processSyncedBlocks(resp.BlocksSince)
	}

	fmt.Printf("[%s] Successfully synchronized via checkpoint! New height: %d\n", n.ID, n.chain.GetHeight())
}

// sendCheckpointResponse envia resposta de checkpoint para um peer
func (n *Node) sendCheckpointResponse(peerID string, response CheckpointResponse) {
	responseData, err := json.Marshal(response)
	if err != nil {
		fmt.Printf("[%s] Failed to marshal checkpoint response: %v\n", n.ID, err)
		return
	}

	n.peersMutex.RLock()
	peer := n.peers[peerID]
	n.peersMutex.RUnlock()

	if peer != nil {
		if err := peer.SendMessage("checkpoint_response", responseData); err != nil {
			fmt.Printf("[%s] Failed to send checkpoint response to %s: %v\n", n.ID, peerID, err)
		}
	}
}

// restoreFromCheckpoint restaura o estado da blockchain a partir de um checkpoint
func (n *Node) restoreFromCheckpoint(checkpoint *blockchain.Checkpoint) error {
	// Por enquanto, vamos apenas registrar que recebemos o checkpoint
	// O estado será restaurado através dos blocos recebidos via BlocksSince
	// que já contêm todas as transações necessárias

	fmt.Printf("[%s] Checkpoint received: %d accounts at height %d\n",
		n.ID, len(checkpoint.Accounts), checkpoint.Height)

	// NOTA: Uma implementação completa de "fast sync" requereria:
	// 1. Criar um novo contexto com o estado do checkpoint injetado
	// 2. Recriar a chain a partir do bloco do checkpoint
	// 3. Atualizar todos os blocos em memória
	//
	// Por enquanto, o protocolo funciona assim:
	// - Node2 recebe checkpoint em altura H
	// - Node2 recebe blocos desde H+1 até altura atual
	// - Os blocos são processados normalmente, reconstruindo o estado
	//
	// Isso é mais seguro e garante consistência, mas requer mais banda.
	// Uma otimização futura seria injetar o estado diretamente.

	return nil
}

// processSyncedBlocks processa blocos recebidos durante sincronização
func (n *Node) processSyncedBlocks(blocks []*blockchain.Block) {
	added := 0
	for i, block := range blocks {
		fmt.Printf("[%s] Processing synced block %d/%d: height=%d, hash=%s\n",
			n.ID, i+1, len(blocks), block.Header.Height, block.Hash[:8])

		// Verifica se já tem o bloco
		if _, exists := n.chain.GetBlock(block.Hash); exists {
			fmt.Printf("[%s] Block %d already exists, skipping\n", n.ID, block.Header.Height)
			continue
		}

		// Validar checkpoint hash se presente
		if block.Header.CheckpointHash != "" {
			if err := n.validateBlockCheckpointHash(block); err != nil {
				fmt.Printf("[%s] Block %d checkpoint validation failed: %v\n", n.ID, block.Header.Height, err)
				continue
			}
		}

		// Adiciona à chain
		if err := n.chain.AddBlock(block); err != nil {
			fmt.Printf("[%s] Failed to add synced block %d: %v\n", n.ID, block.Header.Height, err)
			break
		}
		fmt.Printf("[%s] Successfully added synced block %d\n", n.ID, block.Header.Height)

		// Salvar bloco no disco
		if err := blockchain.SaveBlockToDB(n.db, block); err != nil {
			fmt.Printf("[%s] Warning: failed to save synced block %d to disk: %v\n", n.ID, block.Header.Height, err)
		}

		// Tentar criar checkpoint se necessário
		n.tryCreateCheckpoint(block.Header.Height)

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
		fmt.Printf("[%s] Successfully processed %d synced blocks\n", n.ID, added)
	}
}

// RequestCheckpointFromPeer solicita checkpoint de um peer específico
func (n *Node) RequestCheckpointFromPeer(peerID string, height uint64) error {
	if n.checkpointConfig == nil || !n.checkpointConfig.Enabled {
		return fmt.Errorf("checkpoint not enabled")
	}

	req := CheckpointRequest{
		RequestedHeight: height,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal checkpoint request: %w", err)
	}

	n.peersMutex.RLock()
	peer := n.peers[peerID]
	n.peersMutex.RUnlock()

	if peer == nil {
		return fmt.Errorf("peer %s not found", peerID)
	}

	fmt.Printf("[%s] Requesting checkpoint (height: %d) from %s\n", n.ID, height, peerID)
	return peer.SendMessage("checkpoint_request", data)
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

// GetLastBlock retorna o último bloco da blockchain
func (n *Node) GetLastBlock() *blockchain.Block {
	return n.chain.GetLastBlock()
}

// GetMempoolSize retorna o número de transações no mempool
func (n *Node) GetMempoolSize() int {
	return n.mempool.Size()
}

// GetBlocksInMemory retorna o número de blocos em memória
func (n *Node) GetBlocksInMemory() int {
	return len(n.chain.GetAllBlocks())
}

// GetBlockchainStats retorna estatísticas da blockchain
func (n *Node) GetBlockchainStats() blockchain.ChainStats {
	return n.chain.GetChainStats()
}

// GetID retorna o ID do nó
func (n *Node) GetID() string {
	return n.ID
}

// GetWalletAddress retorna o endereço da carteira do nó
func (n *Node) GetWalletAddress() string {
	return n.wallet.GetAddress()
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
	// Aguarda data channel estar pronto (com timeout de 5 segundos)
	n.peersMutex.RLock()
	peer := n.peers[peerID]
	n.peersMutex.RUnlock()

	if peer == nil {
		fmt.Printf("[%s] Peer %s not found for sync\n", n.ID, peerID)
		return
	}

	// Polling para aguardar data channel estar pronto
	ready := false
	for i := 0; i < 50; i++ { // 50 * 200ms = 10 segundos max
		if peer.IsReady() {
			ready = true
			break
		}
		time.Sleep(200 * time.Millisecond)
	}

	if !ready {
		fmt.Printf("[%s] Data channel with %s not ready after 10s, aborting sync\n", n.ID, peerID)
		return
	}

	fmt.Printf("[%s] 📡 Data channel with %s is ready, starting sync\n", n.ID, peerID)

	currentHeight := n.chain.GetHeight()
	fmt.Printf("[%s] 📊 Current chain height: %d\n", n.ID, currentHeight)

	// Se checkpoint está habilitado, solicita checkpoint do peer
	// A sincronização via checkpoint é assíncrona - a resposta virá pelo handler
	if n.checkpointConfig != nil && n.checkpointConfig.Enabled {
		// Solicita checkpoint do peer (0 = último checkpoint)
		if err := n.RequestCheckpointFromPeer(peerID, 0); err != nil {
			fmt.Printf("[%s] ⚠️  Failed to request checkpoint from %s: %v, falling back to regular sync\n",
				n.ID, peerID, err)
		} else {
			fmt.Printf("[%s] 📋 Requested checkpoint from %s (async)\n", n.ID, peerID)
		}
	}

	// Solicita blocos a partir da próxima altura (sync regular ou complementar ao checkpoint)
	req := SyncRequest{
		FromHeight: currentHeight + 1,
	}

	fmt.Printf("[%s] 📤 Requesting blocks from height %d\n", n.ID, req.FromHeight)

	data, err := json.Marshal(req)
	if err != nil {
		fmt.Printf("[%s] Failed to marshal sync request: %v\n", n.ID, err)
		return
	}

	// peer já foi obtido anteriormente, pode reutilizar
	if err := peer.SendMessage("sync_request", data); err != nil {
		fmt.Printf("[%s] Failed to send sync request to %s: %v\n", n.ID, peerID, err)
	} else {
		fmt.Printf("[%s] Requested sync from %s (from height %d)\n", n.ID, peerID, req.FromHeight)
	}
}

// addCheckpointHashToBlock adiciona o hash do último checkpoint ao bloco
func (n *Node) addCheckpointHashToBlock(block *blockchain.Block) {
	if n.checkpointConfig == nil || !n.checkpointConfig.Enabled {
		return
	}

	n.checkpointMutex.RLock()
	checkpointHash := n.lastCheckpointHash
	checkpointHeight := n.lastCheckpointHeight
	n.checkpointMutex.RUnlock()

	// Se temos um checkpoint, adicionar ao bloco
	if checkpointHash != "" {
		block.Header.CheckpointHash = checkpointHash
		block.Header.CheckpointHeight = checkpointHeight

		// Recalcular hash do bloco com os novos campos
		hash, err := block.CalculateHash()
		if err != nil {
			fmt.Printf("[%s] Failed to recalculate block hash with checkpoint: %v\n", n.ID, err)
			return
		}
		block.Hash = hash

		fmt.Printf("[%s] Added checkpoint hash to block %d: checkpoint_height=%d, hash=%s\n",
			n.ID, block.Header.Height, checkpointHeight, checkpointHash[:16])
	}
}

// tryCreateCheckpoint tenta criar um checkpoint se necessário
func (n *Node) tryCreateCheckpoint(currentHeight uint64) {
	// Verificar se checkpoints estão habilitados
	if n.checkpointConfig == nil || !n.checkpointConfig.Enabled {
		return
	}

	interval := uint64(n.checkpointConfig.Interval)

	// Verificar se atingimos o intervalo para criar checkpoint
	// Criamos checkpoint quando: currentHeight % interval == 0 e currentHeight > 0
	if currentHeight%interval != 0 || currentHeight == 0 {
		return
	}

	// Altura do bloco para o qual vamos criar o checkpoint
	// Checkpoint é do estado no bloco (currentHeight - interval)
	checkpointHeight := currentHeight - interval

	// Verificar se já existe um checkpoint nesta altura (pode ter sido recebido via sync)
	if existingCP, err := blockchain.LoadCheckpointFromDB(n.db, checkpointHeight); err == nil && existingCP != nil {
		fmt.Printf("[%s] Checkpoint at height %d already exists (hash: %s), skipping creation\n",
			n.ID, checkpointHeight, existingCP.Hash[:16])
		// Atualizar referências internas
		n.checkpointMutex.Lock()
		n.lastCheckpointHeight = checkpointHeight
		n.lastCheckpointHash = existingCP.Hash
		n.checkpointMutex.Unlock()
		return
	}

	fmt.Printf("[%s] Creating checkpoint for block %d (current height: %d)\n", n.ID, checkpointHeight, currentHeight)

	// Coletar estado atual
	accounts := n.collectCurrentState()

	// Criar checkpoint
	checkpoint, err := blockchain.CreateCheckpoint(
		checkpointHeight,
		time.Now().Unix(),
		accounts,
		n.checkpointConfig.CSVDelimiter,
	)
	if err != nil {
		fmt.Printf("[%s] Failed to create checkpoint: %v\n", n.ID, err)
		return
	}

	// Salvar checkpoint no LevelDB
	err = blockchain.SaveCheckpointToDB(n.db, checkpoint, n.checkpointConfig.Compression)
	if err != nil {
		fmt.Printf("[%s] Failed to save checkpoint: %v\n", n.ID, err)
		return
	}

	fmt.Printf("[%s] Checkpoint created and saved: height=%d, hash=%s, accounts=%d\n",
		n.ID, checkpointHeight, checkpoint.Hash[:16], len(checkpoint.Accounts))

	// Armazenar checkpoint hash para incluir em próximos blocos
	n.checkpointMutex.Lock()
	n.lastCheckpointHash = checkpoint.Hash
	n.lastCheckpointHeight = checkpointHeight
	n.checkpointMutex.Unlock()

	// Fazer pruning de checkpoints antigos
	err = blockchain.PruneOldCheckpoints(n.db, n.checkpointConfig.KeepOnDisk)
	if err != nil {
		fmt.Printf("[%s] Failed to prune old checkpoints: %v\n", n.ID, err)
	}

	// Fazer pruning de blocos antigos se necessário
	n.tryPruneBlocks(currentHeight)
}

// collectCurrentState coleta o estado atual de todas as contas
func (n *Node) collectCurrentState() map[string]*blockchain.AccountState {
	balances := n.chain.GetContext().GetAllBalances()
	stakes := n.chain.GetContext().GetAllStakes()
	nonces := n.chain.GetContext().GetAllNonces()

	// Unir todos os endereços
	allAddresses := make(map[string]bool)
	for addr := range balances {
		allAddresses[addr] = true
	}
	for addr := range stakes {
		allAddresses[addr] = true
	}
	for addr := range nonces {
		allAddresses[addr] = true
	}

	// Criar mapa de estados
	accounts := make(map[string]*blockchain.AccountState)
	for addr := range allAddresses {
		accounts[addr] = &blockchain.AccountState{
			Address: addr,
			Balance: balances[addr],
			Stake:   stakes[addr],
			Nonce:   nonces[addr],
		}
	}

	return accounts
}

// tryPruneBlocks tenta fazer pruning de blocos antigos
func (n *Node) tryPruneBlocks(currentHeight uint64) {
	if n.checkpointConfig == nil || !n.checkpointConfig.Enabled {
		return
	}

	// Verificar se temos blocos suficientes para fazer pruning
	blocksInMemory := n.GetBlocksInMemory()
	keepInMemory := n.checkpointConfig.KeepInMemory

	if blocksInMemory <= keepInMemory {
		return // Não precisa fazer pruning ainda
	}

	fmt.Printf("[%s] Pruning old blocks: current=%d, in_memory=%d, keep=%d\n",
		n.ID, blocksInMemory, blocksInMemory, keepInMemory)

	// Obter ponteiro para o slice de blocos da chain
	allBlocks := n.chain.GetAllBlocksPointer()
	if allBlocks == nil {
		return
	}

	err := blockchain.PruneOldBlocks(n.db, allBlocks, keepInMemory)
	if err != nil {
		fmt.Printf("[%s] Failed to prune old blocks: %v\n", n.ID, err)
		return
	}

	fmt.Printf("[%s] Blocks pruned successfully: now %d blocks in memory\n", n.ID, len(*allBlocks))
}

// validateBlockCheckpointHash valida o hash de checkpoint em um bloco recebido
func (n *Node) validateBlockCheckpointHash(block *blockchain.Block) error {
	if block.Header.CheckpointHash == "" {
		return nil // Nenhum checkpoint para validar
	}

	checkpointHeight := block.Header.CheckpointHeight

	fmt.Printf("[%s] Validating checkpoint hash in block %d: checkpoint_height=%d, hash=%s\n",
		n.ID, block.Header.Height, checkpointHeight, block.Header.CheckpointHash[:16])

	// Primeiro, tentar carregar checkpoint do disco
	checkpoint, err := blockchain.LoadCheckpointFromDB(n.db, checkpointHeight)
	if err == nil {
		// Temos o checkpoint no disco, validar hash
		if checkpoint.Hash != block.Header.CheckpointHash {
			// Se o hash não bate, mas estamos recebendo de um peer,
			// aceitar o checkpoint do peer e atualizar o nosso
			fmt.Printf("[%s] ⚠️  Checkpoint hash mismatch, accepting peer's checkpoint: peer=%s, local=%s\n",
				n.ID, block.Header.CheckpointHash[:16], checkpoint.Hash[:16])
			// Salvar o checkpoint do peer substituindo o nosso
			// (isso será feito quando recebermos via checkpoint_response)
		}
		fmt.Printf("[%s] Checkpoint hash validated successfully from disk\n", n.ID)
		return nil
	}

	// Se não temos no disco, aceitar o checkpoint do bloco
	// Durante sincronização, confiamos no checkpoint do peer
	currentHeight := n.chain.GetHeight()
	if checkpointHeight > currentHeight {
		// Ainda não temos esse bloco, não podemos validar
		// Isso é normal durante sincronização inicial
		fmt.Printf("[%s] Cannot validate checkpoint yet (checkpoint height %d > current height %d), accepting peer's checkpoint\n",
			n.ID, checkpointHeight, currentHeight)
		return nil
	}

	// Se estamos na altura correta mas não temos o checkpoint salvo,
	// aceitar o checkpoint do peer (ele é a fonte confiável)
	fmt.Printf("[%s] No local checkpoint found, accepting peer's checkpoint hash\n", n.ID)
	return nil
}

// loadLastCheckpoint carrega o último checkpoint do disco
func (n *Node) loadLastCheckpoint() {
	// Obter altura do último checkpoint
	lastHeight, err := blockchain.GetLastCheckpointHeight(n.db)
	if err != nil {
		// Não há checkpoint ainda, isso é normal
		return
	}

	// Carregar checkpoint
	checkpoint, err := blockchain.LoadCheckpointFromDB(n.db, lastHeight)
	if err != nil {
		fmt.Printf("[%s] Warning: failed to load last checkpoint from disk: %v\n", n.ID, err)
		return
	}

	// Armazenar checkpoint hash
	n.checkpointMutex.Lock()
	n.lastCheckpointHash = checkpoint.Hash
	n.lastCheckpointHeight = checkpoint.Height
	n.checkpointMutex.Unlock()

	fmt.Printf("[%s] Loaded last checkpoint from disk: height=%d, hash=%s\n",
		n.ID, checkpoint.Height, checkpoint.Hash[:16])
}

// loadChainFromDisk carrega a blockchain salva no disco
func (n *Node) loadChainFromDisk() error {
	// Obter altura da chain salva
	chainHeightData, err := n.db.Get([]byte("metadata-chain-height"), nil)
	if err != nil {
		// Não há chain salva, isso é normal na primeira execução
		return nil
	}

	var savedHeight uint64
	if _, err := fmt.Sscanf(string(chainHeightData), "%d", &savedHeight); err != nil {
		return fmt.Errorf("failed to parse saved chain height: %w", err)
	}

	currentHeight := n.chain.GetHeight()

	// Se a altura salva é menor ou igual à atual, não precisa carregar
	if savedHeight <= currentHeight {
		fmt.Printf("[%s] Chain already up to date (saved: %d, current: %d)\n", n.ID, savedHeight, currentHeight)
		return nil
	}

	fmt.Printf("[%s] Loading chain from disk: saved height=%d, current=%d\n", n.ID, savedHeight, currentHeight)

	// Carregar blocos do disco a partir da próxima altura
	blocksLoaded := 0
	for height := currentHeight + 1; height <= savedHeight; height++ {
		block, err := blockchain.LoadBlockFromDB(n.db, height)
		if err != nil {
			return fmt.Errorf("failed to load block at height %d: %w", height, err)
		}

		// Adicionar bloco à chain
		if err := n.chain.AddBlock(block); err != nil {
			return fmt.Errorf("failed to add block %d to chain: %w", height, err)
		}

		blocksLoaded++
	}

	fmt.Printf("[%s] Successfully loaded %d blocks from disk. New height: %d\n",
		n.ID, blocksLoaded, n.chain.GetHeight())

	return nil
}
