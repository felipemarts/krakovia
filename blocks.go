package main

// BlockType representa o tipo de um bloco no mundo
type BlockType uint8

const (
	BlockAir BlockType = iota
	BlockGrass
	BlockDirt
	BlockStone
)

// GetBlockUVs retorna as coordenadas UV normalizadas (0-1) para um tipo de bloco
// Atlas é 8x8, cada textura 32x32 pixels (256x256 total)
func GetBlockUVs(blockType BlockType) (uMin, vMin, uMax, vMax float32) {
	// Mapeamento: linha e coluna no atlas 8x8
	var row, col int32

	switch blockType {
	case BlockGrass:
		row, col = 1, 1 // Segunda linha, segunda coluna (grass top)
	case BlockDirt:
		row, col = 1, 0 // Segunda linha, primeira coluna (dirt)
	case BlockStone:
		row, col = 1, 2 // Segunda linha, terceira coluna (stone)
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
