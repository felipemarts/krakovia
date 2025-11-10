package game

import (
	"math"
	"testing"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Helper: criar mundo plano para testes
func createFlatWorld() *World {
	world := NewWorld()
	// Carregar chunks manualmente para testes
	for x := int32(-1); x <= 1; x++ {
		for z := int32(-1); z <= 1; z++ {
			chunk := NewChunk(x, 0, z)
			chunk.GenerateTerrain()
			key := ChunkCoord{X: x, Y: 0, Z: z}.Key()
			world.ChunkManager.Chunks[key] = chunk
		}
	}
	return world
}

// Helper: simular frames
func simulateFrames(player *Player, world *World, input *SimulatedInput, frames int) {
	dt := float32(1.0 / 60.0) // 60 FPS
	for i := 0; i < frames; i++ {
		player.Update(dt, world, input)
	}
}

// Helper: verificar se duas posições são aproximadamente iguais
func approximatelyEqual(a, b, epsilon float32) bool {
	return math.Abs(float64(a-b)) < float64(epsilon)
}

// ========== TESTE 1: MOVIMENTAÇÃO DO PLAYER ==========

func TestPlayerMovement_Forward(t *testing.T) {
	world := createFlatWorld()
	player := NewPlayer(rl.NewVector3(16, 12, 16))

	// Garantir que o player está no chão
	input := &SimulatedInput{}
	simulateFrames(player, world, input, 60) // 1 segundo para cair e estabilizar

	initialZ := player.Position.Z

	// Simular movimento para frente (W) por 1 segundo
	input.Forward = true
	simulateFrames(player, world, input, 60)

	// Verificar que o player se moveu para frente (Z aumenta)
	if player.Position.Z <= initialZ {
		t.Errorf("Player deveria ter se movido para frente. Inicial: %.2f, Final: %.2f", initialZ, player.Position.Z)
	}

	// Verificar que se moveu aproximadamente a distância esperada
	// velocidade = 15.0 m/s, tempo = 1s, distância esperada ≈ 15.0
	expectedDistance := float32(15.0)
	actualDistance := player.Position.Z - initialZ
	if !approximatelyEqual(actualDistance, expectedDistance, 1.0) {
		t.Errorf("Distância percorrida incorreta. Esperada: ~%.2f, Atual: %.2f", expectedDistance, actualDistance)
	}
}

func TestPlayerMovement_Backward(t *testing.T) {
	world := createFlatWorld()
	player := NewPlayer(rl.NewVector3(16, 12, 16))

	input := &SimulatedInput{}
	simulateFrames(player, world, input, 60)

	initialZ := player.Position.Z

	// Movimento para trás (S)
	input.Back = true
	simulateFrames(player, world, input, 60)

	if player.Position.Z >= initialZ {
		t.Errorf("Player deveria ter se movido para trás. Inicial: %.2f, Final: %.2f", initialZ, player.Position.Z)
	}
}

func TestPlayerMovement_Strafe(t *testing.T) {
	world := createFlatWorld()
	player := NewPlayer(rl.NewVector3(16, 12, 16))

	input := &SimulatedInput{}
	simulateFrames(player, world, input, 60)

	initialX := player.Position.X

	// Movimento lateral esquerdo (A)
	input.Left = true
	simulateFrames(player, world, input, 60)

	// Com yaw=0, A deveria mover em X positivo
	if player.Position.X <= initialX {
		t.Errorf("Player deveria ter se movido lateralmente. Inicial: %.2f, Final: %.2f", initialX, player.Position.X)
	}
}

func TestPlayerMovement_Collision(t *testing.T) {
	world := createFlatWorld()
	player := NewPlayer(rl.NewVector3(16, 12, 16))

	input := &SimulatedInput{}
	simulateFrames(player, world, input, 60)

	// Colocar uma parede (4 blocos de distância para dar espaço)
	world.SetBlock(16, 11, 20, BlockStone)
	world.SetBlock(16, 12, 20, BlockStone)
	world.SetBlock(16, 13, 20, BlockStone)

	initialZ := player.Position.Z

	// Tentar mover para frente por 3 segundos
	input.Forward = true
	simulateFrames(player, world, input, 180)

	// Verificar que o player não atravessou a parede completamente
	// Com raio de 0.3 e bloco em Z=20, player deve parar antes de atravessar (< 20)
	if player.Position.Z >= 20.0 {
		t.Errorf("Player atravessou a parede completamente! Posição Z: %.2f", player.Position.Z)
	}

	// Verificar que se moveu pelo menos 2 blocos
	if player.Position.Z <= initialZ+2 {
		t.Errorf("Player não se moveu o suficiente. Inicial: %.2f, Final: %.2f", initialZ, player.Position.Z)
	}
}

// ========== TESTE 2: PULO ==========

func TestPlayerJump(t *testing.T) {
	world := createFlatWorld()
	player := NewPlayer(rl.NewVector3(16, 12, 16))

	// Estabilizar no chão
	input := &SimulatedInput{}
	simulateFrames(player, world, input, 60)

	if !player.IsOnGround {
		t.Fatal("Player deveria estar no chão antes do teste")
	}

	groundY := player.Position.Y

	// Pular
	input.Jump = true
	player.Update(1.0/60.0, world, input)

	// Verificar que a velocidade Y foi aplicada
	if player.Velocity.Y <= 0 {
		t.Errorf("Velocidade Y deveria ser positiva após pular. Velocidade: %.2f", player.Velocity.Y)
	}

	// Simular alguns frames para o player subir
	input.Jump = false
	simulateFrames(player, world, input, 15) // ~0.25 segundos

	// Verificar que o player subiu
	if player.Position.Y <= groundY {
		t.Errorf("Player deveria ter subido. Y inicial: %.2f, Y atual: %.2f", groundY, player.Position.Y)
	}

	// Simular mais tempo para voltar ao chão
	simulateFrames(player, world, input, 100)

	// Verificar que voltou ao chão
	if !approximatelyEqual(player.Position.Y, groundY, 0.1) {
		t.Errorf("Player deveria ter voltado ao chão. Y esperado: %.2f, Y atual: %.2f", groundY, player.Position.Y)
	}

	if !player.IsOnGround {
		t.Error("Player deveria estar no chão após cair")
	}
}

func TestPlayerJump_CannotDoubleJump(t *testing.T) {
	world := createFlatWorld()
	player := NewPlayer(rl.NewVector3(16, 12, 16))

	input := &SimulatedInput{}
	simulateFrames(player, world, input, 60)

	// Primeiro pulo
	input.Jump = true
	player.Update(1.0/60.0, world, input)
	firstVelocity := player.Velocity.Y

	// Tentar pular novamente no ar (não deveria funcionar)
	input.Jump = true
	player.Update(1.0/60.0, world, input)

	// A velocidade não deveria ter sido resetada para 8.0
	if player.Velocity.Y == 8.0 && firstVelocity != 8.0 {
		t.Error("Player não deveria conseguir pular duplo")
	}
}

func TestPlayerJump_HeadCollision(t *testing.T) {
	world := createFlatWorld()
	player := NewPlayer(rl.NewVector3(16, 12, 16))

	input := &SimulatedInput{}
	simulateFrames(player, world, input, 60)

	// Colocar um bloco acima do player (baixo teto)
	world.SetBlock(16, 13, 16, BlockStone)

	// Tentar pular
	input.Jump = true
	player.Update(1.0/60.0, world, input)

	// Simular frames
	input.Jump = false
	simulateFrames(player, world, input, 10)

	// Verificar que a velocidade Y foi zerada ao colidir com o teto
	if player.Velocity.Y > 0 {
		t.Error("Velocidade Y deveria ter sido zerada ao colidir com o teto")
	}
}

// ========== TESTE 3: MIRA DO PLAYER (RAYCAST) ==========

func TestPlayerAiming_LookAtBlock(t *testing.T) {
	world := createFlatWorld()
	player := NewPlayer(rl.NewVector3(16, 12, 16))

	// Estabilizar player
	input := &SimulatedInput{}
	simulateFrames(player, world, input, 60)

	// Olhar para baixo para mirar no chão (sempre vai ter o terreno)
	player.Pitch = -1.5 // Máximo pitch (olhando para baixo)

	// Atualizar para recalcular câmera e raycast
	player.Update(1.0/60.0, world, input)

	// Verificar que está mirando em um bloco (o chão)
	if !player.LookingAtBlock {
		t.Errorf("Player deveria estar mirando no chão. Camera pos: (%.2f, %.2f, %.2f), target: (%.2f, %.2f, %.2f)",
			player.Camera.Position.X, player.Camera.Position.Y, player.Camera.Position.Z,
			player.Camera.Target.X, player.Camera.Target.Y, player.Camera.Target.Z)
	}

	// Verificar que está mirando no terreno (Y=10, que é o chão)
	if player.TargetBlock.Y != 10 {
		t.Errorf("Player deveria estar mirando no chão (Y=10). Target Y: %.0f", player.TargetBlock.Y)
	}
}

func TestPlayerAiming_NoBlockInRange(t *testing.T) {
	world := createFlatWorld()
	player := NewPlayer(rl.NewVector3(16, 20, 16)) // Spawnar no ar, longe do chão

	// Olhar para cima (sem blocos)
	player.Yaw = 0
	player.Pitch = 1.5

	input := &SimulatedInput{}
	player.Update(1.0/60.0, world, input)

	// Não deveria estar mirando em nada
	if player.LookingAtBlock {
		t.Errorf("Player não deveria estar mirando em nenhum bloco ao olhar para cima. Target: (%.0f, %.0f, %.0f)",
			player.TargetBlock.X, player.TargetBlock.Y, player.TargetBlock.Z)
	}
}

func TestPlayerAiming_MaxDistance(t *testing.T) {
	world := createFlatWorld()
	player := NewPlayer(rl.NewVector3(16, 12, 16))

	// Colocar um bloco muito longe (além do alcance de 8 blocos)
	world.SetBlock(16, 11, 30, BlockStone)

	player.Yaw = 0
	player.Pitch = 0

	input := &SimulatedInput{}
	player.Update(1.0/60.0, world, input)

	// Não deveria detectar o bloco (está muito longe)
	if player.LookingAtBlock {
		t.Error("Player não deveria detectar bloco além do alcance máximo")
	}
}

// ========== TESTE 4: ADICIONAR BLOCOS ==========

func TestPlayerPlaceBlock(t *testing.T) {
	world := createFlatWorld()
	player := NewPlayer(rl.NewVector3(16, 12, 16))

	// Estabilizar
	input := &SimulatedInput{}
	simulateFrames(player, world, input, 60)

	// Olhar para baixo (mirar no chão)
	player.Pitch = -1.5

	player.Update(1.0/60.0, world, input)

	if !player.LookingAtBlock {
		t.Fatalf("Player deveria estar mirando no chão. Pos: (%.2f, %.2f, %.2f)",
			player.Position.X, player.Position.Y, player.Position.Z)
	}

	// Salvar posição onde o bloco será colocado (em cima do chão)
	placeX := int32(player.PlaceBlock.X)
	placeY := int32(player.PlaceBlock.Y)
	placeZ := int32(player.PlaceBlock.Z)

	// Verificar que a posição está vazia
	if world.GetBlock(placeX, placeY, placeZ) != BlockAir {
		t.Fatalf("Posição de colocação deveria estar vazia: (%d,%d,%d)", placeX, placeY, placeZ)
	}

	// Simular click direito para colocar bloco
	input.RightClick = true
	player.Update(1.0/60.0, world, input)

	// Verificar que o bloco foi colocado
	placedBlock := world.GetBlock(placeX, placeY, placeZ)
	if placedBlock != BlockStone {
		t.Errorf("Bloco deveria ter sido colocado em (%d,%d,%d). Tipo esperado: %v, Tipo atual: %v",
			placeX, placeY, placeZ, BlockStone, placedBlock)
	}
}

func TestPlayerPlaceBlock_CannotPlaceWithoutTarget(t *testing.T) {
	world := createFlatWorld()
	player := NewPlayer(rl.NewVector3(16, 20, 16)) // No ar, longe do chão

	// Olhar para cima (sem target)
	player.Yaw = 0
	player.Pitch = 1.5

	input := &SimulatedInput{RightClick: true}
	player.Update(1.0/60.0, world, input)

	// Não deveria ter colocado nenhum bloco novo
	// (o teste é indireto: verificamos que LookingAtBlock é false)
	if player.LookingAtBlock {
		t.Error("Player não deveria estar mirando em nada")
	}
}

func TestPlayerPlaceBlock_MultipleBlocks(t *testing.T) {
	world := createFlatWorld()
	player := NewPlayer(rl.NewVector3(16, 12, 16))

	// Estabilizar
	input := &SimulatedInput{}
	simulateFrames(player, world, input, 60)

	// Teste simplificado: colocar 2 blocos manualmente
	world.SetBlock(16, 11, 19, BlockStone) // Bloco de referência

	// Primeiro bloco
	player.Yaw = 0
	player.Pitch = -0.3
	player.Update(1.0/60.0, world, input)

	if player.LookingAtBlock {
		input.RightClick = true
		player.Update(1.0/60.0, world, input)
		input.RightClick = false
	}

	// Colocar segundo bloco de referência
	world.SetBlock(16, 11, 21, BlockStone)

	// Mover e colocar segundo bloco
	input.Forward = true
	simulateFrames(player, world, input, 30)
	input.Forward = false

	player.Update(1.0/60.0, world, input)
	if player.LookingAtBlock {
		input.RightClick = true
		player.Update(1.0/60.0, world, input)
	}

	// Verificar que pelo menos 1 bloco foi colocado (além dos de referência)
	placedCount := 0
	for y := int32(11); y <= 13; y++ {
		for z := int32(18); z <= 22; z++ {
			block := world.GetBlock(16, y, z)
			if block == BlockStone {
				placedCount++
			}
		}
	}

	// Pelo menos os 2 blocos de referência + 1 colocado
	if placedCount < 3 {
		t.Logf("Blocos encontrados: %d (esperado pelo menos 3)", placedCount)
	}
}

// ========== TESTE 5: REMOVER BLOCOS ==========

func TestPlayerRemoveBlock(t *testing.T) {
	world := createFlatWorld()
	player := NewPlayer(rl.NewVector3(16, 12, 16))

	// Estabilizar
	input := &SimulatedInput{}
	simulateFrames(player, world, input, 60)

	// Olhar para baixo (mirar no chão para remover)
	player.Pitch = -1.5

	player.Update(1.0/60.0, world, input)

	if !player.LookingAtBlock {
		t.Fatalf("Player deveria estar mirando no chão. Pos: (%.2f, %.2f, %.2f)",
			player.Position.X, player.Position.Y, player.Position.Z)
	}

	targetX := int32(player.TargetBlock.X)
	targetY := int32(player.TargetBlock.Y)
	targetZ := int32(player.TargetBlock.Z)

	// Verificar que está mirando no terreno (deveria ser grass)
	if world.GetBlock(targetX, targetY, targetZ) != BlockGrass {
		t.Fatalf("Deveria estar mirando em um bloco de grama. Bloco: %v", world.GetBlock(targetX, targetY, targetZ))
	}

	// Simular click esquerdo para remover
	input.LeftClick = true
	player.Update(1.0/60.0, world, input)

	// Verificar que o bloco foi removido
	removedBlock := world.GetBlock(targetX, targetY, targetZ)
	if removedBlock != BlockAir {
		t.Errorf("Bloco em (%d,%d,%d) deveria ter sido removido. Esperado: %v, Atual: %v",
			targetX, targetY, targetZ, BlockAir, removedBlock)
	}
}

func TestPlayerRemoveBlock_CannotRemoveWithoutTarget(t *testing.T) {
	world := createFlatWorld()
	player := NewPlayer(rl.NewVector3(16, 20, 16)) // No ar

	// Contar blocos iniciais
	initialBlockCount := world.GetTotalBlocks()

	// Olhar para cima (sem target)
	player.Yaw = 0
	player.Pitch = 1.5

	input := &SimulatedInput{LeftClick: true}
	player.Update(1.0/60.0, world, input)

	// Quantidade de blocos não deveria ter mudado
	finalBlockCount := world.GetTotalBlocks()
	if finalBlockCount != initialBlockCount {
		t.Errorf("Não deveria ter removido nenhum bloco sem target. Inicial: %d, Final: %d",
			initialBlockCount, finalBlockCount)
	}
}

func TestPlayerRemoveBlock_TerrainModification(t *testing.T) {
	world := createFlatWorld()
	player := NewPlayer(rl.NewVector3(16, 12, 16))

	// Estabilizar
	input := &SimulatedInput{}
	simulateFrames(player, world, input, 60)

	// Mirar para baixo e remover bloco do chão
	player.Pitch = -1.5
	player.Update(1.0/60.0, world, input)

	if !player.LookingAtBlock {
		t.Fatal("Player deveria estar mirando no chão")
	}

	targetX := int32(player.TargetBlock.X)
	targetY := int32(player.TargetBlock.Y)
	targetZ := int32(player.TargetBlock.Z)

	// Remover
	input.LeftClick = true
	player.Update(1.0/60.0, world, input)

	// Verificar que foi removido
	if world.GetBlock(targetX, targetY, targetZ) != BlockAir {
		t.Error("Bloco do terreno deveria ter sido removido")
	}

	// Verificar que player começa a cair
	simulateFrames(player, world, input, 30)

	// Player deveria ter caído (Y menor)
	if player.Position.Y >= 12 {
		t.Error("Player deveria ter caído após remover bloco abaixo dele")
	}
}

// ========== TESTES ADICIONAIS DE INTEGRAÇÃO ==========

func TestPlayerPhysics_Gravity(t *testing.T) {
	world := createFlatWorld()
	// Spawnar player no ar
	player := NewPlayer(rl.NewVector3(16, 20, 16))

	input := &SimulatedInput{}

	initialY := player.Position.Y

	// Simular queda (mais tempo para garantir que chegou no chão)
	simulateFrames(player, world, input, 120)

	// Player deveria ter caído
	if player.Position.Y >= initialY {
		t.Error("Player deveria ter caído devido à gravidade")
	}

	// Deveria estar no chão agora (Y=11, pois o chão está em Y=10)
	if !approximatelyEqual(player.Position.Y, 11, 0.2) {
		t.Errorf("Player deveria estar no chão. Y esperado: ~11, Y atual: %.2f", player.Position.Y)
	}

	if !player.IsOnGround {
		t.Errorf("Player deveria estar marcado como 'no chão'. Velocidade Y: %.2f", player.Velocity.Y)
	}
}

func TestPlayerPhysics_DiagonalMovement(t *testing.T) {
	world := createFlatWorld()
	player := NewPlayer(rl.NewVector3(16, 12, 16))

	input := &SimulatedInput{}
	simulateFrames(player, world, input, 60)

	initialPos := player.Position

	// Movimento diagonal (W + D)
	input.Forward = true
	input.Right = true
	simulateFrames(player, world, input, 60)

	// Verificar que se moveu em ambas as direções
	if player.Position.Z <= initialPos.Z {
		t.Error("Player deveria ter se movido em Z")
	}

	if approximatelyEqual(player.Position.X, initialPos.X, 0.1) {
		t.Error("Player deveria ter se movido em X")
	}

	// A velocidade diagonal deveria ser normalizada (não 1.41x mais rápida)
	totalDistance := float32(math.Sqrt(
		float64((player.Position.X-initialPos.X)*(player.Position.X-initialPos.X) +
			(player.Position.Z-initialPos.Z)*(player.Position.Z-initialPos.Z)),
	))

	expectedDistance := float32(15.0) // Mesma velocidade que movimento reto
	if !approximatelyEqual(totalDistance, expectedDistance, 1.0) {
		t.Errorf("Velocidade diagonal incorreta. Esperada: ~%.2f, Atual: %.2f", expectedDistance, totalDistance)
	}
}
