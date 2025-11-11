package game

// BlockTextureFiles mapeia cada BlockType para o arquivo de textura correspondente
var BlockTextureFiles = map[BlockType]string{
	BlockAir:         "textures/tile_0_0.png",
	BlockGrass:       "textures/tile_1_1.png",
	BlockDirt:        "textures/tile_1_0.png",
	BlockStone:       "textures/tile_1_2.png",
	BlockWood:        "textures/tile_2_0.png",
	BlockLeaves:      "textures/tile_2_1.png",
	BlockSand:        "textures/tile_2_2.png",
	BlockGravel:      "textures/tile_3_0.png",
	BlockCobblestone: "textures/tile_3_1.png",
	BlockPlanks:      "textures/tile_3_2.png",
	BlockBricks:      "textures/tile_4_0.png",
	BlockGlass:       "textures/tile_4_1.png",
	BlockIronOre:     "textures/tile_4_2.png",
	BlockGoldOre:     "textures/tile_5_0.png",
	BlockDiamondOre:  "textures/tile_5_1.png",
	BlockCoal:        "textures/tile_5_2.png",
	BlockSnow:        "textures/tile_6_0.png",
	BlockIce:         "textures/tile_6_1.png",
	BlockObsidian:    "textures/tile_6_2.png",
	BlockBedrock:     "textures/tile_7_0.png",
	BlockWater:       "textures/tile_7_1.png",
	BlockLava:        "textures/tile_7_2.png",
	BlockClay:        "textures/tile_0_1.png",
	BlockMoss:        "textures/tile_0_2.png",
}

// DefaultTextureFile é a textura padrão (posição 0,0)
const DefaultTextureFile = "textures/tile_0_0.png"
