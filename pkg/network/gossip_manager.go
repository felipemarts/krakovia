package network

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// GossipManager gerencia o protocolo gossip
type GossipManager struct {
	nodeID      string
	config      *GossipConfig
	cache       *MessageCache
	validator   *MessageValidator
	metrics     *GossipMetrics
	handlers    map[string]MessageHandler
	handlersMu  sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
}

// MessageHandler processa mensagens recebidas
type MessageHandler func(msg *GossipMessage, fromPeer string) error

// GossipMetrics rastreia estatísticas do protocolo gossip
type GossipMetrics struct {
	messagesSent        int64
	messagesReceived    int64
	messagesDuplicate   int64
	messagesInvalid     int64
	messagesPropagated  int64
	bytesTransferred    int64
	mu                  sync.RWMutex
}

// NewGossipManager cria um novo gerenciador gossip
func NewGossipManager(nodeID string, config *GossipConfig) *GossipManager {
	if config == nil {
		config = DefaultGossipConfig()
	}

	cache := NewMessageCache(config.CacheSize)
	ctx, cancel := context.WithCancel(context.Background())

	gm := &GossipManager{
		nodeID:    nodeID,
		config:    config,
		cache:     cache,
		validator: NewMessageValidator(config, cache),
		metrics:   &GossipMetrics{},
		handlers:  make(map[string]MessageHandler),
		ctx:       ctx,
		cancel:    cancel,
	}

	// Iniciar rotinas de limpeza
	go gm.cleanupLoop()

	return gm
}

// RegisterHandler registra um handler para um tipo de mensagem
func (gm *GossipManager) RegisterHandler(msgType string, handler MessageHandler) {
	gm.handlersMu.Lock()
	defer gm.handlersMu.Unlock()
	gm.handlers[msgType] = handler
}

// CreateMessage cria uma nova mensagem gossip
func (gm *GossipManager) CreateMessage(msgType string, data []byte) (*GossipMessage, error) {
	msg, err := NewGossipMessage(gm.nodeID, msgType, data, gm.config.MaxTTL)
	if err != nil {
		return nil, fmt.Errorf("failed to create gossip message: %w", err)
	}

	// Adicionar ao cache imediatamente
	gm.cache.Add(msg)

	gm.metrics.mu.Lock()
	gm.metrics.messagesSent++
	gm.metrics.bytesTransferred += int64(len(data))
	gm.metrics.mu.Unlock()

	return msg, nil
}

// HandleIncomingMessage processa uma mensagem recebida
func (gm *GossipManager) HandleIncomingMessage(msgData []byte, fromPeer string) (*GossipMessage, []string, error) {
	// Deserializar mensagem
	msg, err := FromJSON(msgData)
	if err != nil {
		gm.metrics.mu.Lock()
		gm.metrics.messagesInvalid++
		gm.metrics.mu.Unlock()
		return nil, nil, fmt.Errorf("failed to parse message: %w", err)
	}

	// Validar mensagem
	if err := gm.validator.ValidateMessage(msg, fromPeer); err != nil {
		if err.Error() == fmt.Sprintf("duplicate message detected: %s", msg.ID) {
			gm.metrics.mu.Lock()
			gm.metrics.messagesDuplicate++
			gm.metrics.mu.Unlock()
		} else {
			gm.metrics.mu.Lock()
			gm.metrics.messagesInvalid++
			gm.metrics.mu.Unlock()
		}
		return nil, nil, err
	}

	// Adicionar ao cache
	gm.cache.Add(msg)

	gm.metrics.mu.Lock()
	gm.metrics.messagesReceived++
	gm.metrics.bytesTransferred += int64(len(msg.Data))
	gm.metrics.mu.Unlock()

	// Chamar handler apropriado
	gm.handlersMu.RLock()
	handler, exists := gm.handlers[msg.Type]
	gm.handlersMu.RUnlock()

	if exists {
		if err := handler(msg, fromPeer); err != nil {
			return nil, nil, fmt.Errorf("handler error: %w", err)
		}
	}

	// Determinar para quais peers propagar
	peersToPropagate := gm.selectPeersForPropagation(fromPeer)

	// Se ainda pode propagar, incrementar hop e propagar
	if msg.ShouldPropagate() && len(peersToPropagate) > 0 {
		msg.IncrementHop()

		gm.metrics.mu.Lock()
		gm.metrics.messagesPropagated++
		gm.metrics.mu.Unlock()

		return msg, peersToPropagate, nil
	}

	return msg, nil, nil
}

// selectPeersForPropagation seleciona peers para propagar a mensagem
func (gm *GossipManager) selectPeersForPropagation(excludePeer string) []string {
	// Esta função será chamada com a lista de peers disponíveis
	// Por enquanto retorna vazio, será integrada com WebRTCClient
	return []string{}
}

// SelectPeersFromList seleciona aleatoriamente N peers da lista (fanout)
func (gm *GossipManager) SelectPeersFromList(availablePeers []string, excludePeer string) []string {
	// Filtrar peer de origem
	filtered := make([]string, 0, len(availablePeers))
	for _, peerID := range availablePeers {
		if peerID != excludePeer && !gm.validator.IsPeerBlocked(peerID) {
			filtered = append(filtered, peerID)
		}
	}

	// Se temos menos peers que fanout, retornar todos
	if len(filtered) <= gm.config.Fanout {
		return filtered
	}

	// Selecionar aleatoriamente N peers (fanout)
	selected := make([]string, gm.config.Fanout)
	perm := rand.Perm(len(filtered))
	for i := 0; i < gm.config.Fanout; i++ {
		selected[i] = filtered[perm[i]]
	}

	return selected
}

// cleanupLoop executa limpezas periódicas
func (gm *GossipManager) cleanupLoop() {
	ticker := time.NewTicker(gm.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-gm.ctx.Done():
			return
		case <-ticker.C:
			// Limpar cache de mensagens antigas
			removed := gm.cache.Cleanup(gm.config.CacheDuration)
			if removed > 0 {
				fmt.Printf("🧹 Gossip cleanup: removed %d old messages from cache\n", removed)
			}

			// Limpar peers bloqueados expirados
			gm.validator.CleanupBlockedPeers()

			// Limpar rate limiter
			gm.validator.rateLimiter.Cleanup(10 * time.Minute)
		}
	}
}

// GetMetrics retorna as métricas atuais
func (gm *GossipManager) GetMetrics() map[string]int64 {
	gm.metrics.mu.RLock()
	defer gm.metrics.mu.RUnlock()

	return map[string]int64{
		"messages_sent":       gm.metrics.messagesSent,
		"messages_received":   gm.metrics.messagesReceived,
		"messages_duplicate":  gm.metrics.messagesDuplicate,
		"messages_invalid":    gm.metrics.messagesInvalid,
		"messages_propagated": gm.metrics.messagesPropagated,
		"bytes_transferred":   gm.metrics.bytesTransferred,
		"cache_size":          int64(gm.cache.Size()),
	}
}

// GetStats retorna estatísticas formatadas
func (gm *GossipManager) GetStats() string {
	metrics := gm.GetMetrics()
	blockedPeers := gm.validator.GetBlockedPeers()

	return fmt.Sprintf(`Gossip Statistics:
  Messages Sent: %d
  Messages Received: %d
  Messages Duplicate: %d
  Messages Invalid: %d
  Messages Propagated: %d
  Bytes Transferred: %d
  Cache Size: %d
  Blocked Peers: %d`,
		metrics["messages_sent"],
		metrics["messages_received"],
		metrics["messages_duplicate"],
		metrics["messages_invalid"],
		metrics["messages_propagated"],
		metrics["bytes_transferred"],
		metrics["cache_size"],
		len(blockedPeers),
	)
}

// IsMessageSeen verifica se uma mensagem já foi vista
func (gm *GossipManager) IsMessageSeen(messageID string) bool {
	return gm.cache.Has(messageID)
}

// Stop para o gerenciador gossip
func (gm *GossipManager) Stop() {
	gm.cancel()
}
