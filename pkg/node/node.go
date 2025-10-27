package node

import (
	"context"
	"fmt"
	"sync"

	"github.com/krakovia/blockchain/pkg/network"
	"github.com/syndtr/goleveldb/leveldb"
)

// Node representa um nó na blockchain
type Node struct {
	ID          string
	Address     string
	db          *leveldb.DB
	webRTC      *network.WebRTCClient
	peers       map[string]*network.Peer
	peersMutex  sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
}

// Config contém as configurações para criar um nó
type Config struct {
	ID              string
	Address         string
	DBPath          string
	SignalingServer string
}

// NewNode cria uma nova instância de nó
func NewNode(config Config) (*Node, error) {
	// Abrir banco de dados LevelDB
	db, err := leveldb.OpenFile(config.DBPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	node := &Node{
		ID:      config.ID,
		Address: config.Address,
		db:      db,
		peers:   make(map[string]*network.Peer),
		ctx:     ctx,
		cancel:  cancel,
	}

	// Inicializar cliente WebRTC
	webRTCClient, err := network.NewWebRTCClient(config.ID, config.SignalingServer, node)
	if err != nil {
		db.Close()
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

	return nil
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
	fmt.Printf("Peer %s connected to node %s\n", peer.ID, n.ID)
}

// RemovePeer remove um peer da lista
func (n *Node) RemovePeer(peerID string) {
	n.peersMutex.Lock()
	defer n.peersMutex.Unlock()
	delete(n.peers, peerID)
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
