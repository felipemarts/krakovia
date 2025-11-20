package game

import (
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

// BlockOrientation representa a orientação do bloco (para onde a frente aponta)
type BlockOrientation uint8

const (
	OrientationNorth BlockOrientation = iota // Frente para -Z (padrão)
	OrientationEast                          // Frente para +X (direita)
	OrientationSouth                         // Frente para +Z (costas)
	OrientationWest                          // Frente para -X (esquerda)
)

// BlockFace representa uma face do bloco
type BlockFace int

const (
	FaceRight  BlockFace = 0 // +X
	FaceLeft   BlockFace = 1 // -X
	FaceTop    BlockFace = 2 // +Y
	FaceBottom BlockFace = 3 // -Y
	FaceFront  BlockFace = 4 // +Z
	FaceBack   BlockFace = 5 // -Z
)

// FaceNames para display na UI
var FaceNames = map[BlockFace]string{
	FaceRight:  "Direita",
	FaceLeft:   "Esquerda",
	FaceTop:    "Topo",
	FaceBottom: "Fundo",
	FaceFront:  "Frente",
	FaceBack:   "Costas",
}

// CustomBlockDefinition define um bloco customizado criado pelo jogador
type CustomBlockDefinition struct {
	ID          uint16                 `json:"id"`           // ID único do bloco (>= CustomBlockIDStart)
	Name        string                 `json:"name"`         // Nome do bloco
	FaceTextures [6]string             `json:"face_textures"` // Caminho das texturas por face
	FaceImages   [6]image.Image        `json:"-"`            // Imagens carregadas (não serializado)
	CreatedAt   int64                  `json:"created_at"`   // Timestamp de criação
}

// CustomBlockIDStart é o ID inicial para blocos customizados
// Blocos padrão (NoBlock, BlockGrass) usam IDs 0-255
// Blocos customizados usam IDs 256+
const CustomBlockIDStart = 256

// CustomBlockManager gerencia todos os blocos customizados do jogador
type CustomBlockManager struct {
	mu sync.RWMutex

	// Blocos customizados registrados
	Blocks map[uint16]*CustomBlockDefinition

	// Próximo ID disponível
	NextID uint16

	// Diretório base para salvar texturas
	TexturesDir string

	// Diretório para dados de blocos
	DataDir string

	// Atlas de texturas para blocos customizados
	Atlas *DynamicAtlasManager

	// Mapeamento de face para slot no atlas
	// Chave: "blockID_faceIndex" (ex: "256_0" para face direita do bloco 256)
	FaceToSlot map[string]int32
}

// NewCustomBlockManager cria um novo gerenciador de blocos customizados
func NewCustomBlockManager() *CustomBlockManager {
	cbm := &CustomBlockManager{
		Blocks:      make(map[uint16]*CustomBlockDefinition),
		NextID:      CustomBlockIDStart,
		TexturesDir: "data/blocks/textures",
		DataDir:     "data/blocks",
		FaceToSlot:  make(map[string]int32),
	}

	// Criar diretórios se não existirem
	os.MkdirAll(cbm.TexturesDir, 0755)
	os.MkdirAll(cbm.DataDir, 0755)

	// Criar atlas grande para blocos customizados (16x16 = 256 slots)
	// Cada bloco usa até 6 slots (um por face)
	cbm.Atlas = NewDynamicAtlasManager(16, 32)

	return cbm
}

// EnsureDefaultBlock garante que o bloco padrão existe com ID 256
func (cbm *CustomBlockManager) EnsureDefaultBlock() error {
	cbm.mu.Lock()
	defer cbm.mu.Unlock()

	// Verificar se o bloco padrão já existe
	if _, exists := cbm.Blocks[DefaultBlockID]; exists {
		return nil
	}

	// Criar o bloco padrão
	block := &CustomBlockDefinition{
		ID:        DefaultBlockID,
		Name:      "Default",
		CreatedAt: 0,
	}

	// Tentar carregar a textura padrão para todas as faces
	file, err := os.Open(DefaultBlockTexturePath)
	if err == nil {
		defer file.Close()

		img, _, decodeErr := image.Decode(file)
		if decodeErr == nil {
			// Converter para RGBA
			bounds := img.Bounds()
			rgbaImg := image.NewRGBA(bounds)
			for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
				for x := bounds.Min.X; x < bounds.Max.X; x++ {
					rgbaImg.Set(x, y, img.At(x, y))
				}
			}

			// Aplicar a mesma textura para todas as faces
			for faceIdx := 0; faceIdx < 6; faceIdx++ {
				block.FaceImages[faceIdx] = rgbaImg
			}
		}
	}

	// Definir caminhos de textura (mesmo que as imagens não tenham sido carregadas)
	for faceIdx := 0; faceIdx < 6; faceIdx++ {
		block.FaceTextures[faceIdx] = DefaultBlockTexturePath
	}

	// Adicionar ao mapa
	cbm.Blocks[block.ID] = block

	// Atualizar próximo ID se necessário
	if block.ID >= cbm.NextID {
		cbm.NextID = block.ID + 1
	}

	return nil
}

// CreateBlock cria um novo bloco customizado
func (cbm *CustomBlockManager) CreateBlock(name string) *CustomBlockDefinition {
	cbm.mu.Lock()
	defer cbm.mu.Unlock()

	block := &CustomBlockDefinition{
		ID:        cbm.NextID,
		Name:      name,
		CreatedAt: 0, // Será definido ao salvar
	}

	cbm.Blocks[block.ID] = block
	cbm.NextID++

	return block
}

// SetFaceTexture define a textura de uma face específica do bloco
func (cbm *CustomBlockManager) SetFaceTexture(blockID uint16, face BlockFace, img image.Image) error {
	cbm.mu.Lock()
	defer cbm.mu.Unlock()

	block, exists := cbm.Blocks[blockID]
	if !exists {
		return fmt.Errorf("bloco %d não encontrado", blockID)
	}

	// Validar tamanho da imagem
	bounds := img.Bounds()
	if bounds.Dx() != 32 || bounds.Dy() != 32 {
		return fmt.Errorf("imagem deve ser 32x32 pixels, recebida: %dx%d", bounds.Dx(), bounds.Dy())
	}

	// Converter para RGBA para garantir compatibilidade
	rgbaImg := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			rgbaImg.Set(x, y, img.At(x, y))
		}
	}

	// Salvar imagem RGBA
	block.FaceImages[face] = rgbaImg

	// Salvar arquivo de textura
	filename := fmt.Sprintf("block_%d_face_%d.png", blockID, face)
	filepath := filepath.Join(cbm.TexturesDir, filename)
	block.FaceTextures[face] = filepath

	// Criar arquivo
	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("erro ao criar arquivo: %w", err)
	}
	defer file.Close()

	err = png.Encode(file, rgbaImg)
	if err != nil {
		return fmt.Errorf("erro ao salvar PNG: %w", err)
	}

	return nil
}

// LoadFaceTextureFromFile carrega uma textura de um arquivo para uma face
func (cbm *CustomBlockManager) LoadFaceTextureFromFile(blockID uint16, face BlockFace, filepath string) error {
	// Abrir arquivo
	file, err := os.Open(filepath)
	if err != nil {
		return fmt.Errorf("erro ao abrir arquivo: %w", err)
	}
	defer file.Close()

	// Decodificar PNG
	img, _, err := image.Decode(file)
	if err != nil {
		return fmt.Errorf("erro ao decodificar imagem: %w", err)
	}

	// Definir textura
	return cbm.SetFaceTexture(blockID, face, img)
}

// BuildAtlas constrói o atlas com todas as texturas dos blocos customizados
func (cbm *CustomBlockManager) BuildAtlas() {
	cbm.mu.Lock()
	defer cbm.mu.Unlock()

	// Limpar mapeamentos anteriores
	cbm.FaceToSlot = make(map[string]int32)

	// Resetar atlas
	cbm.Atlas = NewDynamicAtlasManager(16, 32)

	slotIndex := int32(1) // Slot 0 é reservado para default

	// Adicionar texturas de cada bloco
	for blockID, block := range cbm.Blocks {
		for faceIdx := 0; faceIdx < 6; faceIdx++ {
			img := block.FaceImages[faceIdx]
			if img == nil {
				continue
			}

			// Criar chave única para esta face
			key := fmt.Sprintf("%d_%d", blockID, faceIdx)

			// Criar um BlockType temporário para esta face
			// Usamos o ID do bloco + offset baseado na face
			faceBlockType := BlockType(blockID*6 + uint16(faceIdx))

			// Armazenar no cache
			cbm.Atlas.TextureCache[faceBlockType] = img
			cbm.Atlas.BlockToSlot[faceBlockType] = slotIndex
			cbm.Atlas.SlotToBlock[slotIndex] = faceBlockType
			cbm.Atlas.UsedSlots[slotIndex] = true

			// Mapear para lookup rápido
			cbm.FaceToSlot[key] = slotIndex

			slotIndex++
		}
	}

	cbm.Atlas.AtlasDirty = true
	cbm.Atlas.RebuildAtlas()
	cbm.Atlas.UploadToGPU()
}

// GetFaceUVs retorna as coordenadas UV para uma face específica de um bloco
func (cbm *CustomBlockManager) GetFaceUVs(blockID uint16, face BlockFace, orientation BlockOrientation) (uMin, vMin, uMax, vMax float32) {
	cbm.mu.RLock()
	defer cbm.mu.RUnlock()

	// Aplicar rotação baseada na orientação
	rotatedFace := cbm.rotateFace(face, orientation)

	// Buscar slot para esta face
	key := fmt.Sprintf("%d_%d", blockID, rotatedFace)
	slot, exists := cbm.FaceToSlot[key]
	if !exists {
		// Retornar UVs default (slot 0)
		return 0, 0, 1.0 / 16.0, 1.0 / 16.0
	}

	// Calcular UVs baseado no slot
	col := slot % cbm.Atlas.AtlasGridSize
	row := slot / cbm.Atlas.AtlasGridSize
	tileUV := float32(1.0) / float32(cbm.Atlas.AtlasGridSize)

	uMin = float32(col) * tileUV
	vMin = float32(row) * tileUV
	uMax = uMin + tileUV
	vMax = vMin + tileUV

	return
}

// rotateFace aplica rotação de orientação a uma face
func (cbm *CustomBlockManager) rotateFace(face BlockFace, orientation BlockOrientation) BlockFace {
	// Faces top e bottom não são afetadas por rotação horizontal
	if face == FaceTop || face == FaceBottom {
		return face
	}

	// Mapear faces laterais considerando orientação
	// Orientação North (padrão): sem rotação
	// Orientação East: rotação 90° horário
	// Orientação South: rotação 180°
	// Orientação West: rotação 90° anti-horário

	// Ordem das faces horizontais: Front -> Right -> Back -> Left -> Front...
	horizontalFaces := []BlockFace{FaceFront, FaceRight, FaceBack, FaceLeft}

	// Encontrar índice da face atual
	var currentIdx int
	for i, f := range horizontalFaces {
		if f == face {
			currentIdx = i
			break
		}
	}

	// Aplicar rotação
	rotationSteps := int(orientation)
	newIdx := (currentIdx + rotationSteps) % 4

	return horizontalFaces[newIdx]
}

// SaveBlock salva a definição do bloco em disco
func (cbm *CustomBlockManager) SaveBlock(blockID uint16) error {
	cbm.mu.RLock()
	block, exists := cbm.Blocks[blockID]
	cbm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("bloco %d não encontrado", blockID)
	}

	// Serializar para JSON
	data, err := json.MarshalIndent(block, "", "  ")
	if err != nil {
		return fmt.Errorf("erro ao serializar: %w", err)
	}

	// Salvar arquivo
	filename := fmt.Sprintf("block_%d.json", blockID)
	filepath := filepath.Join(cbm.DataDir, filename)

	err = os.WriteFile(filepath, data, 0644)
	if err != nil {
		return fmt.Errorf("erro ao salvar arquivo: %w", err)
	}

	return nil
}

// LoadAllBlocks carrega todos os blocos salvos do disco
func (cbm *CustomBlockManager) LoadAllBlocks() error {
	cbm.mu.Lock()
	defer cbm.mu.Unlock()

	// Listar arquivos JSON no diretório
	entries, err := os.ReadDir(cbm.DataDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Diretório não existe ainda
		}
		return fmt.Errorf("erro ao listar diretório: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		// Ler arquivo
		filepath := filepath.Join(cbm.DataDir, entry.Name())
		data, err := os.ReadFile(filepath)
		if err != nil {
			fmt.Printf("AVISO: erro ao ler %s: %v\n", entry.Name(), err)
			continue
		}

		// Deserializar
		var block CustomBlockDefinition
		err = json.Unmarshal(data, &block)
		if err != nil {
			fmt.Printf("AVISO: erro ao deserializar %s: %v\n", entry.Name(), err)
			continue
		}

		// Carregar imagens das texturas
		for faceIdx := 0; faceIdx < 6; faceIdx++ {
			texPath := block.FaceTextures[faceIdx]
			if texPath == "" {
				continue
			}

			file, err := os.Open(texPath)
			if err != nil {
				fmt.Printf("AVISO: erro ao abrir textura %s: %v\n", texPath, err)
				continue
			}

			img, _, err := image.Decode(file)
			file.Close()
			if err != nil {
				fmt.Printf("AVISO: erro ao decodificar textura %s: %v\n", texPath, err)
				continue
			}

			// Converter para RGBA para garantir compatibilidade
			bounds := img.Bounds()
			rgbaImg := image.NewRGBA(bounds)
			for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
				for x := bounds.Min.X; x < bounds.Max.X; x++ {
					rgbaImg.Set(x, y, img.At(x, y))
				}
			}

			block.FaceImages[faceIdx] = rgbaImg
		}

		// Adicionar ao mapa
		cbm.Blocks[block.ID] = &block

		// Atualizar próximo ID se necessário
		if block.ID >= cbm.NextID {
			cbm.NextID = block.ID + 1
		}
	}

	return nil
}

// GetBlock retorna um bloco pelo ID
func (cbm *CustomBlockManager) GetBlock(blockID uint16) *CustomBlockDefinition {
	cbm.mu.RLock()
	defer cbm.mu.RUnlock()
	return cbm.Blocks[blockID]
}

// ListBlocks retorna todos os blocos customizados ordenados por ID
func (cbm *CustomBlockManager) ListBlocks() []*CustomBlockDefinition {
	cbm.mu.RLock()
	defer cbm.mu.RUnlock()

	blocks := make([]*CustomBlockDefinition, 0, len(cbm.Blocks))
	for _, block := range cbm.Blocks {
		blocks = append(blocks, block)
	}

	// Ordenar por ID para ordem consistente
	sort.Slice(blocks, func(i, j int) bool {
		return blocks[i].ID < blocks[j].ID
	})

	return blocks
}

// DeleteBlock remove um bloco customizado
func (cbm *CustomBlockManager) DeleteBlock(blockID uint16) error {
	cbm.mu.Lock()
	defer cbm.mu.Unlock()

	block, exists := cbm.Blocks[blockID]
	if !exists {
		return fmt.Errorf("bloco %d não encontrado", blockID)
	}

	// Remover arquivos de textura
	for _, texPath := range block.FaceTextures {
		if texPath != "" {
			os.Remove(texPath)
		}
	}

	// Remover arquivo JSON
	jsonPath := filepath.Join(cbm.DataDir, fmt.Sprintf("block_%d.json", blockID))
	os.Remove(jsonPath)

	// Remover do mapa
	delete(cbm.Blocks, blockID)

	return nil
}

// IsCustomBlock verifica se um BlockType é um bloco customizado
func IsCustomBlock(blockType BlockType) bool {
	return uint16(blockType) >= CustomBlockIDStart
}

// GetCustomBlockID extrai o ID do bloco customizado de um BlockType
func GetCustomBlockID(blockType BlockType) uint16 {
	return uint16(blockType)
}

// EncodeCustomBlockFace codifica um bloco customizado + face em um BlockType único
// Usado para o atlas de texturas por face
// Formato: (blockID - CustomBlockIDStart) * 6 + faceIndex + CustomBlockIDStart + 10000
func EncodeCustomBlockFace(blockID uint16, face BlockFace) BlockType {
	// Offset para separar das texturas de bloco principal
	const faceTextureOffset = 10000
	return BlockType(faceTextureOffset + (blockID-CustomBlockIDStart)*6 + uint16(face))
}

// DecodeCustomBlockFace decodifica um BlockType de face em blockID e face
func DecodeCustomBlockFace(bt BlockType) (blockID uint16, face BlockFace) {
	const faceTextureOffset = 10000
	encoded := uint16(bt) - faceTextureOffset
	blockID = encoded/6 + CustomBlockIDStart
	face = BlockFace(encoded % 6)
	return
}

// IsCustomBlockFace verifica se um BlockType representa uma face de bloco customizado
func IsCustomBlockFace(bt BlockType) bool {
	return uint16(bt) >= 10000
}
