package network

import (
	"fmt"
	"sync"
	"time"
)

// RateLimiter controla a taxa de mensagens por peer
type RateLimiter struct {
	limits map[string]*peerLimit
	mu     sync.RWMutex
	rate   int           // Mensagens permitidas por segundo
	window time.Duration // Janela de tempo para contagem
}

// peerLimit rastreia mensagens de um peer específico
type peerLimit struct {
	count     int
	window    time.Time
	mu        sync.Mutex
	violations int // Contador de violações
}

// NewRateLimiter cria um novo rate limiter
func NewRateLimiter(messagesPerSecond int) *RateLimiter {
	return &RateLimiter{
		limits: make(map[string]*peerLimit),
		rate:   messagesPerSecond,
		window: time.Second,
	}
}

// Allow verifica se o peer pode enviar uma mensagem
func (rl *RateLimiter) Allow(peerID string) bool {
	rl.mu.Lock()
	limit, exists := rl.limits[peerID]
	if !exists {
		limit = &peerLimit{
			count:  0,
			window: time.Now(),
		}
		rl.limits[peerID] = limit
	}
	rl.mu.Unlock()

	limit.mu.Lock()
	defer limit.mu.Unlock()

	now := time.Now()

	// Se passou a janela de tempo, resetar
	if now.Sub(limit.window) >= rl.window {
		limit.count = 0
		limit.window = now
	}

	// Verificar se está dentro do limite
	if limit.count >= rl.rate {
		limit.violations++
		return false
	}

	limit.count++
	return true
}

// GetViolations retorna o número de violações de um peer
func (rl *RateLimiter) GetViolations(peerID string) int {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	if limit, exists := rl.limits[peerID]; exists {
		limit.mu.Lock()
		defer limit.mu.Unlock()
		return limit.violations
	}
	return 0
}

// Reset reseta o contador de um peer
func (rl *RateLimiter) Reset(peerID string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.limits, peerID)
}

// Cleanup remove entradas antigas
func (rl *RateLimiter) Cleanup(maxAge time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	for peerID, limit := range rl.limits {
		limit.mu.Lock()
		if limit.window.Before(cutoff) {
			delete(rl.limits, peerID)
		}
		limit.mu.Unlock()
	}
}

// MessageValidator valida mensagens contra ataques
type MessageValidator struct {
	maxSize      int
	rateLimiter  *RateLimiter
	cache        *MessageCache
	blockedPeers map[string]time.Time
	blockMu      sync.RWMutex
}

// NewMessageValidator cria um novo validador
func NewMessageValidator(config *GossipConfig, cache *MessageCache) *MessageValidator {
	return &MessageValidator{
		maxSize:      config.MaxMessageSize,
		rateLimiter:  NewRateLimiter(config.RateLimitPerSecond),
		cache:        cache,
		blockedPeers: make(map[string]time.Time),
	}
}

// ValidateMessage valida uma mensagem contra vários tipos de ataque
func (mv *MessageValidator) ValidateMessage(msg *GossipMessage, fromPeer string) error {
	// 1. Verificar se peer está bloqueado
	if mv.IsPeerBlocked(fromPeer) {
		return fmt.Errorf("peer %s is blocked", fromPeer)
	}

	// 2. Rate limiting
	if !mv.rateLimiter.Allow(fromPeer) {
		violations := mv.rateLimiter.GetViolations(fromPeer)

		// Se exceder 10 violações, bloquear peer temporariamente
		if violations > 10 {
			mv.BlockPeer(fromPeer, 5*time.Minute)
			return fmt.Errorf("peer %s blocked due to rate limit violations", fromPeer)
		}

		return fmt.Errorf("rate limit exceeded for peer %s", fromPeer)
	}

	// 3. Verificar tamanho da mensagem
	if len(msg.Data) > mv.maxSize {
		return fmt.Errorf("message size (%d) exceeds maximum (%d)", len(msg.Data), mv.maxSize)
	}

	// 4. Validar integridade da mensagem
	if err := msg.Validate(); err != nil {
		return fmt.Errorf("message validation failed: %w", err)
	}

	// 5. Verificar se mensagem já foi vista (ataque de replay)
	if mv.cache.Has(msg.ID) {
		return fmt.Errorf("duplicate message detected: %s", msg.ID)
	}

	// 6. Verificar se não é um ataque de flood com TTL alto
	if msg.TTL > 20 {
		return fmt.Errorf("TTL too high: %d", msg.TTL)
	}

	// 7. Verificar timestamp para evitar ataques de replay com mensagens antigas
	now := time.Now().Unix()
	age := now - msg.Timestamp
	if age > 3600 || age < -60 {
		return fmt.Errorf("invalid timestamp: message age %d seconds", age)
	}

	return nil
}

// BlockPeer bloqueia um peer temporariamente
func (mv *MessageValidator) BlockPeer(peerID string, duration time.Duration) {
	mv.blockMu.Lock()
	defer mv.blockMu.Unlock()
	mv.blockedPeers[peerID] = time.Now().Add(duration)
	fmt.Printf("⚠️  Peer %s blocked for %v due to malicious behavior\n", peerID, duration)
}

// IsPeerBlocked verifica se um peer está bloqueado
func (mv *MessageValidator) IsPeerBlocked(peerID string) bool {
	mv.blockMu.RLock()
	defer mv.blockMu.RUnlock()

	if blockUntil, exists := mv.blockedPeers[peerID]; exists {
		if time.Now().Before(blockUntil) {
			return true
		}
		// Tempo de bloqueio expirou, remover
		delete(mv.blockedPeers, peerID)
	}
	return false
}

// CleanupBlockedPeers remove peers cujo bloqueio expirou
func (mv *MessageValidator) CleanupBlockedPeers() {
	mv.blockMu.Lock()
	defer mv.blockMu.Unlock()

	now := time.Now()
	for peerID, blockUntil := range mv.blockedPeers {
		if now.After(blockUntil) {
			delete(mv.blockedPeers, peerID)
		}
	}
}

// GetBlockedPeers retorna lista de peers bloqueados
func (mv *MessageValidator) GetBlockedPeers() []string {
	mv.blockMu.RLock()
	defer mv.blockMu.RUnlock()

	peers := make([]string, 0, len(mv.blockedPeers))
	for peerID := range mv.blockedPeers {
		peers = append(peers, peerID)
	}
	return peers
}
