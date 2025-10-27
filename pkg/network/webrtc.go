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
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}

	return &WebRTCClient{
		ID:              id,
		SignalingServer: signalingServer,
		config:          config,
		peers:           make(map[string]*Peer),
		handler:         handler,
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
			// Lista de peers disponíveis - conectar a cada um
			for _, peerID := range msg.PeerList {
				if peerID != w.ID {
					go w.ConnectToPeer(peerID)
				}
			}

		case "offer":
			// Recebeu uma oferta de conexão
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

	peerConnection, err := webrtc.NewPeerConnection(w.config)
	if err != nil {
		fmt.Printf("Failed to create peer connection: %v\n", err)
		return
	}

	peer := NewPeer(peerID, peerConnection)
	peer.OnDisconnect = func(id string) {
		w.removePeer(id)
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
	w.signalingConn.WriteJSON(msg)
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
	w.signalingConn.WriteJSON(msg)
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
	w.signalingConn.WriteJSON(msg)
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

// Close fecha todas as conexões
func (w *WebRTCClient) Close() {
	w.peersMutex.Lock()
	defer w.peersMutex.Unlock()

	for _, peer := range w.peers {
		peer.Close()
	}

	if w.signalingConn != nil {
		w.signalingConn.Close()
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

// Broadcast envia uma mensagem para todos os peers
func (w *WebRTCClient) Broadcast(msgType string, data []byte) {
	w.peersMutex.RLock()
	defer w.peersMutex.RUnlock()

	for _, peer := range w.peers {
		if err := peer.SendMessage(msgType, data); err != nil {
			fmt.Printf("Failed to send message to peer %s: %v\n", peer.ID, err)
		}
	}
}
