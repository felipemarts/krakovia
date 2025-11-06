package main

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	ChunkSize   = 32
	RenderDist  = 4  // Chunks para carregar ao redor do player
	BlockSize   = 1.0
)

// BlockType representa o tipo de bloco
type BlockType uint8

const (
	BlockAir BlockType = iota
	BlockGrass
	BlockDirt
	BlockStone
	BlockWood
	BlockLeaves
)

// MinecraftChunk representa um chunk de 32x32x32 blocos
type MinecraftChunk struct {
	Position  rl.Vector3 // Posição do chunk no mundo (em coordenadas de chunk)
	Blocks    [ChunkSize][ChunkSize][ChunkSize]BlockType
	Mesh      rl.Mesh
	Model     rl.Model
	IsDirty   bool // Precisa recriar mesh?
	IsActive  bool // Está carregado?
	// Manter dados da mesh em memória para evitar garbage collection
	Vertices  []float32
	Texcoords []float32
	Normals   []float32
}

// NewMinecraftChunk cria um novo chunk
func NewMinecraftChunk(chunkX, chunkY, chunkZ int32) *MinecraftChunk {
	chunk := &MinecraftChunk{
		Position: rl.NewVector3(float32(chunkX), float32(chunkY), float32(chunkZ)),
		IsDirty:  true,
		IsActive: true,
	}
	return chunk
}

// GetBlock retorna o tipo de bloco em uma posição local do chunk
func (c *MinecraftChunk) GetBlock(x, y, z int32) BlockType {
	if x < 0 || x >= ChunkSize || y < 0 || y >= ChunkSize || z < 0 || z >= ChunkSize {
		return BlockAir
	}
	return c.Blocks[x][y][z]
}

// SetBlock define um bloco em uma posição local do chunk
func (c *MinecraftChunk) SetBlock(x, y, z int32, blockType BlockType) {
	if x < 0 || x >= ChunkSize || y < 0 || y >= ChunkSize || z < 0 || z >= ChunkSize {
		return
	}
	c.Blocks[x][y][z] = blockType
	c.IsDirty = true
}

// GetWorldPosition converte posição local do chunk para posição mundial
func (c *MinecraftChunk) GetWorldPosition(x, y, z int32) rl.Vector3 {
	return rl.NewVector3(
		c.Position.X*ChunkSize+float32(x),
		c.Position.Y*ChunkSize+float32(y),
		c.Position.Z*ChunkSize+float32(z),
	)
}

// GenerateTerrain gera o terreno do chunk com altura variável
func (c *MinecraftChunk) GenerateTerrain() {
	worldX := int32(c.Position.X)
	worldZ := int32(c.Position.Z)

	for x := int32(0); x < ChunkSize; x++ {
		for z := int32(0); z < ChunkSize; z++ {
			// Usar simplex/perlin noise simplificado com sin/cos
			wx := float64(worldX*ChunkSize + x)
			wz := float64(worldZ*ChunkSize + z)

			// Múltiplas octavas para terreno mais interessante
			height := 0.0
			height += math.Sin(wx*0.02) * 8.0
			height += math.Cos(wz*0.02) * 8.0
			height += math.Sin(wx*0.05) * 4.0
			height += math.Cos(wz*0.05) * 4.0
			height += math.Sin(wx*0.1) * 2.0

			terrainHeight := int32(height) + 16 // Altura base = 16

			for y := int32(0); y < ChunkSize; y++ {
				worldY := int32(c.Position.Y)*ChunkSize + y

				if worldY < terrainHeight-3 {
					c.Blocks[x][y][z] = BlockStone
				} else if worldY < terrainHeight-1 {
					c.Blocks[x][y][z] = BlockDirt
				} else if worldY == terrainHeight-1 {
					c.Blocks[x][y][z] = BlockGrass
				} else {
					c.Blocks[x][y][z] = BlockAir
				}
			}
		}
	}

	c.IsDirty = true
}

// GetBlockColor retorna a cor de um tipo de bloco
func GetBlockColor(blockType BlockType) rl.Color {
	switch blockType {
	case BlockGrass:
		return rl.NewColor(34, 139, 34, 255)
	case BlockDirt:
		return rl.NewColor(139, 90, 43, 255)
	case BlockStone:
		return rl.NewColor(128, 128, 128, 255)
	case BlockWood:
		return rl.NewColor(139, 90, 0, 255)
	case BlockLeaves:
		return rl.NewColor(0, 128, 0, 255)
	default:
		return rl.White
	}
}

// ShouldRenderFace verifica se uma face deve ser renderizada (face culling)
func (c *MinecraftChunk) ShouldRenderFace(x, y, z int32, neighbor BlockType) bool {
	return neighbor == BlockAir
}

// BuildMesh cria a mesh do chunk com face culling e texturas
func (c *MinecraftChunk) BuildMesh(world *MinecraftWorld) {
	if !c.IsDirty {
		return
	}

	// Usar arrays da struct para evitar garbage collection
	c.Vertices = make([]float32, 0, 10000)
	c.Texcoords = make([]float32, 0, 10000)
	c.Normals = make([]float32, 0, 10000)

	for x := int32(0); x < ChunkSize; x++ {
		for y := int32(0); y < ChunkSize; y++ {
			for z := int32(0); z < ChunkSize; z++ {
				block := c.GetBlock(x, y, z)
				if block == BlockAir {
					continue
				}

				worldPos := c.GetWorldPosition(x, y, z)

				// Verificar cada face e adicionar apenas as visíveis
				// Face superior (+Y)
				if c.ShouldRenderFace(x, y, z, world.GetBlock(worldPos.X, worldPos.Y+1, worldPos.Z)) {
					c.addFaceVertices(&c.Vertices, &c.Texcoords, &c.Normals, worldPos, 0, block)
				}
				// Face inferior (-Y)
				if c.ShouldRenderFace(x, y, z, world.GetBlock(worldPos.X, worldPos.Y-1, worldPos.Z)) {
					c.addFaceVertices(&c.Vertices, &c.Texcoords, &c.Normals, worldPos, 1, block)
				}
				// Face frente (+Z)
				if c.ShouldRenderFace(x, y, z, world.GetBlock(worldPos.X, worldPos.Y, worldPos.Z+1)) {
					c.addFaceVertices(&c.Vertices, &c.Texcoords, &c.Normals, worldPos, 2, block)
				}
				// Face trás (-Z)
				if c.ShouldRenderFace(x, y, z, world.GetBlock(worldPos.X, worldPos.Y, worldPos.Z-1)) {
					c.addFaceVertices(&c.Vertices, &c.Texcoords, &c.Normals, worldPos, 3, block)
				}
				// Face direita (+X)
				if c.ShouldRenderFace(x, y, z, world.GetBlock(worldPos.X+1, worldPos.Y, worldPos.Z)) {
					c.addFaceVertices(&c.Vertices, &c.Texcoords, &c.Normals, worldPos, 4, block)
				}
				// Face esquerda (-X)
				if c.ShouldRenderFace(x, y, z, world.GetBlock(worldPos.X-1, worldPos.Y, worldPos.Z)) {
					c.addFaceVertices(&c.Vertices, &c.Texcoords, &c.Normals, worldPos, 5, block)
				}
			}
		}
	}

	// Descarregar mesh antiga se existir
	if c.Mesh.VertexCount > 0 {
		rl.UnloadMesh(&c.Mesh)
		c.Mesh = rl.Mesh{}
	}

	// Se modelo antigo existe, resetar (mas não descarregar material pois ele tem a textura)
	if c.Model.MeshCount > 0 {
		c.Model = rl.Model{}
	}

	if len(c.Vertices) == 0 {
		c.IsDirty = false
		c.Mesh = rl.Mesh{}
		c.Model = rl.Model{}
		return
	}

	// Criar nova mesh
	mesh := rl.Mesh{}
	mesh.VertexCount = int32(len(c.Vertices) / 3)
	mesh.TriangleCount = mesh.VertexCount / 3

	// IMPORTANTE: Usar ponteiros para dados armazenados na struct
	mesh.Vertices = &c.Vertices[0]
	mesh.Texcoords = &c.Texcoords[0]
	mesh.Normals = &c.Normals[0]

	// Upload com dynamic=true para copiar dados para VRAM
	rl.UploadMesh(&mesh, true)

	c.Mesh = mesh
	c.Model = rl.LoadModelFromMesh(mesh)

	// Aplicar textura do atlas ao modelo
	if world.AtlasTexture.ID != 0 {
		rl.SetMaterialTexture(c.Model.Materials, rl.MapDiffuse, world.AtlasTexture)
	}

	c.IsDirty = false
}

// addFaceVertices adiciona vértices de uma face do cubo com texturas
func (c *MinecraftChunk) addFaceVertices(vertices, texcoords, normals *[]float32, pos rl.Vector3, face int, blockType BlockType) {
	s := float32(BlockSize)
	x, y, z := pos.X, pos.Y, pos.Z

	// Determinar qual textura usar (0 ou 1 no atlas horizontal)
	texIndex := float32(0)
	if blockType == BlockGrass || blockType == BlockLeaves {
		texIndex = 1 // Verde (lado direito do atlas)
	}

	// Calcular UVs no atlas (atlas 2x1: duas texturas lado a lado)
	uStart := texIndex * 0.5
	uEnd := uStart + 0.5

	// Cada face tem 6 vértices (2 triângulos)
	switch face {
	case 0: // Top (+Y)
		*vertices = append(*vertices,
			x-s/2, y+s/2, z-s/2, x-s/2, y+s/2, z+s/2, x+s/2, y+s/2, z+s/2,
			x-s/2, y+s/2, z-s/2, x+s/2, y+s/2, z+s/2, x+s/2, y+s/2, z-s/2)
		*normals = append(*normals, 0, 1, 0, 0, 1, 0, 0, 1, 0, 0, 1, 0, 0, 1, 0, 0, 1, 0)
		*texcoords = append(*texcoords, uStart, 0, uStart, 1, uEnd, 1, uStart, 0, uEnd, 1, uEnd, 0)
	case 1: // Bottom (-Y)
		*vertices = append(*vertices,
			x-s/2, y-s/2, z-s/2, x+s/2, y-s/2, z+s/2, x-s/2, y-s/2, z+s/2,
			x-s/2, y-s/2, z-s/2, x+s/2, y-s/2, z-s/2, x+s/2, y-s/2, z+s/2)
		*normals = append(*normals, 0, -1, 0, 0, -1, 0, 0, -1, 0, 0, -1, 0, 0, -1, 0, 0, -1, 0)
		*texcoords = append(*texcoords, uStart, 0, uEnd, 1, uStart, 1, uStart, 0, uEnd, 0, uEnd, 1)
	case 2: // Front (+Z)
		*vertices = append(*vertices,
			x-s/2, y-s/2, z+s/2, x-s/2, y+s/2, z+s/2, x+s/2, y+s/2, z+s/2,
			x-s/2, y-s/2, z+s/2, x+s/2, y+s/2, z+s/2, x+s/2, y-s/2, z+s/2)
		*normals = append(*normals, 0, 0, 1, 0, 0, 1, 0, 0, 1, 0, 0, 1, 0, 0, 1, 0, 0, 1)
		*texcoords = append(*texcoords, uStart, 1, uStart, 0, uEnd, 0, uStart, 1, uEnd, 0, uEnd, 1)
	case 3: // Back (-Z)
		*vertices = append(*vertices,
			x-s/2, y-s/2, z-s/2, x+s/2, y+s/2, z-s/2, x-s/2, y+s/2, z-s/2,
			x-s/2, y-s/2, z-s/2, x+s/2, y-s/2, z-s/2, x+s/2, y+s/2, z-s/2)
		*normals = append(*normals, 0, 0, -1, 0, 0, -1, 0, 0, -1, 0, 0, -1, 0, 0, -1, 0, 0, -1)
		*texcoords = append(*texcoords, uEnd, 1, uStart, 0, uEnd, 0, uEnd, 1, uStart, 1, uStart, 0)
	case 4: // Right (+X)
		*vertices = append(*vertices,
			x+s/2, y-s/2, z-s/2, x+s/2, y+s/2, z-s/2, x+s/2, y+s/2, z+s/2,
			x+s/2, y-s/2, z-s/2, x+s/2, y+s/2, z+s/2, x+s/2, y-s/2, z+s/2)
		*normals = append(*normals, 1, 0, 0, 1, 0, 0, 1, 0, 0, 1, 0, 0, 1, 0, 0, 1, 0, 0)
		*texcoords = append(*texcoords, uEnd, 1, uEnd, 0, uStart, 0, uEnd, 1, uStart, 0, uStart, 1)
	case 5: // Left (-X)
		*vertices = append(*vertices,
			x-s/2, y-s/2, z-s/2, x-s/2, y+s/2, z+s/2, x-s/2, y+s/2, z-s/2,
			x-s/2, y-s/2, z-s/2, x-s/2, y-s/2, z+s/2, x-s/2, y+s/2, z+s/2)
		*normals = append(*normals, -1, 0, 0, -1, 0, 0, -1, 0, 0, -1, 0, 0, -1, 0, 0, -1, 0, 0)
		*texcoords = append(*texcoords, uStart, 1, uEnd, 0, uStart, 0, uStart, 1, uEnd, 1, uEnd, 0)
	}
}

// Draw renderiza o chunk
func (c *MinecraftChunk) Draw() {
	if !c.IsActive || c.Model.MeshCount == 0 {
		return
	}

	rl.DrawModel(c.Model, rl.NewVector3(0, 0, 0), 1.0, rl.White)
}

// Unload libera recursos do chunk
func (c *MinecraftChunk) Unload() {
	// Descarregar mesh
	if c.Mesh.VertexCount > 0 {
		rl.UnloadMesh(&c.Mesh)
		c.Mesh = rl.Mesh{}
	}

	// Descarregar materiais do modelo
	if c.Model.MeshCount > 0 && c.Model.MaterialCount > 0 {
		rl.UnloadMaterial(*c.Model.Materials)
	}

	c.Model = rl.Model{}
	c.IsActive = false

	// Limpar dados
	c.Vertices = nil
	c.Texcoords = nil
	c.Normals = nil
}
