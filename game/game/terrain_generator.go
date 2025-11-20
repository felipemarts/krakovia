package game

// TerrainGenerator gera terreno de forma determinística baseado em seed
type TerrainGenerator struct {
	Seed int64
}

// NewTerrainGenerator cria um novo gerador de terreno
func NewTerrainGenerator(seed int64) *TerrainGenerator {
	return &TerrainGenerator{Seed: seed}
}

// hash3D gera um hash determinístico baseado em posição 3D
func (tg *TerrainGenerator) hash3D(x, y, z int32) uint64 {
	h := uint64(tg.Seed)
	h ^= uint64(x) * 0x45d9f3b
	h ^= uint64(y) * 0x45d9f3b * 3
	h ^= uint64(z) * 0x45d9f3b * 7
	h = (h ^ (h >> 16)) * 0x45d9f3b
	h = (h ^ (h >> 16)) * 0x45d9f3b
	h = h ^ (h >> 16)
	return h
}

// GetBlockTypeAt retorna o tipo de bloco para uma posição específica
func (tg *TerrainGenerator) GetBlockTypeAt(x, y, z int32) BlockType {
	// Camada de ar acima de y=8
	if y > 8 {
		return NoBlock
	}

	// Terreno sólido - apenas um tipo de bloco
	if y >= 0 && y <= 8 {
		return BlockType(DefaultBlockID)
	}

	// Abaixo de y=0
	return BlockType(DefaultBlockID)
}
