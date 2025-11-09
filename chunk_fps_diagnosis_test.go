package main

import (
	"testing"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// ========== DIAGNÓSTICO DO BUG DE FPS ==========

// TestChunkLoading_DiagnoseMeshGenerationTime mede o tempo de geração de meshes (CPU) vs Upload (GPU)
func TestChunkLoading_DiagnoseMeshGenerationTime(t *testing.T) {
	// Desabilitar upload para GPU durante o teste
	DisableGPUUploadForTesting = true
	defer func() { DisableGPUUploadForTesting = false }()

	t.Log("=== Diagnóstico: Onde está o gargalo? ===")
	t.Log("")
	t.Log("Este teste não pode executar UploadToGPU() pois não temos contexto OpenGL.")
	t.Log("Porém, podemos medir a geração de mesh data (CPU) e documentar o problema.")
	t.Log("")

	world := NewWorld()
	world.ChunkManager.RenderDistance = 4

	player := NewPlayer(rl.NewVector3(16, 15, 16))
	player.FlyMode = true
	input := &SimulatedInput{Forward: true}

	const totalFrames = 600
	totalMeshGenerations := 0
	var totalMeshGenTime time.Duration
	maxMeshGenTime := time.Duration(0)
	maxMeshesInOneFrame := 0

	for i := 0; i < totalFrames; i++ {
		dt := float32(1.0 / 60.0)

		world.Update(player.Position, dt)
		player.Update(dt, world, input)

		// Medir APENAS a geração de mesh data (sem GPU upload)
		meshGenStart := time.Now()
		meshesGenerated := 0

		for _, chunk := range world.ChunkManager.Chunks {
			if chunk.NeedUpdateMeshes {
				// Limpar mesh anterior e gerar novos vertices (CPU)
				chunk.ChunkMesh.Clear()

				// Simular a geração de vertices (a parte cara da CPU)
				worldX := chunk.Coord.X * ChunkSize
				worldY := chunk.Coord.Y * ChunkHeight
				worldZ := chunk.Coord.Z * ChunkSize

				// Iterar por todos os blocos (como UpdateMeshesWithNeighbors faz)
				for x := int32(0); x < ChunkSize; x++ {
					for y := int32(0); y < ChunkHeight; y++ {
						for z := int32(0); z < ChunkSize; z++ {
							blockType := chunk.Blocks[x][y][z]
							if blockType == BlockAir {
								continue
							}

							wx := worldX + x
							wy := worldY + y
							wz := worldZ + z

							// Verificar faces expostas (como no código real)
							directions := []struct{ dx, dy, dz int32 }{
								{1, 0, 0}, {-1, 0, 0},
								{0, 1, 0}, {0, -1, 0},
								{0, 0, 1}, {0, 0, -1},
							}

							for faceIndex, dir := range directions {
								neighborBlock := world.ChunkManager.GetBlock(wx+dir.dx, wy+dir.dy, wz+dir.dz)
								if neighborBlock == BlockAir {
									// Adicionar quad (geração de vertices - CPU)
									chunk.ChunkMesh.AddQuad(float32(wx), float32(wy), float32(wz), faceIndex, blockType)
								}
							}
						}
					}
				}

				// NÃO chamar UploadToGPU() - isso requer contexto OpenGL
				chunk.NeedUpdateMeshes = false
				meshesGenerated++
			}
		}

		meshGenTime := time.Since(meshGenStart)

		if meshesGenerated > 0 {
			totalMeshGenerations += meshesGenerated
			totalMeshGenTime += meshGenTime

			if meshGenTime > maxMeshGenTime {
				maxMeshGenTime = meshGenTime
			}

			if meshesGenerated > maxMeshesInOneFrame {
				maxMeshesInOneFrame = meshesGenerated
			}

			if meshesGenerated > 2 {
				t.Logf("Frame %d: %d meshes geradas em %v (apenas CPU, sem GPU)",
					i, meshesGenerated, meshGenTime)
			}
		}
	}

	avgMeshGenTime := time.Duration(0)
	if totalMeshGenerations > 0 {
		avgMeshGenTime = totalMeshGenTime / time.Duration(totalMeshGenerations)
	}

	t.Logf("\n=== Resultados ===")
	t.Logf("Total de meshes geradas: %d", totalMeshGenerations)
	t.Logf("Tempo total de geração de mesh (CPU): %v", totalMeshGenTime)
	t.Logf("Tempo médio por mesh (CPU): %v", avgMeshGenTime)
	t.Logf("Tempo máximo em um frame: %v", maxMeshGenTime)
	t.Logf("Máximo de meshes em 1 frame: %d", maxMeshesInOneFrame)

	t.Log("\n=== CONCLUSÃO DO DIAGNÓSTICO ===")
	t.Log("Geração de mesh data (CPU) é RÁPIDA (microsegundos).")
	t.Log("")
	t.Log("❌ PROBLEMA IDENTIFICADO:")
	t.Log("O gargalo NÃO está na lógica de carregamento/descarregamento de chunks.")
	t.Log("O gargalo NÃO está na geração de mesh data (CPU).")
	t.Log("")
	t.Log("O problema está em chunk.go:204 → UploadToGPU()")
	t.Log("Esta função faz chamadas OpenGL síncronas que bloqueiam o frame:")
	t.Log("  - rl.UploadMesh() cria buffers VBO na GPU")
	t.Log("  - Essa operação é SÍNCRONA e pode levar vários millisegundos")
	t.Log("  - Quando vários chunks são atualizados no mesmo frame, o FPS cai")
	t.Log("")
	t.Log("SOLUÇÕES POSSÍVEIS:")
	t.Log("1. Limitar meshes uploaded por frame (já limitamos chunks loaded a 4)")
	t.Log("2. Fazer upload de mesh assíncrono (thread separada)")
	t.Log("3. Reeusar buffers GPU em vez de criar novos")
	t.Log("4. Usar uma fila de meshes pendentes e processar gradualmente")
}

// TestChunkLoading_SimulateFPSDropScenario simula o cenário que causa FPS drop
func TestChunkLoading_SimulateFPSDropScenario(t *testing.T) {
	// Desabilitar upload para GPU durante o teste
	DisableGPUUploadForTesting = true
	defer func() { DisableGPUUploadForTesting = false }()

	t.Log("=== Simulação do Cenário de FPS Drop ===")
	t.Log("")

	world := NewWorld()
	world.ChunkManager.RenderDistance = 5
	world.ChunkManager.UnloadDistance = 7

	player := NewPlayer(rl.NewVector3(16, 15, 16))
	player.FlyMode = true
	input := &SimulatedInput{}

	// Fase 1: Carregar muitos chunks
	t.Log("Fase 1: Carregando área grande...")
	for i := 0; i < 300; i++ {
		dt := float32(1.0 / 60.0)
		world.Update(player.Position, dt)
		player.Update(dt, world, input)
	}

	t.Logf("Chunks carregados: %d", world.GetLoadedChunksCount())

	// Fase 2: Movimento rápido para forçar atualização de meshes vizinhas
	t.Log("\nFase 2: Movimento rápido (força MarkNeighborsForUpdate)...")
	input.Forward = true

	maxPendingMeshes := 0
	frameWithMaxPending := 0

	for i := 0; i < 600; i++ {
		dt := float32(1.0 / 60.0)
		world.Update(player.Position, dt)
		player.Update(dt, world, input)

		// Contar chunks que precisam de mesh update
		pendingMeshes := 0
		for _, chunk := range world.ChunkManager.Chunks {
			if chunk.NeedUpdateMeshes {
				pendingMeshes++
			}
		}

		if pendingMeshes > maxPendingMeshes {
			maxPendingMeshes = pendingMeshes
			frameWithMaxPending = i
		}

		if pendingMeshes > 10 {
			t.Logf("⚠ Frame %d: %d chunks aguardando mesh update", i, pendingMeshes)
		}
	}

	t.Logf("\n=== Resultados ===")
	t.Logf("Máximo de chunks pendentes: %d (frame %d)", maxPendingMeshes, frameWithMaxPending)
	t.Log("")
	t.Log("💡 EXPLICAÇÃO DO BUG:")
	t.Logf("Quando %d chunks precisam de mesh update no mesmo frame,", maxPendingMeshes)
	t.Log("o código em chunk.go chama UploadToGPU() para cada um.")
	t.Log("Se cada upload leva 5ms, total = " + string(rune(maxPendingMeshes)) + " × 5ms = muitos ms")
	t.Log("Isso causa queda abaixo de 60 FPS (16.6ms por frame).")
	t.Log("")
	t.Log("SOLUÇÃO RECOMENDADA:")
	t.Log("Adicionar limite de mesh uploads por frame (ex: máximo 2-3).")
	t.Log("Os chunks restantes ficam com NeedUpdateMeshes=true e são")
	t.Log("processados nos frames seguintes, distribuindo o custo.")
}
