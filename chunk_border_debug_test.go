package main

import (
	"testing"
)

// TestChunkBorderRendering testa especificamente se blocos na borda entre chunks são renderizados corretamente
func TestChunkBorderRendering(t *testing.T) {
	// Criar ChunkManager
	cm := NewChunkManager(5)

	// Criar dois chunks adjacentes, ambos completamente preenchidos com pedra
	chunk0 := NewChunk(0, 0, 0)
	chunk1 := NewChunk(1, 0, 0)

	// Preencher ambos os chunks completamente com pedra
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

	// Testar bloco na borda do chunk 0 (última posição X=31)
	// Este bloco está em x=31 do chunk 0, que é a coordenada mundial 31
	// O vizinho X+ está em x=32 (primeira posição do chunk 1)

	t.Log("=== Testando bloco na borda do chunk 0 (x=31, última posição) ===")

	// Coordenadas mundiais do bloco na borda
	borderX, borderY, borderZ := int32(31), int32(16), int32(16)

	t.Logf("Bloco testado: posição mundial (%d, %d, %d)", borderX, borderY, borderZ)
	t.Logf("Chunk do bloco: %v", GetChunkCoord(borderX, borderY, borderZ))

	// Verificar todos os vizinhos
	neighbors := []struct {
		name     string
		dx, dy, dz int32
	}{
		{"X+ (próximo chunk)", 1, 0, 0},
		{"X- (mesmo chunk)", -1, 0, 0},
		{"Y+", 0, 1, 0},
		{"Y-", 0, -1, 0},
		{"Z+", 0, 0, 1},
		{"Z-", 0, 0, -1},
	}

	for _, n := range neighbors {
		nx := borderX + n.dx
		ny := borderY + n.dy
		nz := borderZ + n.dz

		neighborChunk := GetChunkCoord(nx, ny, nz)
		neighborBlock := cm.GetBlock(nx, ny, nz)

		t.Logf("  Vizinho %s: pos=(%d,%d,%d) chunk=%v tipo=%v",
			n.name, nx, ny, nz, neighborChunk, neighborBlock)
	}

	// Verificar se o bloco está oculto
	isHidden := cm.IsBlockHidden(borderX, borderY, borderZ)
	t.Logf("IsBlockHidden retornou: %v (esperado: true)", isHidden)

	if !isHidden {
		t.Errorf("ERRO: Bloco na borda deveria estar oculto mas está visível!")
	}

	// Agora testar o UpdateMeshesWithNeighbors
	t.Log("\n=== Testando UpdateMeshesWithNeighbors ===")

	// Resetar e atualizar meshes do chunk 0
	chunk0.NeedUpdateMeshes = true
	chunk0.UpdateMeshesWithNeighbors(cm.GetBlock)

	// Contar quantos blocos foram renderizados
	totalBlocks := len(chunk0.GrassTransforms) + len(chunk0.DirtTransforms) + len(chunk0.StoneTransforms)

	t.Logf("Total de blocos renderizados no chunk 0: %d", totalBlocks)
	t.Logf("  Grama: %d", len(chunk0.GrassTransforms))
	t.Logf("  Terra: %d", len(chunk0.DirtTransforms))
	t.Logf("  Pedra: %d", len(chunk0.StoneTransforms))

	// Calcular quantos blocos DEVERIAM ser renderizados
	// Um chunk completamente preenchido cercado por outros chunks cheios
	// deve renderizar apenas os blocos das faces externas

	// Faces do chunk:
	// - Face X- (x=0): 32x32 = 1024 blocos (borda com chunk inexistente)
	// - Face X+ (x=31): 0 blocos (borda com chunk 1)
	// - Face Y- (y=0): 32x32 = 1024 blocos (borda com chunk inexistente)
	// - Face Y+ (y=31): 32x32 = 1024 blocos (borda com chunk inexistente)
	// - Face Z- (z=0): 32x32 = 1024 blocos (borda com chunk inexistente)
	// - Face Z+ (z=31): 32x32 = 1024 blocos (borda com chunk inexistente)
	// Mas há sobreposição nas arestas...

	// O importante é que blocos internos não sejam renderizados
	// Vamos verificar especificamente se o bloco (31, 16, 16) foi renderizado

	// Para verificar isso, vamos testar um bloco interno e um bloco na borda
	t.Log("\n=== Verificando blocos específicos ===")

	// Bloco totalmente interno (não deve ser renderizado)
	internalX, internalY, internalZ := int32(16), int32(16), int32(16)
	isInternalHidden := cm.IsBlockHidden(internalX, internalY, internalZ)
	t.Logf("Bloco interno (%d,%d,%d) está oculto: %v (esperado: true)",
		internalX, internalY, internalZ, isInternalHidden)

	if !isInternalHidden {
		t.Errorf("ERRO: Bloco interno deveria estar oculto!")
	}

	// Bloco na face X- do chunk (deve ser renderizado pois não há chunk em X-)
	edgeX, edgeY, edgeZ := int32(0), int32(16), int32(16)
	isEdgeHidden := cm.IsBlockHidden(edgeX, edgeY, edgeZ)
	t.Logf("Bloco na borda X- (%d,%d,%d) está oculto: %v (esperado: false)",
		edgeX, edgeY, edgeZ, isEdgeHidden)

	if isEdgeHidden {
		t.Errorf("ERRO: Bloco na borda X- (sem vizinho) deveria estar visível!")
	}
}

// TestRealTerrainBorderOcclusion testa oclusão com terreno real (como no jogo)
func TestRealTerrainBorderOcclusion(t *testing.T) {
	// Simular a geração de terreno como no jogo
	cm := NewChunkManager(5)

	// Criar 4 chunks adjacentes no plano Y=0
	coords := []ChunkCoord{
		{X: 0, Y: 0, Z: 0},
		{X: 1, Y: 0, Z: 0},
		{X: 0, Y: 0, Z: 1},
		{X: 1, Y: 0, Z: 1},
	}

	for _, coord := range coords {
		chunk := NewChunk(coord.X, coord.Y, coord.Z)
		chunk.GenerateTerrain() // Gera terreno com ondulações
		cm.Chunks[coord.Key()] = chunk
	}

	// Verificar um bloco na borda entre chunks 0 e 1
	// Posição (31, 10, 16) está na borda do chunk 0
	testX, testY, testZ := int32(31), int32(10), int32(16)

	t.Logf("Testando bloco em terreno real: (%d, %d, %d)", testX, testY, testZ)

	// Ver o que há nessa posição e nos vizinhos
	blockType := cm.GetBlock(testX, testY, testZ)
	t.Logf("Tipo do bloco: %v", blockType)

	if blockType == BlockAir {
		t.Log("Bloco é ar, pulando teste")
		return
	}

	// Verificar vizinhos
	t.Log("Vizinhos:")
	neighborXplus := cm.GetBlock(testX+1, testY, testZ)
	neighborXminus := cm.GetBlock(testX-1, testY, testZ)
	neighborYplus := cm.GetBlock(testX, testY+1, testZ)
	neighborYminus := cm.GetBlock(testX, testY-1, testZ)
	neighborZplus := cm.GetBlock(testX, testY, testZ+1)
	neighborZminus := cm.GetBlock(testX, testY, testZ-1)

	t.Logf("  X+: %v (chunk=%v)", neighborXplus, GetChunkCoord(testX+1, testY, testZ))
	t.Logf("  X-: %v", neighborXminus)
	t.Logf("  Y+: %v", neighborYplus)
	t.Logf("  Y-: %v", neighborYminus)
	t.Logf("  Z+: %v", neighborZplus)
	t.Logf("  Z-: %v", neighborZminus)

	isHidden := cm.IsBlockHidden(testX, testY, testZ)
	t.Logf("IsBlockHidden: %v", isHidden)

	// Verificar se UpdateMeshesWithNeighbors funciona
	chunk0 := cm.Chunks[ChunkCoord{X: 0, Y: 0, Z: 0}.Key()]
	chunk0.NeedUpdateMeshes = true
	chunk0.UpdateMeshesWithNeighbors(cm.GetBlock)

	totalRendered := len(chunk0.GrassTransforms) + len(chunk0.DirtTransforms) + len(chunk0.StoneTransforms)
	t.Logf("Blocos renderizados no chunk (0,0,0): %d", totalRendered)
}
