package main

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	PlayerHeight      = 1.8
	PlayerWidth       = 0.6
	PlayerSpeed       = 5.0
	PlayerJumpForce   = 8.0
	Gravity           = 20.0
	MouseSens         = 0.003
	ReachDistance     = 5.0
	ThirdPersonDist   = 5.0   // Distância da câmera
	ThirdPersonHeight = 2.0   // Altura da câmera
)

// MinecraftPlayer representa o jogador
type MinecraftPlayer struct {
	Position     rl.Vector3
	Velocity     rl.Vector3
	Camera       rl.Camera3D
	Yaw          float32
	Pitch        float32
	IsOnGround   bool
	SelectedBlock BlockType
}

// NewMinecraftPlayer cria um novo player
func NewMinecraftPlayer(startPos rl.Vector3) *MinecraftPlayer {
	player := &MinecraftPlayer{
		Position:      startPos,
		Velocity:      rl.NewVector3(0, 0, 0),
		Yaw:           0,
		Pitch:         0,
		SelectedBlock: BlockDirt,
	}

	player.Camera = rl.Camera3D{
		Position:   player.GetEyePosition(),
		Target:     rl.Vector3Add(player.GetEyePosition(), player.GetForward()),
		Up:         rl.NewVector3(0, 1, 0),
		Fovy:       70.0,
		Projection: rl.CameraPerspective,
	}

	return player
}

// GetEyePosition retorna a posição dos olhos do player
func (p *MinecraftPlayer) GetEyePosition() rl.Vector3 {
	return rl.NewVector3(p.Position.X, p.Position.Y+PlayerHeight*0.9, p.Position.Z)
}

// GetForward retorna o vetor de direção frontal
func (p *MinecraftPlayer) GetForward() rl.Vector3 {
	return rl.NewVector3(
		float32(math.Cos(float64(p.Pitch))*math.Sin(float64(p.Yaw))),
		float32(math.Sin(float64(p.Pitch))),
		float32(math.Cos(float64(p.Pitch))*math.Cos(float64(p.Yaw))),
	)
}

// GetRight retorna o vetor de direção lateral direita
func (p *MinecraftPlayer) GetRight() rl.Vector3 {
	forward := p.GetForward()
	up := rl.NewVector3(0, 1, 0)
	return rl.Vector3Normalize(rl.Vector3CrossProduct(forward, up))
}

// Update atualiza o player (física, input, etc)
func (p *MinecraftPlayer) Update(world *MinecraftWorld, dt float32) {
	// Controle de mouse
	p.updateMouseLook(dt)

	// Movimento
	p.updateMovement(dt)

	// Física (gravidade e colisão)
	p.updatePhysics(world, dt)

	// Atualizar câmera para terceira pessoa
	forward := p.GetForward()

	// Posição da câmera atrás e acima do player
	cameraOffset := rl.Vector3Scale(forward, -ThirdPersonDist)
	cameraOffset.Y += ThirdPersonHeight

	p.Camera.Position = rl.Vector3Add(p.Position, cameraOffset)
	p.Camera.Position.Y += PlayerHeight / 2

	// Câmera olha para o player
	p.Camera.Target = rl.NewVector3(p.Position.X, p.Position.Y+PlayerHeight/2, p.Position.Z)

	// Seleção de blocos
	p.updateBlockSelection()
}

// updateMouseLook atualiza rotação com mouse
func (p *MinecraftPlayer) updateMouseLook(dt float32) {
	mouseDelta := rl.GetMouseDelta()

	p.Yaw -= mouseDelta.X * MouseSens // Invertido
	p.Pitch -= mouseDelta.Y * MouseSens

	// Limitar pitch
	if p.Pitch > 1.5 {
		p.Pitch = 1.5
	}
	if p.Pitch < -1.5 {
		p.Pitch = -1.5
	}
}

// updateMovement processa input de movimento
func (p *MinecraftPlayer) updateMovement(dt float32) {
	forward := p.GetForward()
	right := p.GetRight()

	// Zerar componente Y para movimento horizontal
	forward.Y = 0
	forward = rl.Vector3Normalize(forward)
	right.Y = 0
	right = rl.Vector3Normalize(right)

	moveDir := rl.NewVector3(0, 0, 0)

	// WASD
	if rl.IsKeyDown(rl.KeyW) {
		moveDir = rl.Vector3Add(moveDir, forward)
	}
	if rl.IsKeyDown(rl.KeyS) {
		moveDir = rl.Vector3Subtract(moveDir, forward)
	}
	if rl.IsKeyDown(rl.KeyD) {
		moveDir = rl.Vector3Add(moveDir, right)
	}
	if rl.IsKeyDown(rl.KeyA) {
		moveDir = rl.Vector3Subtract(moveDir, right)
	}

	// Normalizar e aplicar velocidade
	if rl.Vector3Length(moveDir) > 0 {
		moveDir = rl.Vector3Normalize(moveDir)
		p.Velocity.X = moveDir.X * PlayerSpeed
		p.Velocity.Z = moveDir.Z * PlayerSpeed
	} else {
		p.Velocity.X = 0
		p.Velocity.Z = 0
	}

	// Pulo
	if rl.IsKeyPressed(rl.KeySpace) && p.IsOnGround {
		p.Velocity.Y = PlayerJumpForce
		p.IsOnGround = false
	}
}

// updatePhysics aplica física e colisão
func (p *MinecraftPlayer) updatePhysics(world *MinecraftWorld, dt float32) {
	// Gravidade
	p.Velocity.Y -= Gravity * dt

	// Tentar mover no eixo X
	newPosX := p.Position
	newPosX.X += p.Velocity.X * dt
	if !p.checkCollision(world, newPosX) {
		p.Position.X = newPosX.X
	}

	// Tentar mover no eixo Z
	newPosZ := p.Position
	newPosZ.Z += p.Velocity.Z * dt
	if !p.checkCollision(world, newPosZ) {
		p.Position.Z = newPosZ.Z
	}

	// Tentar mover no eixo Y
	newPosY := p.Position
	newPosY.Y += p.Velocity.Y * dt

	if p.checkCollision(world, newPosY) {
		p.Velocity.Y = 0
		if p.Velocity.Y < 0 {
			p.IsOnGround = true
		}
	} else {
		p.Position.Y = newPosY.Y
		p.IsOnGround = false
	}

	// Verificar se está no chão
	groundCheck := p.Position
	groundCheck.Y -= 0.1
	if p.checkCollision(world, groundCheck) {
		p.IsOnGround = true
	}
}

// checkCollision verifica colisão da caixa do player
func (p *MinecraftPlayer) checkCollision(world *MinecraftWorld, pos rl.Vector3) bool {
	// Verificar vários pontos ao redor da caixa do player
	hw := float32(PlayerWidth / 2.0) // half width

	// Pontos para verificar (base, meio, topo)
	checkPoints := []rl.Vector3{
		// Base
		{pos.X - hw, pos.Y, pos.Z - hw},
		{pos.X + hw, pos.Y, pos.Z - hw},
		{pos.X - hw, pos.Y, pos.Z + hw},
		{pos.X + hw, pos.Y, pos.Z + hw},
		// Meio
		{pos.X - hw, pos.Y + float32(PlayerHeight/2), pos.Z - hw},
		{pos.X + hw, pos.Y + float32(PlayerHeight/2), pos.Z - hw},
		{pos.X - hw, pos.Y + float32(PlayerHeight/2), pos.Z + hw},
		{pos.X + hw, pos.Y + float32(PlayerHeight/2), pos.Z + hw},
		// Topo
		{pos.X - hw, pos.Y + float32(PlayerHeight), pos.Z - hw},
		{pos.X + hw, pos.Y + float32(PlayerHeight), pos.Z - hw},
		{pos.X - hw, pos.Y + float32(PlayerHeight), pos.Z + hw},
		{pos.X + hw, pos.Y + float32(PlayerHeight), pos.Z + hw},
	}

	for _, point := range checkPoints {
		block := world.GetBlock(point.X, point.Y, point.Z)
		if block != BlockAir {
			return true
		}
	}

	return false
}

// updateBlockSelection atualiza seleção de blocos com scroll do mouse
func (p *MinecraftPlayer) updateBlockSelection() {
	wheelMove := rl.GetMouseWheelMove()
	if wheelMove > 0 {
		p.SelectedBlock++
		if p.SelectedBlock > BlockLeaves {
			p.SelectedBlock = BlockGrass
		}
	} else if wheelMove < 0 {
		p.SelectedBlock--
		if p.SelectedBlock < BlockGrass {
			p.SelectedBlock = BlockLeaves
		}
	}

	// Atalhos numéricos
	if rl.IsKeyPressed(rl.KeyOne) {
		p.SelectedBlock = BlockGrass
	} else if rl.IsKeyPressed(rl.KeyTwo) {
		p.SelectedBlock = BlockDirt
	} else if rl.IsKeyPressed(rl.KeyThree) {
		p.SelectedBlock = BlockStone
	} else if rl.IsKeyPressed(rl.KeyFour) {
		p.SelectedBlock = BlockWood
	} else if rl.IsKeyPressed(rl.KeyFive) {
		p.SelectedBlock = BlockLeaves
	}
}

// Raycast para interação com blocos
func (p *MinecraftPlayer) Raycast(world *MinecraftWorld) (hit bool, hitPos rl.Vector3, normal rl.Vector3) {
	start := p.GetEyePosition()
	direction := p.GetForward()

	// DDA (Digital Differential Analyzer) para raycast em grade de voxels
	step := float32(0.1)
	maxDist := float32(ReachDistance)

	for dist := float32(0); dist < maxDist; dist += step {
		pos := rl.Vector3Add(start, rl.Vector3Scale(direction, dist))

		block := world.GetBlock(pos.X, pos.Y, pos.Z)
		if block != BlockAir {
			// Encontrou bloco
			hitPos = rl.NewVector3(float32(math.Floor(float64(pos.X))), float32(math.Floor(float64(pos.Y))), float32(math.Floor(float64(pos.Z))))

			// Calcular normal aproximado
			prevPos := rl.Vector3Subtract(pos, rl.Vector3Scale(direction, step))
			dx := math.Floor(float64(pos.X)) - math.Floor(float64(prevPos.X))
			dy := math.Floor(float64(pos.Y)) - math.Floor(float64(prevPos.Y))
			dz := math.Floor(float64(pos.Z)) - math.Floor(float64(prevPos.Z))

			normal = rl.NewVector3(float32(dx), float32(dy), float32(dz))

			return true, hitPos, normal
		}
	}

	return false, rl.NewVector3(0, 0, 0), rl.NewVector3(0, 0, 0)
}

// HandleBlockInteraction processa cliques do mouse para adicionar/remover blocos
func (p *MinecraftPlayer) HandleBlockInteraction(world *MinecraftWorld) {
	hit, hitPos, normal := p.Raycast(world)

	if !hit {
		return
	}

	// Botão esquerdo: remover bloco
	if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		world.SetBlock(hitPos.X, hitPos.Y, hitPos.Z, BlockAir)
	}

	// Botão direito: adicionar bloco
	if rl.IsMouseButtonPressed(rl.MouseRightButton) {
		// Colocar bloco na face adjacente
		newPos := rl.Vector3Add(hitPos, normal)

		// Verificar se não está colocando dentro do player
		playerBox := p.Position
		if !p.checkCollision(world, playerBox) ||
		   math.Abs(float64(newPos.X-p.Position.X)) > 1 ||
		   math.Abs(float64(newPos.Y-p.Position.Y)) > 1 ||
		   math.Abs(float64(newPos.Z-p.Position.Z)) > 1 {
			world.SetBlock(newPos.X, newPos.Y, newPos.Z, p.SelectedBlock)
		}
	}
}

// GetSelectedBlockName retorna o nome do bloco selecionado
func (p *MinecraftPlayer) GetSelectedBlockName() string {
	switch p.SelectedBlock {
	case BlockGrass:
		return "Grass"
	case BlockDirt:
		return "Dirt"
	case BlockStone:
		return "Stone"
	case BlockWood:
		return "Wood"
	case BlockLeaves:
		return "Leaves"
	default:
		return "Unknown"
	}
}

// Draw renderiza o player (cápsula)
func (p *MinecraftPlayer) Draw() {
	capsuleHeight := float32(PlayerHeight - PlayerWidth)
	radius := float32(PlayerWidth / 2)

	// Posição do corpo (cilindro)
	bodyPos := rl.NewVector3(p.Position.X, p.Position.Y+radius+capsuleHeight/2, p.Position.Z)

	// Desenhar cilindro (corpo)
	rl.DrawCylinder(bodyPos, radius, radius, capsuleHeight, 8, rl.NewColor(100, 150, 200, 255))
	rl.DrawCylinderWires(bodyPos, radius, radius, capsuleHeight, 8, rl.Black)

	// Desenhar esferas (topo e base)
	topPos := rl.NewVector3(p.Position.X, p.Position.Y+radius+capsuleHeight, p.Position.Z)
	bottomPos := rl.NewVector3(p.Position.X, p.Position.Y+radius, p.Position.Z)

	rl.DrawSphere(topPos, radius, rl.NewColor(100, 150, 200, 255))
	rl.DrawSphereWires(topPos, radius, 8, 8, rl.Black)

	rl.DrawSphere(bottomPos, radius, rl.NewColor(100, 150, 200, 255))
	rl.DrawSphereWires(bottomPos, radius, 8, 8, rl.Black)
}
