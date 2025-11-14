package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Config configuração da API HTTP
type Config struct {
	Enabled  bool
	Address  string
	Username string
	Password string
}

// Server servidor HTTP da API
type Server struct {
	config *Config
	node   NodeInterface
	server *http.Server
}

// NodeInterface interface que o node deve implementar
type NodeInterface interface {
	GetID() string
	GetWalletAddress() string
	GetChainHeight() uint64
	GetBalance() uint64
	GetStake() uint64
	GetNonce() uint64
	GetMempoolSize() int
	GetPeers() []PeerInfo
	GetLastBlock() BlockInfo
	IsMining() bool
	StartMining() error
	StopMining()
	CreateTransaction(to string, amount, fee uint64, data string) (TxInfo, error)
	CreateStakeTransaction(amount, fee uint64) (TxInfo, error)
	CreateUnstakeTransaction(amount, fee uint64) (TxInfo, error)
}

// PeerInfo informações de um peer
type PeerInfo interface {
	GetID() string
}

// BlockInfo informações de um bloco
type BlockInfo interface {
	GetHeight() uint64
	GetHash() string
	GetTimestamp() int64
	GetTransactionCount() int
}

// TxInfo informações de uma transação
type TxInfo interface {
	GetID() string
}

// NewServer cria um novo servidor API
func NewServer(node NodeInterface, config *Config) *Server {
	return &Server{
		config: config,
		node:   node,
	}
}

// Start inicia o servidor HTTP
func (s *Server) Start() error {
	if !s.config.Enabled {
		return nil
	}

	mux := http.NewServeMux()

	// UI
	mux.HandleFunc("/", s.handleUI)

	// API endpoints
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/wallet", s.handleWallet)
	mux.HandleFunc("/api/peers", s.handlePeers)
	mux.HandleFunc("/api/lastblock", s.handleLastBlock)
	mux.HandleFunc("/api/mining/start", s.handleStartMining)
	mux.HandleFunc("/api/mining/stop", s.handleStopMining)
	mux.HandleFunc("/api/transaction/send", s.handleSendTransaction)
	mux.HandleFunc("/api/transaction/stake", s.handleStakeTransaction)
	mux.HandleFunc("/api/transaction/unstake", s.handleUnstakeTransaction)

	s.server = &http.Server{
		Addr:    s.config.Address,
		Handler: s.authMiddleware(mux),
	}

	go func() {
		fmt.Printf("Starting API server on %s\n", s.config.Address)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("API server error: %v\n", err)
		}
	}()

	return nil
}

// Stop para o servidor HTTP
func (s *Server) Stop() error {
	if s.server != nil {
		return s.server.Close()
	}
	return nil
}

// authMiddleware middleware de autenticação básica
func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Permitir acesso à UI sem autenticação para facilitar desenvolvimento
		if r.URL.Path == "/" {
			next.ServeHTTP(w, r)
			return
		}

		// Verificar autenticação básica nas rotas /api
		if s.config.Username != "" && s.config.Password != "" {
			username, password, ok := r.BasicAuth()
			if !ok || username != s.config.Username || password != s.config.Password {
				w.Header().Set("WWW-Authenticate", `Basic realm="Krakovia Node API"`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

// handleUI serve a interface HTML
func (s *Server) handleUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(htmlUI))
}

// handleStatus retorna status do node
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"node_id":       s.node.GetID(),
		"chain_height":  s.node.GetChainHeight(),
		"balance":       s.node.GetBalance(),
		"stake":         s.node.GetStake(),
		"nonce":         s.node.GetNonce(),
		"mempool_size":  s.node.GetMempoolSize(),
		"peer_count":    len(s.node.GetPeers()),
		"mining":        s.node.IsMining(),
		"timestamp":     time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(status)
}

// handleWallet retorna informações da wallet
func (s *Server) handleWallet(w http.ResponseWriter, r *http.Request) {
	wallet := map[string]interface{}{
		"address": s.node.GetWalletAddress(),
		"balance": s.node.GetBalance(),
		"stake":   s.node.GetStake(),
		"nonce":   s.node.GetNonce(),
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(wallet)
}

// handlePeers retorna lista de peers
func (s *Server) handlePeers(w http.ResponseWriter, r *http.Request) {
	peers := s.node.GetPeers()
	peerList := make([]map[string]string, 0, len(peers))

	for _, peer := range peers {
		peerList = append(peerList, map[string]string{
			"id": peer.GetID(),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"peers": peerList,
		"count": len(peerList),
	})
}

// handleLastBlock retorna último bloco
func (s *Server) handleLastBlock(w http.ResponseWriter, r *http.Request) {
	block := s.node.GetLastBlock()

	blockData := map[string]interface{}{
		"height":     block.GetHeight(),
		"hash":       block.GetHash(),
		"timestamp":  block.GetTimestamp(),
		"tx_count":   block.GetTransactionCount(),
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(blockData)
}

// handleStartMining inicia mineração
func (s *Server) handleStartMining(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := s.node.StartMining(); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status": "mining started",
	})
}

// handleStopMining para mineração
func (s *Server) handleStopMining(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.node.StopMining()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status": "mining stopped",
	})
}

// handleSendTransaction cria uma transação
func (s *Server) handleSendTransaction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		To     string `json:"to"`
		Amount uint64 `json:"amount"`
		Fee    uint64 `json:"fee"`
		Data   string `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	tx, err := s.node.CreateTransaction(req.To, req.Amount, req.Fee, req.Data)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status": "transaction created",
		"tx_id":  tx.GetID(),
	})
}

// handleStakeTransaction cria uma transação de stake
func (s *Server) handleStakeTransaction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Amount uint64 `json:"amount"`
		Fee    uint64 `json:"fee"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	tx, err := s.node.CreateStakeTransaction(req.Amount, req.Fee)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status": "stake transaction created",
		"tx_id":  tx.GetID(),
	})
}

// handleUnstakeTransaction cria uma transação de unstake
func (s *Server) handleUnstakeTransaction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Amount uint64 `json:"amount"`
		Fee    uint64 `json:"fee"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	tx, err := s.node.CreateUnstakeTransaction(req.Amount, req.Fee)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status": "unstake transaction created",
		"tx_id":  tx.GetID(),
	})
}
