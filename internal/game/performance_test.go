package game

import (
	"testing"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Teste de performance: medir tempo de carregamento de chunks
func TestPerformance_ChunkGeneration(t *testing.T) {
	world := NewWorld()

	start := time.Now()

	// Gerar 9 chunks (3x3)
	for x := int32(-1); x <= 1; x++ {
		for z := int32(-1); z <= 1; z++ {
			chunk := NewChunk(x, 0, z)
			chunk.GenerateTerrain()
			key := ChunkCoord{X: x, Y: 0, Z: z}.Key()
			world.ChunkManager.Chunks[key] = chunk
		}
	}

	elapsed := time.Since(start)
	t.Logf("Tempo para gerar 9 chunks: %v", elapsed)

	// Deve ser rápido (< 100ms)
	if elapsed > 100*time.Millisecond {
		t.Errorf("Geração de chunks muito lenta: %v (esperado < 100ms)", elapsed)
	}
}

// Teste de performance: medir tempo de update de meshes
func TestPerformance_MeshUpdate(t *testing.T) {
	// SKIP: Este teste requer contexto OpenGL ativo para UploadToGPU
	t.Skip("Teste requer contexto OpenGL ativo (não disponível em testes)")

	chunk := NewChunk(0, 0, 0)
	chunk.GenerateTerrain()

	start := time.Now()

	// Atualizar mesh 100 vezes
	for i := 0; i < 100; i++ {
		chunk.UpdateMeshes()
	}

	elapsed := time.Since(start)
	avgTime := elapsed / 100
	t.Logf("Tempo médio para UpdateMeshes: %v", avgTime)

	// Cada update deve ser muito rápido (< 1ms)
	if avgTime > time.Millisecond {
		t.Errorf("UpdateMeshes muito lento: %v (esperado < 1ms)", avgTime)
	}
}

// Teste de performance: simular game loop
func TestPerformance_GameLoop(t *testing.T) {
	world := createChunkedFlatWorld()
	player := NewPlayer(rl.NewVector3(16, 15, 16))
	input := &SimulatedInput{}

	// Simular 600 frames (10 segundos a 60 FPS)
	totalFrames := 600
	start := time.Now()

	for i := 0; i < totalFrames; i++ {
		dt := float32(1.0 / 60.0)
		world.Update(player.Position, dt)
		player.Update(dt, world, input)
	}

	elapsed := time.Since(start)
	avgFrameTime := elapsed / time.Duration(totalFrames)
	fps := float64(totalFrames) / elapsed.Seconds()

	t.Logf("Simulação de %d frames:", totalFrames)
	t.Logf("  Tempo total: %v", elapsed)
	t.Logf("  Tempo médio por frame: %v", avgFrameTime)
	t.Logf("  FPS simulado: %.2f", fps)

	// Deve manter pelo menos 60 FPS (< 16.6ms por frame)
	if avgFrameTime > 16666*time.Microsecond {
		t.Errorf("Performance insuficiente: %.2f FPS (esperado >= 60)", fps)
	}
}

// Teste de performance: GetBlock (chamado frequentemente na colisão)
func TestPerformance_GetBlock(t *testing.T) {
	world := createChunkedFlatWorld()

	start := time.Now()

	// Simular muitas chamadas de GetBlock (como na colisão)
	iterations := 100000
	for i := 0; i < iterations; i++ {
		world.GetBlock(16, 10, 16)
	}

	elapsed := time.Since(start)
	avgTime := elapsed / time.Duration(iterations)
	t.Logf("Tempo médio para GetBlock: %v", avgTime)

	// Deve ser muito rápido (< 100ns)
	if avgTime > 100*time.Nanosecond {
		t.Errorf("GetBlock muito lento: %v (esperado < 100ns)", avgTime)
	}
}

// Teste de performance: CheckCollision
func TestPerformance_CheckCollision(t *testing.T) {
	world := createChunkedFlatWorld()
	player := NewPlayer(rl.NewVector3(16, 11, 16))

	start := time.Now()

	// Simular muitas verificações de colisão
	iterations := 10000
	for i := 0; i < iterations; i++ {
		testPos := rl.NewVector3(16, 11, 16)
		player.CheckCollision(testPos, world)
	}

	elapsed := time.Since(start)
	avgTime := elapsed / time.Duration(iterations)
	t.Logf("Tempo médio para CheckCollision: %v", avgTime)

	// Deve ser razoavelmente rápido (< 10µs)
	if avgTime > 10*time.Microsecond {
		t.Errorf("CheckCollision muito lento: %v (esperado < 10µs)", avgTime)
	}
}

// Teste de performance: carregamento gradual de chunks
func TestPerformance_ChunkLoading(t *testing.T) {
	world := NewWorld()
	player := NewPlayer(rl.NewVector3(16, 15, 16))

	// Simular movimento do player para forçar carregamento de chunks
	input := &SimulatedInput{Forward: true}

	start := time.Now()
	frameCount := 0
	slowFrames := 0

	// Simular 300 frames (5 segundos)
	for i := 0; i < 300; i++ {
		frameStart := time.Now()
		dt := float32(1.0 / 60.0)

		world.Update(player.Position, dt)
		player.Update(dt, world, input)

		frameTime := time.Since(frameStart)
		frameCount++

		// Contar frames lentos (> 16.6ms)
		if frameTime > 16666*time.Microsecond {
			slowFrames++
			t.Logf("Frame lento #%d: %v", i, frameTime)
		}
	}

	elapsed := time.Since(start)
	avgFrameTime := elapsed / time.Duration(frameCount)

	t.Logf("Carregamento gradual de chunks:")
	t.Logf("  Chunks carregados: %d", world.GetLoadedChunksCount())
	t.Logf("  Tempo total: %v", elapsed)
	t.Logf("  Tempo médio por frame: %v", avgFrameTime)
	t.Logf("  Frames lentos: %d/%d (%.1f%%)", slowFrames, frameCount, float64(slowFrames)/float64(frameCount)*100)

	// Menos de 10% dos frames devem ser lentos
	if slowFrames > frameCount/10 {
		t.Errorf("Muitos frames lentos: %d/%d (esperado < %d)", slowFrames, frameCount, frameCount/10)
	}
}

// Teste de stress: muitos chunks
func TestPerformance_ManyChunks(t *testing.T) {
	world := NewWorld()

	// Gerar área maior de chunks (7x7 = 49 chunks)
	start := time.Now()

	for x := int32(-3); x <= 3; x++ {
		for z := int32(-3); z <= 3; z++ {
			chunk := NewChunk(x, 0, z)
			chunk.GenerateTerrain()
			key := ChunkCoord{X: x, Y: 0, Z: z}.Key()
			world.ChunkManager.Chunks[key] = chunk
		}
	}

	elapsed := time.Since(start)
	t.Logf("Tempo para gerar 49 chunks: %v", elapsed)
	t.Logf("Total de blocos: %d", world.GetTotalBlocks())

	// Simular alguns frames para ver o impacto
	player := NewPlayer(rl.NewVector3(0, 15, 0))
	input := &SimulatedInput{}

	frameStart := time.Now()
	for i := 0; i < 60; i++ {
		dt := float32(1.0 / 60.0)
		world.Update(player.Position, dt)
		player.Update(dt, world, input)
	}
	frameTime := time.Since(frameStart) / 60

	t.Logf("Tempo médio por frame com 49 chunks: %v", frameTime)

	// Deve manter performance aceitável mesmo com muitos chunks
	if frameTime > 20*time.Millisecond {
		t.Errorf("Performance degradou muito com muitos chunks: %v", frameTime)
	}
}

// Benchmark: geração de chunk
func BenchmarkChunkGeneration(b *testing.B) {
	for i := 0; i < b.N; i++ {
		chunk := NewChunk(0, 0, 0)
		chunk.GenerateTerrain()
	}
}

// Benchmark: UpdateMeshes
func BenchmarkUpdateMeshes(b *testing.B) {
	// SKIP: Este benchmark requer contexto OpenGL ativo para UploadToGPU
	b.Skip("Benchmark requer contexto OpenGL ativo (não disponível em testes)")

	chunk := NewChunk(0, 0, 0)
	chunk.GenerateTerrain()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		chunk.UpdateMeshes()
	}
}

// Benchmark: GetBlock
func BenchmarkGetBlock(b *testing.B) {
	world := createChunkedFlatWorld()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		world.GetBlock(16, 10, 16)
	}
}

// Benchmark: CheckCollision
func BenchmarkCheckCollision(b *testing.B) {
	world := createChunkedFlatWorld()
	player := NewPlayer(rl.NewVector3(16, 11, 16))
	testPos := rl.NewVector3(16, 11, 16)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		player.CheckCollision(testPos, world)
	}
}
