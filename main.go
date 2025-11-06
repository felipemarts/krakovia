package main

import (
	"fmt"
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	screenWidth  = 1280
	screenHeight = 720
	blockSize    = 1.0
)

func main() {
	rl.InitWindow(screenWidth, screenHeight, "Krakovia - Minecraft em Go")
	defer rl.CloseWindow()

	rl.SetTargetFPS(60)
	rl.DisableCursor()

	// Inicializar jogador
	player := NewPlayer(rl.NewVector3(8, 20, 8))

	// Inicializar mundo
	world := NewWorld()

	// Gerar terreno inicial
	world.GenerateTerrain()

	for !rl.WindowShouldClose() {
		dt := rl.GetFrameTime()

		// Atualizar jogador
		player.Update(dt, world)

		// Renderizar
		rl.BeginDrawing()
		rl.ClearBackground(rl.SkyBlue)

		rl.BeginMode3D(player.Camera)

		// Renderizar mundo
		world.Render()

		// Renderizar jogador como cápsula
		player.RenderPlayer()

		// Desenhar highlight para indicar onde o bloco será removido
		if player.LookingAtBlock {
			// Centralizar o wireframe no meio do bloco
			centerPos := rl.NewVector3(
				player.TargetBlock.X + 0.5,
				player.TargetBlock.Y + 0.5,
				player.TargetBlock.Z + 0.5,
			)
			rl.DrawCubeWiresV(centerPos, rl.NewVector3(1.01, 1.01, 1.01), rl.Red)
		}

		rl.EndMode3D()

		// UI
		rl.DrawText("WASD - Mover | Espaço - Pular | Mouse - Olhar", 10, 10, 20, rl.Black)
		rl.DrawText("Click Esquerdo - Remover | Click Direito - Colocar", 10, 35, 20, rl.Black)
		rl.DrawText(fmt.Sprintf("Posição: (%.1f, %.1f, %.1f)", player.Position.X, player.Position.Y, player.Position.Z), 10, 60, 20, rl.Black)
		rl.DrawText(fmt.Sprintf("FPS: %d", rl.GetFPS()), 10, screenHeight-30, 20, rl.Green)

		// Crosshair
		rl.DrawLine(screenWidth/2-10, screenHeight/2, screenWidth/2+10, screenHeight/2, rl.White)
		rl.DrawLine(screenWidth/2, screenHeight/2-10, screenWidth/2, screenHeight/2+10, rl.White)

		rl.EndDrawing()
	}
}

// Player representa o jogador
type Player struct {
	Position        rl.Vector3
	Velocity        rl.Vector3
	Camera          rl.Camera3D
	Yaw             float32
	Pitch           float32
	IsOnGround      bool
	LookingAtBlock  bool
	TargetBlock     rl.Vector3
	PlaceBlock      rl.Vector3
	Height          float32
	Radius          float32
	CameraDistance  float32
}

func NewPlayer(position rl.Vector3) *Player {
	player := &Player{
		Position:       position,
		Velocity:       rl.NewVector3(0, 0, 0),
		Yaw:            0,
		Pitch:          0.3, // Olhando um pouco para baixo
		Height:         1.8,
		Radius:         0.3,
		CameraDistance: 5.0,
	}

	// Câmera em terceira pessoa
	player.Camera = rl.Camera3D{
		Position:   rl.NewVector3(position.X, position.Y+2, position.Z+5),
		Target:     rl.NewVector3(position.X, position.Y+1, position.Z),
		Up:         rl.NewVector3(0, 1, 0),
		Fovy:       60.0,
		Projection: rl.CameraPerspective,
	}

	return player
}

func (p *Player) Update(dt float32, world *World) {
	// Controle do mouse
	mouseDelta := rl.GetMouseDelta()
	sensitivity := float32(0.003)

	p.Yaw -= mouseDelta.X * sensitivity
	p.Pitch -= mouseDelta.Y * sensitivity

	// Limitar pitch
	if p.Pitch > 1.5 {
		p.Pitch = 1.5
	}
	if p.Pitch < -1.5 {
		p.Pitch = -1.5
	}

	// Calcular direção frontal e lateral
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
	speed := float32(4.3)
	moveInput := rl.NewVector3(0, 0, 0)

	if rl.IsKeyDown(rl.KeyW) {
		moveInput = rl.Vector3Add(moveInput, forward)
	}
	if rl.IsKeyDown(rl.KeyS) {
		moveInput = rl.Vector3Subtract(moveInput, forward)
	}
	if rl.IsKeyDown(rl.KeyA) {
		moveInput = rl.Vector3Add(moveInput, right)
	}
	if rl.IsKeyDown(rl.KeyD) {
		moveInput = rl.Vector3Subtract(moveInput, right)
	}

	// Normalizar movimento diagonal
	if rl.Vector3Length(moveInput) > 0 {
		moveInput = rl.Vector3Normalize(moveInput)
		moveInput = rl.Vector3Scale(moveInput, speed)
	}

	p.Velocity.X = moveInput.X
	p.Velocity.Z = moveInput.Z

	// Gravidade
	gravity := float32(-20.0)
	p.Velocity.Y += gravity * dt

	// Pulo
	if rl.IsKeyPressed(rl.KeySpace) && p.IsOnGround {
		p.Velocity.Y = 8.0
		p.IsOnGround = false
	}

	// Aplicar velocidade com detecção de colisão
	p.ApplyMovement(dt, world)

	// Atualizar câmera em terceira pessoa
	// Posição do alvo (jogador)
	targetHeight := float32(1.0)
	p.Camera.Target = rl.NewVector3(p.Position.X, p.Position.Y+targetHeight, p.Position.Z)

	// Calcular posição da câmera atrás do jogador
	camX := p.Position.X - float32(math.Sin(float64(p.Yaw))*math.Cos(float64(p.Pitch)))*p.CameraDistance
	camY := p.Position.Y + targetHeight + float32(math.Sin(float64(p.Pitch)))*p.CameraDistance
	camZ := p.Position.Z - float32(math.Cos(float64(p.Yaw))*math.Cos(float64(p.Pitch)))*p.CameraDistance

	p.Camera.Position = rl.NewVector3(camX, camY, camZ)

	// Raycasting para colocar/remover blocos
	p.RaycastBlocks(world)

	// Interação com blocos
	if rl.IsMouseButtonPressed(rl.MouseLeftButton) && p.LookingAtBlock {
		// Remover bloco
		world.SetBlock(int32(p.TargetBlock.X), int32(p.TargetBlock.Y), int32(p.TargetBlock.Z), BlockAir)
	}

	if rl.IsMouseButtonPressed(rl.MouseRightButton) && p.LookingAtBlock {
		// Colocar bloco
		world.SetBlock(int32(p.PlaceBlock.X), int32(p.PlaceBlock.Y), int32(p.PlaceBlock.Z), BlockStone)
	}
}

func (p *Player) RenderPlayer() {
	// Desenhar cápsula representando o jogador
	// Corpo (cilindro)
	bodyHeight := p.Height - p.Radius*2
	bodyPos := rl.NewVector3(p.Position.X, p.Position.Y+p.Radius+bodyHeight/2, p.Position.Z)
	rl.DrawCylinder(bodyPos, p.Radius, p.Radius, bodyHeight, 8, rl.Blue)
	rl.DrawCylinderWires(bodyPos, p.Radius, p.Radius, bodyHeight, 8, rl.DarkBlue)

	// Esfera superior (cabeça)
	topSpherePos := rl.NewVector3(p.Position.X, p.Position.Y+p.Height-p.Radius, p.Position.Z)
	rl.DrawSphere(topSpherePos, p.Radius, rl.Blue)
	rl.DrawSphereWires(topSpherePos, p.Radius, 8, 8, rl.DarkBlue)

	// Esfera inferior (pés)
	bottomSpherePos := rl.NewVector3(p.Position.X, p.Position.Y+p.Radius, p.Position.Z)
	rl.DrawSphere(bottomSpherePos, p.Radius, rl.Blue)
	rl.DrawSphereWires(bottomSpherePos, p.Radius, 8, 8, rl.DarkBlue)

	// Indicador de direção (pequeno cubo na frente)
	dirX := float32(math.Sin(float64(p.Yaw))) * (p.Radius + 0.1)
	dirZ := float32(math.Cos(float64(p.Yaw))) * (p.Radius + 0.1)
	dirPos := rl.NewVector3(p.Position.X+dirX, p.Position.Y+p.Height/2, p.Position.Z+dirZ)
	rl.DrawCube(dirPos, 0.1, 0.1, 0.1, rl.Red)

	// Visualizar cilindro de colisão (semi-transparente)
	collisionPos := rl.NewVector3(p.Position.X, p.Position.Y+p.Height/2, p.Position.Z)
	rl.DrawCylinderWires(collisionPos, p.Radius, p.Radius, p.Height, 12, rl.Yellow)

	// Desenhar círculo no chão mostrando o raio de colisão
	floorPos := rl.NewVector3(p.Position.X, p.Position.Y+0.01, p.Position.Z)
	rl.DrawCircle3D(floorPos, p.Radius, rl.NewVector3(1, 0, 0), 90, rl.Fade(rl.Yellow, 0.3))
}

func (p *Player) ApplyMovement(dt float32, world *World) {
	// Movimento horizontal (X)
	newPosX := p.Position
	newPosX.X += p.Velocity.X * dt
	if !p.CheckCollision(newPosX, world) {
		p.Position.X = newPosX.X
	}

	// Movimento horizontal (Z)
	newPosZ := p.Position
	newPosZ.Z += p.Velocity.Z * dt
	if !p.CheckCollision(newPosZ, world) {
		p.Position.Z = newPosZ.Z
	}

	// Movimento vertical (Y)
	newPosY := p.Position
	newPosY.Y += p.Velocity.Y * dt

	if !p.CheckCollision(newPosY, world) {
		p.Position.Y = newPosY.Y
		p.IsOnGround = false
	} else {
		if p.Velocity.Y < 0 {
			// Colidiu com o chão
			p.IsOnGround = true
			p.Velocity.Y = 0
		} else {
			// Colidiu com o teto
			p.Velocity.Y = 0
		}
	}
}

func (p *Player) CheckCollision(newPos rl.Vector3, world *World) bool {
	// Verificar colisão cilíndrica apropriada
	minX := int32(math.Floor(float64(newPos.X - p.Radius)))
	maxX := int32(math.Floor(float64(newPos.X + p.Radius)))
	minY := int32(math.Floor(float64(newPos.Y)))
	maxY := int32(math.Floor(float64(newPos.Y + p.Height)))
	minZ := int32(math.Floor(float64(newPos.Z - p.Radius)))
	maxZ := int32(math.Floor(float64(newPos.Z + p.Radius)))

	for x := minX; x <= maxX; x++ {
		for y := minY; y <= maxY; y++ {
			for z := minZ; z <= maxZ; z++ {
				if world.GetBlock(x, y, z) != BlockAir {
					// Verificar se realmente colide com o cilindro do jogador
					// Centro do bloco
					blockCenterX := float32(x) + 0.5
					blockCenterZ := float32(z) + 0.5

					// Distância horizontal do centro do jogador ao centro do bloco
					dx := newPos.X - blockCenterX
					dz := newPos.Z - blockCenterZ
					distSq := dx*dx + dz*dz

					// Colisão cilíndrica: verificar se a distância é menor que a soma dos raios
					// (raio do jogador + raio do bloco que é 0.5)
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
	// Raycast a partir de uma posição de mira de ombro (sobre o ombro direito)
	eyeHeight := float32(1.5)
	shoulderOffset := float32(0.3) // Offset para a direita
	shoulderUp := float32(0.2)     // Offset para cima

	// Calcular direção "direita" baseada no yaw
	rightX := float32(math.Cos(float64(p.Yaw)))
	rightZ := float32(-math.Sin(float64(p.Yaw)))

	// Posição de mira sobre o ombro direito
	rayOrigin := rl.NewVector3(
		p.Position.X + rightX*shoulderOffset,
		p.Position.Y + eyeHeight + shoulderUp,
		p.Position.Z + rightZ*shoulderOffset,
	)
	rayDir := rl.Vector3Normalize(rl.Vector3Subtract(p.Camera.Target, p.Camera.Position))

	maxDistance := float32(8.0)
	p.LookingAtBlock = false

	// Posição inicial do voxel
	voxelX := int32(math.Floor(float64(rayOrigin.X)))
	voxelY := int32(math.Floor(float64(rayOrigin.Y)))
	voxelZ := int32(math.Floor(float64(rayOrigin.Z)))

	// Direção do passo (1 ou -1)
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

	// Armazenar voxel anterior para colocação de blocos
	prevVoxelX, prevVoxelY, prevVoxelZ := voxelX, voxelY, voxelZ

	// DDA traversal
	for t := float32(0); t < maxDistance; {
		// Verificar se o voxel atual contém um bloco
		if world.GetBlock(voxelX, voxelY, voxelZ) != BlockAir {
			p.LookingAtBlock = true
			p.TargetBlock = rl.NewVector3(float32(voxelX), float32(voxelY), float32(voxelZ))
			p.PlaceBlock = rl.NewVector3(float32(prevVoxelX), float32(prevVoxelY), float32(prevVoxelZ))
			return
		}

		// Armazenar voxel atual antes de avançar
		prevVoxelX, prevVoxelY, prevVoxelZ = voxelX, voxelY, voxelZ

		// Avançar para o próximo voxel
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

// World representa o mundo voxel
type World struct {
	Blocks map[int64]BlockType
	SizeX  int32
	SizeY  int32
	SizeZ  int32
}

type BlockType uint8

const (
	BlockAir BlockType = iota
	BlockGrass
	BlockDirt
	BlockStone
)

func NewWorld() *World {
	return &World{
		Blocks: make(map[int64]BlockType),
		SizeX:  32,
		SizeY:  64,
		SizeZ:  32,
	}
}

func (w *World) GetBlockIndex(x, y, z int32) int64 {
	return int64(x) | (int64(y) << 20) | (int64(z) << 40)
}

func (w *World) SetBlock(x, y, z int32, block BlockType) {
	if x < 0 || x >= w.SizeX || y < 0 || y >= w.SizeY || z < 0 || z >= w.SizeZ {
		return
	}

	idx := w.GetBlockIndex(x, y, z)
	if block == BlockAir {
		delete(w.Blocks, idx)
	} else {
		w.Blocks[idx] = block
	}
}

func (w *World) GetBlock(x, y, z int32) BlockType {
	if x < 0 || x >= w.SizeX || y < 0 || y >= w.SizeY || z < 0 || z >= w.SizeZ {
		return BlockAir
	}

	idx := w.GetBlockIndex(x, y, z)
	if block, exists := w.Blocks[idx]; exists {
		return block
	}
	return BlockAir
}

func (w *World) GenerateTerrain() {
	// Gerar terreno simples
	for x := int32(0); x < w.SizeX; x++ {
		for z := int32(0); z < w.SizeZ; z++ {
			// Altura base + variação simples
			height := int32(10 + int32(math.Sin(float64(x)*0.3)*3+math.Cos(float64(z)*0.3)*3))

			for y := int32(0); y <= height; y++ {
				if y == height {
					w.SetBlock(x, y, z, BlockGrass)
				} else if y >= height-3 {
					w.SetBlock(x, y, z, BlockDirt)
				} else {
					w.SetBlock(x, y, z, BlockStone)
				}
			}
		}
	}
}

func (w *World) Render() {
	// Renderizar todos os blocos
	for idx, blockType := range w.Blocks {
		if blockType == BlockAir {
			continue
		}

		// Decodificar posição
		x := int32(idx & 0xFFFFF)
		y := int32((idx >> 20) & 0xFFFFF)
		z := int32((idx >> 40) & 0xFFFFF)

		// Ajustar para valores negativos se necessário
		if x >= 0x80000 {
			x -= 0x100000
		}
		if y >= 0x80000 {
			y -= 0x100000
		}
		if z >= 0x80000 {
			z -= 0x100000
		}

		// Centralizar o cubo (DrawCubeV desenha do centro)
		pos := rl.NewVector3(float32(x)+0.5, float32(y)+0.5, float32(z)+0.5)

		// Escolher cor baseada no tipo
		var color rl.Color
		switch blockType {
		case BlockGrass:
			color = rl.Green
		case BlockDirt:
			color = rl.Brown
		case BlockStone:
			color = rl.Gray
		default:
			color = rl.White
		}

		// Desenhar cubo
		rl.DrawCubeV(pos, rl.NewVector3(1, 1, 1), color)
		rl.DrawCubeWiresV(pos, rl.NewVector3(1.01, 1.01, 1.01), rl.Black)
	}
}
