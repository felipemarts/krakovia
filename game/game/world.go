package game

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

// World representa o mundo voxel com sistema de chunks
type World struct {
	ChunkManager     *ChunkManager
	GrassMesh        rl.Mesh
	Material         rl.Material
	TextureAtlas     rl.Texture2D
	RenderDistance   int32
	TerrainGenerator *TerrainGenerator

	// Sistema de atlas dinâmico
	DynamicAtlas  *DynamicAtlasManager
	VisibleBlocks *VisibleBlocksTracker
}

func NewWorld() *World {
	renderDistance := int32(5)
	w := &World{
		ChunkManager:     NewChunkManager(renderDistance),
		RenderDistance:   renderDistance,
		TerrainGenerator: NewTerrainGenerator(12345), // Seed fixo para testes
	}
	return w
}

// InitWorldGraphics inicializa recursos gráficos do mundo (deve ser chamado após rl.InitWindow)
func (w *World) InitWorldGraphics() {
	// Inicializar atlas dinâmico 4x4
	w.DynamicAtlas = NewDynamicAtlasManager(4, 32)
	w.VisibleBlocks = NewVisibleBlocksTracker()

	// Carregar texturas de todos os tipos conhecidos
	for blockType, texFile := range BlockTextureFiles {
		err := w.DynamicAtlas.LoadTexture(blockType, texFile)
		if err != nil {
			// Apenas aviso, não falha
			// fmt.Printf("AVISO: Erro ao carregar textura %s: %v\n", texFile, err)
		}
	}

	// Build inicial do atlas
	w.DynamicAtlas.RebuildAtlas()
	w.DynamicAtlas.UploadToGPU()

	// Carregar texture atlas antigo (backup, caso necessário)
	w.TextureAtlas = w.DynamicAtlas.AtlasTexture

	// Criar material com textura
	w.Material = rl.LoadMaterialDefault()

	// IMPORTANTE: Definir a textura no mapa de difusa do material
	diffuseMap := w.Material.GetMap(rl.MapDiffuse)
	diffuseMap.Texture = w.DynamicAtlas.AtlasTexture

	// Criar meshes com UVs customizadas para cada tipo de bloco
	// NOTA: Essas meshes não são mais usadas com o sistema de chunks
	w.GrassMesh = CreateTexturedCubeMesh(BlockGrass)
}

func (w *World) SetBlock(x, y, z int32, block BlockType) {
	w.ChunkManager.SetBlock(x, y, z, block)
}

func (w *World) GetBlock(x, y, z int32) BlockType {
	return w.ChunkManager.GetBlock(x, y, z)
}

func (w *World) IsBlockHidden(x, y, z int32) bool {
	return w.ChunkManager.IsBlockHidden(x, y, z)
}

// Update atualiza o mundo (carrega/descarrega chunks baseado na posição do jogador)
func (w *World) Update(playerPos rl.Vector3, dt float32) {
	// Atualizar chunks (carrega/descarrega)
	w.ChunkManager.Update(playerPos, dt, w.TerrainGenerator)

	// Gerenciar atlas dinamicamente (apenas quando necessário)
	w.UpdateDynamicAtlas()
}

// UpdateDynamicAtlas atualiza o atlas apenas quando novos chunks foram carregados
func (w *World) UpdateDynamicAtlas() {
	if w.DynamicAtlas == nil {
		return
	}

	// OTIMIZAÇÃO: Só verificar se novos chunks foram carregados
	if !w.ChunkManager.NewChunksLoaded {
		return
	}

	w.ChunkManager.NewChunksLoaded = false
	atlasChanged := false

	// Verificar apenas chunks recém-gerados
	for _, chunk := range w.ChunkManager.Chunks {
		if !chunk.IsGenerated {
			continue
		}

		// Amostragem LEVE: verificar apenas alguns blocos por chunk
		// Isso captura a maioria dos tipos sem custo alto
		uniqueTypes := make(map[BlockType]bool)

		for x := int32(0); x < ChunkSize; x += 8 {
			for y := int32(0); y < ChunkHeight; y += 8 {
				for z := int32(0); z < ChunkSize; z += 8 {
					blockType := chunk.Blocks[x][y][z]
					if blockType != BlockAir {
						uniqueTypes[blockType] = true
					}
				}
			}
		}

		// Verificar se os tipos encontrados estão no atlas
		for blockType := range uniqueTypes {
			w.DynamicAtlas.mu.RLock()
			_, exists := w.DynamicAtlas.BlockToSlot[blockType]
			w.DynamicAtlas.mu.RUnlock()

			if !exists {
				w.DynamicAtlas.AllocateSlot(blockType)
				atlasChanged = true
			}
		}
	}

	// Se o atlas mudou, rebuild
	if atlasChanged {
		w.DynamicAtlas.RebuildAtlas()
		w.DynamicAtlas.UploadToGPU()

		// Atualizar material
		diffuseMap := w.Material.GetMap(rl.MapDiffuse)
		diffuseMap.Texture = w.DynamicAtlas.AtlasTexture

		// Marcar todos os chunks para atualização
		for _, chunk := range w.ChunkManager.Chunks {
			chunk.NeedUpdateMeshes = true
		}
	}
}

func (w *World) Render(playerPos rl.Vector3) {
	w.ChunkManager.Render(w.GrassMesh, w.Material, playerPos, w.VisibleBlocks, w.DynamicAtlas)
}

// GetTotalBlocks retorna o número total de blocos (para debug/UI)
func (w *World) GetTotalBlocks() int {
	return w.ChunkManager.GetTotalBlocks()
}

// GetLoadedChunksCount retorna o número de chunks carregados (para debug/UI)
func (w *World) GetLoadedChunksCount() int {
	return w.ChunkManager.GetLoadedChunksCount()
}
