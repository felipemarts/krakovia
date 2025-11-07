package main

import (
	"testing"
)

// TestChunkBorderVisibilityBug testa o bug específico onde blocos na divisa entre chunks são sempre visíveis
func TestChunkBorderVisibilityBug(t *testing.T) {
	// Criar ChunkManager
	cm := NewChunkManager(5)

	// Criar um cubo 2x2x2 de chunks completamente preenchidos
	// para garantir que o chunk central (0,0,0) está completamente cercado
	for cx := int32(-1); cx <= 1; cx++ {
		for cy := int32(-1); cy <= 1; cy++ {
			for cz := int32(-1); cz <= 1; cz++ {
				chunk := NewChunk(cx, cy, cz)
				// Preencher completamente com pedra
				for x := int32(0); x < ChunkSize; x++ {
					for y := int32(0); y < ChunkHeight; y++ {
						for z := int32(0); z < ChunkSize; z++ {
							chunk.Blocks[x][y][z] = BlockStone
						}
					}
				}
				cm.Chunks[ChunkCoord{X: cx, Y: cy, Z: cz}.Key()] = chunk
			}
		}
	}

	chunk0 := cm.Chunks[ChunkCoord{X: 0, Y: 0, Z: 0}.Key()]

	// Atualizar meshes do chunk 0 sem considerar vizinhos
	t.Log("=== Testando UpdateMeshes (sem chunks vizinhos) ===")
	chunk0.NeedUpdateMeshes = true
	chunk0.UpdateMeshes()
	blocksWithoutNeighbors := len(chunk0.GrassTransforms) + len(chunk0.DirtTransforms) + len(chunk0.StoneTransforms)
	t.Logf("Blocos renderizados SEM considerar chunks vizinhos: %d", blocksWithoutNeighbors)

	// Atualizar meshes do chunk 0 COM consideração de vizinhos
	t.Log("\n=== Testando UpdateMeshesWithNeighbors (com chunks vizinhos) ===")
	chunk0.NeedUpdateMeshes = true
	chunk0.UpdateMeshesWithNeighbors(cm.GetBlock)
	blocksWithNeighbors := len(chunk0.GrassTransforms) + len(chunk0.DirtTransforms) + len(chunk0.StoneTransforms)
	t.Logf("Blocos renderizados COM considerar chunks vizinhos: %d", blocksWithNeighbors)

	// O chunk (0,0,0) está COMPLETAMENTE cercado por outros chunks também preenchidos
	// Portanto, NENHUMA face externa deveria ser visível
	// Todos os blocos deveriam estar ocultos = 0 blocos renderizados

	// Testar especificamente blocos na borda X=31 (interface com chunk 1)
	t.Log("\n=== Testando blocos específicos na borda X=31 ===")

	borderBlocksVisible := 0
	for y := int32(0); y < ChunkHeight; y++ {
		for z := int32(0); z < ChunkSize; z++ {
			wx, wy, wz := int32(31), y, z

			// Verificar se o bloco está oculto
			isHidden := cm.IsBlockHidden(wx, wy, wz)

			if !isHidden {
				borderBlocksVisible++
				if borderBlocksVisible <= 5 { // Mostrar apenas os primeiros 5
					t.Logf("  Bloco (%d,%d,%d) está VISÍVEL (deveria estar oculto!)", wx, wy, wz)

					// Debug: verificar vizinhos
					for _, dir := range []struct{ name string; dx, dy, dz int32 }{
						{"X+", 1, 0, 0}, {"X-", -1, 0, 0},
						{"Y+", 0, 1, 0}, {"Y-", 0, -1, 0},
						{"Z+", 0, 0, 1}, {"Z-", 0, 0, -1},
					} {
						nb := cm.GetBlock(wx+dir.dx, wy+dir.dy, wz+dir.dz)
						t.Logf("    Vizinho %s: %v", dir.name, nb)
					}
				}
			}
		}
	}

	t.Logf("Total de blocos VISÍVEIS na borda X=31: %d de %d (esperado: 0)",
		borderBlocksVisible, ChunkSize*ChunkHeight)

	if borderBlocksVisible > 0 {
		t.Errorf("BUG CONFIRMADO: %d blocos na borda X=31 estão visíveis quando deveriam estar ocultos!",
			borderBlocksVisible)
	}

	// Verificar se UpdateMeshesWithNeighbors está respeitando a oclusão
	t.Log("\n=== Comparando contagens ===")

	// Um chunk totalmente preenchido COMPLETAMENTE cercado por outros chunks preenchidos
	// deveria renderizar 0 blocos (todos ocultos)
	t.Logf("Blocos renderizados SEM considerar vizinhos: %d", blocksWithoutNeighbors)
	t.Logf("Blocos renderizados COM considerar vizinhos: %d (esperado: 0)", blocksWithNeighbors)

	if blocksWithNeighbors > 0 {
		t.Errorf("Chunk completamente cercado ainda renderiza %d blocos! Esperado: 0",
			blocksWithNeighbors)
	}
}

// TestUpdateMeshesCallingCorrectFunction testa se UpdateMeshes está sendo chamado corretamente no Render
func TestUpdateMeshesCallingCorrectFunction(t *testing.T) {
	// Criar ChunkManager
	cm := NewChunkManager(5)

	// Criar chunk único
	chunk := NewChunk(0, 0, 0)

	// Preencher com pedra
	for x := int32(0); x < ChunkSize; x++ {
		for y := int32(0); y < ChunkHeight; y++ {
			for z := int32(0); z < ChunkSize; z++ {
				chunk.Blocks[x][y][z] = BlockStone
			}
		}
	}

	cm.Chunks[ChunkCoord{X: 0, Y: 0, Z: 0}.Key()] = chunk

	// Marcar como necessitando atualização
	chunk.NeedUpdateMeshes = true

	t.Log("=== Testando chamada de UpdateMeshes via chunk.Render ===")

	// Simular o que Render faz no chunk.go
	// Nota: Render chama UpdateMeshesWithNeighbors quando getBlockFunc != nil
	getBlockFunc := cm.GetBlock
	if chunk.NeedUpdateMeshes && getBlockFunc != nil {
		t.Log("Chamando UpdateMeshesWithNeighbors com cm.GetBlock")
		chunk.UpdateMeshesWithNeighbors(getBlockFunc)
	} else if chunk.NeedUpdateMeshes {
		t.Log("Chamando UpdateMeshes (SEM getBlockFunc)")
		chunk.UpdateMeshes()
	}

	totalBlocks := len(chunk.GrassTransforms) + len(chunk.DirtTransforms) + len(chunk.StoneTransforms)
	t.Logf("Blocos renderizados: %d", totalBlocks)

	// Verificar se foi usado o método correto
	if !chunk.NeedUpdateMeshes {
		t.Log("✓ NeedUpdateMeshes foi resetado corretamente")
	} else {
		t.Error("✗ NeedUpdateMeshes não foi resetado!")
	}
}
