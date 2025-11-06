package main

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// World representa o mundo voxel
type World struct {
	Blocks           map[int64]BlockType
	SizeX            int32
	SizeY            int32
	SizeZ            int32
	GrassMesh        rl.Mesh
	DirtMesh         rl.Mesh
	StoneMesh        rl.Mesh
	Material         rl.Material
	TextureAtlas     rl.Texture2D
	// Instanced rendering: transforms agrupados por tipo de bloco
	GrassTransforms  []rl.Matrix
	DirtTransforms   []rl.Matrix
	StoneTransforms  []rl.Matrix
	NeedUpdateMeshes bool
}

func NewWorld() *World {
	w := &World{
		Blocks:           make(map[int64]BlockType),
		SizeX:            32,
		SizeY:            64,
		SizeZ:            32,
		NeedUpdateMeshes: true,
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

func (w *World) GetBlockIndex(x, y, z int32) int64 {
	return int64(x) | (int64(y) << 20) | (int64(z) << 40)
}

func (w *World) SetBlock(x, y, z int32, block BlockType) {
	if x < 0 || x >= w.SizeX || y < 0 || y >= w.SizeY || z < 0 || z >= w.SizeZ {
		return
	}

	idx := w.GetBlockIndex(x, y, z)
	if block == BlockAir {
		delete(w.Blocks, idx)
	} else {
		w.Blocks[idx] = block
	}

	// Marcar que as meshes precisam ser atualizadas
	w.NeedUpdateMeshes = true
}

func (w *World) GetBlock(x, y, z int32) BlockType {
	if x < 0 || x >= w.SizeX || y < 0 || y >= w.SizeY || z < 0 || z >= w.SizeZ {
		return BlockAir
	}

	idx := w.GetBlockIndex(x, y, z)
	if block, exists := w.Blocks[idx]; exists {
		return block
	}
	return BlockAir
}

func (w *World) GenerateTerrain() {
	// Gerar terreno simples
	for x := int32(0); x < w.SizeX; x++ {
		for z := int32(0); z < w.SizeZ; z++ {
			// Altura base + variação simples
			height := int32(10 + int32(math.Sin(float64(x)*0.3)*3+math.Cos(float64(z)*0.3)*3))

			for y := int32(0); y <= height; y++ {
				if y == height {
					w.SetBlock(x, y, z, BlockGrass)
				} else if y >= height-3 {
					w.SetBlock(x, y, z, BlockDirt)
				} else {
					w.SetBlock(x, y, z, BlockStone)
				}
			}
		}
	}
}

// UpdateMeshes atualiza os arrays de transformações agrupados por tipo de bloco
func (w *World) UpdateMeshes() {
	// Limpar arrays (reutilizar memória)
	w.GrassTransforms = w.GrassTransforms[:0]
	w.DirtTransforms = w.DirtTransforms[:0]
	w.StoneTransforms = w.StoneTransforms[:0]

	// Iterar por todos os blocos e criar matrizes de transformação agrupadas por tipo
	for idx, blockType := range w.Blocks {
		if blockType == BlockAir {
			continue
		}

		// Decodificar posição
		x := int32(idx & 0xFFFFF)
		y := int32((idx >> 20) & 0xFFFFF)
		z := int32((idx >> 40) & 0xFFFFF)

		// Ajustar para valores negativos se necessário
		if x >= 0x80000 {
			x -= 0x100000
		}
		if y >= 0x80000 {
			y -= 0x100000
		}
		if z >= 0x80000 {
			z -= 0x100000
		}

		// Criar matriz de transformação (translação para a posição do bloco)
		// Centralizar o cubo (+0.5 em cada eixo)
		transform := rl.MatrixTranslate(float32(x)+0.5, float32(y)+0.5, float32(z)+0.5)

		// Adicionar ao array correspondente ao tipo de bloco
		switch blockType {
		case BlockGrass:
			w.GrassTransforms = append(w.GrassTransforms, transform)
		case BlockDirt:
			w.DirtTransforms = append(w.DirtTransforms, transform)
		case BlockStone:
			w.StoneTransforms = append(w.StoneTransforms, transform)
		}
	}

	w.NeedUpdateMeshes = false
}

func (w *World) Render() {
	// Atualizar meshes se necessário
	if w.NeedUpdateMeshes {
		w.UpdateMeshes()
	}

	// Renderizar blocos de grama (1 draw call para todos)
	if len(w.GrassTransforms) > 0 {
		DrawMeshInstanced(w.GrassMesh, w.Material, w.GrassTransforms)
	}

	// Renderizar blocos de terra (1 draw call para todos)
	if len(w.DirtTransforms) > 0 {
		DrawMeshInstanced(w.DirtMesh, w.Material, w.DirtTransforms)
	}

	// Renderizar blocos de pedra (1 draw call para todos)
	if len(w.StoneTransforms) > 0 {
		DrawMeshInstanced(w.StoneMesh, w.Material, w.StoneTransforms)
	}
}
