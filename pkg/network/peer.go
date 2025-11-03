package network

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/pion/webrtc/v3"
)

// Peer representa uma conexão peer-to-peer
type Peer struct {
	ID              string
	Connection      *webrtc.PeerConnection
	DataChannel     *webrtc.DataChannel
	dataChannelMux  sync.RWMutex
	dataChannelReady bool
	OnMessage       func(msgType string, data []byte)
	OnDisconnect    func(peerID string)
}

// Message representa uma mensagem entre peers
type Message struct {
	Type string `json:"type"`
	Data []byte `json:"data"`
}

// NewPeer cria um novo peer
func NewPeer(id string, connection *webrtc.PeerConnection) *Peer {
	return &Peer{
		ID:         id,
		Connection: connection,
	}
}

// SetDataChannel define o data channel e configura handlers
func (p *Peer) SetDataChannel(dc *webrtc.DataChannel) {
	p.dataChannelMux.Lock()
	p.DataChannel = dc
	p.dataChannelMux.Unlock()

	// Handler quando o canal abre
	dc.OnOpen(func() {
		fmt.Printf("Data channel '%s' open with peer %s\n", dc.Label(), p.ID)
		p.dataChannelMux.Lock()
		p.dataChannelReady = true
		p.dataChannelMux.Unlock()
	})

	// Handler para mensagens recebidas
	dc.OnMessage(func(msg webrtc.DataChannelMessage) {
		var message Message
		if err := json.Unmarshal(msg.Data, &message); err != nil {
			fmt.Printf("Failed to unmarshal message from peer %s: %v\n", p.ID, err)
			return
		}

		if p.OnMessage != nil {
			p.OnMessage(message.Type, message.Data)
		}
	})

	// Handler quando o canal fecha
	dc.OnClose(func() {
		fmt.Printf("Data channel closed with peer %s\n", p.ID)
		p.dataChannelMux.Lock()
		p.dataChannelReady = false
		p.dataChannelMux.Unlock()
		if p.OnDisconnect != nil {
			p.OnDisconnect(p.ID)
		}
	})
}

// IsReady retorna se o data channel está pronto para enviar mensagens
func (p *Peer) IsReady() bool {
	p.dataChannelMux.RLock()
	defer p.dataChannelMux.RUnlock()
	return p.dataChannelReady && p.DataChannel != nil
}

// SendMessage envia uma mensagem para o peer
func (p *Peer) SendMessage(msgType string, data []byte) error {
	p.dataChannelMux.RLock()
	dc := p.DataChannel
	ready := p.dataChannelReady
	p.dataChannelMux.RUnlock()

	if dc == nil || !ready {
		return fmt.Errorf("data channel not established")
	}

	message := Message{
		Type: msgType,
		Data: data,
	}

	messageBytes, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	return dc.Send(messageBytes)
}

// Close fecha a conexão com o peer
func (p *Peer) Close() error {
	p.dataChannelMux.Lock()
	defer p.dataChannelMux.Unlock()

	if p.DataChannel != nil {
		if err := p.DataChannel.Close(); err != nil {
			return err
		}
	}

	if p.Connection != nil {
		if err := p.Connection.Close(); err != nil {
			return err
		}
	}

	return nil
}
