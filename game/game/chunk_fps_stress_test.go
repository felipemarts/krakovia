package game

import (
	"testing"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// ========== TESTES DE STRESS PARA DETECTAR QUEDAS DE FPS REAIS ==========

// TestChunkLoading_StressTestRapidMovement simula movimento rápido que força carregamento massivo
func TestChunkLoading_StressTestRapidMovement(t *testing.T) {
	world := NewWorld()
	// Usar renderDistance grande para forçar mais carregamentos
	world.ChunkManager.RenderDistance = 5
	world.ChunkManager.UnloadDistance = 7

	player := NewPlayer(rl.NewVector3(16, 15, 16))
	player.FlyMode = true
	input := &SimulatedInput{Forward: true}

	const totalFrames = 1200 // 20 segundos
	const fpsDropThreshold = 100 * time.Millisecond

	type FrameMetrics struct {
		frameNum      int
		frameTime     time.Duration
		chunksLoaded  int
		chunksTotal   int
		playerPos     rl.Vector3
		playerChunk   ChunkCoord
		chunkChanged  bool
	}

	var worstFrames []FrameMetrics
	lastChunkCount := 0
	lastPlayerChunk := ChunkCoord{X: 0, Y: 0, Z: 0}
	fpsDrops := 0

	t.Log("=== Teste de Stress: Movimento Rápido ===")

	for i := 0; i < totalFrames; i++ {
		frameStart := time.Now()
		dt := float32(1.0 / 60.0)

		world.Update(player.Position, dt)
		player.Update(dt, world, input)

		frameTime := time.Since(frameStart)
		currentChunkCount := world.GetLoadedChunksCount()
		currentPlayerChunk := GetChunkCoordFromFloat(player.Position.X, player.Position.Y, player.Position.Z)

		chunksLoaded := 0
		if currentChunkCount > lastChunkCount {
			chunksLoaded = currentChunkCount - lastChunkCount
		}

		chunkChanged := currentPlayerChunk != lastPlayerChunk

		// Detectar frames problemáticos
		if frameTime > fpsDropThreshold {
			fpsDrops++
			metric := FrameMetrics{
				frameNum:     i,
				frameTime:    frameTime,
				chunksLoaded: chunksLoaded,
				chunksTotal:  currentChunkCount,
				playerPos:    player.Position,
				playerChunk:  currentPlayerChunk,
				chunkChanged: chunkChanged,
			}
			worstFrames = append(worstFrames, metric)

			t.Logf("⚠ FPS DROP #%d: Frame %d - %v (chunks: %d, carregados: %d, mudou chunk: %v)",
				fpsDrops, i, frameTime, currentChunkCount, chunksLoaded, chunkChanged)
		}

		lastChunkCount = currentChunkCount
		lastPlayerChunk = currentPlayerChunk
	}

	t.Logf("\n=== Resultados ===")
	t.Logf("Total de frames: %d", totalFrames)
	t.Logf("FPS drops detectados: %d (%.2f%%)", fpsDrops, float64(fpsDrops)/float64(totalFrames)*100)
	t.Logf("Chunks finais: %d", world.GetLoadedChunksCount())

	if fpsDrops > 0 {
		t.Errorf("❌ BUG DETECTADO: %d quedas de FPS durante movimento rápido!", fpsDrops)
	} else {
		t.Log("✓ Nenhuma queda de FPS durante movimento rápido")
	}
}

// TestChunkLoading_StressTestDirectionChanges simula mudanças bruscas de direção
func TestChunkLoading_StressTestDirectionChanges(t *testing.T) {
	world := NewWorld()
	world.ChunkManager.RenderDistance = 4
	world.ChunkManager.UnloadDistance = 6

	player := NewPlayer(rl.NewVector3(16, 15, 16))
	player.FlyMode = true
	input := &SimulatedInput{}

	const totalFrames = 1200
	const fpsDropThreshold = 100 * time.Millisecond

	fpsDrops := 0
	lastChunkCount := 0
	directionChangeFrames := 0

	t.Log("=== Teste de Stress: Mudanças Bruscas de Direção ===")

	for i := 0; i < totalFrames; i++ {
		// Mudar direção a cada 60 frames (1 segundo)
		if i%60 == 0 {
			input.Forward = false
			input.Back = false
			input.Left = false
			input.Right = false

			// Alternar entre direções
			switch (i / 60) % 4 {
			case 0:
				input.Forward = true
			case 1:
				input.Right = true
			case 2:
				input.Back = true
			case 3:
				input.Left = true
			}
			directionChangeFrames++
		}

		frameStart := time.Now()
		dt := float32(1.0 / 60.0)

		world.Update(player.Position, dt)
		player.Update(dt, world, input)

		frameTime := time.Since(frameStart)
		currentChunkCount := world.GetLoadedChunksCount()

		chunksChanged := currentChunkCount - lastChunkCount

		if frameTime > fpsDropThreshold {
			fpsDrops++
			t.Logf("⚠ FPS DROP: Frame %d - %v (chunks: %d, delta: %+d)",
				i, frameTime, currentChunkCount, chunksChanged)
		}

		lastChunkCount = currentChunkCount
	}

	t.Logf("\n=== Resultados ===")
	t.Logf("Mudanças de direção: %d", directionChangeFrames)
	t.Logf("FPS drops: %d (%.2f%%)", fpsDrops, float64(fpsDrops)/float64(totalFrames)*100)

	if fpsDrops > 0 {
		t.Errorf("❌ BUG DETECTADO: %d quedas de FPS durante mudanças de direção!", fpsDrops)
	}
}

// TestChunkLoading_StressTestMassiveUnload força descarregamento massivo de chunks
func TestChunkLoading_StressTestMassiveUnload(t *testing.T) {
	world := NewWorld()
	world.ChunkManager.RenderDistance = 5
	world.ChunkManager.UnloadDistance = 6 // Pequena diferença = descarrega rápido

	player := NewPlayer(rl.NewVector3(16, 15, 16))
	player.FlyMode = true
	input := &SimulatedInput{}

	const fpsDropThreshold = 100 * time.Millisecond

	// Fase 1: Carregar muitos chunks
	t.Log("Fase 1: Carregando área grande...")
	for i := 0; i < 300; i++ {
		dt := float32(1.0 / 60.0)
		world.Update(player.Position, dt)
		player.Update(dt, world, input)
	}

	initialChunks := world.GetLoadedChunksCount()
	t.Logf("Chunks carregados: %d", initialChunks)

	// Fase 2: Mover MUITO rápido para forçar descarregamento massivo
	t.Log("Fase 2: Movimento rápido para forçar descarregamento...")
	input.Forward = true

	fpsDrops := 0
	maxUnloadInSingleFrame := 0
	worstUnloadFrameTime := time.Duration(0)
	lastChunkCount := initialChunks

	for i := 0; i < 600; i++ {
		frameStart := time.Now()
		dt := float32(1.0 / 60.0)

		world.Update(player.Position, dt)
		player.Update(dt, world, input)

		frameTime := time.Since(frameStart)
		currentChunkCount := world.GetLoadedChunksCount()

		// Detectar descarregamento
		if currentChunkCount < lastChunkCount {
			chunksUnloaded := lastChunkCount - currentChunkCount

			if chunksUnloaded > maxUnloadInSingleFrame {
				maxUnloadInSingleFrame = chunksUnloaded
				worstUnloadFrameTime = frameTime
			}

			if frameTime > fpsDropThreshold {
				fpsDrops++
				t.Logf("⚠ FPS DROP durante DESCARREGAMENTO: Frame %d - %v (%d chunks removidos)",
					i, frameTime, chunksUnloaded)
			}
		}

		lastChunkCount = currentChunkCount
	}

	t.Logf("\n=== Resultados ===")
	t.Logf("Chunks iniciais: %d", initialChunks)
	t.Logf("Chunks finais: %d", world.GetLoadedChunksCount())
	t.Logf("Maior descarregamento em 1 frame: %d chunks (tempo: %v)", maxUnloadInSingleFrame, worstUnloadFrameTime)
	t.Logf("FPS drops: %d", fpsDrops)

	if fpsDrops > 0 {
		t.Errorf("❌ BUG DETECTADO: %d quedas de FPS durante descarregamento massivo!", fpsDrops)
	}
}

// TestChunkLoading_StressTestContinuousLoad carregamento contínuo sem parar
func TestChunkLoading_StressTestContinuousLoad(t *testing.T) {
	world := NewWorld()
	// RenderDistance grande = muitos chunks = mais trabalho
	world.ChunkManager.RenderDistance = 6
	world.ChunkManager.UnloadDistance = 8

	player := NewPlayer(rl.NewVector3(16, 15, 16))
	player.FlyMode = true
	input := &SimulatedInput{Forward: true}

	const totalFrames = 1800 // 30 segundos
	const fpsDropThreshold = 100 * time.Millisecond

	fpsDrops := 0
	totalChunksLoaded := 0
	consecutiveFPSDrops := 0
	maxConsecutiveFPSDrops := 0
	lastChunkCount := 0

	t.Log("=== Teste de Stress: Carregamento Contínuo (30 segundos) ===")

	for i := 0; i < totalFrames; i++ {
		frameStart := time.Now()
		dt := float32(1.0 / 60.0)

		world.Update(player.Position, dt)
		player.Update(dt, world, input)

		frameTime := time.Since(frameStart)
		currentChunkCount := world.GetLoadedChunksCount()

		if currentChunkCount > lastChunkCount {
			totalChunksLoaded += currentChunkCount - lastChunkCount
		}

		if frameTime > fpsDropThreshold {
			fpsDrops++
			consecutiveFPSDrops++

			if consecutiveFPSDrops > maxConsecutiveFPSDrops {
				maxConsecutiveFPSDrops = consecutiveFPSDrops
			}

			if consecutiveFPSDrops == 1 {
				t.Logf("⚠ FPS DROP: Frame %d - %v (chunks: %d)", i, frameTime, currentChunkCount)
			} else if consecutiveFPSDrops > 3 {
				t.Logf("⚠⚠ MÚLTIPLOS FPS DROPS CONSECUTIVOS: %d frames seguidos!", consecutiveFPSDrops)
			}
		} else {
			consecutiveFPSDrops = 0
		}

		lastChunkCount = currentChunkCount

		// Log de progresso a cada 5 segundos
		if i > 0 && i%300 == 0 {
			t.Logf("Progresso: %d segundos, chunks: %d, FPS drops até agora: %d",
				i/60, currentChunkCount, fpsDrops)
		}
	}

	t.Logf("\n=== Resultados ===")
	t.Logf("Duração: 30 segundos (%d frames)", totalFrames)
	t.Logf("Total de chunks carregados: %d", totalChunksLoaded)
	t.Logf("Chunks finais: %d", world.GetLoadedChunksCount())
	t.Logf("FPS drops totais: %d (%.2f%%)", fpsDrops, float64(fpsDrops)/float64(totalFrames)*100)
	t.Logf("Máximo de FPS drops consecutivos: %d", maxConsecutiveFPSDrops)

	if fpsDrops > 0 {
		t.Errorf("❌ BUG DETECTADO: %d quedas de FPS durante carregamento contínuo!", fpsDrops)
		if maxConsecutiveFPSDrops > 5 {
			t.Errorf("❌❌ GRAVE: %d FPS drops CONSECUTIVOS detectados!", maxConsecutiveFPSDrops)
		}
	}
}

// TestChunkLoading_StressTestChunkThrashing simula "thrashing" de chunks (carrega/descarrega repetidamente)
func TestChunkLoading_StressTestChunkThrashing(t *testing.T) {
	world := NewWorld()
	world.ChunkManager.RenderDistance = 3
	world.ChunkManager.UnloadDistance = 4 // Muito próximo = mais thrashing

	player := NewPlayer(rl.NewVector3(16, 15, 16))
	player.FlyMode = true
	input := &SimulatedInput{}

	const totalFrames = 1200
	const fpsDropThreshold = 100 * time.Millisecond

	fpsDrops := 0
	thrashingEvents := 0
	lastChunkCount := 0

	t.Log("=== Teste de Stress: Chunk Thrashing (ir e voltar) ===")

	for i := 0; i < totalFrames; i++ {
		// Alternar entre frente e trás a cada 30 frames (0.5 segundos)
		// Isso força carregar e descarregar os mesmos chunks repetidamente
		if i%30 == 0 {
			input.Forward = !input.Forward
			input.Back = !input.Back
		}

		frameStart := time.Now()
		dt := float32(1.0 / 60.0)

		world.Update(player.Position, dt)
		player.Update(dt, world, input)

		frameTime := time.Since(frameStart)
		currentChunkCount := world.GetLoadedChunksCount()

		// Detectar thrashing (chunks sendo adicionados e removidos frequentemente)
		if currentChunkCount != lastChunkCount {
			thrashingEvents++
		}

		if frameTime > fpsDropThreshold {
			fpsDrops++
			chunksChanged := currentChunkCount - lastChunkCount
			t.Logf("⚠ FPS DROP: Frame %d - %v (chunks: %d, delta: %+d)",
				i, frameTime, currentChunkCount, chunksChanged)
		}

		lastChunkCount = currentChunkCount
	}

	t.Logf("\n=== Resultados ===")
	t.Logf("Eventos de thrashing: %d", thrashingEvents)
	t.Logf("FPS drops: %d (%.2f%%)", fpsDrops, float64(fpsDrops)/float64(totalFrames)*100)

	if fpsDrops > 0 {
		t.Errorf("❌ BUG DETECTADO: %d quedas de FPS durante chunk thrashing!", fpsDrops)
	}
}

// TestChunkLoading_EdgeCase_MaxChunksPerFrame testa o limite de chunks carregados por frame
func TestChunkLoading_EdgeCase_MaxChunksPerFrame(t *testing.T) {
	world := NewWorld()
	world.ChunkManager.RenderDistance = 8 // MUITO grande
	world.ChunkManager.UnloadDistance = 10

	player := NewPlayer(rl.NewVector3(16, 15, 16))
	player.FlyMode = true
	input := &SimulatedInput{}

	const fpsDropThreshold = 100 * time.Millisecond

	maxChunksLoadedPerFrame := 0
	frameWithMaxLoading := 0
	frameTimeWithMaxLoading := time.Duration(0)
	lastChunkCount := 0
	fpsDrops := 0

	t.Log("=== Teste de Edge Case: Máximo de Chunks por Frame ===")

	for i := 0; i < 600; i++ {
		frameStart := time.Now()
		dt := float32(1.0 / 60.0)

		world.Update(player.Position, dt)
		player.Update(dt, world, input)

		frameTime := time.Since(frameStart)
		currentChunkCount := world.GetLoadedChunksCount()

		chunksLoaded := 0
		if currentChunkCount > lastChunkCount {
			chunksLoaded = currentChunkCount - lastChunkCount

			if chunksLoaded > maxChunksLoadedPerFrame {
				maxChunksLoadedPerFrame = chunksLoaded
				frameWithMaxLoading = i
				frameTimeWithMaxLoading = frameTime
			}
		}

		if frameTime > fpsDropThreshold {
			fpsDrops++
			t.Logf("⚠ FPS DROP: Frame %d - %v (%d chunks carregados neste frame)",
				i, frameTime, chunksLoaded)
		}

		lastChunkCount = currentChunkCount
	}

	t.Logf("\n=== Resultados ===")
	t.Logf("Máximo de chunks carregados em 1 frame: %d", maxChunksLoadedPerFrame)
	t.Logf("Aconteceu no frame: %d", frameWithMaxLoading)
	t.Logf("Tempo daquele frame: %v", frameTimeWithMaxLoading)
	t.Logf("FPS drops totais: %d", fpsDrops)

	// O sistema limita a 4 chunks por frame, verificar se está funcionando
	if maxChunksLoadedPerFrame > 4 {
		t.Logf("⚠ AVISO: Sistema carregou %d chunks em 1 frame (limite esperado: 4)", maxChunksLoadedPerFrame)
	}

	if fpsDrops > 0 {
		t.Errorf("❌ BUG DETECTADO: %d quedas de FPS com renderDistance=%d!",
			fpsDrops, world.ChunkManager.RenderDistance)
	}
}
