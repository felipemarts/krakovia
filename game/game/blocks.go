package game

// BlockType representa o tipo de um bloco no mundo
type BlockType uint8

const (
	BlockAir BlockType = iota
	BlockGrass
	BlockDirt
	BlockStone
	BlockWood
	BlockLeaves
	BlockSand
	BlockGravel
	BlockCobblestone
	BlockPlanks
	BlockBricks
	BlockGlass
	BlockIronOre
	BlockGoldOre
	BlockDiamondOre
	BlockCoal
	BlockSnow
	BlockIce
	BlockObsidian
	BlockBedrock
	BlockWater
	BlockLava
	BlockClay
	BlockMoss
)

// GetBlockUVs retorna as coordenadas UV normalizadas (0-1) para um tipo de bloco
// Atlas é 8x8, cada textura 32x32 pixels (256x256 total)
func GetBlockUVs(blockType BlockType) (uMin, vMin, uMax, vMax float32) {
	// Mapeamento: linha e coluna no atlas 8x8
	var row, col int32

	switch blockType {
	case BlockGrass:
		row, col = 1, 1
	case BlockDirt:
		row, col = 1, 0
	case BlockStone:
		row, col = 1, 2
	case BlockWood:
		row, col = 2, 0
	case BlockLeaves:
		row, col = 2, 1
	case BlockSand:
		row, col = 2, 2
	case BlockGravel:
		row, col = 3, 0
	case BlockCobblestone:
		row, col = 3, 1
	case BlockPlanks:
		row, col = 3, 2
	case BlockBricks:
		row, col = 4, 0
	case BlockGlass:
		row, col = 4, 1
	case BlockIronOre:
		row, col = 4, 2
	case BlockGoldOre:
		row, col = 5, 0
	case BlockDiamondOre:
		row, col = 5, 1
	case BlockCoal:
		row, col = 5, 2
	case BlockSnow:
		row, col = 6, 0
	case BlockIce:
		row, col = 6, 1
	case BlockObsidian:
		row, col = 6, 2
	case BlockBedrock:
		row, col = 7, 0
	case BlockWater:
		row, col = 7, 1
	case BlockLava:
		row, col = 7, 2
	case BlockClay:
		row, col = 0, 1
	case BlockMoss:
		row, col = 0, 2
	default:
		row, col = 0, 0 // Default: primeira textura
	}

	// Atlas 8x8, cada tile é 1/8 do tamanho total
	tileSize := float32(1.0 / 8.0)

	uMin = float32(col) * tileSize
	vMin = float32(row) * tileSize
	uMax = uMin + tileSize
	vMax = vMin + tileSize

	return
}
