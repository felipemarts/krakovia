package game

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"sync"
	"unsafe"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// DynamicAtlasManager gerencia um atlas de texturas dinâmico
type DynamicAtlasManager struct {
	mu sync.RWMutex

	// Configuração
	AtlasGridSize  int32 // Ex: 4 para atlas 4x4
	TileSize       int32 // Ex: 32 pixels
	AtlasPixelSize int32 // AtlasGridSize * TileSize

	// Cache de texturas carregadas
	TextureCache map[BlockType]image.Image // BlockType → imagem 32x32

	// Mapeamento de slots
	BlockToSlot map[BlockType]int32   // BlockType → posição no atlas (0-15 para 4x4)
	SlotToBlock map[int32]BlockType   // posição → BlockType
	UsedSlots   map[int32]bool        // quais slots estão ocupados
	NextSlot    int32                 // próximo slot disponível

	// Atlas atual
	AtlasImage   *image.RGBA      // Imagem do atlas montado
	AtlasTexture rl.Texture2D     // Textura no GPU
	AtlasDirty   bool             // Precisa rebuild?

	// Estatísticas
	LoadedTextures int
	RebuildCount   int
}

// NewDynamicAtlasManager cria um novo gerenciador de atlas dinâmico
func NewDynamicAtlasManager(gridSize, tileSize int32) *DynamicAtlasManager {
	dam := &DynamicAtlasManager{
		AtlasGridSize:  gridSize,
		TileSize:       tileSize,
		AtlasPixelSize: gridSize * tileSize,
		TextureCache:   make(map[BlockType]image.Image),
		BlockToSlot:    make(map[BlockType]int32),
		SlotToBlock:    make(map[int32]BlockType),
		UsedSlots:      make(map[int32]bool),
		NextSlot:       1, // Slot 0 reservado para default
	}

	// Criar atlas vazio
	dam.AtlasImage = image.NewRGBA(image.Rect(0, 0, int(dam.AtlasPixelSize), int(dam.AtlasPixelSize)))

	// Carregar textura default no slot 0
	dam.LoadTexture(BlockAir, DefaultTextureFile)
	dam.BlockToSlot[BlockAir] = 0
	dam.SlotToBlock[0] = BlockAir
	dam.UsedSlots[0] = true

	return dam
}

// LoadTexture carrega uma textura individual do arquivo
func (dam *DynamicAtlasManager) LoadTexture(blockType BlockType, filePath string) error {
	dam.mu.Lock()
	defer dam.mu.Unlock()

	// Se já carregou, retorna
	if _, exists := dam.TextureCache[blockType]; exists {
		return nil
	}

	// Carregar arquivo
	file, err := os.Open("assets/" + filePath)
	if err != nil {
		return fmt.Errorf("erro ao abrir %s: %w", filePath, err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return fmt.Errorf("erro ao decodificar %s: %w", filePath, err)
	}

	dam.TextureCache[blockType] = img
	dam.LoadedTextures++

	return nil
}

// AllocateSlot aloca um slot no atlas para um BlockType
func (dam *DynamicAtlasManager) AllocateSlot(blockType BlockType) int32 {
	dam.mu.Lock()
	defer dam.mu.Unlock()

	// Se já tem slot, retorna
	if slot, exists := dam.BlockToSlot[blockType]; exists {
		return slot
	}

	// Verificar se há slots disponíveis
	maxSlots := dam.AtlasGridSize * dam.AtlasGridSize
	if dam.NextSlot >= maxSlots {
		// Atlas cheio, retorna slot default
		fmt.Printf("AVISO: Atlas cheio! BlockType %d usando textura default\n", blockType)
		return 0
	}

	// Alocar próximo slot
	slot := dam.NextSlot
	dam.NextSlot++

	dam.BlockToSlot[blockType] = slot
	dam.SlotToBlock[slot] = blockType
	dam.UsedSlots[slot] = true

	dam.AtlasDirty = true

	return slot
}

// FreeSlot libera um slot (quando chunk é descarregado e textura não é mais necessária)
func (dam *DynamicAtlasManager) FreeSlot(blockType BlockType) {
	dam.mu.Lock()
	defer dam.mu.Unlock()

	slot, exists := dam.BlockToSlot[blockType]
	if !exists || slot == 0 { // Não liberar slot 0 (default)
		return
	}

	delete(dam.BlockToSlot, blockType)
	delete(dam.SlotToBlock, slot)
	delete(dam.UsedSlots, slot)

	dam.AtlasDirty = true
}

// RebuildAtlas reconstrói a imagem do atlas
func (dam *DynamicAtlasManager) RebuildAtlas() {
	dam.mu.Lock()
	defer dam.mu.Unlock()

	if !dam.AtlasDirty {
		return
	}

	// Limpar atlas (preto com alpha)
	for y := 0; y < int(dam.AtlasPixelSize); y++ {
		for x := 0; x < int(dam.AtlasPixelSize); x++ {
			dam.AtlasImage.Set(x, y, color.RGBA{0, 0, 0, 255})
		}
	}

	// Copiar cada textura para seu slot
	for blockType, slot := range dam.BlockToSlot {
		img, exists := dam.TextureCache[blockType]
		if !exists {
			continue
		}

		// Calcular posição no grid
		col := slot % dam.AtlasGridSize
		row := slot / dam.AtlasGridSize

		destX := int(col * dam.TileSize)
		destY := int(row * dam.TileSize)

		// Copiar pixels
		for y := 0; y < int(dam.TileSize); y++ {
			for x := 0; x < int(dam.TileSize); x++ {
				// Garantir que não exceda os limites da imagem fonte
				srcBounds := img.Bounds()
				if x < srcBounds.Dx() && y < srcBounds.Dy() {
					srcColor := img.At(srcBounds.Min.X+x, srcBounds.Min.Y+y)
					dam.AtlasImage.Set(destX+x, destY+y, srcColor)
				}
			}
		}
	}

	dam.AtlasDirty = false
	dam.RebuildCount++
}

// UploadToGPU faz upload do atlas para GPU
func (dam *DynamicAtlasManager) UploadToGPU() {
	dam.mu.Lock()
	defer dam.mu.Unlock()

	// Descarregar textura antiga se existir
	if dam.AtlasTexture.ID != 0 {
		rl.UnloadTexture(dam.AtlasTexture)
	}

	// Converter image.RGBA para Raylib Image
	raylibImg := rl.Image{
		Data:    unsafe.Pointer(&dam.AtlasImage.Pix[0]),
		Width:   dam.AtlasPixelSize,
		Height:  dam.AtlasPixelSize,
		Mipmaps: 1,
		Format:  rl.UncompressedR8g8b8a8,
	}

	// Upload para GPU
	dam.AtlasTexture = rl.LoadTextureFromImage(&raylibImg)
	rl.SetTextureFilter(dam.AtlasTexture, rl.FilterPoint)
}

// GetBlockUVs retorna UVs para um BlockType
func (dam *DynamicAtlasManager) GetBlockUVs(blockType BlockType) (uMin, vMin, uMax, vMax float32) {
	dam.mu.RLock()
	defer dam.mu.RUnlock()

	slot, exists := dam.BlockToSlot[blockType]
	if !exists {
		slot = 0 // Default
	}

	col := slot % dam.AtlasGridSize
	row := slot / dam.AtlasGridSize

	tileUV := float32(1.0) / float32(dam.AtlasGridSize)

	uMin = float32(col) * tileUV
	vMin = float32(row) * tileUV
	uMax = uMin + tileUV
	vMax = vMin + tileUV

	return
}

// SaveAtlasDebug salva atlas atual em arquivo para debug
func (dam *DynamicAtlasManager) SaveAtlasDebug(filename string) error {
	dam.mu.RLock()
	defer dam.mu.RUnlock()

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	return png.Encode(file, dam.AtlasImage)
}

// PrintStats imprime estatísticas do atlas
func (dam *DynamicAtlasManager) PrintStats() {
	dam.mu.RLock()
	defer dam.mu.RUnlock()

	fmt.Printf("=== Dynamic Atlas Stats ===\n")
	fmt.Printf("Grid Size: %dx%d (max %d textures)\n", dam.AtlasGridSize, dam.AtlasGridSize, dam.AtlasGridSize*dam.AtlasGridSize)
	fmt.Printf("Loaded Textures: %d\n", dam.LoadedTextures)
	fmt.Printf("Allocated Slots: %d\n", len(dam.UsedSlots))
	fmt.Printf("Rebuild Count: %d\n", dam.RebuildCount)
	fmt.Printf("Atlas Dirty: %v\n", dam.AtlasDirty)
	fmt.Printf("==========================\n")
}
