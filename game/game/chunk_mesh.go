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

	// Definir vértices, normais e UVs específicas por face
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
		// UVs para face +X
		cm.Texcoords = append(cm.Texcoords,
			uMax, vMax,
			uMax, vMin,
			uMin, vMin,
			uMin, vMax,
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
		// UVs para face -X
		cm.Texcoords = append(cm.Texcoords,
			uMax, vMax,
			uMax, vMin,
			uMin, vMin,
			uMin, vMax,
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
		// UVs para face +Y (topo)
		cm.Texcoords = append(cm.Texcoords,
			uMin, vMin,
			uMin, vMax,
			uMax, vMax,
			uMax, vMin,
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
		// UVs para face -Y (fundo)
		cm.Texcoords = append(cm.Texcoords,
			uMin, vMax,
			uMin, vMin,
			uMax, vMin,
			uMax, vMax,
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
		// UVs para face +Z (frente)
		cm.Texcoords = append(cm.Texcoords,
			uMax, vMax,
			uMax, vMin,
			uMin, vMin,
			uMin, vMax,
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
		// UVs para face -Z (trás)
		cm.Texcoords = append(cm.Texcoords,
			uMin, vMax,
			uMin, vMin,
			uMax, vMin,
			uMax, vMax,
		)
	}

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

	// Definir vértices, normais e UVs específicas por face
	// UVs são ajustadas para cada face garantir orientação correta da textura
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
		// UVs para face +X: textura olhando para -X
		cm.Texcoords = append(cm.Texcoords,
			uMax, vMax, // bottom-left do quad -> top-right da textura
			uMax, vMin, // top-left do quad -> bottom-right da textura
			uMin, vMin, // top-right do quad -> bottom-left da textura
			uMin, vMax, // bottom-right do quad -> top-left da textura
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
		// UVs para face -X: textura olhando para +X
		cm.Texcoords = append(cm.Texcoords,
			uMax, vMax,
			uMax, vMin,
			uMin, vMin,
			uMin, vMax,
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
		// UVs para face +Y (topo): textura vista de cima
		cm.Texcoords = append(cm.Texcoords,
			uMin, vMin,
			uMin, vMax,
			uMax, vMax,
			uMax, vMin,
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
		// UVs para face -Y (fundo): textura vista de baixo
		cm.Texcoords = append(cm.Texcoords,
			uMin, vMax,
			uMin, vMin,
			uMax, vMin,
			uMax, vMax,
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
		// UVs para face +Z (frente): textura olhando para -Z
		cm.Texcoords = append(cm.Texcoords,
			uMax, vMax,
			uMax, vMin,
			uMin, vMin,
			uMin, vMax,
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
		// UVs para face -Z (trás): textura olhando para +Z
		cm.Texcoords = append(cm.Texcoords,
			uMin, vMax,
			uMin, vMin,
			uMax, vMin,
			uMax, vMax,
		)
	}

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

// AddQuadWithCustomUVs adiciona um quad com UVs customizadas (para blocos customizados)
func (cm *ChunkMesh) AddQuadWithCustomUVs(x, y, z float32, face int, uMin, vMin, uMax, vMax float32) {
	vertexOffset := uint16(len(cm.Vertices) / 3)

	// Definir vértices, normais e UVs específicas por face
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
		cm.Texcoords = append(cm.Texcoords,
			uMax, vMax,
			uMax, vMin,
			uMin, vMin,
			uMin, vMax,
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
		cm.Texcoords = append(cm.Texcoords,
			uMax, vMax,
			uMax, vMin,
			uMin, vMin,
			uMin, vMax,
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
		cm.Texcoords = append(cm.Texcoords,
			uMin, vMin,
			uMin, vMax,
			uMax, vMax,
			uMax, vMin,
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
		cm.Texcoords = append(cm.Texcoords,
			uMin, vMax,
			uMin, vMin,
			uMax, vMin,
			uMax, vMax,
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
		cm.Texcoords = append(cm.Texcoords,
			uMax, vMax,
			uMax, vMin,
			uMin, vMin,
			uMin, vMax,
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
		cm.Texcoords = append(cm.Texcoords,
			uMin, vMax,
			uMin, vMin,
			uMax, vMin,
			uMax, vMax,
		)
	}

	// Índices
	cm.Indices = append(cm.Indices,
		vertexOffset+0, vertexOffset+1, vertexOffset+2,
		vertexOffset+0, vertexOffset+2, vertexOffset+3,
	)
}
