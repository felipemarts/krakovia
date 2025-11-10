package game

import (
	"testing"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// ========== TESTES QUE SIMULAM O CENÁRIO REAL DO JOGO ==========

// TestChunkLoading_RealScenario_WithMeshUpdates simula o cenário real incluindo UpdateMeshes
func TestChunkLoading_RealScenario_WithMeshUpdates(t *testing.T) {
	// Desabilitar upload para GPU durante o teste
	DisableGPUUploadForTesting = true
	defer func() { DisableGPUUploadForTesting = false }()

	world := NewWorld()
	world.ChunkManager.RenderDistance = 4

	player := NewPlayer(rl.NewVector3(16, 15, 16))
	player.FlyMode = true
	input := &SimulatedInput{Forward: true}

	const totalFrames = 600
	const fpsDropThreshold = 33 * time.Millisecond

	type FrameAnalysis struct {
		frameNum         int
		frameTime        time.Duration
		updateTime       time.Duration // Tempo do world.Update
		chunksLoaded     int
		chunksNeedUpdate int // Chunks que precisam de mesh update
		playerChunk      ChunkCoord
	}

	fpsDrops := 0
	var problematicFrames []FrameAnalysis
	lastChunkCount := 0

	t.Log("=== Teste de Cenário Real: Incluindo UpdateMeshes ===")

	for i := 0; i < totalFrames; i++ {
		frameStart := time.Now()
		dt := float32(1.0 / 60.0)

		updateStart := time.Now()
		world.Update(player.Position, dt)
		updateTime := time.Since(updateStart)

		player.Update(dt, world, input)

		// Simular o UpdateMeshes que acontece no jogo real
		// No jogo real, chunks com NeedUpdateMeshes=true têm suas meshes atualizadas
		// NOTA: Não fazemos Upload para GPU pois não temos contexto OpenGL em testes
		chunksNeedingUpdate := 0
		meshUpdateStart := time.Now()
		for _, chunk := range world.ChunkManager.Chunks {
			if chunk.NeedUpdateMeshes {
				chunksNeedingUpdate++
				// Gerar mesh (sem upload para GPU)
				chunk.UpdateMeshesWithNeighbors(world.ChunkManager.GetBlock)
				// NÃO chamar UploadToGPU() pois não temos contexto OpenGL
				chunk.NeedUpdateMeshes = false
			}
		}
		meshUpdateTime := time.Since(meshUpdateStart)

		totalFrameTime := time.Since(frameStart)
		currentChunkCount := world.GetLoadedChunksCount()
		currentPlayerChunk := GetChunkCoordFromFloat(player.Position.X, player.Position.Y, player.Position.Z)

		chunksLoaded := 0
		if currentChunkCount > lastChunkCount {
			chunksLoaded = currentChunkCount - lastChunkCount
		}

		analysis := FrameAnalysis{
			frameNum:         i,
			frameTime:        totalFrameTime,
			updateTime:       updateTime,
			chunksLoaded:     chunksLoaded,
			chunksNeedUpdate: chunksNeedingUpdate,
			playerChunk:      currentPlayerChunk,
		}

		if totalFrameTime > fpsDropThreshold {
			fpsDrops++
			problematicFrames = append(problematicFrames, analysis)
			t.Logf("⚠ FPS DROP #%d: Frame %d - %v (update: %v, meshes: %v, chunks loaded: %d, meshes updated: %d)",
				fpsDrops, i, totalFrameTime, updateTime, meshUpdateTime, chunksLoaded, chunksNeedingUpdate)
		} else if chunksNeedingUpdate > 0 {
			// Log quando meshes são atualizadas mesmo sem FPS drop
			t.Logf("Frame %d: %d meshes atualizadas (tempo total: %v, mesh update: %v)",
				i, chunksNeedingUpdate, totalFrameTime, meshUpdateTime)
		}

		lastChunkCount = currentChunkCount
	}

	t.Logf("\n=== Resultados do Cenário Real ===")
	t.Logf("Total de frames: %d", totalFrames)
	t.Logf("FPS drops detectados: %d (%.2f%%)", fpsDrops, float64(fpsDrops)/float64(totalFrames)*100)
	t.Logf("Chunks finais: %d", world.GetLoadedChunksCount())

	if fpsDrops > 0 {
		t.Errorf("❌ BUG DETECTADO: %d quedas de FPS no cenário real!", fpsDrops)
		t.Log("\nFrames problemáticos:")
		for _, frame := range problematicFrames {
			t.Logf("  Frame %d: %v (chunks need update: %d)",
				frame.frameNum, frame.frameTime, frame.chunksNeedUpdate)
		}
	} else {
		t.Log("✓ Nenhuma queda de FPS detectada no cenário real")
	}
}

// TestChunkLoading_MeshUpdateBottleneck testa especificamente se UpdateMeshes causa gargalo
func TestChunkLoading_MeshUpdateBottleneck(t *testing.T) {
	// Desabilitar upload para GPU durante o teste
	DisableGPUUploadForTesting = true
	defer func() { DisableGPUUploadForTesting = false }()

	world := NewWorld()
	world.ChunkManager.RenderDistance = 5 // Maior = mais chunks para atualizar

	player := NewPlayer(rl.NewVector3(16, 15, 16))
	player.FlyMode = true
	input := &SimulatedInput{Forward: true}

	const totalFrames = 600
	const fpsDropThreshold = 33 * time.Millisecond

	type MeshUpdateStats struct {
		frameNum          int
		numMeshesUpdated  int
		meshUpdateTime    time.Duration
		totalFrameTime    time.Duration
		meshTimePercentage float64
	}

	var meshStats []MeshUpdateStats
	fpsDrops := 0
	totalMeshUpdates := 0

	t.Log("=== Teste de Gargalo: UpdateMeshes ===")

	for i := 0; i < totalFrames; i++ {
		frameStart := time.Now()
		dt := float32(1.0 / 60.0)

		world.Update(player.Position, dt)
		player.Update(dt, world, input)

		// Medir tempo de UpdateMeshes separadamente (sem GPU upload)
		meshStart := time.Now()
		meshesUpdated := 0
		for _, chunk := range world.ChunkManager.Chunks {
			if chunk.NeedUpdateMeshes {
				chunk.UpdateMeshesWithNeighbors(world.ChunkManager.GetBlock)
				chunk.NeedUpdateMeshes = false
				meshesUpdated++
			}
		}
		meshTime := time.Since(meshStart)
		totalTime := time.Since(frameStart)

		if meshesUpdated > 0 {
			totalMeshUpdates += meshesUpdated
			meshPercentage := float64(meshTime) / float64(totalTime) * 100

			stat := MeshUpdateStats{
				frameNum:          i,
				numMeshesUpdated:  meshesUpdated,
				meshUpdateTime:    meshTime,
				totalFrameTime:    totalTime,
				meshTimePercentage: meshPercentage,
			}
			meshStats = append(meshStats, stat)

			if totalTime > fpsDropThreshold {
				fpsDrops++
				t.Logf("⚠ FPS DROP: Frame %d - %v (UpdateMeshes: %d chunks, %v, %.1f%% do tempo do frame)",
					i, totalTime, meshesUpdated, meshTime, meshPercentage)
			} else if meshPercentage > 50 {
				t.Logf("⚠ AVISO: Frame %d - UpdateMeshes consumiu %.1f%% do tempo (%d meshes, %v)",
					i, meshPercentage, meshesUpdated, meshTime)
			}
		}
	}

	// Análise estatística
	if len(meshStats) > 0 {
		maxMeshesInFrame := 0
		maxMeshTime := time.Duration(0)
		var totalMeshTime time.Duration

		for _, stat := range meshStats {
			if stat.numMeshesUpdated > maxMeshesInFrame {
				maxMeshesInFrame = stat.numMeshesUpdated
			}
			if stat.meshUpdateTime > maxMeshTime {
				maxMeshTime = stat.meshUpdateTime
			}
			totalMeshTime += stat.meshUpdateTime
		}

		avgMeshTime := totalMeshTime / time.Duration(len(meshStats))

		t.Logf("\n=== Estatísticas de UpdateMeshes ===")
		t.Logf("Frames com mesh updates: %d de %d", len(meshStats), totalFrames)
		t.Logf("Total de mesh updates: %d", totalMeshUpdates)
		t.Logf("Máximo de meshes em 1 frame: %d", maxMeshesInFrame)
		t.Logf("Tempo máximo de mesh update: %v", maxMeshTime)
		t.Logf("Tempo médio de mesh update: %v", avgMeshTime)
		t.Logf("FPS drops: %d", fpsDrops)

		if maxMeshTime > 16*time.Millisecond {
			t.Logf("⚠ GARGALO DETECTADO: UpdateMeshes levou %v (> 16ms)", maxMeshTime)
		}
	}

	if fpsDrops > 0 {
		t.Errorf("❌ BUG DETECTADO: %d quedas de FPS causadas por UpdateMeshes!", fpsDrops)
	}
}

// TestChunkLoading_NeighborMarkingPerformance testa se marcar vizinhos causa problemas
func TestChunkLoading_NeighborMarkingPerformance(t *testing.T) {
	// Desabilitar upload para GPU durante o teste
	DisableGPUUploadForTesting = true
	defer func() { DisableGPUUploadForTesting = false }()

	world := NewWorld()
	world.ChunkManager.RenderDistance = 4

	player := NewPlayer(rl.NewVector3(16, 15, 16))
	player.FlyMode = true
	input := &SimulatedInput{Forward: true}

	const totalFrames = 600
	const fpsDropThreshold = 33 * time.Millisecond

	fpsDrops := 0
	totalChunksMarked := 0
	maxChunksMarkedInFrame := 0

	t.Log("=== Teste: Performance de MarkNeighborsForUpdate ===")

	for i := 0; i < totalFrames; i++ {
		frameStart := time.Now()
		dt := float32(1.0 / 60.0)

		lastChunkCount := world.GetLoadedChunksCount()
		world.Update(player.Position, dt)
		player.Update(dt, world, input)
		currentChunkCount := world.GetLoadedChunksCount()

		// Contar quantos chunks foram marcados para update neste frame
		chunksMarked := 0
		for _, chunk := range world.ChunkManager.Chunks {
			if chunk.NeedUpdateMeshes {
				chunksMarked++
			}
		}

		if chunksMarked > maxChunksMarkedInFrame {
			maxChunksMarkedInFrame = chunksMarked
		}

		if currentChunkCount > lastChunkCount {
			totalChunksMarked += chunksMarked
		}

		frameTime := time.Since(frameStart)

		if frameTime > fpsDropThreshold {
			fpsDrops++
			t.Logf("⚠ FPS DROP: Frame %d - %v (chunks marcados: %d)",
				i, frameTime, chunksMarked)
		}
	}

	t.Logf("\n=== Resultados ===")
	t.Logf("Total de chunks marcados: %d", totalChunksMarked)
	t.Logf("Máximo de chunks marcados em 1 frame: %d", maxChunksMarkedInFrame)
	t.Logf("FPS drops: %d", fpsDrops)

	if fpsDrops > 0 {
		t.Errorf("❌ BUG DETECTADO: %d quedas de FPS!", fpsDrops)
	}
}

// TestChunkLoading_RealWorld_30Seconds simula 30 segundos de jogo real
func TestChunkLoading_RealWorld_30Seconds(t *testing.T) {
	// Desabilitar upload para GPU durante o teste
	DisableGPUUploadForTesting = true
	defer func() { DisableGPUUploadForTesting = false }()

	world := NewWorld()
	world.ChunkManager.RenderDistance = 5 // Valor típico do jogo

	player := NewPlayer(rl.NewVector3(16, 15, 16))
	player.FlyMode = true
	input := &SimulatedInput{Forward: true}

	const totalFrames = 1800 // 30 segundos
	const fpsDropThreshold = 33 * time.Millisecond
	const severeDropThreshold = 50 * time.Millisecond

	fpsDrops := 0
	severeDrops := 0
	totalMeshUpdates := 0

	// Estatísticas por segundo
	type SecondStats struct {
		second       int
		avgFrameTime time.Duration
		fpsDrops     int
		meshUpdates  int
		chunksLoaded int
	}

	secondStats := make([]SecondStats, 30)

	t.Log("=== Teste de Mundo Real: 30 Segundos de Jogo ===")

	for i := 0; i < totalFrames; i++ {
		frameStart := time.Now()
		dt := float32(1.0 / 60.0)

		world.Update(player.Position, dt)
		player.Update(dt, world, input)

		// UpdateMeshes como no jogo real (sem GPU upload)
		meshesUpdated := 0
		for _, chunk := range world.ChunkManager.Chunks {
			if chunk.NeedUpdateMeshes {
				chunk.UpdateMeshesWithNeighbors(world.ChunkManager.GetBlock)
				chunk.NeedUpdateMeshes = false
				meshesUpdated++
				totalMeshUpdates++
			}
		}

		frameTime := time.Since(frameStart)

		// Estatísticas por segundo
		second := i / 60
		if second < 30 {
			secondStats[second].second = second
			secondStats[second].avgFrameTime += frameTime
			secondStats[second].meshUpdates += meshesUpdated

			if frameTime > fpsDropThreshold {
				secondStats[second].fpsDrops++
			}
		}

		if frameTime > severeDropThreshold {
			severeDrops++
			t.Logf("⚠⚠ SEVERE DROP: Frame %d (%.1fs) - %v (meshes: %d)",
				i, float64(i)/60.0, frameTime, meshesUpdated)
		} else if frameTime > fpsDropThreshold {
			fpsDrops++
		}

		// Log a cada 5 segundos
		if i > 0 && i%300 == 0 {
			t.Logf("Progresso: %.0fs - chunks: %d, FPS drops: %d, severe: %d",
				float64(i)/60.0, world.GetLoadedChunksCount(), fpsDrops+severeDrops, severeDrops)
		}
	}

	// Calcular médias
	for i := range secondStats {
		secondStats[i].avgFrameTime /= 60
		secondStats[i].chunksLoaded = world.GetLoadedChunksCount()
	}

	t.Logf("\n=== Resultados de 30 Segundos ===")
	t.Logf("Total de frames: %d", totalFrames)
	t.Logf("FPS drops (>33ms): %d (%.2f%%)", fpsDrops+severeDrops,
		float64(fpsDrops+severeDrops)/float64(totalFrames)*100)
	t.Logf("Severe drops (>50ms): %d", severeDrops)
	t.Logf("Total mesh updates: %d", totalMeshUpdates)
	t.Logf("Chunks finais: %d", world.GetLoadedChunksCount())

	// Encontrar o segundo com mais problemas
	worstSecond := 0
	maxDrops := 0
	for i, stat := range secondStats {
		if stat.fpsDrops > maxDrops {
			maxDrops = stat.fpsDrops
			worstSecond = i
		}
	}

	if maxDrops > 0 {
		t.Logf("\nPior segundo: %d (FPS drops: %d, mesh updates: %d)",
			worstSecond, secondStats[worstSecond].fpsDrops, secondStats[worstSecond].meshUpdates)
	}

	if severeDrops > 0 {
		t.Errorf("❌❌ BUG GRAVE DETECTADO: %d quedas severas de FPS (>50ms)!", severeDrops)
	} else if fpsDrops > 0 {
		t.Errorf("❌ BUG DETECTADO: %d quedas de FPS!", fpsDrops)
	} else {
		t.Log("✓ Teste de 30 segundos passou sem quedas de FPS")
	}
}
