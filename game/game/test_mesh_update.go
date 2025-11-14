package game

import (
	"fmt"
	"testing"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// NOTA: Este teste foi desabilitado porque mudamos de instanced rendering (arrays de transforms)
// para mesh combinada. A lógica de ocultação agora funciona gerando apenas faces visíveis
// ao invés de pular blocos inteiros. Os testes em chunk_occlusion_test.go cobrem a lógica
// de detecção de blocos ocultos.

// TestMeshUpdateDetailed testa detalhadamente o UpdateMeshesWithNeighbors
func _TestMeshUpdateDetailed(t *testing.T) {
	t.Skip("Teste desabilitado - API mudou para mesh combinada")
	/*
	// Criar ChunkManager com dois chunks adjacentes
	cm := NewChunkManager(5)

	chunk0 := NewChunk(0, 0, 0)
	chunk1 := NewChunk(1, 0, 0)

	// Preencher completamente com pedra
	for x := int32(0); x < ChunkSize; x++ {
		for y := int32(0); y < ChunkHeight; y++ {
			for z := int32(0); z < ChunkSize; z++ {
				chunk0.Blocks[x][y][z] = BlockStone
				chunk1.Blocks[x][y][z] = BlockStone
			}
		}
	}

	cm.Chunks[ChunkCoord{X: 0, Y: 0, Z: 0}.Key()] = chunk0
	cm.Chunks[ChunkCoord{X: 1, Y: 0, Z: 0}.Key()] = chunk1

	// Contar quantos blocos na face X+ (x=31) deveriam ser ocultos
	faceXplus := 0
	faceXplusHidden := 0

	for y := int32(0); y < ChunkHeight; y++ {
		for z := int32(0); z < ChunkSize; z++ {
			worldX := int32(31) // Última coluna do chunk 0
			worldY := int32(y)
			worldZ := int32(z)

			faceXplus++

			if cm.IsBlockHidden(worldX, worldY, worldZ) {
				faceXplusHidden++
			}
		}
	}

	t.Logf("Face X+ do chunk 0:")
	t.Logf("  Total de blocos: %d", faceXplus)
	t.Logf("  Ocultos: %d", faceXplusHidden)
	t.Logf("  Visíveis: %d", faceXplus-faceXplusHidden)

	// Agora atualizar meshes e verificar
	chunk0.UpdateMeshesWithNeighbors(cm.GetBlock)

	// Verificar se algum bloco da face X+ foi renderizado
	blocksRenderedFromFaceXplus := 0

	// Percorrer os transforms renderizados
	worldX0 := chunk0.Coord.X * ChunkSize
	worldY0 := chunk0.Coord.Y * ChunkHeight
	worldZ0 := chunk0.Coord.Z * ChunkSize

	for _, transform := range chunk0.StoneTransforms {
		// Extrair posição do transform (é a posição + 0.5)
		// transform.m12, m13, m14 são as coordenadas de translação
		x := transform.M12
		y := transform.M13
		z := transform.M14

		// Remover o offset de 0.5 para obter coordenadas mundiais
		wx := int32(x - 0.5)
		wy := int32(y - 0.5)
		wz := int32(z - 0.5)

		// Verificar se está na face X+ do chunk 0
		localX := wx - worldX0
		if localX == 31 {
			blocksRenderedFromFaceXplus++
			if blocksRenderedFromFaceXplus <= 5 { // Mostrar apenas os primeiros 5
				t.Logf("  Bloco renderizado na face X+: mundo=(%d,%d,%d) local=(%d,%d,%d)",
					wx, wy, wz, localX, wy-worldY0, wz-worldZ0)
			}
		}
	}

	t.Logf("\nBlocos DA FACE X+ que foram renderizados: %d", blocksRenderedFromFaceXplus)
	t.Logf("Total de blocos renderizados: %d", len(chunk0.StoneTransforms))

	if blocksRenderedFromFaceXplus > 0 {
		t.Errorf("ERRO: Blocos da face X+ não deveriam ser renderizados pois têm vizinho (chunk 1)!")

		// Debug: verificar um bloco específico
		testX, testY, testZ := int32(31), int32(16), int32(16)
		t.Logf("\nDebug do bloco (%d,%d,%d):", testX, testY, testZ)

		// Ver o que UpdateMeshesWithNeighbors vê
		isHiddenByFunc := true
		directions := []struct{ dx, dy, dz int32; name string }{
			{1, 0, 0, "X+"},
			{-1, 0, 0, "X-"},
			{0, 1, 0, "Y+"},
			{0, -1, 0, "Y-"},
			{0, 0, 1, "Z+"},
			{0, 0, -1, "Z-"},
		}

		for _, dir := range directions {
			neighborBlock := cm.GetBlock(testX+dir.dx, testY+dir.dy, testZ+dir.dz)
			t.Logf("  Vizinho %s: tipo=%v", dir.name, neighborBlock)
			if neighborBlock == BlockAir {
				isHiddenByFunc = false
			}
		}

		t.Logf("  Deveria estar oculto: %v", isHiddenByFunc)
		t.Logf("  IsBlockHidden diz: %v", cm.IsBlockHidden(testX, testY, testZ))
	}
	*/
}

// TestCompareOcclusionMethods compara os dois métodos de detecção de oclusão
func _TestCompareOcclusionMethods(t *testing.T) {
	t.Skip("Teste desabilitado - API mudou para mesh combinada")
	/*
	cm := NewChunkManager(5)

	chunk0 := NewChunk(0, 0, 0)
	chunk1 := NewChunk(1, 0, 0)

	// Preencher completamente
	for x := int32(0); x < ChunkSize; x++ {
		for y := int32(0); y < ChunkHeight; y++ {
			for z := int32(0); z < ChunkSize; z++ {
				chunk0.Blocks[x][y][z] = BlockStone
				chunk1.Blocks[x][y][z] = BlockStone
			}
		}
	}

	cm.Chunks[ChunkCoord{X: 0, Y: 0, Z: 0}.Key()] = chunk0
	cm.Chunks[ChunkCoord{X: 1, Y: 0, Z: 0}.Key()] = chunk1

	// Testar alguns blocos específicos na borda
	testCases := []struct {
		x, y, z int32
		desc    string
	}{
		{31, 16, 16, "Centro da face X+"},
		{31, 0, 0, "Canto da face X+ (inferior)"},
		{31, 31, 31, "Canto da face X+ (superior)"},
	}

	for _, tc := range testCases {
		t.Logf("\n=== Testando: %s (%d,%d,%d) ===", tc.desc, tc.x, tc.y, tc.z)

		// Método 1: ChunkManager.IsBlockHidden
		method1 := cm.IsBlockHidden(tc.x, tc.y, tc.z)

		// Método 2: Simular o que UpdateMeshesWithNeighbors faz
		method2 := true
		directions := []struct{ dx, dy, dz int32 }{
			{1, 0, 0}, {-1, 0, 0}, {0, 1, 0}, {0, -1, 0}, {0, 0, 1}, {0, 0, -1},
		}

		for _, dir := range directions {
			neighborBlock := cm.GetBlock(tc.x+dir.dx, tc.y+dir.dy, tc.z+dir.dz)
			if neighborBlock == BlockAir {
				method2 = false
				break
			}
		}

		t.Logf("  ChunkManager.IsBlockHidden: %v", method1)
		t.Logf("  UpdateMeshesWithNeighbors logic: %v", method2)

		if method1 != method2 {
			t.Errorf("  INCONSISTÊNCIA! Os dois métodos deram resultados diferentes!")
		}

		if method1 == false {
			t.Errorf("  ERRO! Bloco deveria estar oculto mas foi detectado como visível!")
		}
	}
	*/
}

// Helper para imprimir matriz (para debug)
func printMatrix(m rl.Matrix) string {
	return fmt.Sprintf("[%.1f, %.1f, %.1f]", m.M12, m.M13, m.M14)
}
