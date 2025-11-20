package game

// BlockTextureFiles mapeia cada BlockType para o arquivo de textura correspondente
var BlockTextureFiles = map[BlockType]string{
	BlockAir:   "textures/tile_0_0.png",
	BlockGrass: "textures/tile_3_2.png",
}

// DefaultTextureFile é a textura padrão (posição 0,0)
const DefaultTextureFile = "textures/tile_0_0.png"
