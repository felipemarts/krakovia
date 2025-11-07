package main

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	ChunkSize   = 32
	ChunkHeight = 32
)

// ChunkCoord representa as coordenadas de um chunk no mundo
type ChunkCoord struct {
	X, Y, Z int32
}

// Chunk representa um pedaço 32x32x32 do mundo
type Chunk struct {
	Coord            ChunkCoord
	Blocks           [ChunkSize][ChunkHeight][ChunkSize]BlockType
	GrassTransforms  []rl.Matrix
	DirtTransforms   []rl.Matrix
	StoneTransforms  []rl.Matrix
	NeedUpdateMeshes bool
	IsGenerated      bool
}

// NewChunk cria um novo chunk nas coordenadas especificadas
func NewChunk(x, y, z int32) *Chunk {
	// Pré-alocar arrays de transforms com capacidade estimada
	// Para um mundo plano, cada chunk terá no máximo ChunkSize*ChunkSize blocos (1024)
	estimatedCapacity := ChunkSize * ChunkSize

	return &Chunk{
		Coord:            ChunkCoord{X: x, Y: y, Z: z},
		GrassTransforms:  make([]rl.Matrix, 0, estimatedCapacity),
		DirtTransforms:   make([]rl.Matrix, 0, estimatedCapacity),
		StoneTransforms:  make([]rl.Matrix, 0, estimatedCapacity),
		NeedUpdateMeshes: true,
		IsGenerated:      false,
	}
}

// GetBlock retorna o tipo de bloco nas coordenadas locais do chunk (0-31)
func (c *Chunk) GetBlock(x, y, z int32) BlockType {
	if x < 0 || x >= ChunkSize || y < 0 || y >= ChunkHeight || z < 0 || z >= ChunkSize {
		return BlockAir
	}
	return c.Blocks[x][y][z]
}

// SetBlock define o tipo de bloco nas coordenadas locais do chunk (0-31)
func (c *Chunk) SetBlock(x, y, z int32, block BlockType) {
	if x < 0 || x >= ChunkSize || y < 0 || y >= ChunkHeight || z < 0 || z >= ChunkSize {
		return
	}
	c.Blocks[x][y][z] = block
	c.NeedUpdateMeshes = true
}

// GenerateTerrain gera o terreno para este chunk
func (c *Chunk) GenerateTerrain() {
	// Posição mundial do chunk
	worldY := c.Coord.Y * ChunkHeight

	// Mundo plano com apenas 1 bloco de espessura na altura y=10
	flatHeight := int32(10)

	// Verificar se este chunk contém a altura do plano
	if worldY <= flatHeight && worldY+ChunkHeight > flatHeight {
		// Calcular y local dentro do chunk
		localY := flatHeight - worldY

		// Criar camada plana de grama diretamente (sem SetBlock para evitar marcar NeedUpdateMeshes múltiplas vezes)
		for x := int32(0); x < ChunkSize; x++ {
			for z := int32(0); z < ChunkSize; z++ {
				c.Blocks[x][localY][z] = BlockGrass
			}
		}
	}

	c.IsGenerated = true
	c.NeedUpdateMeshes = true
	// Gerar meshes imediatamente após gerar terreno
	c.UpdateMeshes()
}

// UpdateMeshes atualiza os arrays de transformações agrupados por tipo de bloco
func (c *Chunk) UpdateMeshes() {
	// Limpar arrays (reutilizar memória)
	c.GrassTransforms = c.GrassTransforms[:0]
	c.DirtTransforms = c.DirtTransforms[:0]
	c.StoneTransforms = c.StoneTransforms[:0]

	// Posição mundial do chunk
	worldX := c.Coord.X * ChunkSize
	worldY := c.Coord.Y * ChunkHeight
	worldZ := c.Coord.Z * ChunkSize

	// Iterar por todos os blocos do chunk
	for x := int32(0); x < ChunkSize; x++ {
		for y := int32(0); y < ChunkHeight; y++ {
			for z := int32(0); z < ChunkSize; z++ {
				blockType := c.Blocks[x][y][z]
				if blockType == BlockAir {
					continue
				}

				// Calcular posição mundial do bloco
				wx := float32(worldX + x)
				wy := float32(worldY + y)
				wz := float32(worldZ + z)

				// Criar matriz de transformação (translação para a posição do bloco)
				// Centralizar o cubo (+0.5 em cada eixo)
				transform := rl.MatrixTranslate(wx+0.5, wy+0.5, wz+0.5)

				// Adicionar ao array correspondente ao tipo de bloco
				switch blockType {
				case BlockGrass:
					c.GrassTransforms = append(c.GrassTransforms, transform)
				case BlockDirt:
					c.DirtTransforms = append(c.DirtTransforms, transform)
				case BlockStone:
					c.StoneTransforms = append(c.StoneTransforms, transform)
				}
			}
		}
	}

	c.NeedUpdateMeshes = false
}

// Render renderiza o chunk usando instanced rendering
func (c *Chunk) Render(grassMesh, dirtMesh, stoneMesh rl.Mesh, material rl.Material) {
	// Atualizar meshes se necessário
	if c.NeedUpdateMeshes {
		c.UpdateMeshes()
	}

	// Renderizar blocos de grama (1 draw call para todos)
	if len(c.GrassTransforms) > 0 {
		DrawMeshInstanced(grassMesh, material, c.GrassTransforms)
	}

	// Renderizar blocos de terra (1 draw call para todos)
	if len(c.DirtTransforms) > 0 {
		DrawMeshInstanced(dirtMesh, material, c.DirtTransforms)
	}

	// Renderizar blocos de pedra (1 draw call para todos)
	if len(c.StoneTransforms) > 0 {
		DrawMeshInstanced(stoneMesh, material, c.StoneTransforms)
	}
}

// GetChunkCoord retorna as coordenadas do chunk que contém a posição mundial
func GetChunkCoord(worldX, worldY, worldZ int32) ChunkCoord {
	// Para coordenadas negativas, precisamos ajustar o cálculo
	// Exemplo: -1 / 32 = 0 (errado), mas (-1 - 31) / 32 = -1 (correto)
	cx := worldX / ChunkSize
	if worldX < 0 && worldX%ChunkSize != 0 {
		cx--
	}

	cy := worldY / ChunkHeight
	if worldY < 0 && worldY%ChunkHeight != 0 {
		cy--
	}

	cz := worldZ / ChunkSize
	if worldZ < 0 && worldZ%ChunkSize != 0 {
		cz--
	}

	return ChunkCoord{X: cx, Y: cy, Z: cz}
}

// GetChunkCoordFromFloat retorna as coordenadas do chunk que contém a posição mundial (float)
func GetChunkCoordFromFloat(worldX, worldY, worldZ float32) ChunkCoord {
	// Usar math.Floor corretamente para floats
	cx := int32(math.Floor(float64(worldX) / float64(ChunkSize)))
	cy := int32(math.Floor(float64(worldY) / float64(ChunkHeight)))
	cz := int32(math.Floor(float64(worldZ) / float64(ChunkSize)))
	return ChunkCoord{X: cx, Y: cy, Z: cz}
}

// ChunkKey gera uma chave única para um chunk baseada em suas coordenadas
func (cc ChunkCoord) Key() int64 {
	return int64(cc.X) | (int64(cc.Y) << 20) | (int64(cc.Z) << 40)
}
