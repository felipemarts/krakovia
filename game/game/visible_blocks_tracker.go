package game

import "sync"

// VisibleBlocksTracker rastreia quais tipos de blocos estão visíveis
type VisibleBlocksTracker struct {
	mu sync.RWMutex

	// Conta quantos chunks visíveis usam cada tipo de bloco
	BlockUsageCount map[BlockType]int
}

// NewVisibleBlocksTracker cria um novo rastreador
func NewVisibleBlocksTracker() *VisibleBlocksTracker {
	return &VisibleBlocksTracker{
		BlockUsageCount: make(map[BlockType]int),
	}
}

// RegisterChunk registra blocos de um chunk como visíveis
func (vbt *VisibleBlocksTracker) RegisterChunk(chunk *Chunk) {
	vbt.mu.Lock()
	defer vbt.mu.Unlock()

	// Contar tipos únicos no chunk
	uniqueBlocks := make(map[BlockType]bool)

	for x := int32(0); x < ChunkSize; x++ {
		for y := int32(0); y < ChunkHeight; y++ {
			for z := int32(0); z < ChunkSize; z++ {
				blockType := chunk.Blocks[x][y][z]
				if blockType != NoBlock {
					uniqueBlocks[blockType] = true
				}
			}
		}
	}

	// Incrementar contadores
	for blockType := range uniqueBlocks {
		vbt.BlockUsageCount[blockType]++
	}
}

// UnregisterChunk remove blocos de um chunk dos visíveis
func (vbt *VisibleBlocksTracker) UnregisterChunk(chunk *Chunk) {
	vbt.mu.Lock()
	defer vbt.mu.Unlock()

	// Contar tipos únicos no chunk
	uniqueBlocks := make(map[BlockType]bool)

	for x := int32(0); x < ChunkSize; x++ {
		for y := int32(0); y < ChunkHeight; y++ {
			for z := int32(0); z < ChunkSize; z++ {
				blockType := chunk.Blocks[x][y][z]
				if blockType != NoBlock {
					uniqueBlocks[blockType] = true
				}
			}
		}
	}

	// Decrementar contadores
	for blockType := range uniqueBlocks {
		if count, exists := vbt.BlockUsageCount[blockType]; exists {
			if count <= 1 {
				delete(vbt.BlockUsageCount, blockType)
			} else {
				vbt.BlockUsageCount[blockType] = count - 1
			}
		}
	}
}

// GetRequiredBlocks retorna lista de blocos que devem estar no atlas
func (vbt *VisibleBlocksTracker) GetRequiredBlocks() []BlockType {
	vbt.mu.RLock()
	defer vbt.mu.RUnlock()

	blocks := make([]BlockType, 0, len(vbt.BlockUsageCount))
	for blockType := range vbt.BlockUsageCount {
		blocks = append(blocks, blockType)
	}

	return blocks
}

// Clear limpa todos os registros
func (vbt *VisibleBlocksTracker) Clear() {
	vbt.mu.Lock()
	defer vbt.mu.Unlock()

	vbt.BlockUsageCount = make(map[BlockType]int)
}
