package game

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	cameraTargetHeight              = 1.0
	cameraTargetOffsetUp            = 1.0
	cameraTargetOffsetRight         = -0.6
	cameraTransitionSpeed           = 8.0
	cameraAutoFirstPersonSwitch     = 0.8
	cameraCollisionProbeStep        = 0.2
	cameraCollisionPadding          = 0.3
	cameraFirstPersonForwardOffset  = 0.05
	cameraFirstPersonBlendThreshold = 0.15
)

// Player representa o jogador
type Player struct {
	Position            rl.Vector3
	Velocity            rl.Vector3
	Camera              rl.Camera3D
	Yaw                 float32
	Pitch               float32
	IsOnGround          bool
	LookingAtBlock      bool
	TargetBlock         rl.Vector3
	PlaceBlock          rl.Vector3
	Height              float32
	Radius              float32
	CameraDistance      float32
	FirstPerson         bool
	ThirdPersonDistance float32
	FirstPersonDistance float32
	FlyMode             bool
	ShowCollisionBody   bool
}

func NewPlayer(position rl.Vector3) *Player {
	player := &Player{
		Position:            position,
		Velocity:            rl.NewVector3(0, 0, 0),
		Yaw:                 0,
		Pitch:               0.3, // Olhando um pouco para baixo
		Height:              1.8,
		Radius:              0.3,
		CameraDistance:      5.0,
		ThirdPersonDistance: 5.0,
		FirstPersonDistance: 0.35,
	}

	// CÃ¢mera em terceira pessoa
	player.Camera = rl.Camera3D{
		Position:   rl.NewVector3(position.X, position.Y+2, position.Z+5),
		Target:     rl.NewVector3(position.X, position.Y+1, position.Z),
		Up:         rl.NewVector3(0, 1, 0),
		Fovy:       60.0,
		Projection: rl.CameraPerspective,
	}

	return player
}

func (p *Player) Update(dt float32, world *World, input Input) {
	// Toggle fly mode com tecla P
	if input.IsFlyTogglePressed() {
		p.FlyMode = !p.FlyMode
		if p.FlyMode {
			// Ao ativar fly mode, zerar velocidade vertical
			p.Velocity.Y = 0
		}
	}

	// Alternar modos de cÃ¢mera com a tecla V
	if input.IsCameraTogglePressed() {
		p.FirstPerson = !p.FirstPerson
	}

	// Toggle visualização do corpo de colisão com tecla K
	if input.IsCollisionTogglePressed() {
		p.ShowCollisionBody = !p.ShowCollisionBody
	}

	// Controle do mouse
	mouseDelta := input.GetMouseDelta()
	sensitivity := float32(0.003)

	p.Yaw -= mouseDelta.X * sensitivity
	p.Pitch -= mouseDelta.Y * sensitivity // Mantém sensação natural em primeira e terceira pessoa

	// Limitar pitch
	if p.Pitch > 1.5 {
		p.Pitch = 1.5
	}
	if p.Pitch < -1.5 {
		p.Pitch = -1.5
	}

	// Calcular direÃ§Ã£o frontal e lateral
	forward := rl.NewVector3(
		float32(math.Sin(float64(p.Yaw))),
		0,
		float32(math.Cos(float64(p.Yaw))),
	)
	right := rl.NewVector3(
		float32(math.Sin(float64(p.Yaw+math.Pi/2))),
		0,
		float32(math.Cos(float64(p.Yaw+math.Pi/2))),
	)

	// Movimento WASD
	speed := float32(15.0)
	moveInput := rl.NewVector3(0, 0, 0)

	if input.IsForwardPressed() {
		moveInput = rl.Vector3Add(moveInput, forward)
	}
	if input.IsBackPressed() {
		moveInput = rl.Vector3Subtract(moveInput, forward)
	}
	if input.IsLeftPressed() {
		moveInput = rl.Vector3Add(moveInput, right)
	}
	if input.IsRightPressed() {
		moveInput = rl.Vector3Subtract(moveInput, right)
	}

	// Normalizar movimento diagonal
	if rl.Vector3Length(moveInput) > 0 {
		moveInput = rl.Vector3Normalize(moveInput)
		moveInput = rl.Vector3Scale(moveInput, speed)
	}

	p.Velocity.X = moveInput.X
	p.Velocity.Z = moveInput.Z

	// LÃ³gica de fÃ­sica diferente baseado no modo fly
	if p.FlyMode {
		// No modo fly: sem gravidade, controle vertical com Shift/Ctrl
		flySpeed := float32(15.0)
		p.Velocity.Y = 0

		if input.IsFlyUpPressed() {
			p.Velocity.Y = flySpeed
		}
		if input.IsFlyDownPressed() {
			p.Velocity.Y = -flySpeed
		}

		// No modo fly, aplicar movimento sem colisÃµes
		p.Position.X += p.Velocity.X * dt
		p.Position.Y += p.Velocity.Y * dt
		p.Position.Z += p.Velocity.Z * dt
	} else {
		// Modo normal: gravidade e colisÃµes ativas
		gravity := float32(-20.0)
		p.Velocity.Y += gravity * dt

		// Pulo
		if input.IsJumpPressed() && p.IsOnGround {
			p.Velocity.Y = 8.0
			p.IsOnGround = false
		}

		// Aplicar velocidade com detecÃ§Ã£o de colisÃ£o
		p.ApplyMovement(dt, world)
	}

	// Atualizar câmera considerando colisões e transições suaves
	p.updateCamera(dt, world)

	// Raycasting para colocar/remover blocos
	p.RaycastBlocks(world)

	// InteraÃ§Ã£o com blocos
	if input.IsLeftClickPressed() && p.LookingAtBlock {
		// Remover bloco
		world.SetBlock(int32(p.TargetBlock.X), int32(p.TargetBlock.Y), int32(p.TargetBlock.Z), BlockAir)
	}

	if input.IsRightClickPressed() && p.LookingAtBlock {
		// Colocar bloco - mas verificar se não colide com o jogador
		placePos := rl.NewVector3(
			float32(int32(p.PlaceBlock.X))+0.5,
			float32(int32(p.PlaceBlock.Y)),
			float32(int32(p.PlaceBlock.Z))+0.5,
		)

		// Verificar se o bloco que vai ser colocado não colide com o jogador
		if !p.wouldBlockCollideWithPlayer(placePos) {
			world.SetBlock(int32(p.PlaceBlock.X), int32(p.PlaceBlock.Y), int32(p.PlaceBlock.Z), BlockStone)
		}
	}
}

func (p *Player) updateCamera(dt float32, world *World) {
	desiredDistance := p.ThirdPersonDistance
	if p.FirstPerson {
		desiredDistance = p.FirstPersonDistance
	}

	p.CameraDistance = smoothApproach(p.CameraDistance, desiredDistance, dt, cameraTransitionSpeed)
	if p.CameraDistance < p.FirstPersonDistance {
		p.CameraDistance = p.FirstPersonDistance
	}

	right := rl.NewVector3(
		float32(math.Cos(float64(p.Yaw))),
		0,
		float32(-math.Sin(float64(p.Yaw))),
	)

	head := rl.NewVector3(p.Position.X, p.Position.Y+cameraTargetHeight, p.Position.Z)
	totalRange := p.ThirdPersonDistance - p.FirstPersonDistance
	shoulderBlend := float32(1.0)
	if totalRange > 0 {
		shoulderBlend = clamp01((p.CameraDistance - p.FirstPersonDistance) / totalRange)
	}

	dynamicOffsetRight := cameraTargetOffsetRight * shoulderBlend
	pivot := rl.Vector3Add(head, rl.Vector3Scale(right, dynamicOffsetRight))
	pivot = rl.Vector3Add(pivot, rl.NewVector3(0, cameraTargetOffsetUp, 0))

	forward := rl.NewVector3(
		float32(math.Sin(float64(p.Yaw)))*float32(math.Cos(float64(p.Pitch))),
		float32(math.Sin(float64(p.Pitch))),
		float32(math.Cos(float64(p.Yaw)))*float32(math.Cos(float64(p.Pitch))),
	)
	if rl.Vector3Length(forward) == 0 {
		forward = rl.NewVector3(0, 0, 1)
	} else {
		forward = rl.Vector3Normalize(forward)
	}
	backward := rl.Vector3Scale(forward, -1)

	collisionDistance := p.resolveCameraCollision(world, pivot, backward, p.CameraDistance)

	useFirstPerson := false
	if collisionDistance < cameraAutoFirstPersonSwitch {
		useFirstPerson = true
	} else if p.CameraDistance <= p.FirstPersonDistance+cameraFirstPersonBlendThreshold {
		// Ainda estamos no "túnel" de transição próximo ao jogador, manter primeira pessoa
		useFirstPerson = true
	} else if p.FirstPerson {
		// Deseja primeira pessoa, mas aguardar aproximação suave
		useFirstPerson = false
	}

	var cameraPos rl.Vector3
	var cameraTarget rl.Vector3

	if useFirstPerson {
		viewPivot := rl.Vector3Add(head, rl.NewVector3(0, cameraTargetOffsetUp*0.5, 0))
		cameraPos = rl.Vector3Add(viewPivot, rl.Vector3Scale(forward, cameraFirstPersonForwardOffset))
		cameraTarget = rl.Vector3Add(cameraPos, forward)
	} else {
		cameraPos = rl.Vector3Add(pivot, rl.Vector3Scale(backward, collisionDistance))
		cameraTarget = rl.Vector3Add(pivot, forward)
	}

	p.Camera.Position = cameraPos
	p.Camera.Target = cameraTarget
}

func (p *Player) resolveCameraCollision(world *World, pivot, backward rl.Vector3, desired float32) float32 {
	if world == nil {
		return desired
	}

	maxDistance := desired
	if maxDistance < p.FirstPersonDistance {
		maxDistance = p.FirstPersonDistance
	}

	steps := int(maxDistance/cameraCollisionProbeStep) + 1
	for i := 0; i <= steps; i++ {
		distance := float32(i) * cameraCollisionProbeStep
		if distance > maxDistance {
			distance = maxDistance
		}

		point := rl.Vector3Add(pivot, rl.Vector3Scale(backward, distance))
		if p.isCameraObstructed(world, point) {
			clipped := distance - cameraCollisionPadding
			if clipped < p.FirstPersonDistance {
				clipped = p.FirstPersonDistance
			}
			if clipped < 0 {
				clipped = 0
			}
			return clipped
		}
	}

	return maxDistance
}

func (p *Player) isCameraObstructed(world *World, point rl.Vector3) bool {
	if world == nil {
		return false
	}

	x := int32(math.Floor(float64(point.X)))
	y := int32(math.Floor(float64(point.Y)))
	z := int32(math.Floor(float64(point.Z)))

	return world.GetBlock(x, y, z) != BlockAir
}

func smoothApproach(current, target, dt, speed float32) float32 {
	if speed <= 0 || dt <= 0 {
		return target
	}

	factor := 1 - float32(math.Exp(float64(-speed*dt)))
	return current + (target-current)*factor
}

func clamp01(value float32) float32 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

func (p *Player) RenderPlayer() {
	if !p.ShowCollisionBody {
		return
	}

	base := rl.NewVector3(p.Position.X, p.Position.Y, p.Position.Z)
	top := rl.NewVector3(p.Position.X, p.Position.Y+p.Height, p.Position.Z)

	fillColor := rl.Color{R: 255, G: 229, B: 153, A: 80}
	wireColor := rl.Color{R: 255, G: 140, B: 0, A: 255}

	rl.DrawCylinderEx(base, top, p.Radius, p.Radius, 20, fillColor)
	rl.DrawCylinderWiresEx(base, top, p.Radius, p.Radius, 12, wireColor)
}

func (p *Player) ApplyMovement(dt float32, world *World) {
	// Limitar delta time para evitar tunneling em caso de lag
	// Subdividir movimentos grandes em steps menores
	maxDt := float32(0.016) // ~60 FPS
	remainingDt := dt

	for remainingDt > 0 {
		stepDt := remainingDt
		if stepDt > maxDt {
			stepDt = maxDt
		}
		remainingDt -= stepDt

		// Movimento horizontal (X)
		newPosX := p.Position
		newPosX.X += p.Velocity.X * stepDt
		if !p.CheckCollision(newPosX, world) {
			p.Position.X = newPosX.X
		}

		// Movimento horizontal (Z)
		newPosZ := p.Position
		newPosZ.Z += p.Velocity.Z * stepDt
		if !p.CheckCollision(newPosZ, world) {
			p.Position.Z = newPosZ.Z
		}

		// Movimento vertical (Y)
		newPosY := p.Position
		newPosY.Y += p.Velocity.Y * stepDt

		if !p.CheckCollision(newPosY, world) {
			p.Position.Y = newPosY.Y
			// SÃ³ marcar como nÃ£o no chÃ£o se estamos realmente nos movendo para cima ou caindo
			if p.Velocity.Y != 0 {
				p.IsOnGround = false
			}
		} else {
			if p.Velocity.Y < 0 {
				// Colidiu com o chÃ£o
				p.IsOnGround = true
				p.Velocity.Y = 0
			} else if p.Velocity.Y > 0 {
				// Colidiu com o teto
				p.Velocity.Y = 0
			}
		}
	}

	// VerificaÃ§Ã£o extra: check colisÃ£o abaixo para garantir IsOnGround correto
	checkBelowPos := p.Position
	checkBelowPos.Y -= 0.01 // Verificar ligeiramente abaixo
	if p.CheckCollision(checkBelowPos, world) {
		p.IsOnGround = true
	}
}

// wouldBlockCollideWithPlayer verifica se um bloco na posição dada colidiria com o jogador
func (p *Player) wouldBlockCollideWithPlayer(blockPos rl.Vector3) bool {
	// blockPos é o centro do bloco (x+0.5, y, z+0.5)
	// Verificar colisão do cilindro do jogador com o bloco

	// Distância horizontal do centro do jogador ao centro do bloco
	dx := p.Position.X - blockPos.X
	dz := p.Position.Z - blockPos.Z
	distSq := dx*dx + dz*dz

	// Colisão horizontal (cilíndrica)
	maxDist := p.Radius + 0.5
	if distSq >= maxDist*maxDist {
		return false // Muito longe horizontalmente
	}

	// Colisão vertical
	// O bloco ocupa de blockPos.Y até blockPos.Y+1
	// O jogador ocupa de p.Position.Y até p.Position.Y+p.Height
	blockBottom := blockPos.Y
	blockTop := blockPos.Y + 1.0
	playerBottom := p.Position.Y
	playerTop := p.Position.Y + p.Height

	// Verificar se há sobreposição vertical
	if playerTop <= blockBottom || playerBottom >= blockTop {
		return false // Sem sobreposição vertical
	}

	return true // Colide!
}

func (p *Player) CheckCollision(newPos rl.Vector3, world *World) bool {
	// Verificar colisÃ£o cilÃ­ndrica apropriada
	minX := int32(math.Floor(float64(newPos.X - p.Radius)))
	maxX := int32(math.Floor(float64(newPos.X + p.Radius)))
	minY := int32(math.Floor(float64(newPos.Y)))
	maxY := int32(math.Floor(float64(newPos.Y + p.Height)))
	minZ := int32(math.Floor(float64(newPos.Z - p.Radius)))
	maxZ := int32(math.Floor(float64(newPos.Z + p.Radius)))

	for x := minX; x <= maxX; x++ {
		for y := minY; y <= maxY; y++ {
			for z := minZ; z <= maxZ; z++ {
				blockType := world.GetBlock(x, y, z)
				if blockType != BlockAir {
					// OtimizaÃ§Ã£o: ignorar colisÃ£o com blocos completamente ocultos
					// (eles nÃ£o podem ser alcanÃ§ados pelo jogador)
					if world.IsBlockHidden(x, y, z) {
						continue
					}

					// Verificar se realmente colide com o cilindro do jogador
					// Centro do bloco
					blockCenterX := float32(x) + 0.5
					blockCenterZ := float32(z) + 0.5

					// DistÃ¢ncia horizontal do centro do jogador ao centro do bloco
					dx := newPos.X - blockCenterX
					dz := newPos.Z - blockCenterZ
					distSq := dx*dx + dz*dz

					// ColisÃ£o cilÃ­ndrica: verificar se a distÃ¢ncia Ã© menor que a soma dos raios
					// (raio do jogador + raio do bloco que Ã© 0.5)
					maxDist := p.Radius + 0.5
					if distSq < maxDist*maxDist {
						return true
					}
				}
			}
		}
	}

	return false
}

func (p *Player) RaycastBlocks(world *World) {
	// Raycast diretamente da cÃ¢mera na direÃ§Ã£o que ela estÃ¡ apontando
	// Isso garante que o raycast sempre acerte onde o crosshair aponta
	rayOrigin := p.Camera.Position
	rayDir := rl.Vector3Normalize(rl.Vector3Subtract(p.Camera.Target, p.Camera.Position))

	maxDistance := float32(10.0)
	p.LookingAtBlock = false

	// PosiÃ§Ã£o inicial do voxel
	voxelX := int32(math.Floor(float64(rayOrigin.X)))
	voxelY := int32(math.Floor(float64(rayOrigin.Y)))
	voxelZ := int32(math.Floor(float64(rayOrigin.Z)))

	// DireÃ§Ã£o do passo (1 ou -1)
	stepX := int32(1)
	if rayDir.X < 0 {
		stepX = -1
	}
	stepY := int32(1)
	if rayDir.Y < 0 {
		stepY = -1
	}
	stepZ := int32(1)
	if rayDir.Z < 0 {
		stepZ = -1
	}

	// Calcular tMax e tDelta
	var tMaxX, tMaxY, tMaxZ float32
	var tDeltaX, tDeltaY, tDeltaZ float32

	if rayDir.X != 0 {
		if rayDir.X > 0 {
			tMaxX = (float32(voxelX+1) - rayOrigin.X) / rayDir.X
		} else {
			tMaxX = (float32(voxelX) - rayOrigin.X) / rayDir.X
		}
		tDeltaX = float32(math.Abs(float64(1.0 / rayDir.X)))
	} else {
		tMaxX = float32(math.MaxFloat32)
		tDeltaX = float32(math.MaxFloat32)
	}

	if rayDir.Y != 0 {
		if rayDir.Y > 0 {
			tMaxY = (float32(voxelY+1) - rayOrigin.Y) / rayDir.Y
		} else {
			tMaxY = (float32(voxelY) - rayOrigin.Y) / rayDir.Y
		}
		tDeltaY = float32(math.Abs(float64(1.0 / rayDir.Y)))
	} else {
		tMaxY = float32(math.MaxFloat32)
		tDeltaY = float32(math.MaxFloat32)
	}

	if rayDir.Z != 0 {
		if rayDir.Z > 0 {
			tMaxZ = (float32(voxelZ+1) - rayOrigin.Z) / rayDir.Z
		} else {
			tMaxZ = (float32(voxelZ) - rayOrigin.Z) / rayDir.Z
		}
		tDeltaZ = float32(math.Abs(float64(1.0 / rayDir.Z)))
	} else {
		tMaxZ = float32(math.MaxFloat32)
		tDeltaZ = float32(math.MaxFloat32)
	}

	// Armazenar voxel anterior para colocaÃ§Ã£o de blocos
	prevVoxelX, prevVoxelY, prevVoxelZ := voxelX, voxelY, voxelZ

	// DDA traversal
	for t := float32(0); t < maxDistance; {
		// Verificar se o voxel atual contÃ©m um bloco
		if world.GetBlock(voxelX, voxelY, voxelZ) != BlockAir {
			p.LookingAtBlock = true
			p.TargetBlock = rl.NewVector3(float32(voxelX), float32(voxelY), float32(voxelZ))
			p.PlaceBlock = rl.NewVector3(float32(prevVoxelX), float32(prevVoxelY), float32(prevVoxelZ))
			return
		}

		// Armazenar voxel atual antes de avanÃ§ar
		prevVoxelX, prevVoxelY, prevVoxelZ = voxelX, voxelY, voxelZ

		// AvanÃ§ar para o prÃ³ximo voxel
		if tMaxX < tMaxY {
			if tMaxX < tMaxZ {
				voxelX += stepX
				t = tMaxX
				tMaxX += tDeltaX
			} else {
				voxelZ += stepZ
				t = tMaxZ
				tMaxZ += tDeltaZ
			}
		} else {
			if tMaxY < tMaxZ {
				voxelY += stepY
				t = tMaxY
				tMaxY += tDeltaY
			} else {
				voxelZ += stepZ
				t = tMaxZ
				tMaxZ += tDeltaZ
			}
		}
	}
}
