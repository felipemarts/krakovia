package main

import (
	"math"
	"testing"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// ========== TESTES DE PERFORMANCE DE FPS DURANTE CARREGAMENTO DE CHUNKS ==========

// TestChunkLoading_NoFPSDrops verifica se há quedas de FPS ao carregar chunks
func TestChunkLoading_NoFPSDrops(t *testing.T) {
	world := NewWorld()
	world.ChunkManager.RenderDistance = 4

	player := NewPlayer(rl.NewVector3(16, 15, 16))
	player.FlyMode = true
	input := &SimulatedInput{Forward: true}

	// Métricas de performance
	const targetFrameTime = 16666 * time.Microsecond // 60 FPS = ~16.6ms por frame
	const totalFrames = 600                          // 10 segundos
	slowFrameThreshold := targetFrameTime * 2        // Frames > 33ms são considerados lentos

	slowFrames := 0
	var slowestFrame time.Duration
	slowestFrameNum := 0
	frameTimings := make([]time.Duration, totalFrames)

	for i := 0; i < totalFrames; i++ {
		frameStart := time.Now()
		dt := float32(1.0 / 60.0)

		world.Update(player.Position, dt)
		player.Update(dt, world, input)

		frameTime := time.Since(frameStart)
		frameTimings[i] = frameTime

		if frameTime > slowFrameThreshold {
			slowFrames++
			if frameTime > slowestFrame {
				slowestFrame = frameTime
				slowestFrameNum = i
			}
		}
	}

	// Calcular estatísticas
	var totalTime time.Duration
	for _, ft := range frameTimings {
		totalTime += ft
	}
	avgFrameTime := totalTime / time.Duration(totalFrames)

	// Logging detalhado
	t.Logf("=== Performance durante carregamento de chunks ===")
	t.Logf("Total de frames: %d", totalFrames)
	t.Logf("Tempo médio por frame: %v (target: %v)", avgFrameTime, targetFrameTime)
	t.Logf("Frames lentos (>%v): %d de %d (%.1f%%)",
		slowFrameThreshold, slowFrames, totalFrames,
		float64(slowFrames)/float64(totalFrames)*100)
	t.Logf("Frame mais lento: #%d com %v", slowestFrameNum, slowestFrame)
	t.Logf("Chunks carregados: %d", world.GetLoadedChunksCount())
	t.Logf("Posição final: (%.2f, %.2f, %.2f)", player.Position.X, player.Position.Y, player.Position.Z)

	// Critérios de falha
	slowFramePercentage := float64(slowFrames) / float64(totalFrames) * 100

	if slowFramePercentage > 10.0 {
		t.Errorf("Muitos frames lentos: %.1f%% (esperado < 10%%)", slowFramePercentage)
	}

	if slowestFrame > 50*time.Millisecond {
		t.Errorf("Frame mais lento é muito ruim: %v (esperado < 50ms)", slowestFrame)
	}
}

// TestChunkUnloading_NoFPSDrops verifica se há quedas de FPS ao descarregar chunks
func TestChunkUnloading_NoFPSDrops(t *testing.T) {
	world := NewWorld()
	world.ChunkManager.RenderDistance = 2
	world.ChunkManager.UnloadDistance = 4

	player := NewPlayer(rl.NewVector3(16, 15, 16))
	player.FlyMode = true
	input := &SimulatedInput{}

	// Carregar muitos chunks inicialmente
	t.Log("Fase 1: Carregando chunks iniciais...")
	for i := 0; i < 180; i++ { // 3 segundos
		dt := float32(1.0 / 60.0)
		world.Update(player.Position, dt)
		player.Update(dt, world, input)
	}

	initialChunks := world.GetLoadedChunksCount()
	t.Logf("Chunks carregados: %d", initialChunks)

	// Agora mover longe para forçar descarregamento e medir performance
	t.Log("Fase 2: Movendo longe para forçar descarregamento...")
	input.Forward = true

	const totalFrames = 600
	const targetFrameTime = 16666 * time.Microsecond
	slowFrameThreshold := targetFrameTime * 2

	slowFrames := 0
	var slowestFrame time.Duration
	slowestFrameNum := 0
	chunksUnloadedDuringTest := 0
	lastChunkCount := initialChunks

	for i := 0; i < totalFrames; i++ {
		frameStart := time.Now()
		dt := float32(1.0 / 60.0)

		world.Update(player.Position, dt)
		player.Update(dt, world, input)

		frameTime := time.Since(frameStart)

		// Detectar quando chunks são descarregados
		currentChunkCount := world.GetLoadedChunksCount()
		if currentChunkCount < lastChunkCount {
			chunksUnloaded := lastChunkCount - currentChunkCount
			chunksUnloadedDuringTest += chunksUnloaded
			t.Logf("Frame #%d: %d chunks descarregados (frame time: %v)",
				i, chunksUnloaded, frameTime)
		}
		lastChunkCount = currentChunkCount

		if frameTime > slowFrameThreshold {
			slowFrames++
			if frameTime > slowestFrame {
				slowestFrame = frameTime
				slowestFrameNum = i
			}
		}
	}

	finalChunks := world.GetLoadedChunksCount()

	// Logging
	t.Logf("=== Performance durante descarregamento de chunks ===")
	t.Logf("Chunks iniciais: %d", initialChunks)
	t.Logf("Chunks finais: %d", finalChunks)
	t.Logf("Total de chunks descarregados: %d", chunksUnloadedDuringTest)
	t.Logf("Frames lentos (>%v): %d de %d (%.1f%%)",
		slowFrameThreshold, slowFrames, totalFrames,
		float64(slowFrames)/float64(totalFrames)*100)
	t.Logf("Frame mais lento: #%d com %v", slowestFrameNum, slowestFrame)

	// Critérios de falha
	slowFramePercentage := float64(slowFrames) / float64(totalFrames) * 100

	if slowFramePercentage > 10.0 {
		t.Errorf("Muitos frames lentos durante descarregamento: %.1f%% (esperado < 10%%)",
			slowFramePercentage)
	}

	if slowestFrame > 50*time.Millisecond {
		t.Errorf("Frame mais lento durante descarregamento: %v (esperado < 50ms)", slowestFrame)
	}
}

// TestChunkLoading_FrameTimeSpikes detecta picos de frame time durante carregamento
func TestChunkLoading_FrameTimeSpikes(t *testing.T) {
	world := NewWorld()
	world.ChunkManager.RenderDistance = 4

	player := NewPlayer(rl.NewVector3(16, 15, 16))
	player.FlyMode = true
	input := &SimulatedInput{Forward: true}

	const totalFrames = 600
	frameTimings := make([]time.Duration, totalFrames)
	chunksLoadedPerFrame := make([]int, totalFrames)
	lastChunkCount := 0

	// Coletar dados
	for i := 0; i < totalFrames; i++ {
		frameStart := time.Now()
		dt := float32(1.0 / 60.0)

		world.Update(player.Position, dt)
		player.Update(dt, world, input)

		frameTimings[i] = time.Since(frameStart)

		currentChunkCount := world.GetLoadedChunksCount()
		chunksLoadedPerFrame[i] = currentChunkCount - lastChunkCount
		lastChunkCount = currentChunkCount
	}

	// Analisar correlação entre carregamento de chunks e frame time
	t.Log("=== Análise de picos de frame time ===")

	maxFrameTime := time.Duration(0)
	maxFrameIndex := 0
	framesWithChunkLoading := 0
	slowFramesDuringLoading := 0

	for i := 0; i < totalFrames; i++ {
		if frameTimings[i] > maxFrameTime {
			maxFrameTime = frameTimings[i]
			maxFrameIndex = i
		}

		// Frames onde chunks foram carregados
		if chunksLoadedPerFrame[i] > 0 {
			framesWithChunkLoading++
			if frameTimings[i] > 16666*time.Microsecond*2 {
				slowFramesDuringLoading++
			}
		}
	}

	t.Logf("Frame mais lento: #%d com %v (%d chunks carregados nesse frame)",
		maxFrameIndex, maxFrameTime, chunksLoadedPerFrame[maxFrameIndex])
	t.Logf("Frames com carregamento de chunks: %d", framesWithChunkLoading)
	t.Logf("Frames lentos durante carregamento: %d de %d (%.1f%%)",
		slowFramesDuringLoading, framesWithChunkLoading,
		float64(slowFramesDuringLoading)/float64(framesWithChunkLoading)*100)

	// Listar os 5 frames mais lentos e quantos chunks foram carregados
	t.Log("\nTop 5 frames mais lentos:")
	type FrameInfo struct {
		index        int
		time         time.Duration
		chunksLoaded int
	}
	topFrames := make([]FrameInfo, 0, 5)

	for i := 0; i < totalFrames; i++ {
		info := FrameInfo{i, frameTimings[i], chunksLoadedPerFrame[i]}

		// Inserir ordenado
		inserted := false
		for j := 0; j < len(topFrames); j++ {
			if info.time > topFrames[j].time {
				topFrames = append(topFrames[:j], append([]FrameInfo{info}, topFrames[j:]...)...)
				inserted = true
				break
			}
		}
		if !inserted && len(topFrames) < 5 {
			topFrames = append(topFrames, info)
		}
		if len(topFrames) > 5 {
			topFrames = topFrames[:5]
		}
	}

	for i, frame := range topFrames {
		t.Logf("  #%d: Frame %d - %v (%d chunks carregados)",
			i+1, frame.index, frame.time, frame.chunksLoaded)
	}

	// Verificar se picos estão correlacionados com carregamento
	if maxFrameTime > 50*time.Millisecond {
		t.Errorf("Pico de frame time muito alto: %v (frame #%d, chunks carregados: %d)",
			maxFrameTime, maxFrameIndex, chunksLoadedPerFrame[maxFrameIndex])
	}
}

// TestChunkLoading_ConsistentFrameTime verifica consistência de frame time
func TestChunkLoading_ConsistentFrameTime(t *testing.T) {
	world := NewWorld()
	world.ChunkManager.RenderDistance = 3

	player := NewPlayer(rl.NewVector3(16, 15, 16))
	player.FlyMode = true
	input := &SimulatedInput{Forward: true}

	const totalFrames = 600
	frameTimings := make([]time.Duration, totalFrames)

	// Coletar frame times
	for i := 0; i < totalFrames; i++ {
		frameStart := time.Now()
		dt := float32(1.0 / 60.0)

		world.Update(player.Position, dt)
		player.Update(dt, world, input)

		frameTimings[i] = time.Since(frameStart)
	}

	// Calcular média e desvio padrão
	var sum time.Duration
	for _, ft := range frameTimings {
		sum += ft
	}
	avg := sum / time.Duration(totalFrames)

	var varianceSum float64
	avgNanos := float64(avg.Nanoseconds())
	for _, ft := range frameTimings {
		diff := float64(ft.Nanoseconds()) - avgNanos
		varianceSum += diff * diff
	}
	variance := varianceSum / float64(totalFrames)
	stdDevNanos := math.Sqrt(variance)
	stdDev := time.Duration(stdDevNanos) * time.Nanosecond

	// Calcular percentis
	sortedTimings := make([]time.Duration, totalFrames)
	copy(sortedTimings, frameTimings)

	// Bubble sort simples (suficiente para 600 elementos)
	for i := 0; i < len(sortedTimings); i++ {
		for j := i + 1; j < len(sortedTimings); j++ {
			if sortedTimings[i] > sortedTimings[j] {
				sortedTimings[i], sortedTimings[j] = sortedTimings[j], sortedTimings[i]
			}
		}
	}

	p50 := sortedTimings[totalFrames/2]
	p95 := sortedTimings[totalFrames*95/100]
	p99 := sortedTimings[totalFrames*99/100]

	t.Log("=== Análise de consistência de frame time ===")
	t.Logf("Média: %v", avg)
	t.Logf("Desvio padrão: %v", stdDev)
	t.Logf("P50 (mediana): %v", p50)
	t.Logf("P95: %v", p95)
	t.Logf("P99: %v", p99)
	t.Logf("Min: %v", sortedTimings[0])
	t.Logf("Max: %v", sortedTimings[totalFrames-1])

	// Critérios de consistência
	// Nota: Em testes sem GPU/renderização real, os valores são muito pequenos (microsegundos)
	// então usamos critérios mais flexíveis
	if p99 > 33*time.Millisecond {
		t.Errorf("P99 muito alto: %v (esperado < 33ms)", p99)
	}

	if p95 > 25*time.Millisecond {
		t.Errorf("P95 muito alto: %v (esperado < 25ms)", p95)
	}

	// Verificar se há picos extremos (mais de 100ms)
	if sortedTimings[totalFrames-1] > 100*time.Millisecond {
		t.Errorf("Frame mais lento é extremo: %v (esperado < 100ms)", sortedTimings[totalFrames-1])
	}

	// Verificar se P99 está muito distante da mediana (indica picos isolados)
	if p99 > p50*100 && p99 > 10*time.Millisecond {
		t.Logf("AVISO: P99 (%.3fms) está %.1fx maior que a mediana (%.3fms) - possíveis picos durante carregamento",
			float64(p99.Microseconds())/1000.0,
			float64(p99)/float64(p50),
			float64(p50.Microseconds())/1000.0)
	}

	// Logging de variabilidade para informação (não falha)
	variabilityRatio := float64(sortedTimings[totalFrames-1]) / float64(avg)
	t.Logf("Razão de variabilidade (max/avg): %.2fx", variabilityRatio)

	// Apenas avisar se a variabilidade for muito alta em conjunto com valores absolutos altos
	if variabilityRatio > 50.0 && sortedTimings[totalFrames-1] > 50*time.Millisecond {
		t.Errorf("Frame times muito inconsistentes: max é %.2fx a média (max=%v)",
			variabilityRatio, sortedTimings[totalFrames-1])
	}
}

// TestChunkLoading_DetectFPSDropsOnChunkChange detecta quedas de FPS especificamente quando chunks mudam
func TestChunkLoading_DetectFPSDropsOnChunkChange(t *testing.T) {
	world := NewWorld()
	world.ChunkManager.RenderDistance = 3

	player := NewPlayer(rl.NewVector3(16, 15, 16))
	player.FlyMode = true
	input := &SimulatedInput{Forward: true}

	const totalFrames = 600
	const fpsDropThreshold = 33 * time.Millisecond // Pior que 30 FPS

	// Rastrear estado dos chunks e frame times
	type FrameData struct {
		frameTime      time.Duration
		chunkCount     int
		chunksAdded    int
		chunksRemoved  int
		playerChunk    ChunkCoord
		chunkChanged   bool
	}

	frames := make([]FrameData, totalFrames)
	lastChunkCount := 0
	lastPlayerChunk := ChunkCoord{X: 0, Y: 0, Z: 0}

	// Coletar dados detalhados
	for i := 0; i < totalFrames; i++ {
		frameStart := time.Now()
		dt := float32(1.0 / 60.0)

		world.Update(player.Position, dt)
		player.Update(dt, world, input)

		frameTime := time.Since(frameStart)
		currentChunkCount := world.GetLoadedChunksCount()
		currentPlayerChunk := GetChunkCoordFromFloat(player.Position.X, player.Position.Y, player.Position.Z)

		chunksAdded := 0
		chunksRemoved := 0
		if currentChunkCount > lastChunkCount {
			chunksAdded = currentChunkCount - lastChunkCount
		} else if currentChunkCount < lastChunkCount {
			chunksRemoved = lastChunkCount - currentChunkCount
		}

		chunkChanged := currentPlayerChunk != lastPlayerChunk

		frames[i] = FrameData{
			frameTime:     frameTime,
			chunkCount:    currentChunkCount,
			chunksAdded:   chunksAdded,
			chunksRemoved: chunksRemoved,
			playerChunk:   currentPlayerChunk,
			chunkChanged:  chunkChanged,
		}

		lastChunkCount = currentChunkCount
		lastPlayerChunk = currentPlayerChunk
	}

	// Analisar frames problemáticos
	t.Log("=== Detecção de quedas de FPS correlacionadas com chunks ===")

	fpsDropsDuringChunkChange := 0
	fpsDropsTotal := 0
	chunkChangeFrames := 0
	totalChunksLoaded := 0
	totalChunksUnloaded := 0

	for i, frame := range frames {
		if frame.chunkChanged {
			chunkChangeFrames++
		}

		if frame.chunksAdded > 0 {
			totalChunksLoaded += frame.chunksAdded
		}
		if frame.chunksRemoved > 0 {
			totalChunksUnloaded += frame.chunksRemoved
		}

		// Detectar FPS drops
		if frame.frameTime > fpsDropThreshold {
			fpsDropsTotal++

			// Verificar se foi durante mudança de chunk ou carregamento
			if frame.chunkChanged || frame.chunksAdded > 0 || frame.chunksRemoved > 0 {
				fpsDropsDuringChunkChange++
				t.Logf("QUEDA DE FPS #%d: Frame %d - %v (chunk mudou: %v, +%d chunks, -%d chunks)",
					fpsDropsDuringChunkChange, i, frame.frameTime,
					frame.chunkChanged, frame.chunksAdded, frame.chunksRemoved)
			}
		}
	}

	t.Logf("Total de frames: %d", totalFrames)
	t.Logf("Mudanças de chunk do jogador: %d", chunkChangeFrames)
	t.Logf("Total de chunks carregados: %d", totalChunksLoaded)
	t.Logf("Total de chunks descarregados: %d", totalChunksUnloaded)
	t.Logf("FPS drops totais (>%v): %d", fpsDropThreshold, fpsDropsTotal)
	t.Logf("FPS drops durante mudança de chunks: %d de %d (%.1f%%)",
		fpsDropsDuringChunkChange, fpsDropsTotal,
		func() float64 {
			if fpsDropsTotal > 0 {
				return float64(fpsDropsDuringChunkChange) / float64(fpsDropsTotal) * 100
			}
			return 0
		}())

	// Critérios de falha
	if fpsDropsTotal > totalFrames/10 {
		t.Errorf("PROBLEMA: Muitos FPS drops (%d de %d frames = %.1f%%)",
			fpsDropsTotal, totalFrames,
			float64(fpsDropsTotal)/float64(totalFrames)*100)
	}

	if fpsDropsDuringChunkChange > 0 {
		t.Errorf("PROBLEMA DETECTADO: %d quedas de FPS ocorreram durante carregamento/descarregamento de chunks",
			fpsDropsDuringChunkChange)
		t.Error("Isso indica que o sistema de chunks está causando travamentos!")
	}

	// Se não houve FPS drops, reportar sucesso
	if fpsDropsTotal == 0 {
		t.Log("✓ Nenhuma queda de FPS detectada durante teste")
	}
}
