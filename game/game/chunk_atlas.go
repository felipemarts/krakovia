package game

import (
	"image"
	"image/color"
	"math"
	"unsafe"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// ChunkAtlas gerencia o atlas de texturas específico de um chunk
type ChunkAtlas struct {
	// Mapeamento de blocos usados neste chunk
	UsedBlocks map[BlockType]int32 // BlockType → índice no atlas local

	// Atlas local
	GridSize       int32        // Tamanho do grid (ex: 4 para 4x4)
	TileSize       int32        // 32 pixels
	AtlasImage     *image.RGBA  // Imagem do atlas
	AtlasTexture   rl.Texture2D // Textura no GPU
	Material       rl.Material  // Material específico do chunk
	NeedsRebuild   bool         // Precisa reconstruir?
	IsUploaded     bool         // Já foi feito upload para GPU?
}

// NewChunkAtlas cria um novo atlas para um chunk
func NewChunkAtlas(gridSize, tileSize int32) *ChunkAtlas {
	atlasPixelSize := gridSize * tileSize
	return &ChunkAtlas{
		UsedBlocks: make(map[BlockType]int32),
		GridSize:   gridSize,
		TileSize:   tileSize,
		AtlasImage: image.NewRGBA(image.Rect(0, 0, int(atlasPixelSize), int(atlasPixelSize))),
		NeedsRebuild: true,
		IsUploaded: false,
	}
}

// AddBlockType adiciona um tipo de bloco ao atlas (se ainda não existe)
func (ca *ChunkAtlas) AddBlockType(blockType BlockType) {
	if _, exists := ca.UsedBlocks[blockType]; exists {
		return // Já existe
	}

	// Adicionar no próximo slot disponível
	index := int32(len(ca.UsedBlocks))
	ca.UsedBlocks[blockType] = index
	ca.NeedsRebuild = true
}

// RebuildAtlas reconstrói o atlas com as texturas necessárias
func (ca *ChunkAtlas) RebuildAtlas(textureCache map[BlockType]image.Image) {
	if !ca.NeedsRebuild {
		return
	}

	// Calcular tamanho necessário do grid
	numTextures := len(ca.UsedBlocks)
	requiredGridSize := int32(math.Ceil(math.Sqrt(float64(numTextures))))

	// Usar no mínimo 2x2, no máximo 8x8
	if requiredGridSize < 2 {
		requiredGridSize = 2
	}
	if requiredGridSize > 8 {
		requiredGridSize = 8
	}

	ca.GridSize = requiredGridSize
	atlasPixelSize := ca.GridSize * ca.TileSize

	// Recriar imagem se o tamanho mudou
	if ca.AtlasImage.Bounds().Dx() != int(atlasPixelSize) {
		ca.AtlasImage = image.NewRGBA(image.Rect(0, 0, int(atlasPixelSize), int(atlasPixelSize)))
	}

	// Limpar atlas
	for y := 0; y < int(atlasPixelSize); y++ {
		for x := 0; x < int(atlasPixelSize); x++ {
			ca.AtlasImage.Set(x, y, color.RGBA{0, 0, 0, 255})
		}
	}

	// Copiar cada textura para seu slot
	for blockType, index := range ca.UsedBlocks {
		img, exists := textureCache[blockType]
		if !exists {
			continue
		}

		// Calcular posição no grid
		col := index % ca.GridSize
		row := index / ca.GridSize

		destX := int(col * ca.TileSize)
		destY := int(row * ca.TileSize)

		// Copiar pixels
		srcBounds := img.Bounds()
		for y := 0; y < int(ca.TileSize) && y < srcBounds.Dy(); y++ {
			for x := 0; x < int(ca.TileSize) && x < srcBounds.Dx(); x++ {
				srcColor := img.At(srcBounds.Min.X+x, srcBounds.Min.Y+y)
				ca.AtlasImage.Set(destX+x, destY+y, srcColor)
			}
		}
	}

	ca.NeedsRebuild = false
}

// UploadToGPU faz upload do atlas para a GPU
func (ca *ChunkAtlas) UploadToGPU() {
	// Descarregar textura antiga se existir
	if ca.IsUploaded && ca.AtlasTexture.ID != 0 {
		rl.UnloadTexture(ca.AtlasTexture)
	}

	atlasPixelSize := ca.GridSize * ca.TileSize

	// Converter para Raylib Image
	raylibImg := rl.Image{
		Data:    unsafe.Pointer(&ca.AtlasImage.Pix[0]),
		Width:   atlasPixelSize,
		Height:  atlasPixelSize,
		Mipmaps: 1,
		Format:  rl.UncompressedR8g8b8a8,
	}

	// Upload para GPU
	ca.AtlasTexture = rl.LoadTextureFromImage(&raylibImg)
	rl.SetTextureFilter(ca.AtlasTexture, rl.FilterPoint)

	// Criar ou atualizar material
	if !ca.IsUploaded {
		ca.Material = rl.LoadMaterialDefault()
	}

	diffuseMap := ca.Material.GetMap(rl.MapDiffuse)
	diffuseMap.Texture = ca.AtlasTexture

	ca.IsUploaded = true
}

// GetBlockUVs retorna as coordenadas UV para um tipo de bloco
func (ca *ChunkAtlas) GetBlockUVs(blockType BlockType) (uMin, vMin, uMax, vMax float32) {
	index, exists := ca.UsedBlocks[blockType]
	if !exists {
		// Se não existe, usar primeira posição (default)
		index = 0
	}

	col := index % ca.GridSize
	row := index / ca.GridSize

	tileUV := float32(1.0) / float32(ca.GridSize)

	uMin = float32(col) * tileUV
	vMin = float32(row) * tileUV
	uMax = uMin + tileUV
	vMax = vMin + tileUV

	return
}

// Unload descarrega recursos do atlas
func (ca *ChunkAtlas) Unload() {
	if ca.IsUploaded && ca.AtlasTexture.ID != 0 {
		rl.UnloadTexture(ca.AtlasTexture)
		ca.IsUploaded = false
	}
}
