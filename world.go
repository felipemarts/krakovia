package main

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

// World representa o mundo voxel com sistema de chunks
type World struct {
	ChunkManager   *ChunkManager
	GrassMesh      rl.Mesh
	DirtMesh       rl.Mesh
	StoneMesh      rl.Mesh
	Material       rl.Material
	TextureAtlas   rl.Texture2D
	RenderDistance int32
}

func NewWorld() *World {
	renderDistance := int32(2) // Carregar chunks em um raio de 2 chunks com carregamento gradual
	w := &World{
		ChunkManager:   NewChunkManager(renderDistance),
		RenderDistance: renderDistance,
	}
	return w
}

// InitWorldGraphics inicializa recursos gráficos do mundo (deve ser chamado após rl.InitWindow)
func (w *World) InitWorldGraphics() {
	// Carregar texture atlas
	w.TextureAtlas = rl.LoadTexture("texture_atlas.png")

	// Configurar filtro de textura para pixel art (sem blur)
	rl.SetTextureFilter(w.TextureAtlas, rl.FilterPoint)

	// Criar material com textura
	w.Material = rl.LoadMaterialDefault()

	// IMPORTANTE: Definir a textura no mapa de difusa do material
	diffuseMap := w.Material.GetMap(rl.MapDiffuse)
	diffuseMap.Texture = w.TextureAtlas

	// Criar meshes com UVs customizadas para cada tipo de bloco
	w.GrassMesh = CreateTexturedCubeMesh(BlockGrass)
	w.DirtMesh = CreateTexturedCubeMesh(BlockDirt)
	w.StoneMesh = CreateTexturedCubeMesh(BlockStone)
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
	w.ChunkManager.Update(playerPos, dt)
}

func (w *World) Render(playerPos rl.Vector3) {
	w.ChunkManager.Render(w.GrassMesh, w.DirtMesh, w.StoneMesh, w.Material, playerPos)
}

// GetTotalBlocks retorna o número total de blocos (para debug/UI)
func (w *World) GetTotalBlocks() int {
	return w.ChunkManager.GetTotalBlocks()
}

// GetLoadedChunksCount retorna o número de chunks carregados (para debug/UI)
func (w *World) GetLoadedChunksCount() int {
	return w.ChunkManager.GetLoadedChunksCount()
}
