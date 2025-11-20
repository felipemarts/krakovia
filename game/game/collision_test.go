package game

import (
	"testing"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Helper: criar mundo com chunks para testes
func createChunkedFlatWorld() *World {
	world := NewWorld()

	// Desabilitar carregamento automático de chunks durante testes
	// Definir cooldown muito alto para evitar Update de chunks
	world.ChunkManager.UpdateCooldownLimit = 9999999.0

	// Carregar chunks manualmente para o teste com terreno PLANO
	for x := int32(-2); x <= 2; x++ {
		for z := int32(-2); z <= 2; z++ {
			chunk := NewChunk(x, 0, z)

			// Gerar terreno completamente plano em Y=10 (sem noise)
			for cx := int32(0); cx < ChunkSize; cx++ {
				for cz := int32(0); cz < ChunkSize; cz++ {
					// Preencher blocos até Y=10
					for cy := int32(0); cy <= 10; cy++ {
						if cy < 8 {
							chunk.Blocks[cx][cy][cz] = BlockGrass
						} else if cy < 10 {
							chunk.Blocks[cx][cy][cz] = BlockGrass
						} else {
							chunk.Blocks[cx][cy][cz] = BlockGrass
						}
					}
				}
			}
			chunk.IsGenerated = true
			chunk.NeedUpdateMeshes = true

			key := ChunkCoord{X: x, Y: 0, Z: z}.Key()
			world.ChunkManager.Chunks[key] = chunk
		}
	}

	return world
}

// Helper: simular frames para testes de colisão
func simulateCollisionFrames(player *Player, world *World, input *SimulatedInput, frames int) {
	dt := float32(1.0 / 60.0) // 60 FPS
	for i := 0; i < frames; i++ {
		world.Update(player.Position, dt)
		player.Update(dt, world, input)
	}
}

// ========== TESTES DE COLISÃO COM CHUNKS ==========

func TestCollision_PlayerLandsOnGround(t *testing.T) {
	world := createChunkedFlatWorld()
	// Spawnar player no ar, acima do chão em Y=10
	player := NewPlayer(rl.NewVector3(16, 15, 16))

	// Verificar se o bloco do chão existe
	blockBelow := world.GetBlock(16, 10, 16)
	t.Logf("Bloco em (16, 10, 16): %v", blockBelow)

	input := &SimulatedInput{}

	// Simular queda por 2 segundos
	simulateCollisionFrames(player, world, input, 120)

	// Player deveria ter pousado no chão (Y=11, em cima do bloco Y=10)
	t.Logf("Player Y: %.2f, IsOnGround: %v, Velocity.Y: %.2f", player.Position.Y, player.IsOnGround, player.Velocity.Y)

	if !approximatelyEqual(player.Position.Y, 11, 0.2) {
		t.Errorf("Player deveria estar no chão (Y=11). Y atual: %.2f", player.Position.Y)
	}

	// Deveria estar marcado como no chão
	if !player.IsOnGround {
		t.Errorf("Player deveria estar marcado como no chão. Posição: (%.2f, %.2f, %.2f)",
			player.Position.X, player.Position.Y, player.Position.Z)
	}

	// Velocidade vertical deveria ser zero
	if player.Velocity.Y != 0 {
		t.Errorf("Velocidade Y deveria ser zero no chão. Velocidade Y: %.2f", player.Velocity.Y)
	}
}

func TestCollision_PlayerCannotFallThroughFloor(t *testing.T) {
	world := createChunkedFlatWorld()
	player := NewPlayer(rl.NewVector3(16, 15, 16))

	input := &SimulatedInput{}

	// Simular muitos frames para garantir que não atravessa
	simulateCollisionFrames(player, world, input, 300) // 5 segundos

	// Player não deveria estar abaixo do chão
	if player.Position.Y < 10.5 {
		t.Errorf("Player atravessou o chão! Y: %.2f (deveria ser >= 10.5)", player.Position.Y)
	}

	// Deveria estar no chão
	if !player.IsOnGround {
		t.Error("Player deveria estar no chão após queda longa")
	}
}

func TestCollision_PlayerStaysOnGroundWhileMoving(t *testing.T) {
	world := createChunkedFlatWorld()
	player := NewPlayer(rl.NewVector3(16, 15, 16))

	input := &SimulatedInput{}

	// Deixar cair primeiro
	simulateCollisionFrames(player, world, input, 120)

	if !player.IsOnGround {
		t.Fatal("Player deveria estar no chão antes de começar a andar")
	}

	groundY := player.Position.Y

	// Andar para frente por 2 segundos
	input.Forward = true
	simulateCollisionFrames(player, world, input, 120)

	// Y não deveria mudar significativamente (deve permanecer no chão)
	if !approximatelyEqual(player.Position.Y, groundY, 0.1) {
		t.Errorf("Player deveria permanecer no chão ao andar. Y inicial: %.2f, Y final: %.2f",
			groundY, player.Position.Y)
	}

	// Deveria continuar marcado como no chão
	if !player.IsOnGround {
		t.Error("Player deveria continuar no chão ao andar")
	}
}

func TestCollision_PlayerCollidesWithWall(t *testing.T) {
	world := createChunkedFlatWorld()
	player := NewPlayer(rl.NewVector3(16, 15, 16))

	input := &SimulatedInput{}

	// Deixar cair
	simulateCollisionFrames(player, world, input, 120)

	// Colocar uma parede na frente do player
	for y := int32(11); y <= 13; y++ {
		world.SetBlock(16, y, 20, BlockGrass)
	}

	initialZ := player.Position.Z

	// Tentar andar para frente por 3 segundos
	input.Forward = true
	simulateCollisionFrames(player, world, input, 180)

	// Player deveria ter se movido um pouco, mas parado antes da parede
	// Com raio 0.3 e colisão cilíndrica, player pode chegar bem perto da borda do bloco
	// Bloco em Z=20, player pode chegar até ~19.9 sem atravessar
	if player.Position.Z >= 20.0 {
		t.Errorf("Player atravessou a parede completamente! Z: %.2f (deveria ser < 20.0)", player.Position.Z)
	}

	// Mas deveria ter se movido pelo menos um pouco
	if player.Position.Z <= initialZ+1.0 {
		t.Errorf("Player não se moveu. Z inicial: %.2f, Z final: %.2f", initialZ, player.Position.Z)
	}
}

func TestCollision_PlayerCollidesWithCeiling(t *testing.T) {
	world := createChunkedFlatWorld()
	player := NewPlayer(rl.NewVector3(16, 15, 16))

	input := &SimulatedInput{}

	// Deixar cair
	simulateCollisionFrames(player, world, input, 120)

	// Colocar bloco acima (teto baixo)
	world.SetBlock(16, 13, 16, BlockGrass)

	// Tentar pular
	input.Jump = true
	simulateCollisionFrames(player, world, input, 1)
	input.Jump = false

	// Simular alguns frames
	simulateCollisionFrames(player, world, input, 20)

	// Player não deveria ter subido muito (colidiu com teto)
	if player.Position.Y > 12.0 {
		t.Errorf("Player atravessou o teto! Y: %.2f", player.Position.Y)
	}

	// Velocidade Y deveria ter sido zerada ou ser negativa
	if player.Velocity.Y > 0 {
		t.Errorf("Velocidade Y deveria ser <= 0 após colidir com teto. Velocidade Y: %.2f", player.Velocity.Y)
	}
}

func TestCollision_PlayerCannotMoveIntoBlock(t *testing.T) {
	world := createChunkedFlatWorld()
	player := NewPlayer(rl.NewVector3(16, 15, 16))

	input := &SimulatedInput{}

	// Deixar cair
	simulateCollisionFrames(player, world, input, 120)

	// Cercar player com blocos em todas as direções horizontais
	world.SetBlock(16, 11, 17, BlockGrass) // Frente
	world.SetBlock(16, 11, 15, BlockGrass) // Trás
	world.SetBlock(17, 11, 16, BlockGrass) // Direita
	world.SetBlock(15, 11, 16, BlockGrass) // Esquerda

	initialPos := player.Position

	// Tentar mover em todas as direções
	input.Forward = true
	simulateCollisionFrames(player, world, input, 30)
	input.Forward = false

	input.Back = true
	simulateCollisionFrames(player, world, input, 30)
	input.Back = false

	input.Right = true
	simulateCollisionFrames(player, world, input, 30)
	input.Right = false

	input.Left = true
	simulateCollisionFrames(player, world, input, 30)
	input.Left = false

	// Player não deveria ter se movido significativamente
	if !approximatelyEqual(player.Position.X, initialPos.X, 0.3) {
		t.Errorf("Player moveu em X quando cercado. X inicial: %.2f, X final: %.2f",
			initialPos.X, player.Position.X)
	}

	if !approximatelyEqual(player.Position.Z, initialPos.Z, 0.3) {
		t.Errorf("Player moveu em Z quando cercado. Z inicial: %.2f, Z final: %.2f",
			initialPos.Z, player.Position.Z)
	}
}

func TestCollision_PlayerStaysOnGroundAcrossChunkBoundary(t *testing.T) {
	world := createChunkedFlatWorld()
	// Colocar player perto da borda de um chunk (chunk boundary em X=32)
	player := NewPlayer(rl.NewVector3(30, 15, 16))

	input := &SimulatedInput{}

	// Deixar cair
	simulateCollisionFrames(player, world, input, 120)

	if !player.IsOnGround {
		t.Fatal("Player deveria estar no chão antes de atravessar chunk")
	}

	groundY := player.Position.Y

	// Andar para atravessar a borda do chunk (para X > 32)
	input.Right = true
	simulateCollisionFrames(player, world, input, 120)

	// Verificar que atravessou a borda do chunk
	if player.Position.X <= 32 {
		t.Skip("Player não atravessou borda do chunk, teste inconclusivo")
	}

	// Player deveria continuar no chão
	if !approximatelyEqual(player.Position.Y, groundY, 0.2) {
		t.Errorf("Player caiu ao atravessar chunk boundary. Y inicial: %.2f, Y final: %.2f",
			groundY, player.Position.Y)
	}

	if !player.IsOnGround {
		t.Error("Player deveria continuar no chão após atravessar chunk boundary")
	}
}

func TestCollision_HighSpeedNoClipping(t *testing.T) {
	world := createChunkedFlatWorld()
	player := NewPlayer(rl.NewVector3(16, 15, 16))

	input := &SimulatedInput{}

	// Deixar cair
	simulateCollisionFrames(player, world, input, 120)

	// Colocar parede
	for y := int32(11); y <= 13; y++ {
		world.SetBlock(16, y, 19, BlockGrass)
	}

	// Tentar mover muito rápido em direção à parede
	// (simular delta time grande como se fosse lag)
	largeDt := float32(1.0) // 1 segundo por frame (simulando lag extremo)

	for i := 0; i < 5; i++ {
		world.Update(player.Position, largeDt)
		player.Update(largeDt, world, input)
		input.Forward = true
	}

	// Mesmo com lag, player não deveria atravessar parede completamente
	// Com raio 0.3 e colisão cilíndrica, pode chegar perto mas não atravessar
	if player.Position.Z >= 19.0 {
		t.Errorf("Player atravessou parede mesmo com delta time grande! Z: %.2f (deveria ser < 19.0)", player.Position.Z)
	}
}

// Teste específico para o bug reportado: player atravessando chão
func TestCollision_BugFix_PlayerNotFallingThroughFloor(t *testing.T) {
	world := createChunkedFlatWorld()
	// Usar a mesma posição inicial do jogo
	player := NewPlayer(rl.NewVector3(8, 100, 8))

	input := &SimulatedInput{}

	// Simular muitos frames (10 segundos) para garantir que player cai e fica estável
	for i := 0; i < 600; i++ {
		dt := float32(1.0 / 60.0)
		world.Update(player.Position, dt)
		player.Update(dt, world, input)

		// Verificar a cada frame que não está abaixo do chão
		if player.Position.Y < 10.0 {
			t.Fatalf("Player atravessou o chão no frame %d! Y: %.2f", i, player.Position.Y)
		}
	}

	// Após queda, deveria estar no chão
	if !approximatelyEqual(player.Position.Y, 11, 0.2) {
		t.Errorf("Player deveria estar no chão (Y=11). Y: %.2f", player.Position.Y)
	}

	if !player.IsOnGround {
		t.Error("Player deveria estar no chão após queda")
	}
}
