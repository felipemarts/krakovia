package node

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/krakovia/blockchain/pkg/network"
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
}

// NewNode cria uma nova instância de nó
func NewNode(config Config) (*Node, error) {
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

	ctx, cancel := context.WithCancel(context.Background())

	// Criar sistema de descoberta de peers
	discovery := network.NewPeerDiscovery(config.ID, config.MaxPeers, config.MinPeers)

	node := &Node{
		ID:                config.ID,
		Address:           config.Address,
		db:                db,
		peers:             make(map[string]*network.Peer),
		discovery:         discovery,
		ctx:               ctx,
		cancel:            cancel,
		discoveryInterval: time.Duration(config.DiscoveryInterval) * time.Second,
	}

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
	fmt.Printf("Peer %s connected to node %s\n", peer.ID, n.ID)
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
