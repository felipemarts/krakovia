package network

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// GossipMessage representa uma mensagem no protocolo gossip
type GossipMessage struct {
	ID          string    `json:"id"`           // UUID único
	OriginID    string    `json:"origin_id"`    // ID do nó que originou a mensagem
	Type        string    `json:"type"`         // Tipo da mensagem (block, transaction, etc)
	Data        []byte    `json:"data"`         // Dados da mensagem
	Timestamp   int64     `json:"timestamp"`    // Unix timestamp
	TTL         int       `json:"ttl"`          // Time to live (hops)
	HopCount    int       `json:"hop_count"`    // Número de hops já realizados
	Signature   string    `json:"signature"`    // Assinatura para validação (futuro)
	Hash        string    `json:"hash"`         // Hash da mensagem para integridade
}

// GossipConfig configurações do protocolo gossip
type GossipConfig struct {
	Fanout              int           // Número de peers para propagar (default: 3)
	MaxTTL              int           // TTL máximo (default: 10)
	CacheSize           int           // Tamanho do cache de mensagens vistas (default: 10000)
	CacheDuration       time.Duration // Quanto tempo manter mensagens no cache (default: 5min)
	RateLimitPerSecond  int           // Limite de mensagens por segundo por peer (default: 100)
	MaxMessageSize      int           // Tamanho máximo de mensagem em bytes (default: 1MB)
	CleanupInterval     time.Duration // Intervalo de limpeza do cache (default: 1min)
}

// DefaultGossipConfig retorna configuração padrão
func DefaultGossipConfig() *GossipConfig {
	return &GossipConfig{
		Fanout:              3,
		MaxTTL:              10,
		CacheSize:           10000,
		CacheDuration:       5 * time.Minute,
		RateLimitPerSecond:  100,
		MaxMessageSize:      1024 * 1024, // 1MB
		CleanupInterval:     1 * time.Minute,
	}
}

// MessageCache armazena mensagens já vistas
type MessageCache struct {
	messages map[string]*cacheEntry
	mu       sync.RWMutex
	maxSize  int
}

type cacheEntry struct {
	timestamp time.Time
	message   *GossipMessage
}

// NewMessageCache cria um novo cache de mensagens
func NewMessageCache(maxSize int) *MessageCache {
	return &MessageCache{
		messages: make(map[string]*cacheEntry),
		maxSize:  maxSize,
	}
}

// Has verifica se a mensagem já foi vista
func (mc *MessageCache) Has(messageID string) bool {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	_, exists := mc.messages[messageID]
	return exists
}

// Add adiciona uma mensagem ao cache
func (mc *MessageCache) Add(msg *GossipMessage) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	// Se cache está cheio, remover entrada mais antiga
	if len(mc.messages) >= mc.maxSize {
		mc.removeOldestUnsafe()
	}

	mc.messages[msg.ID] = &cacheEntry{
		timestamp: time.Now(),
		message:   msg,
	}
}

// removeOldestUnsafe remove a entrada mais antiga (não thread-safe)
func (mc *MessageCache) removeOldestUnsafe() {
	var oldestID string
	var oldestTime time.Time

	for id, entry := range mc.messages {
		if oldestID == "" || entry.timestamp.Before(oldestTime) {
			oldestID = id
			oldestTime = entry.timestamp
		}
	}

	if oldestID != "" {
		delete(mc.messages, oldestID)
	}
}

// Cleanup remove mensagens antigas do cache
func (mc *MessageCache) Cleanup(maxAge time.Duration) int {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	removed := 0

	for id, entry := range mc.messages {
		if entry.timestamp.Before(cutoff) {
			delete(mc.messages, id)
			removed++
		}
	}

	return removed
}

// Size retorna o tamanho atual do cache
func (mc *MessageCache) Size() int {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return len(mc.messages)
}

// NewGossipMessage cria uma nova mensagem gossip
func NewGossipMessage(originID, msgType string, data []byte, ttl int) (*GossipMessage, error) {
	msg := &GossipMessage{
		ID:        uuid.New().String(),
		OriginID:  originID,
		Type:      msgType,
		Data:      data,
		Timestamp: time.Now().Unix(),
		TTL:       ttl,
		HopCount:  0,
	}

	// Calcular hash para integridade
	hash, err := msg.calculateHash()
	if err != nil {
		return nil, fmt.Errorf("failed to calculate hash: %w", err)
	}
	msg.Hash = hash

	return msg, nil
}

// calculateHash calcula o hash da mensagem
func (gm *GossipMessage) calculateHash() (string, error) {
	// Criar cópia sem hash para cálculo
	temp := *gm
	temp.Hash = ""
	temp.Signature = ""

	data, err := json.Marshal(temp)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

// Validate valida a integridade da mensagem
func (gm *GossipMessage) Validate() error {
	// Verificar campos obrigatórios
	if gm.ID == "" {
		return fmt.Errorf("message ID is empty")
	}
	if gm.OriginID == "" {
		return fmt.Errorf("origin ID is empty")
	}
	if gm.Type == "" {
		return fmt.Errorf("message type is empty")
	}

	// Verificar TTL
	if gm.TTL < 0 {
		return fmt.Errorf("invalid TTL: %d", gm.TTL)
	}

	// Verificar hop count
	if gm.HopCount < 0 {
		return fmt.Errorf("invalid hop count: %d", gm.HopCount)
	}

	// Verificar se hop count não excedeu TTL
	if gm.HopCount > gm.TTL {
		return fmt.Errorf("hop count (%d) exceeds TTL (%d)", gm.HopCount, gm.TTL)
	}

	// Verificar timestamp (não pode ser futuro ou muito antigo)
	now := time.Now().Unix()
	if gm.Timestamp > now+60 { // 1 minuto de tolerância para clock skew
		return fmt.Errorf("timestamp is in the future")
	}
	if gm.Timestamp < now-3600 { // Mensagens com mais de 1 hora são rejeitadas
		return fmt.Errorf("timestamp is too old")
	}

	// Verificar hash
	calculatedHash, err := gm.calculateHash()
	if err != nil {
		return fmt.Errorf("failed to calculate hash: %w", err)
	}
	if calculatedHash != gm.Hash {
		return fmt.Errorf("hash mismatch: expected %s, got %s", gm.Hash, calculatedHash)
	}

	return nil
}

// IncrementHop incrementa o contador de hops
func (gm *GossipMessage) IncrementHop() {
	gm.HopCount++
}

// ShouldPropagate verifica se a mensagem deve continuar sendo propagada
func (gm *GossipMessage) ShouldPropagate() bool {
	return gm.HopCount < gm.TTL
}

// Clone cria uma cópia da mensagem
func (gm *GossipMessage) Clone() *GossipMessage {
	clone := *gm
	// Copiar slice para evitar shared memory
	if len(gm.Data) > 0 {
		clone.Data = make([]byte, len(gm.Data))
		copy(clone.Data, gm.Data)
	}
	return &clone
}

// ToJSON serializa a mensagem para JSON
func (gm *GossipMessage) ToJSON() ([]byte, error) {
	return json.Marshal(gm)
}

// FromJSON desserializa uma mensagem de JSON
func FromJSON(data []byte) (*GossipMessage, error) {
	var msg GossipMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}
