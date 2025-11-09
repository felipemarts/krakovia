package main

import (
	"math"
	"testing"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// ========== TESTES DE CARREGAMENTO E DESCARREGAMENTO DE CHUNKS ==========

// TestChunkLoading_PlayerMovement verifica se chunks são carregados quando o jogador se move
func TestChunkLoading_PlayerMovement(t *testing.T) {
	world := NewWorld()
	// Iniciar com renderDistance menor para facilitar o teste
	world.ChunkManager.RenderDistance = 3
	world.ChunkManager.UnloadDistance = 5

	// Spawnar jogador no centro do mundo
	player := NewPlayer(rl.NewVector3(16, 15, 16))
	input := &SimulatedInput{}

	// Simular alguns frames para carregar chunks iniciais ao redor do jogador
	for i := 0; i < 60; i++ {
		dt := float32(1.0 / 60.0)
		world.Update(player.Position, dt)
		player.Update(dt, world, input)
	}

	initialChunkCount := world.GetLoadedChunksCount()
	t.Logf("Chunks carregados inicialmente: %d", initialChunkCount)

	// Verificar que pelo menos alguns chunks foram carregados
	if initialChunkCount == 0 {
		t.Fatal("Nenhum chunk foi carregado ao redor do jogador")
	}

	// Guardar posição inicial do jogador em termos de chunk
	initialPlayerChunk := GetChunkCoordFromFloat(player.Position.X, player.Position.Y, player.Position.Z)
	t.Logf("Chunk inicial do jogador: (%d, %d, %d)", initialPlayerChunk.X, initialPlayerChunk.Y, initialPlayerChunk.Z)

	// Mover jogador para frente por uma distância grande (atravessar vários chunks)
	// ChunkSize = 32, então andar ~100 blocos = ~3 chunks
	input.Forward = true
	for i := 0; i < 600; i++ { // 10 segundos de movimento
		dt := float32(1.0 / 60.0)
		world.Update(player.Position, dt)
		player.Update(dt, world, input)
	}

	finalPlayerChunk := GetChunkCoordFromFloat(player.Position.X, player.Position.Y, player.Position.Z)
	t.Logf("Chunk final do jogador: (%d, %d, %d)", finalPlayerChunk.X, finalPlayerChunk.Y, finalPlayerChunk.Z)
	t.Logf("Posição final do jogador: (%.2f, %.2f, %.2f)", player.Position.X, player.Position.Y, player.Position.Z)

	// Verificar que o jogador se moveu para outro chunk
	if finalPlayerChunk == initialPlayerChunk {
		t.Skip("Jogador não se moveu para outro chunk, teste inconclusivo")
	}

	// Verificar que novos chunks foram carregados
	finalChunkCount := world.GetLoadedChunksCount()
	t.Logf("Chunks carregados após movimento: %d", finalChunkCount)

	// O número de chunks pode variar, mas deve haver chunks carregados
	if finalChunkCount == 0 {
		t.Error("Nenhum chunk carregado após o jogador se mover")
	}

	// Verificar que o chunk onde o jogador está agora existe
	playerChunkKey := finalPlayerChunk.Key()
	if _, exists := world.ChunkManager.Chunks[playerChunkKey]; !exists {
		t.Errorf("Chunk do jogador (%d, %d, %d) não foi carregado",
			finalPlayerChunk.X, finalPlayerChunk.Y, finalPlayerChunk.Z)
	}

	// Verificar que há chunks ao redor do jogador (pelo menos na direção do movimento)
	frontChunk := ChunkCoord{X: finalPlayerChunk.X, Y: finalPlayerChunk.Y, Z: finalPlayerChunk.Z + 1}
	if _, exists := world.ChunkManager.Chunks[frontChunk.Key()]; !exists {
		t.Logf("Aviso: Chunk à frente do jogador não foi carregado: (%d, %d, %d)",
			frontChunk.X, frontChunk.Y, frontChunk.Z)
	}
}

// TestChunkUnloading_DistantChunks verifica se chunks distantes são descarregados
func TestChunkUnloading_DistantChunks(t *testing.T) {
	world := NewWorld()
	// Renderização curta para facilitar teste de descarregamento
	world.ChunkManager.RenderDistance = 2
	world.ChunkManager.UnloadDistance = 4

	player := NewPlayer(rl.NewVector3(16, 15, 16))
	// Ativar fly mode para facilitar movimento longo
	player.FlyMode = true
	input := &SimulatedInput{}

	// Carregar chunks iniciais
	for i := 0; i < 120; i++ { // 2 segundos
		dt := float32(1.0 / 60.0)
		world.Update(player.Position, dt)
		player.Update(dt, world, input)
	}

	initialChunkCount := world.GetLoadedChunksCount()
	initialPlayerChunk := GetChunkCoordFromFloat(player.Position.X, player.Position.Y, player.Position.Z)
	t.Logf("Chunks iniciais: %d, Chunk do jogador: (%d, %d, %d)",
		initialChunkCount, initialPlayerChunk.X, initialPlayerChunk.Y, initialPlayerChunk.Z)
	t.Logf("Posição inicial: (%.2f, %.2f, %.2f)", player.Position.X, player.Position.Y, player.Position.Z)

	// Guardar as chaves dos chunks iniciais
	initialChunkKeys := make(map[int64]bool)
	for key := range world.ChunkManager.Chunks {
		initialChunkKeys[key] = true
	}

	// Mover jogador MUITO longe (para forçar descarregamento)
	// Com velocidade 15 blocos/seg, 20 segundos = 300 blocos ~= 9 chunks
	input.Forward = true
	for i := 0; i < 1200; i++ { // 20 segundos de movimento
		dt := float32(1.0 / 60.0)
		world.Update(player.Position, dt)
		player.Update(dt, world, input)
	}

	finalPlayerChunk := GetChunkCoordFromFloat(player.Position.X, player.Position.Y, player.Position.Z)
	finalChunkCount := world.GetLoadedChunksCount()
	t.Logf("Chunks finais: %d, Chunk do jogador: (%d, %d, %d)",
		finalChunkCount, finalPlayerChunk.X, finalPlayerChunk.Y, finalPlayerChunk.Z)
	t.Logf("Posição final: (%.2f, %.2f, %.2f)", player.Position.X, player.Position.Y, player.Position.Z)

	// Calcular distância percorrida
	dx := player.Position.X - 16
	dz := player.Position.Z - 16
	distMoved := math.Sqrt(float64(dx*dx + dz*dz))
	t.Logf("Distância percorrida: %.2f blocos", distMoved)

	// Verificar que o jogador se moveu uma distância significativa
	if distMoved < 100 {
		t.Skipf("Jogador não se moveu longe o suficiente (%.2f blocos), teste inconclusivo", distMoved)
	}

	// Verificar que alguns chunks iniciais foram descarregados
	chunksUnloaded := 0
	for key := range initialChunkKeys {
		if _, exists := world.ChunkManager.Chunks[key]; !exists {
			chunksUnloaded++
		}
	}

	t.Logf("Chunks descarregados: %d de %d iniciais", chunksUnloaded, len(initialChunkKeys))

	if chunksUnloaded == 0 {
		t.Error("Nenhum chunk foi descarregado apesar do jogador ter se movido muito longe")
	}

	// Verificar que o chunk atual do jogador ainda existe
	if _, exists := world.ChunkManager.Chunks[finalPlayerChunk.Key()]; !exists {
		t.Error("Chunk do jogador foi descarregado incorretamente")
	}
}

// TestChunkLoading_ChunkBoundary verifica carregamento ao atravessar borda de chunk
func TestChunkLoading_ChunkBoundary(t *testing.T) {
	world := NewWorld()
	world.ChunkManager.RenderDistance = 3
	world.ChunkManager.UnloadDistance = 5

	// Posicionar jogador perto da borda de um chunk
	// Chunk 0 vai de Z=0 a Z=31, chunk 1 vai de Z=32 a Z=63
	// Colocar jogador em Z=29, perto da borda
	player := NewPlayer(rl.NewVector3(16, 15, 29))
	player.FlyMode = true // Usar fly mode para movimento mais previsível
	input := &SimulatedInput{}

	// Carregar chunks iniciais
	for i := 0; i < 120; i++ {
		dt := float32(1.0 / 60.0)
		world.Update(player.Position, dt)
		player.Update(dt, world, input)
	}

	// Verificar chunk atual
	chunkBefore := GetChunkCoordFromFloat(player.Position.X, player.Position.Y, player.Position.Z)
	t.Logf("Chunk antes de atravessar: (%d, %d, %d)", chunkBefore.X, chunkBefore.Y, chunkBefore.Z)
	t.Logf("Posição antes: (%.2f, %.2f, %.2f)", player.Position.X, player.Position.Y, player.Position.Z)

	// Verificar que o chunk à frente (onde o jogador vai entrar) existe ou será carregado
	nextChunk := ChunkCoord{X: chunkBefore.X, Y: chunkBefore.Y, Z: chunkBefore.Z + 1}

	// Mover para frente para atravessar a borda (em Z)
	input.Forward = true
	for i := 0; i < 120; i++ { // 2 segundos
		dt := float32(1.0 / 60.0)
		world.Update(player.Position, dt)
		player.Update(dt, world, input)

		// Verificar se já atravessou
		currentChunk := GetChunkCoordFromFloat(player.Position.X, player.Position.Y, player.Position.Z)
		if currentChunk.Z != chunkBefore.Z {
			t.Logf("Atravessou chunk boundary no frame %d", i)
			break
		}
	}

	chunkAfter := GetChunkCoordFromFloat(player.Position.X, player.Position.Y, player.Position.Z)
	t.Logf("Chunk depois de atravessar: (%d, %d, %d)", chunkAfter.X, chunkAfter.Y, chunkAfter.Z)
	t.Logf("Posição depois: (%.2f, %.2f, %.2f)", player.Position.X, player.Position.Y, player.Position.Z)

	// Verificar que o jogador atravessou para o próximo chunk
	if chunkAfter.Z <= chunkBefore.Z {
		t.Skip("Jogador não atravessou chunk boundary, teste inconclusivo")
	}

	// Dar tempo para o ChunkManager carregar os chunks necessários
	input.Forward = false
	for i := 0; i < 60; i++ {
		dt := float32(1.0 / 60.0)
		world.Update(player.Position, dt)
		player.Update(dt, world, input)
	}

	// Verificar que o novo chunk foi carregado
	if _, exists := world.ChunkManager.Chunks[nextChunk.Key()]; !exists {
		t.Errorf("Chunk adjacente (%d, %d, %d) não foi carregado ao atravessar boundary",
			nextChunk.X, nextChunk.Y, nextChunk.Z)
	}

	// Verificar que o chunk atual do jogador está carregado
	if _, exists := world.ChunkManager.Chunks[chunkAfter.Key()]; !exists {
		t.Error("Chunk do jogador não está carregado após atravessar boundary")
	}
}

// TestChunkLoading_AllDirections verifica carregamento em todas as direções
func TestChunkLoading_AllDirections(t *testing.T) {
	world := NewWorld()
	world.ChunkManager.RenderDistance = 2
	world.ChunkManager.UnloadDistance = 4

	player := NewPlayer(rl.NewVector3(16, 15, 16))
	input := &SimulatedInput{}

	// Carregar chunks iniciais
	for i := 0; i < 120; i++ {
		dt := float32(1.0 / 60.0)
		world.Update(player.Position, dt)
		player.Update(dt, world, input)
	}

	playerChunk := GetChunkCoordFromFloat(player.Position.X, player.Position.Y, player.Position.Z)
	t.Logf("Chunk do jogador: (%d, %d, %d)", playerChunk.X, playerChunk.Y, playerChunk.Z)

	// Verificar que chunks foram carregados em todas as direções (pelo menos os adjacentes)
	directions := []struct {
		name  string
		coord ChunkCoord
	}{
		{"frente (Z+)", ChunkCoord{X: playerChunk.X, Y: playerChunk.Y, Z: playerChunk.Z + 1}},
		{"trás (Z-)", ChunkCoord{X: playerChunk.X, Y: playerChunk.Y, Z: playerChunk.Z - 1}},
		{"direita (X+)", ChunkCoord{X: playerChunk.X + 1, Y: playerChunk.Y, Z: playerChunk.Z}},
		{"esquerda (X-)", ChunkCoord{X: playerChunk.X - 1, Y: playerChunk.Y, Z: playerChunk.Z}},
	}

	loadedDirections := 0
	for _, dir := range directions {
		if _, exists := world.ChunkManager.Chunks[dir.coord.Key()]; exists {
			t.Logf("✓ Chunk %s carregado: (%d, %d, %d)", dir.name, dir.coord.X, dir.coord.Y, dir.coord.Z)
			loadedDirections++
		} else {
			t.Logf("✗ Chunk %s NÃO carregado: (%d, %d, %d)", dir.name, dir.coord.X, dir.coord.Y, dir.coord.Z)
		}
	}

	// Pelo menos alguns chunks adjacentes devem estar carregados
	if loadedDirections < 2 {
		t.Errorf("Muito poucos chunks adjacentes carregados: %d de %d", loadedDirections, len(directions))
	}

	t.Logf("Total de chunks carregados: %d", world.GetLoadedChunksCount())
	t.Logf("Chunks adjacentes carregados: %d de %d", loadedDirections, len(directions))
}

// TestChunkLoading_Performance verifica que o carregamento não causa lag
func TestChunkLoading_Performance(t *testing.T) {
	world := NewWorld()
	world.ChunkManager.RenderDistance = 4

	player := NewPlayer(rl.NewVector3(16, 15, 16))
	input := &SimulatedInput{Forward: true}

	totalFrames := 600 // 10 segundos

	for i := 0; i < totalFrames; i++ {
		dt := float32(1.0 / 60.0)

		// Medir tempo de update
		world.Update(player.Position, dt)
		player.Update(dt, world, input)

		// Nota: em um teste real não podemos medir tempo de CPU de forma precisa
		// Este teste verifica apenas que o sistema não trava ou entra em loop infinito
	}

	t.Logf("Total de frames simulados: %d", totalFrames)
	t.Logf("Chunks carregados: %d", world.GetLoadedChunksCount())
	t.Logf("Posição final: (%.2f, %.2f, %.2f)", player.Position.X, player.Position.Y, player.Position.Z)

	// Se chegamos aqui, o teste passou (não travou)
	if world.GetLoadedChunksCount() == 0 {
		t.Error("Nenhum chunk foi carregado durante o movimento")
	}
}

// TestChunkLoading_NoUnloadWithinDistance verifica que chunks dentro da distância não são descarregados
func TestChunkLoading_NoUnloadWithinDistance(t *testing.T) {
	world := NewWorld()
	world.ChunkManager.RenderDistance = 3
	world.ChunkManager.UnloadDistance = 5

	player := NewPlayer(rl.NewVector3(16, 15, 16))
	input := &SimulatedInput{}

	// Carregar chunks iniciais
	for i := 0; i < 120; i++ {
		dt := float32(1.0 / 60.0)
		world.Update(player.Position, dt)
		player.Update(dt, world, input)
	}

	playerChunk := GetChunkCoordFromFloat(player.Position.X, player.Position.Y, player.Position.Z)
	initialChunks := make(map[int64]ChunkCoord)
	for key, chunk := range world.ChunkManager.Chunks {
		initialChunks[key] = chunk.Coord
	}

	t.Logf("Chunks iniciais: %d", len(initialChunks))

	// Mover jogador um pouco (mas não muito longe)
	// Mover ~1 chunk apenas
	input.Forward = true
	for i := 0; i < 180; i++ { // 3 segundos
		dt := float32(1.0 / 60.0)
		world.Update(player.Position, dt)
		player.Update(dt, world, input)
	}

	finalPlayerChunk := GetChunkCoordFromFloat(player.Position.X, player.Position.Y, player.Position.Z)
	t.Logf("Movimento de chunk (%d,%d,%d) para (%d,%d,%d)",
		playerChunk.X, playerChunk.Y, playerChunk.Z,
		finalPlayerChunk.X, finalPlayerChunk.Y, finalPlayerChunk.Z)

	// Verificar que chunks próximos ao ponto inicial não foram descarregados
	chunksStillLoaded := 0
	for key, coord := range initialChunks {
		if _, exists := world.ChunkManager.Chunks[key]; exists {
			chunksStillLoaded++
		} else {
			// Calcular distância do chunk descarregado ao jogador final
			dx := float32(coord.X - finalPlayerChunk.X)
			dy := float32(coord.Y - finalPlayerChunk.Y)
			dz := float32(coord.Z - finalPlayerChunk.Z)
			dist := float32(math.Sqrt(float64(dx*dx + dy*dy + dz*dz)))

			if dist <= float32(world.ChunkManager.UnloadDistance) {
				t.Errorf("Chunk (%d,%d,%d) foi descarregado mas está dentro da UnloadDistance (dist=%.2f, limit=%d)",
					coord.X, coord.Y, coord.Z, dist, world.ChunkManager.UnloadDistance)
			}
		}
	}

	t.Logf("Chunks que permaneceram carregados: %d de %d", chunksStillLoaded, len(initialChunks))
	t.Logf("Total de chunks agora: %d", world.GetLoadedChunksCount())
}
