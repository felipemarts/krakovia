package main

import (
	"fmt"
	"unsafe"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// CreateTexturedCubeMesh cria uma mesh de cubo com UVs para o texture atlas
func CreateTexturedCubeMesh(blockType BlockType) rl.Mesh {
	// Não usar GenMeshCube pois ele já faz upload automático
	// Vamos criar a mesh manualmente
	mesh := rl.Mesh{}
	mesh.VertexCount = 24
	mesh.TriangleCount = 12

	// Obter UVs para o tipo de bloco
	uMin, vMin, uMax, vMax := GetBlockUVs(blockType)

	// Debug: imprimir os UVs calculados
	fmt.Printf("Bloco %v: UV=[%.3f,%.3f] -> [%.3f,%.3f]\n", blockType, uMin, vMin, uMax, vMax)

	// Gerar mesh base e pegar UVs padrão
	tempMesh := rl.GenMeshCube(1.0, 1.0, 1.0)

	// Copiar dados da mesh temporária
	mesh.Vertices = tempMesh.Vertices
	mesh.Normals = tempMesh.Normals
	mesh.Indices = tempMesh.Indices

	// Criar novos UVs modificados
	if tempMesh.Texcoords != nil {
		// Converter ponteiro para slice
		oldTexcoords := unsafe.Slice(tempMesh.Texcoords, tempMesh.VertexCount*2)

		// Alocar novo array de UVs
		newTexcoords := make([]float32, mesh.VertexCount*2)

		// Aplicar UVs para todos os vértices
		for i := int32(0); i < mesh.VertexCount; i++ {
			// Pegar UV original (0-1)
			origU := oldTexcoords[i*2]
			origV := oldTexcoords[i*2+1]

			// Mapear para a região do atlas
			newTexcoords[i*2] = uMin + origU*(uMax-uMin)
			newTexcoords[i*2+1] = vMin + origV*(vMax-vMin)
		}

		// Atribuir novo array
		mesh.Texcoords = &newTexcoords[0]
	}

	// Upload para GPU
	rl.UploadMesh(&mesh, false)

	return mesh
}

// DrawMeshInstanced desenha múltiplas instâncias de uma mesh com diferentes transformações
// usando geometry batching - combina todas as geometrias em uma única mesh para 1 draw call
func DrawMeshInstanced(mesh rl.Mesh, material rl.Material, transforms []rl.Matrix) {
	if len(transforms) == 0 {
		return
	}

	// Se houver apenas uma instância, usar draw normal
	if len(transforms) == 1 {
		rl.DrawMesh(mesh, material, transforms[0])
		return
	}

	// Criar mesh combinada com todas as instâncias transformadas
	batchedMesh := CreateBatchedMesh(mesh, transforms)
	defer rl.UnloadMesh(&batchedMesh)

	// Um único draw call para todas as instâncias!
	rl.DrawMesh(batchedMesh, material, rl.MatrixIdentity())
}

// CreateBatchedMesh cria uma única mesh contendo todas as instâncias transformadas
func CreateBatchedMesh(baseMesh rl.Mesh, transforms []rl.Matrix) rl.Mesh {
	instanceCount := int32(len(transforms))
	verticesPerInstance := baseMesh.VertexCount
	trianglesPerInstance := baseMesh.TriangleCount

	// Alocar nova mesh com espaço para todas as instâncias
	batchedMesh := rl.Mesh{
		VertexCount:   verticesPerInstance * instanceCount,
		TriangleCount: trianglesPerInstance * instanceCount,
	}

	// Converter ponteiros para slices usando unsafe
	baseVertices := unsafe.Slice(baseMesh.Vertices, verticesPerInstance*3)
	baseTexcoords := unsafe.Slice(baseMesh.Texcoords, verticesPerInstance*2)
	baseNormals := unsafe.Slice(baseMesh.Normals, verticesPerInstance*3)
	baseIndices := unsafe.Slice(baseMesh.Indices, trianglesPerInstance*3)

	// Alocar arrays para mesh combinada
	newVertices := make([]float32, batchedMesh.VertexCount*3)
	newTexcoords := make([]float32, batchedMesh.VertexCount*2)
	newNormals := make([]float32, batchedMesh.VertexCount*3)
	newIndices := make([]uint16, batchedMesh.TriangleCount*3)

	// Copiar e transformar dados de cada instância
	for i := int32(0); i < instanceCount; i++ {
		transform := transforms[i]
		vertexOffset := i * verticesPerInstance
		indexOffset := i * verticesPerInstance
		triangleOffset := i * trianglesPerInstance

		// Transformar vértices e normais
		for v := int32(0); v < verticesPerInstance; v++ {
			// Posição original
			x := baseVertices[v*3]
			y := baseVertices[v*3+1]
			z := baseVertices[v*3+2]

			// Aplicar transformação
			pos := rl.Vector3Transform(rl.NewVector3(x, y, z), transform)

			// Armazenar no array combinado
			idx := (vertexOffset + v) * 3
			newVertices[idx] = pos.X
			newVertices[idx+1] = pos.Y
			newVertices[idx+2] = pos.Z

			// Normal (apenas rotação, sem translação)
			nx := baseNormals[v*3]
			ny := baseNormals[v*3+1]
			nz := baseNormals[v*3+2]
			normal := rl.Vector3Transform(rl.NewVector3(nx, ny, nz), rl.MatrixIdentity())
			newNormals[idx] = normal.X
			newNormals[idx+1] = normal.Y
			newNormals[idx+2] = normal.Z

			// Coordenadas de textura (sem transformação)
			texIdx := (vertexOffset + v) * 2
			newTexcoords[texIdx] = baseTexcoords[v*2]
			newTexcoords[texIdx+1] = baseTexcoords[v*2+1]
		}

		// Copiar índices (ajustando offset)
		for t := int32(0); t < trianglesPerInstance*3; t++ {
			newIndices[triangleOffset*3+t] = baseIndices[t] + uint16(indexOffset)
		}
	}

	// Atribuir dados à mesh
	batchedMesh.Vertices = &newVertices[0]
	batchedMesh.Texcoords = &newTexcoords[0]
	batchedMesh.Normals = &newNormals[0]
	batchedMesh.Indices = &newIndices[0]

	// Upload para GPU
	rl.UploadMesh(&batchedMesh, false)

	return batchedMesh
}
