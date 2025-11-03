package network

import (
	"fmt"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
)

// PeerHandler define a interface para lidar com eventos de peers
type PeerHandler interface {
	AddPeer(peer *Peer)
	RemovePeer(peerID string)
}

// WebRTCClient gerencia conexões WebRTC
type WebRTCClient struct {
	ID              string
	SignalingServer string
	config          webrtc.Configuration
	peers           map[string]*Peer
	peersMutex      sync.RWMutex
	signalingConn   *websocket.Conn
	signalingMux    sync.Mutex
	handler         PeerHandler
	discovery       *PeerDiscovery
	gossipManager   *GossipManager
}

// SignalingMessage representa uma mensagem do servidor de signaling
type SignalingMessage struct {
	Type     string                     `json:"type"`
	From     string                     `json:"from"`
	To       string                     `json:"to"`
	SDP      *webrtc.SessionDescription `json:"sdp,omitempty"`
	ICE      *webrtc.ICECandidateInit   `json:"ice,omitempty"`
	PeerList []string                   `json:"peerList,omitempty"`
}

// NewWebRTCClient cria um novo cliente WebRTC
func NewWebRTCClient(id, signalingServer string, handler PeerHandler) (*WebRTCClient, error) {
	return NewWebRTCClientWithDiscovery(id, signalingServer, handler, nil)
}

// NewWebRTCClientWithDiscovery cria um novo cliente WebRTC com sistema de descoberta
func NewWebRTCClientWithDiscovery(id, signalingServer string, handler PeerHandler, discovery *PeerDiscovery) (*WebRTCClient, error) {
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}

	// Criar gerenciador gossip
	gossipConfig := DefaultGossipConfig()
	gossipManager := NewGossipManager(id, gossipConfig)

	return &WebRTCClient{
		ID:              id,
		SignalingServer: signalingServer,
		config:          config,
		peers:           make(map[string]*Peer),
		handler:         handler,
		discovery:       discovery,
		gossipManager:   gossipManager,
	}, nil
}

// Connect conecta ao servidor de signaling
func (w *WebRTCClient) Connect() error {
	conn, _, err := websocket.DefaultDialer.Dial(w.SignalingServer, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to signaling server: %w", err)
	}

	w.signalingConn = conn

	// Registrar no servidor de signaling
	registerMsg := SignalingMessage{
		Type: "register",
		From: w.ID,
	}

	if err := conn.WriteJSON(registerMsg); err != nil {
		return fmt.Errorf("failed to register with signaling server: %w", err)
	}

	// Iniciar goroutine para receber mensagens do signaling server
	go w.handleSignalingMessages()

	return nil
}

// handleSignalingMessages processa mensagens do servidor de signaling
func (w *WebRTCClient) handleSignalingMessages() {
	for {
		var msg SignalingMessage
		err := w.signalingConn.ReadJSON(&msg)
		if err != nil {
			fmt.Printf("Error reading signaling message: %v\n", err)
			return
		}

		switch msg.Type {
		case "peer-list":
			fmt.Printf("[%s] Received peer list: %v\n", w.ID, msg.PeerList)
			// Lista de peers disponíveis
			if w.discovery != nil {
				// Usar sistema de descoberta para selecionar peers
				w.peersMutex.RLock()
				currentlyConnected := make(map[string]bool)
				for peerID := range w.peers {
					currentlyConnected[peerID] = true
				}
				w.peersMutex.RUnlock()

				// Adicionar todos os peers à lista de conhecidos
				for _, peerID := range msg.PeerList {
					w.discovery.AddKnownPeer(peerID)
				}

				// Selecionar quais peers conectar
				toConnect := w.discovery.SelectPeersToConnect(msg.PeerList, currentlyConnected)
				fmt.Printf("[%s] Selected peers to connect: %v\n", w.ID, toConnect)
				for _, peerID := range toConnect {
					go func(pid string) {
					if err := w.ConnectToPeer(pid); err != nil {
						fmt.Printf("Failed to connect to peer %s: %v\n", pid, err)
					}
				}(peerID)
				}
			} else {
				// Modo legado: conectar a todos
				for _, peerID := range msg.PeerList {
					if peerID != w.ID {
						go func(pid string) {
					if err := w.ConnectToPeer(pid); err != nil {
						fmt.Printf("Failed to connect to peer %s: %v\n", pid, err)
					}
				}(peerID)
					}
				}
			}

		case "offer":
			// Recebeu uma oferta de conexão - verificar se deve aceitar
			if w.discovery != nil && !w.discovery.ShouldAcceptNewPeer() {
				fmt.Printf("Rejecting offer from %s (peer limit reached)\n", msg.From)
				return
			}
			go w.handleOffer(msg.From, msg.SDP)

		case "answer":
			// Recebeu uma resposta a uma oferta
			go w.handleAnswer(msg.From, msg.SDP)

		case "ice":
			// Recebeu um ICE candidate
			go w.handleICE(msg.From, msg.ICE)
		}
	}
}

// ConnectToPeer inicia uma conexão com outro peer
func (w *WebRTCClient) ConnectToPeer(peerID string) error {
	fmt.Printf("Connecting to peer %s\n", peerID)

	// Criar peer connection
	peerConnection, err := webrtc.NewPeerConnection(w.config)
	if err != nil {
		return fmt.Errorf("failed to create peer connection: %w", err)
	}

	peer := NewPeer(peerID, peerConnection)

	// Criar data channel
	dataChannel, err := peerConnection.CreateDataChannel("data", nil)
	if err != nil {
		return fmt.Errorf("failed to create data channel: %w", err)
	}

	peer.SetDataChannel(dataChannel)
	peer.OnDisconnect = func(id string) {
		w.removePeer(id)
	}
	peer.OnMessage = func(msgType string, data []byte) {
		// Se for mensagem gossip, processar através do gossip manager
		if msgType == "gossip" {
			w.handleGossipMessage(data, peerID)
		}
		// Outros tipos de mensagem podem ser tratados aqui
	}

	// Configurar ICE candidate handler
	peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate != nil {
			w.sendICECandidate(peerID, candidate)
		}
	})

	// Criar oferta
	offer, err := peerConnection.CreateOffer(nil)
	if err != nil {
		return fmt.Errorf("failed to create offer: %w", err)
	}

	if err := peerConnection.SetLocalDescription(offer); err != nil {
		return fmt.Errorf("failed to set local description: %w", err)
	}

	// Enviar oferta via signaling
	w.sendOffer(peerID, &offer)

	w.addPeer(peer)

	return nil
}

// handleOffer processa uma oferta recebida
func (w *WebRTCClient) handleOffer(peerID string, sdp *webrtc.SessionDescription) {
	fmt.Printf("Received offer from peer %s\n", peerID)

	// Verificar se já existe uma conexão com este peer (oferta simultânea)
	w.peersMutex.RLock()
	existingPeer, exists := w.peers[peerID]
	w.peersMutex.RUnlock()

	if exists && existingPeer.Connection != nil {
		// Oferta simultânea detectada. Para evitar duplicação,
		// apenas o nó com ID lexicograficamente menor aceita a offer.
		// O outro deve descartar a offer recebida.
		if w.ID > peerID {
			// Nosso ID é maior, então mantemos nossa conexão e ignoramos a offer dele
			fmt.Printf("Ignoring simultaneous offer from %s (our ID is greater)\n", peerID)
			return
		}
		// Nosso ID é menor, então descartamos nossa conexão e aceitamos a offer dele
		fmt.Printf("Accepting simultaneous offer from %s (our ID is smaller)\n", peerID)
		w.removePeer(peerID) // Remover a conexão que iniciamos
	}

	peerConnection, err := webrtc.NewPeerConnection(w.config)
	if err != nil {
		fmt.Printf("Failed to create peer connection: %v\n", err)
		return
	}

	peer := NewPeer(peerID, peerConnection)
	peer.OnDisconnect = func(id string) {
		w.removePeer(id)
	}
	peer.OnMessage = func(msgType string, data []byte) {
		// Se for mensagem gossip, processar através do gossip manager
		if msgType == "gossip" {
			w.handleGossipMessage(data, peerID)
		}
		// Outros tipos de mensagem podem ser tratados aqui
	}

	// Handler para data channel recebido
	peerConnection.OnDataChannel(func(dc *webrtc.DataChannel) {
		peer.SetDataChannel(dc)
	})

	// Configurar ICE candidate handler
	peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate != nil {
			w.sendICECandidate(peerID, candidate)
		}
	})

	// Definir remote description
	if err := peerConnection.SetRemoteDescription(*sdp); err != nil {
		fmt.Printf("Failed to set remote description: %v\n", err)
		return
	}

	// Criar resposta
	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		fmt.Printf("Failed to create answer: %v\n", err)
		return
	}

	if err := peerConnection.SetLocalDescription(answer); err != nil {
		fmt.Printf("Failed to set local description: %v\n", err)
		return
	}

	// Enviar resposta
	w.sendAnswer(peerID, &answer)

	w.addPeer(peer)
}

// handleAnswer processa uma resposta recebida
func (w *WebRTCClient) handleAnswer(peerID string, sdp *webrtc.SessionDescription) {
	fmt.Printf("Received answer from peer %s\n", peerID)

	w.peersMutex.RLock()
	peer, exists := w.peers[peerID]
	w.peersMutex.RUnlock()

	if !exists {
		fmt.Printf("Peer %s not found\n", peerID)
		return
	}

	if err := peer.Connection.SetRemoteDescription(*sdp); err != nil {
		fmt.Printf("Failed to set remote description: %v\n", err)
		return
	}
}

// handleICE processa um ICE candidate recebido
func (w *WebRTCClient) handleICE(peerID string, ice *webrtc.ICECandidateInit) {
	w.peersMutex.RLock()
	peer, exists := w.peers[peerID]
	w.peersMutex.RUnlock()

	if !exists {
		fmt.Printf("Peer %s not found for ICE candidate\n", peerID)
		return
	}

	if err := peer.Connection.AddICECandidate(*ice); err != nil {
		fmt.Printf("Failed to add ICE candidate: %v\n", err)
		return
	}
}

// sendOffer envia uma oferta via signaling
func (w *WebRTCClient) sendOffer(to string, sdp *webrtc.SessionDescription) {
	msg := SignalingMessage{
		Type: "offer",
		From: w.ID,
		To:   to,
		SDP:  sdp,
	}
	w.signalingMux.Lock()
	if err := w.signalingConn.WriteJSON(msg); err != nil {
		fmt.Printf("Failed to send signaling message: %v\n", err)
	}
	w.signalingMux.Unlock()
}

// sendAnswer envia uma resposta via signaling
func (w *WebRTCClient) sendAnswer(to string, sdp *webrtc.SessionDescription) {
	msg := SignalingMessage{
		Type: "answer",
		From: w.ID,
		To:   to,
		SDP:  sdp,
	}
	w.signalingMux.Lock()
	if err := w.signalingConn.WriteJSON(msg); err != nil {
		fmt.Printf("Failed to send signaling message: %v\n", err)
	}
	w.signalingMux.Unlock()
}

// sendICECandidate envia um ICE candidate via signaling
func (w *WebRTCClient) sendICECandidate(to string, candidate *webrtc.ICECandidate) {
	init := candidate.ToJSON()
	msg := SignalingMessage{
		Type: "ice",
		From: w.ID,
		To:   to,
		ICE:  &init,
	}
	w.signalingMux.Lock()
	if err := w.signalingConn.WriteJSON(msg); err != nil {
		fmt.Printf("Failed to send signaling message: %v\n", err)
	}
	w.signalingMux.Unlock()
}

// addPeer adiciona um peer
func (w *WebRTCClient) addPeer(peer *Peer) {
	w.peersMutex.Lock()
	w.peers[peer.ID] = peer
	w.peersMutex.Unlock()

	if w.handler != nil {
		w.handler.AddPeer(peer)
	}
}

// removePeer remove um peer
func (w *WebRTCClient) removePeer(peerID string) {
	w.peersMutex.Lock()
	delete(w.peers, peerID)
	w.peersMutex.Unlock()

	if w.handler != nil {
		w.handler.RemovePeer(peerID)
	}
}

// RequestPeerList solicita a lista de peers do servidor de signaling
func (w *WebRTCClient) RequestPeerList() {
	msg := SignalingMessage{
		Type: "get-peers",
		From: w.ID,
	}
	w.signalingMux.Lock()
	if err := w.signalingConn.WriteJSON(msg); err != nil {
		fmt.Printf("Failed to send signaling message: %v\n", err)
	}
	w.signalingMux.Unlock()
}

// DisconnectPeer desconecta de um peer específico
func (w *WebRTCClient) DisconnectPeer(peerID string) error {
	w.peersMutex.RLock()
	peer, exists := w.peers[peerID]
	w.peersMutex.RUnlock()

	if !exists {
		return fmt.Errorf("peer %s not found", peerID)
	}

	// Fechar conexão
	if err := peer.Close(); err != nil {
		return fmt.Errorf("failed to close peer connection: %w", err)
	}

	// Remover da lista
	w.removePeer(peerID)

	return nil
}

// Close fecha todas as conexões
func (w *WebRTCClient) Close() {
	// Parar gossip manager
	if w.gossipManager != nil {
		w.gossipManager.Stop()
	}

	w.peersMutex.Lock()
	defer w.peersMutex.Unlock()

	for _, peer := range w.peers {
		if err := peer.Close(); err != nil {
			fmt.Printf("Error closing peer connection: %v\n", err)
		}
	}

	if w.signalingConn != nil {
		if err := w.signalingConn.Close(); err != nil {
			fmt.Printf("Error closing signaling connection: %v\n", err)
		}
	}
}

// SendToPeer envia uma mensagem para um peer específico
func (w *WebRTCClient) SendToPeer(peerID string, msgType string, data []byte) error {
	w.peersMutex.RLock()
	peer, exists := w.peers[peerID]
	w.peersMutex.RUnlock()

	if !exists {
		return fmt.Errorf("peer %s not found", peerID)
	}

	return peer.SendMessage(msgType, data)
}

// Broadcast envia uma mensagem para todos os peers (método legado - deprecado)
// Use GossipBroadcast para comunicação eficiente
func (w *WebRTCClient) Broadcast(msgType string, data []byte) {
	w.peersMutex.RLock()
	defer w.peersMutex.RUnlock()

	for _, peer := range w.peers {
		if err := peer.SendMessage(msgType, data); err != nil {
			fmt.Printf("Failed to send message to peer %s: %v\n", peer.ID, err)
		}
	}
}

// GossipBroadcast envia uma mensagem usando o protocolo gossip
func (w *WebRTCClient) GossipBroadcast(msgType string, data []byte) error {
	// Criar mensagem gossip
	gossipMsg, err := w.gossipManager.CreateMessage(msgType, data)
	if err != nil {
		return fmt.Errorf("failed to create gossip message: %w", err)
	}

	// Obter lista de peers
	w.peersMutex.RLock()
	peerIDs := make([]string, 0, len(w.peers))
	for peerID := range w.peers {
		peerIDs = append(peerIDs, peerID)
	}
	w.peersMutex.RUnlock()

	// Selecionar peers para fanout inicial
	selectedPeers := w.gossipManager.SelectPeersFromList(peerIDs, "")

	// Enviar para peers selecionados
	return w.sendGossipToPeers(gossipMsg, selectedPeers)
}

// handleGossipMessage processa uma mensagem gossip recebida
func (w *WebRTCClient) handleGossipMessage(msgData []byte, fromPeer string) {
	// Processar mensagem através do gossip manager
	gossipMsg, peersToPropagate, err := w.gossipManager.HandleIncomingMessage(msgData, fromPeer)
	if err != nil {
		// Erro silencioso para mensagens duplicadas (esperado no gossip)
		if err.Error() != fmt.Sprintf("duplicate message detected: %s", "") {
			fmt.Printf("Error handling gossip message from %s: %v\n", fromPeer, err)
		}
		return
	}

	// Se deve propagar, enviar para outros peers
	if len(peersToPropagate) > 0 {
		// Obter lista de peers conectados
		w.peersMutex.RLock()
		connectedPeers := make([]string, 0, len(w.peers))
		for peerID := range w.peers {
			connectedPeers = append(connectedPeers, peerID)
		}
		w.peersMutex.RUnlock()

		// Selecionar peers para propagar (excluindo quem enviou)
		peersToSend := w.gossipManager.SelectPeersFromList(connectedPeers, fromPeer)

		// Propagar para peers selecionados
		if err := w.sendGossipToPeers(gossipMsg, peersToSend); err != nil {
			fmt.Printf("Failed to propagate gossip message: %v\n", err)
		}
	}
}

// sendGossipToPeers envia uma mensagem gossip para uma lista de peers
func (w *WebRTCClient) sendGossipToPeers(msg *GossipMessage, peerIDs []string) error {
	// Serializar mensagem
	msgData, err := msg.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize message: %w", err)
	}

	w.peersMutex.RLock()
	defer w.peersMutex.RUnlock()

	var lastErr error
	successCount := 0

	for _, peerID := range peerIDs {
		peer, exists := w.peers[peerID]
		if !exists {
			continue
		}

		// Enviar como tipo "gossip"
		if err := peer.SendMessage("gossip", msgData); err != nil {
			lastErr = err
			fmt.Printf("Failed to send gossip to peer %s: %v\n", peerID, err)
		} else {
			successCount++
		}
	}

	if successCount == 0 && lastErr != nil {
		return fmt.Errorf("failed to send to any peer: %w", lastErr)
	}

	return nil
}

// RegisterGossipHandler registra um handler para mensagens gossip
func (w *WebRTCClient) RegisterGossipHandler(msgType string, handler MessageHandler) {
	w.gossipManager.RegisterHandler(msgType, handler)
}

// GetGossipMetrics retorna métricas do protocolo gossip
func (w *WebRTCClient) GetGossipMetrics() map[string]int64 {
	return w.gossipManager.GetMetrics()
}

// GetGossipStats retorna estatísticas formatadas do gossip
func (w *WebRTCClient) GetGossipStats() string {
	return w.gossipManager.GetStats()
}
