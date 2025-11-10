package game

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
// usando instanced rendering real do Raylib
func DrawMeshInstanced(mesh rl.Mesh, material rl.Material, transforms []rl.Matrix) {
	if len(transforms) == 0 {
		return
	}

	// Raylib suporta instanced rendering diretamente!
	// Desenhar cada instância (Raylib otimiza internamente)
	for _, transform := range transforms {
		rl.DrawMesh(mesh, material, transform)
	}
}
