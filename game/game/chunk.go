package game

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
	ChunkMesh        *ChunkMesh  // Mesh combinada de todo o chunk
	ChunkAtlas       *ChunkAtlas // Atlas de texturas específico deste chunk
	NeedUpdateMeshes bool
	IsGenerated      bool
}

// NewChunk cria um novo chunk nas coordenadas especificadas
func NewChunk(x, y, z int32) *Chunk {
	return &Chunk{
		Coord:            ChunkCoord{X: x, Y: y, Z: z},
		ChunkMesh:        NewChunkMesh(),
		ChunkAtlas:       NewChunkAtlas(16, 32), // Atlas 8x8 = 64 slots
		NeedUpdateMeshes: true,
		IsGenerated:      false,
	}
}

// GetBlock retorna o tipo de bloco nas coordenadas locais do chunk (0-31)
func (c *Chunk) GetBlock(x, y, z int32) BlockType {
	if x < 0 || x >= ChunkSize || y < 0 || y >= ChunkHeight || z < 0 || z >= ChunkSize {
		return NoBlock
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

// GenerateTerrain gera o terreno para este chunk (versão antiga - mantida para compatibilidade)
func (c *Chunk) GenerateTerrain() {
	// Posição mundial do chunk
	worldX := c.Coord.X * ChunkSize
	worldZ := c.Coord.Z * ChunkSize
	worldY := c.Coord.Y * ChunkHeight

	// Altura base do terreno
	baseHeight := int32(10)

	// Gerar terreno apenas se este chunk pode conter o terreno
	if worldY <= baseHeight+3 && worldY+ChunkHeight > baseHeight-3 {
		for x := int32(0); x < ChunkSize; x++ {
			for z := int32(0); z < ChunkSize; z++ {
				// Calcular posição mundial do bloco
				wx := worldX + x
				wz := worldZ + z

				// Usar noise simples baseado em seno para criar ondulações
				// Combinar múltiplas frequências para terreno mais interessante
				noise := math.Sin(float64(wx)*0.1) * math.Cos(float64(wz)*0.1)
				noise += math.Sin(float64(wx)*0.05) * math.Cos(float64(wz)*0.05) * 0.5

				// Converter noise (-1 a 1) para variação de altura (0 a 3 blocos)
				heightVariation := int32(noise * 1.5)
				terrainHeight := baseHeight + heightVariation

				// Preencher blocos até a altura do terreno
				for y := int32(0); y < ChunkHeight; y++ {
					worldBlockY := worldY + y

					if worldBlockY <= terrainHeight {
						// Usar apenas um tipo de bloco
						c.Blocks[x][y][z] = BlockType(DefaultBlockID)
					}
				}
			}
		}
	}

	c.IsGenerated = true
	c.NeedUpdateMeshes = true
	// Meshes serão atualizadas no primeiro render
}

// GenerateTerrainWithGenerator gera terreno usando TerrainGenerator
func (c *Chunk) GenerateTerrainWithGenerator(tg *TerrainGenerator) {
	worldX := c.Coord.X * ChunkSize
	worldY := c.Coord.Y * ChunkHeight
	worldZ := c.Coord.Z * ChunkSize

	for x := int32(0); x < ChunkSize; x++ {
		for y := int32(0); y < ChunkHeight; y++ {
			for z := int32(0); z < ChunkSize; z++ {
				wx := worldX + x
				wy := worldY + y
				wz := worldZ + z

				blockType := tg.GetBlockTypeAt(wx, wy, wz)
				c.Blocks[x][y][z] = blockType
			}
		}
	}

	c.IsGenerated = true
	c.NeedUpdateMeshes = true
}

// IsBlockHiddenLocal verifica oclusão apenas dentro do chunk (otimização parcial)
// NOTA: Não considera chunks vizinhos - use UpdateMeshesWithNeighbors para oclusão completa
func (c *Chunk) IsBlockHiddenLocal(x, y, z int32) bool {
	// Verificar todas as 6 direções (cima, baixo, norte, sul, leste, oeste)
	directions := []struct{ dx, dy, dz int32 }{
		{1, 0, 0},  // Direita
		{-1, 0, 0}, // Esquerda
		{0, 1, 0},  // Cima
		{0, -1, 0}, // Baixo
		{0, 0, 1},  // Frente
		{0, 0, -1}, // Trás
	}

	for _, dir := range directions {
		nx, ny, nz := x+dir.dx, y+dir.dy, z+dir.dz

		// Se o vizinho está fora do chunk, considerar como exposto (visível)
		if nx < 0 || nx >= ChunkSize || ny < 0 || ny >= ChunkHeight || nz < 0 || nz >= ChunkSize {
			return false
		}

		// Se o vizinho é ar, o bloco está exposto (visível)
		if c.Blocks[nx][ny][nz] == NoBlock {
			return false
		}
	}

	// Todas as 6 faces estão bloqueadas - bloco está completamente oculto
	return true
}

// UpdateMeshes atualiza a mesh sem considerar chunks vizinhos (fallback)
func (c *Chunk) UpdateMeshes(atlas *DynamicAtlasManager) {
	// Usar a versão com vizinhos, mas retornar NoBlock para blocos fora do chunk
	c.UpdateMeshesWithNeighbors(func(x, y, z int32) BlockType {
		// Converter para coordenadas locais
		localX := x - c.Coord.X*ChunkSize
		localY := y - c.Coord.Y*ChunkHeight
		localZ := z - c.Coord.Z*ChunkSize

		// Se está fora do chunk, retornar ar
		if localX < 0 || localX >= ChunkSize || localY < 0 || localY >= ChunkHeight || localZ < 0 || localZ >= ChunkSize {
			return NoBlock
		}

		return c.Blocks[localX][localY][localZ]
	}, atlas)
}

// UpdateMeshesWithNeighbors atualiza meshes considerando chunks vizinhos
func (c *Chunk) UpdateMeshesWithNeighbors(getBlockFunc func(x, y, z int32) BlockType, globalAtlas *DynamicAtlasManager) {
	// Limpar mesh anterior
	c.ChunkMesh.Clear()

	// Resetar atlas do chunk
	c.ChunkAtlas.UsedBlocks = make(map[BlockType]int32)
	c.ChunkAtlas.BlockOrder = make([]BlockType, 0)
	c.ChunkAtlas.NeedsRebuild = true

	// Posição mundial do chunk
	worldX := c.Coord.X * ChunkSize
	worldY := c.Coord.Y * ChunkHeight
	worldZ := c.Coord.Z * ChunkSize

	// Direções das 6 faces: +X, -X, +Y, -Y, +Z, -Z
	directions := []struct{ dx, dy, dz int32 }{
		{1, 0, 0},  // 0: Face +X (direita)
		{-1, 0, 0}, // 1: Face -X (esquerda)
		{0, 1, 0},  // 2: Face +Y (topo)
		{0, -1, 0}, // 3: Face -Y (fundo)
		{0, 0, 1},  // 4: Face +Z (frente)
		{0, 0, -1}, // 5: Face -Z (trás)
	}

	// Estrutura para armazenar faces visíveis
	type visibleFace struct {
		wx, wy, wz int32
		faceIndex  int
		blockType  BlockType
	}
	var visibleFaces []visibleFace

	// PASSAGEM 1: Coletar todos os BlockTypes e faces visíveis
	for x := int32(0); x < ChunkSize; x++ {
		for y := int32(0); y < ChunkHeight; y++ {
			for z := int32(0); z < ChunkSize; z++ {
				blockType := c.Blocks[x][y][z]
				if blockType == NoBlock {
					continue
				}

				// Adicionar tipo de bloco ao atlas do chunk
				c.ChunkAtlas.AddBlockType(blockType)

				// Calcular posição mundial do bloco
				wx := worldX + x
				wy := worldY + y
				wz := worldZ + z

				// Para cada face, verificar se está exposta
				for faceIndex, dir := range directions {
					neighborBlock := getBlockFunc(wx+dir.dx, wy+dir.dy, wz+dir.dz)

					// Se o vizinho é ar, a face está exposta
					if neighborBlock == NoBlock {
						// Para blocos customizados, usar textura específica da face
						if IsCustomBlock(blockType) {
							faceBlockType := EncodeCustomBlockFace(GetCustomBlockID(blockType), BlockFace(faceIndex))
							c.ChunkAtlas.AddBlockType(faceBlockType)
							visibleFaces = append(visibleFaces, visibleFace{wx, wy, wz, faceIndex, faceBlockType})
						} else {
							visibleFaces = append(visibleFaces, visibleFace{wx, wy, wz, faceIndex, blockType})
						}
					}
				}
			}
		}
	}

	// Rebuildar atlas do chunk ANTES de gerar a mesh
	// Isso garante que o GridSize esteja correto para o cálculo dos UVs
	if globalAtlas != nil {
		c.ChunkAtlas.RebuildAtlas(globalAtlas.TextureCache)
		c.ChunkAtlas.UploadToGPU()
	}

	// PASSAGEM 2: Gerar a mesh com os UVs corretos (agora que o GridSize foi atualizado)
	for _, face := range visibleFaces {
		c.ChunkMesh.AddQuadWithChunkAtlas(float32(face.wx), float32(face.wy), float32(face.wz), face.faceIndex, face.blockType, c.ChunkAtlas)
	}

	// Upload mesh para GPU
	c.ChunkMesh.UploadToGPU()

	c.NeedUpdateMeshes = false
}

// Render renderiza o chunk usando mesh combinada
// NOTA: A atualização de meshes agora é feita no ChunkManager.Render() com limite por frame
func (c *Chunk) Render(grassMesh, dirtMesh, stoneMesh rl.Mesh, material rl.Material, getBlockFunc func(x, y, z int32) BlockType) {
	// Renderizar mesh combinada (1 draw call para TODO o chunk!)
	if c.ChunkMesh.Uploaded {
		rl.DrawMesh(c.ChunkMesh.Mesh, material, rl.MatrixIdentity())
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
	// Usar deslocamento com máscaras para evitar colisões com números negativos
	// Cada coordenada tem 20 bits (suporta valores de -524288 a 524287)
	const mask20 = 0xFFFFF // 20 bits

	// Converter para não-negativo adicionando offset
	// Isso mapeia -524288 para 0, 0 para 524288, etc.
	const offset = 524288 // 2^19

	x := int64(cc.X+offset) & mask20
	y := int64(cc.Y+offset) & mask20
	z := int64(cc.Z+offset) & mask20

	return x | (y << 20) | (z << 40)
}
