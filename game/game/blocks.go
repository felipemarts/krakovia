package game

// BlockType representa o tipo de um bloco no mundo
// 0 = sem bloco, >= 256 = bloco customizado
type BlockType uint16

// NoBlock representa ausência de bloco
const NoBlock BlockType = 0

// DefaultBlockTexturePath é o caminho da textura padrão
const DefaultBlockTexturePath = "assets/default_block.png"

// DefaultBlockID é o ID do bloco padrão (primeiro bloco customizado)
const DefaultBlockID uint16 = 256

// GetBlockUVs retorna as coordenadas UV normalizadas (0-1) para um tipo de bloco
// Atlas é 8x8, cada textura 32x32 pixels (256x256 total)
func GetBlockUVs(blockType BlockType) (uMin, vMin, uMax, vMax float32) {
	// Mapeamento: linha e coluna no atlas 8x8
	var row, col int32

	switch blockType {
	case BlockType(DefaultBlockID):
		row, col = 0, 0
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
