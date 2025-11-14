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
		return BlockAir
	}

	// Camada de superfície (y=8)
	if y == 8 {
		h := tg.hash3D(x, y, z)
		surfaceTypes := []BlockType{
			BlockGrass, BlockSand, BlockGravel, BlockStone,
			BlockSnow, BlockMoss, BlockClay,
		}
		return surfaceTypes[h%uint64(len(surfaceTypes))]
	}

	// Camadas intermediárias superiores (y=6-7)
	if y >= 6 && y < 8 {
		h := tg.hash3D(x, y, z)
		upperTypes := []BlockType{
			BlockDirt, BlockCobblestone, BlockGravel,
			BlockCoal, BlockClay, BlockStone,
		}
		return upperTypes[h%uint64(len(upperTypes))]
	}

	// Camadas intermediárias (y=4-5)
	if y >= 4 && y < 6 {
		h := tg.hash3D(x, y, z)
		midTypes := []BlockType{
			BlockDirt, BlockCobblestone, BlockGravel,
			BlockCoal, BlockIronOre, BlockStone,
			BlockClay, BlockObsidian,
		}
		return midTypes[h%uint64(len(midTypes))]
	}

	// Camadas profundas (y=2-3)
	if y >= 2 && y < 4 {
		h := tg.hash3D(x, y, z)
		deepTypes := []BlockType{
			BlockStone, BlockCobblestone, BlockIronOre,
			BlockGoldOre, BlockDiamondOre, BlockObsidian,
			BlockCoal, BlockBedrock,
		}
		return deepTypes[h%uint64(len(deepTypes))]
	}

	// Camada mais profunda (y=0-1) - mais minérios raros
	if y >= 0 && y < 2 {
		h := tg.hash3D(x, y, z)
		deepestTypes := []BlockType{
			BlockStone, BlockBedrock, BlockObsidian,
			BlockDiamondOre, BlockGoldOre, BlockLava,
			BlockIronOre,
		}
		return deepestTypes[h%uint64(len(deepestTypes))]
	}

	// Abaixo de y=0, apenas bedrock
	return BlockBedrock
}
