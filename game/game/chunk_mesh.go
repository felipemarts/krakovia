package game

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

// DisableGPUUploadForTesting desabilita upload de GPU para testes
// Esta variável deve ser definida para true em ambientes de teste
var DisableGPUUploadForTesting = false

// ChunkMesh representa uma mesh customizada para um chunk
type ChunkMesh struct {
	Vertices  []float32
	Texcoords []float32
	Normals   []float32
	Indices   []uint16
	Mesh      rl.Mesh
	Uploaded  bool
}

// NewChunkMesh cria uma nova mesh vazia para um chunk
func NewChunkMesh() *ChunkMesh {
	return &ChunkMesh{
		Vertices:  make([]float32, 0, 10000),
		Texcoords: make([]float32, 0, 10000),
		Normals:   make([]float32, 0, 10000),
		Indices:   make([]uint16, 0, 10000),
		Uploaded:  false,
	}
}

// AddQuad adiciona um quad (face de bloco) à mesh
func (cm *ChunkMesh) AddQuad(x, y, z float32, face int, blockType BlockType, atlas *DynamicAtlasManager) {
	var uMin, vMin, uMax, vMax float32
	if atlas != nil {
		uMin, vMin, uMax, vMax = atlas.GetBlockUVs(blockType)
	} else {
		uMin, vMin, uMax, vMax = GetBlockUVs(blockType)
	}

	vertexOffset := uint16(len(cm.Vertices) / 3)

	// Definir vértices e normais baseado na face
	switch face {
	case 0: // Face +X (direita)
		cm.Vertices = append(cm.Vertices,
			x+1, y, z, // 0
			x+1, y+1, z, // 1
			x+1, y+1, z+1, // 2
			x+1, y, z+1, // 3
		)
		cm.Normals = append(cm.Normals,
			1, 0, 0,
			1, 0, 0,
			1, 0, 0,
			1, 0, 0,
		)

	case 1: // Face -X (esquerda)
		cm.Vertices = append(cm.Vertices,
			x, y, z+1, // 0
			x, y+1, z+1, // 1
			x, y+1, z, // 2
			x, y, z, // 3
		)
		cm.Normals = append(cm.Normals,
			-1, 0, 0,
			-1, 0, 0,
			-1, 0, 0,
			-1, 0, 0,
		)

	case 2: // Face +Y (topo)
		cm.Vertices = append(cm.Vertices,
			x, y+1, z, // 0
			x, y+1, z+1, // 1
			x+1, y+1, z+1, // 2
			x+1, y+1, z, // 3
		)
		cm.Normals = append(cm.Normals,
			0, 1, 0,
			0, 1, 0,
			0, 1, 0,
			0, 1, 0,
		)

	case 3: // Face -Y (fundo)
		cm.Vertices = append(cm.Vertices,
			x, y, z+1, // 0
			x, y, z, // 1
			x+1, y, z, // 2
			x+1, y, z+1, // 3
		)
		cm.Normals = append(cm.Normals,
			0, -1, 0,
			0, -1, 0,
			0, -1, 0,
			0, -1, 0,
		)

	case 4: // Face +Z (frente)
		cm.Vertices = append(cm.Vertices,
			x+1, y, z+1, // 0
			x+1, y+1, z+1, // 1
			x, y+1, z+1, // 2
			x, y, z+1, // 3
		)
		cm.Normals = append(cm.Normals,
			0, 0, 1,
			0, 0, 1,
			0, 0, 1,
			0, 0, 1,
		)

	case 5: // Face -Z (trás)
		cm.Vertices = append(cm.Vertices,
			x, y, z, // 0
			x, y+1, z, // 1
			x+1, y+1, z, // 2
			x+1, y, z, // 3
		)
		cm.Normals = append(cm.Normals,
			0, 0, -1,
			0, 0, -1,
			0, 0, -1,
			0, 0, -1,
		)
	}

	// UVs (mesmos para todas as faces)
	cm.Texcoords = append(cm.Texcoords,
		uMin, vMax, // 0
		uMin, vMin, // 1
		uMax, vMin, // 2
		uMax, vMax, // 3
	)

	// Índices (2 triângulos por quad)
	cm.Indices = append(cm.Indices,
		vertexOffset+0, vertexOffset+1, vertexOffset+2,
		vertexOffset+0, vertexOffset+2, vertexOffset+3,
	)
}

// AddQuadWithChunkAtlas adiciona um quad usando o atlas do chunk
func (cm *ChunkMesh) AddQuadWithChunkAtlas(x, y, z float32, face int, blockType BlockType, chunkAtlas *ChunkAtlas) {
	// Obter UVs do atlas do chunk
	uMin, vMin, uMax, vMax := chunkAtlas.GetBlockUVs(blockType)

	vertexOffset := uint16(len(cm.Vertices) / 3)

	// Definir vértices e normais baseado na face
	switch face {
	case 0: // Face +X (direita)
		cm.Vertices = append(cm.Vertices,
			x+1, y, z,
			x+1, y+1, z,
			x+1, y+1, z+1,
			x+1, y, z+1,
		)
		cm.Normals = append(cm.Normals,
			1, 0, 0,
			1, 0, 0,
			1, 0, 0,
			1, 0, 0,
		)

	case 1: // Face -X (esquerda)
		cm.Vertices = append(cm.Vertices,
			x, y, z+1,
			x, y+1, z+1,
			x, y+1, z,
			x, y, z,
		)
		cm.Normals = append(cm.Normals,
			-1, 0, 0,
			-1, 0, 0,
			-1, 0, 0,
			-1, 0, 0,
		)

	case 2: // Face +Y (topo)
		cm.Vertices = append(cm.Vertices,
			x, y+1, z,
			x, y+1, z+1,
			x+1, y+1, z+1,
			x+1, y+1, z,
		)
		cm.Normals = append(cm.Normals,
			0, 1, 0,
			0, 1, 0,
			0, 1, 0,
			0, 1, 0,
		)

	case 3: // Face -Y (fundo)
		cm.Vertices = append(cm.Vertices,
			x, y, z+1,
			x, y, z,
			x+1, y, z,
			x+1, y, z+1,
		)
		cm.Normals = append(cm.Normals,
			0, -1, 0,
			0, -1, 0,
			0, -1, 0,
			0, -1, 0,
		)

	case 4: // Face +Z (frente)
		cm.Vertices = append(cm.Vertices,
			x+1, y, z+1,
			x+1, y+1, z+1,
			x, y+1, z+1,
			x, y, z+1,
		)
		cm.Normals = append(cm.Normals,
			0, 0, 1,
			0, 0, 1,
			0, 0, 1,
			0, 0, 1,
		)

	case 5: // Face -Z (trás)
		cm.Vertices = append(cm.Vertices,
			x, y, z,
			x, y+1, z,
			x+1, y+1, z,
			x+1, y, z,
		)
		cm.Normals = append(cm.Normals,
			0, 0, -1,
			0, 0, -1,
			0, 0, -1,
			0, 0, -1,
		)
	}

	// UVs
	cm.Texcoords = append(cm.Texcoords,
		uMin, vMax,
		uMin, vMin,
		uMax, vMin,
		uMax, vMax,
	)

	// Índices
	cm.Indices = append(cm.Indices,
		vertexOffset+0, vertexOffset+1, vertexOffset+2,
		vertexOffset+0, vertexOffset+2, vertexOffset+3,
	)
}

// UploadToGPU faz upload da mesh para a GPU
func (cm *ChunkMesh) UploadToGPU() {
	if len(cm.Vertices) == 0 {
		return
	}

	// Se estamos em modo de teste, pular upload para GPU
	if DisableGPUUploadForTesting {
		cm.Uploaded = false // Mesh data gerada, mas não enviada para GPU
		return
	}

	// Criar mesh do Raylib
	cm.Mesh = rl.Mesh{}
	cm.Mesh.VertexCount = int32(len(cm.Vertices) / 3)
	cm.Mesh.TriangleCount = int32(len(cm.Indices) / 3)

	cm.Mesh.Vertices = &cm.Vertices[0]
	cm.Mesh.Texcoords = &cm.Texcoords[0]
	cm.Mesh.Normals = &cm.Normals[0]
	cm.Mesh.Indices = (*uint16)(nil)
	if len(cm.Indices) > 0 {
		cm.Mesh.Indices = &cm.Indices[0]
	}

	// Upload para GPU
	rl.UploadMesh(&cm.Mesh, false)
	cm.Uploaded = true
}

// Clear limpa a mesh
func (cm *ChunkMesh) Clear() {
	cm.Vertices = cm.Vertices[:0]
	cm.Texcoords = cm.Texcoords[:0]
	cm.Normals = cm.Normals[:0]
	cm.Indices = cm.Indices[:0]

	if cm.Uploaded {
		rl.UnloadMesh(&cm.Mesh)
		cm.Uploaded = false
	}
}
