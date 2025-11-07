package main

import (
	"testing"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// TestGradualChunkLoading testa o cenário real onde chunks são carregados gradualmente
// Este é o cenário que causava o bug: chunks vizinhos são carregados depois,
// mas o chunk original não atualizava suas meshes
func TestGradualChunkLoading(t *testing.T) {
	cm := NewChunkManager(5)

	t.Log("=== Passo 1: Criar chunk central (0,0,0) ===")
	// Simular LoadChunksAroundPlayer criando o primeiro chunk
	chunk0 := NewChunk(0, 0, 0)
	for x := int32(0); x < ChunkSize; x++ {
		for y := int32(0); y < ChunkHeight; y++ {
			for z := int32(0); z < ChunkSize; z++ {
				chunk0.Blocks[x][y][z] = BlockStone
			}
		}
	}
	cm.Chunks[ChunkCoord{X: 0, Y: 0, Z: 0}.Key()] = chunk0
	cm.MarkNeighborsForUpdate(ChunkCoord{X: 0, Y: 0, Z: 0}) // Nenhum vizinho existe ainda

	// Simular primeiro render - chunk não tem vizinhos
	t.Log("Renderizando chunk (0,0,0) sem vizinhos...")
	chunk0.UpdateMeshesWithNeighbors(cm.GetBlock)
	blocksBeforeNeighbors := len(chunk0.GrassTransforms) + len(chunk0.DirtTransforms) + len(chunk0.StoneTransforms)
	t.Logf("Blocos visíveis no chunk (0,0,0) SEM vizinhos: %d", blocksBeforeNeighbors)

	// Deve ter muitos blocos visíveis nas bordas (faces externas)
	if blocksBeforeNeighbors == 0 {
		t.Error("Chunk sem vizinhos deveria ter blocos visíveis nas faces externas!")
	}

	t.Log("\n=== Passo 2: Carregar chunk vizinho em X+ (1,0,0) ===")
	chunk1 := NewChunk(1, 0, 0)
	for x := int32(0); x < ChunkSize; x++ {
		for y := int32(0); y < ChunkHeight; y++ {
			for z := int32(0); z < ChunkSize; z++ {
				chunk1.Blocks[x][y][z] = BlockStone
			}
		}
	}
	cm.Chunks[ChunkCoord{X: 1, Y: 0, Z: 0}.Key()] = chunk1
	cm.MarkNeighborsForUpdate(ChunkCoord{X: 1, Y: 0, Z: 0}) // Deve marcar chunk0 para atualização!

	// Verificar se chunk0 foi marcado para atualização
	if !chunk0.NeedUpdateMeshes {
		t.Error("BUG: Chunk (0,0,0) deveria ser marcado para atualização após vizinho ser carregado!")
	} else {
		t.Log("✓ Chunk (0,0,0) foi corretamente marcado para atualização")
	}

	t.Log("\n=== Passo 3: Renderizar chunk (0,0,0) novamente (simular próximo frame) ===")
	// Simular o que acontece no próximo frame de render
	chunk0.UpdateMeshesWithNeighbors(cm.GetBlock)
	blocksAfterNeighbor := len(chunk0.GrassTransforms) + len(chunk0.DirtTransforms) + len(chunk0.StoneTransforms)
	t.Logf("Blocos visíveis no chunk (0,0,0) COM vizinho em X+: %d", blocksAfterNeighbor)

	// Deve ter MENOS blocos agora, pois a face X+ está oculta
	if blocksAfterNeighbor >= blocksBeforeNeighbors {
		t.Errorf("BUG: Chunk não reduziu blocos após vizinho ser carregado! Antes: %d, Depois: %d",
			blocksBeforeNeighbors, blocksAfterNeighbor)
	} else {
		reduction := blocksBeforeNeighbors - blocksAfterNeighbor
		t.Logf("✓ Redução de %d blocos após carregar vizinho (esperado ~1024)", reduction)

		// Face X+ tem 32x32 = 1024 blocos
		// Nem todos podem estar ocultos por causa de outras faces expostas
		if reduction < 800 {
			t.Errorf("Redução muito pequena! Esperado pelo menos ~800, obtido %d", reduction)
		}
	}

	t.Log("\n=== Passo 4: Carregar todos os vizinhos restantes ===")
	// Carregar chunks em todas as direções
	neighbors := []ChunkCoord{
		{X: -1, Y: 0, Z: 0},  // X-
		{X: 0, Y: 1, Z: 0},   // Y+
		{X: 0, Y: -1, Z: 0},  // Y-
		{X: 0, Y: 0, Z: 1},   // Z+
		{X: 0, Y: 0, Z: -1},  // Z-
		{X: 1, Y: 0, Z: 1},   // Diagonais também
		{X: -1, Y: 0, Z: 1},
		{X: 1, Y: 0, Z: -1},
		{X: -1, Y: 0, Z: -1},
		{X: 0, Y: 1, Z: 1},
		{X: 0, Y: 1, Z: -1},
		{X: 0, Y: -1, Z: 1},
		{X: 0, Y: -1, Z: -1},
		{X: 1, Y: 1, Z: 0},
		{X: 1, Y: -1, Z: 0},
		{X: -1, Y: 1, Z: 0},
		{X: -1, Y: -1, Z: 0},
	}

	for _, coord := range neighbors {
		chunk := NewChunk(coord.X, coord.Y, coord.Z)
		for x := int32(0); x < ChunkSize; x++ {
			for y := int32(0); y < ChunkHeight; y++ {
				for z := int32(0); z < ChunkSize; z++ {
					chunk.Blocks[x][y][z] = BlockStone
				}
			}
		}
		cm.Chunks[coord.Key()] = chunk
		cm.MarkNeighborsForUpdate(coord)
	}

	// Renderizar novamente
	chunk0.UpdateMeshesWithNeighbors(cm.GetBlock)
	blocksFinalCount := len(chunk0.GrassTransforms) + len(chunk0.DirtTransforms) + len(chunk0.StoneTransforms)
	t.Logf("Blocos visíveis no chunk (0,0,0) COM TODOS vizinhos: %d (esperado: 0)", blocksFinalCount)

	if blocksFinalCount > 0 {
		t.Errorf("BUG: Chunk completamente cercado ainda tem %d blocos visíveis!", blocksFinalCount)
	}
}

// TestChunkManagerMarkNeighborsInLoadSequence testa se MarkNeighborsForUpdate é chamado
// durante o carregamento de chunks via LoadChunksAroundPlayer
func TestChunkManagerMarkNeighborsInLoadSequence(t *testing.T) {
	cm := NewChunkManager(2)
	playerPos := rl.Vector3{X: 16, Y: 16, Z: 16} // Centro do chunk (0,0,0)

	t.Log("=== Passo 1: Carregar chunks ao redor do jogador ===")
	// Primeira chamada - nenhum chunk existe
	cm.LoadChunksAroundPlayer(playerPos)

	// Verificar quantos chunks foram criados
	initialCount := cm.GetLoadedChunksCount()
	t.Logf("Chunks carregados após primeira chamada: %d", initialCount)

	if initialCount == 0 {
		t.Fatal("Nenhum chunk foi carregado!")
	}

	// Obter o chunk central
	chunk0, exists := cm.Chunks[ChunkCoord{X: 0, Y: 0, Z: 0}.Key()]
	if !exists {
		t.Fatal("Chunk (0,0,0) não foi criado!")
	}

	t.Log("\n=== Passo 2: Simular múltiplos frames para carregar mais chunks ===")
	// Como LoadChunksAroundPlayer só carrega maxChunksPerFrame chunks por vez,
	// precisamos chamar várias vezes para carregar todos os vizinhos
	for i := 0; i < 20; i++ {
		cm.LoadChunksAroundPlayer(playerPos)
	}

	finalCount := cm.GetLoadedChunksCount()
	t.Logf("Chunks carregados após várias chamadas: %d", finalCount)

	// Verificar se o chunk central foi marcado para atualização
	// (deve ter sido marcado quando vizinhos foram carregados)
	t.Logf("Chunk (0,0,0) precisa atualizar meshes: %v", chunk0.NeedUpdateMeshes)

	t.Log("\n=== Passo 3: Verificar que chunks vizinhos foram criados ===")
	neighborCoords := []ChunkCoord{
		{X: 1, Y: 0, Z: 0},
		{X: -1, Y: 0, Z: 0},
		{X: 0, Y: 1, Z: 0},
		{X: 0, Y: -1, Z: 0},
	}

	neighborsExist := 0
	for _, coord := range neighborCoords {
		if _, exists := cm.Chunks[coord.Key()]; exists {
			neighborsExist++
			t.Logf("✓ Chunk vizinho %v existe", coord)
		}
	}

	if neighborsExist > 0 {
		t.Logf("Total de vizinhos diretos carregados: %d de %d", neighborsExist, len(neighborCoords))
	}
}
