package signaling

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Client representa um cliente conectado ao servidor de signaling
type Client struct {
	ID       string
	Conn     *websocket.Conn
	Send     chan []byte
	connMux  sync.Mutex
}

// Server é o servidor de signaling WebSocket
type Server struct {
	clients      map[string]*Client
	clientsMutex sync.RWMutex
	register     chan *Client
	unregister   chan *Client
	broadcast    chan []byte
}

// Message representa uma mensagem de signaling
type Message struct {
	Type     string                     `json:"type"`
	From     string                     `json:"from"`
	To       string                     `json:"to"`
	SDP      *webrtc.SessionDescription `json:"sdp,omitempty"`
	ICE      *webrtc.ICECandidateInit   `json:"ice,omitempty"`
	PeerList []string                   `json:"peerList,omitempty"`
}

// NewServer cria um novo servidor de signaling
func NewServer() *Server {
	return &Server{
		clients:    make(map[string]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte),
	}
}

// Run inicia o servidor de signaling
func (s *Server) Run() {
	for {
		select {
		case client := <-s.register:
			s.clientsMutex.Lock()
			s.clients[client.ID] = client
			s.clientsMutex.Unlock()

			fmt.Printf("Client %s registered\n", client.ID)

			// Enviar lista de peers existentes para o novo cliente
			s.sendPeerList(client)

			// Notificar outros clientes sobre o novo peer
			s.notifyNewPeer(client.ID)

		case client := <-s.unregister:
			s.clientsMutex.Lock()
			if _, ok := s.clients[client.ID]; ok {
				delete(s.clients, client.ID)
				close(client.Send)
				fmt.Printf("Client %s unregistered\n", client.ID)
			}
			s.clientsMutex.Unlock()
		}
	}
}

// sendPeerList envia a lista de peers conectados para um cliente
func (s *Server) sendPeerList(client *Client) {
	s.clientsMutex.RLock()
	defer s.clientsMutex.RUnlock()

	peerList := make([]string, 0)
	for id := range s.clients {
		if id != client.ID {
			peerList = append(peerList, id)
		}
	}

	fmt.Printf("Sending peer list to %s: %v\n", client.ID, peerList)

	msg := Message{
		Type:     "peer-list",
		PeerList: peerList,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error marshaling peer list: %v", err)
		return
	}

	select {
	case client.Send <- data:
		fmt.Printf("Peer list sent to %s\n", client.ID)
	default:
		fmt.Printf("Failed to send peer list to %s (channel blocked)\n", client.ID)
		close(client.Send)
		delete(s.clients, client.ID)
	}
}

// notifyNewPeer notifica todos os clientes sobre um novo peer
func (s *Server) notifyNewPeer(newPeerID string) {
	s.clientsMutex.RLock()
	defer s.clientsMutex.RUnlock()

	msg := Message{
		Type:     "peer-list",
		PeerList: []string{newPeerID},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error marshaling new peer notification: %v", err)
		return
	}

	for id, client := range s.clients {
		if id != newPeerID {
			select {
			case client.Send <- data:
			default:
				close(client.Send)
				delete(s.clients, id)
			}
		}
	}
}

// HandleWebSocket gerencia conexões WebSocket
func (s *Server) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Error upgrading connection: %v", err)
		return
	}

	client := &Client{
		Conn: conn,
		Send: make(chan []byte, 256),
	}

	// Ler goroutine - recebe mensagens do cliente
	go s.readPump(client)

	// Write goroutine - envia mensagens para o cliente
	go s.writePump(client)
}

// readPump lê mensagens do cliente
func (s *Server) readPump(client *Client) {
	defer func() {
		s.unregister <- client
		client.Conn.Close()
	}()

	for {
		var msg Message
		err := client.Conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		switch msg.Type {
		case "register":
			// Registrar cliente
			client.ID = msg.From
			s.register <- client

		case "get-peers":
			// Cliente solicitou lista de peers
			s.sendPeerList(client)

		case "offer", "answer", "ice":
			// Encaminhar mensagem para o destinatário
			s.forwardMessage(msg)
		}
	}
}

// writePump envia mensagens para o cliente
func (s *Server) writePump(client *Client) {
	defer client.Conn.Close()

	for message := range client.Send {
		client.connMux.Lock()
		err := client.Conn.WriteMessage(websocket.TextMessage, message)
		client.connMux.Unlock()

		if err != nil {
			log.Printf("Error writing message: %v", err)
			return
		}
	}
}

// forwardMessage encaminha uma mensagem de um cliente para outro
func (s *Server) forwardMessage(msg Message) {
	s.clientsMutex.RLock()
	defer s.clientsMutex.RUnlock()

	if targetClient, ok := s.clients[msg.To]; ok {
		data, err := json.Marshal(msg)
		if err != nil {
			log.Printf("Error marshaling message: %v", err)
			return
		}

		select {
		case targetClient.Send <- data:
		default:
			close(targetClient.Send)
			delete(s.clients, msg.To)
		}
	}
}

// Start inicia o servidor HTTP
func (s *Server) Start(addr string) error {
	go s.Run()

	// Usar um ServeMux próprio ao invés do global para evitar conflitos em testes
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.HandleWebSocket)

	fmt.Printf("Signaling server started on %s\n", addr)
	return http.ListenAndServe(addr, mux)
}
