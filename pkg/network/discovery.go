package network

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// PeerInfo contém informações sobre um peer
type PeerInfo struct {
	ID            string
	ConnectedAt   time.Time
	LastSeen      time.Time
	MessageCount  int64
	IsConnected   bool
}

// PeerDiscovery gerencia a descoberta e seleção de peers
type PeerDiscovery struct {
	knownPeers   map[string]*PeerInfo
	peersMutex   sync.RWMutex
	maxPeers     int
	minPeers     int
	nodeID       string
}

// NewPeerDiscovery cria uma nova instância de descoberta de peers
func NewPeerDiscovery(nodeID string, maxPeers, minPeers int) *PeerDiscovery {
	return &PeerDiscovery{
		knownPeers: make(map[string]*PeerInfo),
		maxPeers:   maxPeers,
		minPeers:   minPeers,
		nodeID:     nodeID,
	}
}

// AddKnownPeer adiciona um peer à lista de peers conhecidos
func (pd *PeerDiscovery) AddKnownPeer(peerID string) {
	pd.peersMutex.Lock()
	defer pd.peersMutex.Unlock()

	if peerID == pd.nodeID {
		return // Não adicionar a si mesmo
	}

	if _, exists := pd.knownPeers[peerID]; !exists {
		pd.knownPeers[peerID] = &PeerInfo{
			ID:          peerID,
			ConnectedAt: time.Now(),
			LastSeen:    time.Now(),
			IsConnected: false,
		}
	}
}

// MarkPeerConnected marca um peer como conectado
func (pd *PeerDiscovery) MarkPeerConnected(peerID string) {
	pd.peersMutex.Lock()
	defer pd.peersMutex.Unlock()

	if peer, exists := pd.knownPeers[peerID]; exists {
		peer.IsConnected = true
		peer.ConnectedAt = time.Now()
		peer.LastSeen = time.Now()
	} else {
		pd.knownPeers[peerID] = &PeerInfo{
			ID:          peerID,
			ConnectedAt: time.Now(),
			LastSeen:    time.Now(),
			IsConnected: true,
		}
	}
}

// MarkPeerDisconnected marca um peer como desconectado
func (pd *PeerDiscovery) MarkPeerDisconnected(peerID string) {
	pd.peersMutex.Lock()
	defer pd.peersMutex.Unlock()

	if peer, exists := pd.knownPeers[peerID]; exists {
		peer.IsConnected = false
		peer.LastSeen = time.Now()
	}
}

// UpdatePeerActivity atualiza a atividade de um peer
func (pd *PeerDiscovery) UpdatePeerActivity(peerID string) {
	pd.peersMutex.Lock()
	defer pd.peersMutex.Unlock()

	if peer, exists := pd.knownPeers[peerID]; exists {
		peer.LastSeen = time.Now()
		peer.MessageCount++
	}
}

// GetConnectedPeersCount retorna o número de peers conectados
func (pd *PeerDiscovery) GetConnectedPeersCount() int {
	pd.peersMutex.RLock()
	defer pd.peersMutex.RUnlock()

	count := 0
	for _, peer := range pd.knownPeers {
		if peer.IsConnected {
			count++
		}
	}
	return count
}

// ShouldAcceptNewPeer verifica se deve aceitar um novo peer
func (pd *PeerDiscovery) ShouldAcceptNewPeer() bool {
	return pd.GetConnectedPeersCount() < pd.maxPeers
}

// NeedsMorePeers verifica se precisa de mais peers
func (pd *PeerDiscovery) NeedsMorePeers() bool {
	return pd.GetConnectedPeersCount() < pd.minPeers
}

// SelectPeersToConnect seleciona peers para conectar
func (pd *PeerDiscovery) SelectPeersToConnect(availablePeers []string, currentlyConnected map[string]bool) []string {
	pd.peersMutex.Lock()
	defer pd.peersMutex.Unlock()

	// Nota: peers já devem ter sido adicionados via AddKnownPeer antes de chamar este método

	connectedCount := 0
	for _, peer := range pd.knownPeers {
		if peer.IsConnected {
			connectedCount++
		}
	}

	// Calcular quantos peers precisamos conectar
	needCount := pd.minPeers - connectedCount
	if needCount <= 0 {
		return []string{}
	}

	// Selecionar peers para conectar
	var candidates []string
	for _, peerID := range availablePeers {
		if peerID == pd.nodeID {
			continue
		}
		if currentlyConnected[peerID] {
			continue
		}
		candidates = append(candidates, peerID)
	}

	// Embaralhar e selecionar
	rand.Shuffle(len(candidates), func(i, j int) {
		candidates[i], candidates[j] = candidates[j], candidates[i]
	})

	maxToConnect := pd.maxPeers - connectedCount
	if len(candidates) < maxToConnect {
		maxToConnect = len(candidates)
	}

	return candidates[:min(needCount, maxToConnect)]
}

// SelectPeersToDisconnect seleciona peers para desconectar quando exceder o limite
func (pd *PeerDiscovery) SelectPeersToDisconnect(connectedPeerIDs []string) []string {
	pd.peersMutex.RLock()
	defer pd.peersMutex.RUnlock()

	connectedCount := len(connectedPeerIDs)
	if connectedCount <= pd.maxPeers {
		return []string{}
	}

	// Calcular quantos precisamos desconectar
	disconnectCount := connectedCount - pd.maxPeers

	// Criar slice de peers com suas métricas
	type peerScore struct {
		id    string
		score float64
	}

	var scores []peerScore
	for _, peerID := range connectedPeerIDs {
		if peer, exists := pd.knownPeers[peerID]; exists {
			// Score baseado em: tempo de conexão e atividade
			connectionTime := time.Since(peer.ConnectedAt).Seconds()
			activityScore := float64(peer.MessageCount) / max(connectionTime, 1)

			// Quanto menor o score, mais provável de ser desconectado
			score := connectionTime + (activityScore * 100)
			scores = append(scores, peerScore{id: peerID, score: score})
		}
	}

	// Ordenar por score (menores primeiro = menos ativos/recentes)
	// Implementação simples: peers com menor score são desconectados
	var toDisconnect []string
	for i := 0; i < min(disconnectCount, len(scores)); i++ {
		minIdx := i
		for j := i + 1; j < len(scores); j++ {
			if scores[j].score < scores[minIdx].score {
				minIdx = j
			}
		}
		scores[i], scores[minIdx] = scores[minIdx], scores[i]
		toDisconnect = append(toDisconnect, scores[i].id)
	}

	return toDisconnect
}

// GetPeerStats retorna estatísticas dos peers
func (pd *PeerDiscovery) GetPeerStats() map[string]interface{} {
	pd.peersMutex.RLock()
	defer pd.peersMutex.RUnlock()

	connected := 0
	known := len(pd.knownPeers)

	for _, peer := range pd.knownPeers {
		if peer.IsConnected {
			connected++
		}
	}

	return map[string]interface{}{
		"connected": connected,
		"known":     known,
		"max":       pd.maxPeers,
		"min":       pd.minPeers,
		"need_more": connected < pd.minPeers,
		"at_limit":  connected >= pd.maxPeers,
	}
}

// PrintStats imprime estatísticas dos peers
func (pd *PeerDiscovery) PrintStats() {
	stats := pd.GetPeerStats()
	fmt.Printf("Peer Stats - Connected: %d/%d (min: %d, max: %d) | Known: %d\n",
		stats["connected"], stats["max"], stats["min"], stats["max"], stats["known"])
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
