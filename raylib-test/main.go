package main

import (
	"fmt"
	"math"
	"math/rand"

	rl "github.com/gen2brain/raylib-go/raylib"
)

func main() {
	// Initialize window
	screenWidth := int32(1024)
	screenHeight := int32(768)

	rl.InitWindow(screenWidth, screenHeight, "Voxel World - 10k Voxels Instanced")
	defer rl.CloseWindow()

	rl.SetTargetFPS(60)

	// Setup camera
	camera := rl.Camera3D{
		Position:   rl.NewVector3(50.0, 30.0, 50.0),
		Target:     rl.NewVector3(0.0, 0.0, 0.0),
		Up:         rl.NewVector3(0.0, 1.0, 0.0),
		Fovy:       45.0,
		Projection: rl.CameraPerspective,
	}

	// Carregar modelo com animação
	model := rl.LoadModel("../model.glb")
	defer rl.UnloadModel(model)

	// Carregar animações do modelo
	modelAnimations := rl.LoadModelAnimations("../model.glb")
	animsCount := int32(len(modelAnimations))
	defer rl.UnloadModelAnimations(modelAnimations)

	animFrameCounter := int32(0)
	currentAnim := int32(0)

	// Criar mundo de voxels com capacidade para 10k voxels
	// Agora com texturas!
	voxelWorld := NewVoxelWorld(10000, "voxel_atlas.png", 2)
	defer voxelWorld.Unload()

	// Gerar 10 mil voxels com texturas aleatórias
	generateVoxelTerrainWithTextures(voxelWorld, 10000)

	// Atualizar dados de instancing
	voxelWorld.UpdateInstanceData()

	fmt.Printf("Voxel World initialized!\n")
	fmt.Printf("Total voxels: %d\n", voxelWorld.ActiveInstances)
	fmt.Printf("Rendering with single draw call using instancing\n")
	fmt.Printf("Atlas texture ID: %d\n", voxelWorld.AtlasTexture.ID)
	fmt.Printf("Atlas size: %dx%d\n", voxelWorld.AtlasTexture.Width, voxelWorld.AtlasTexture.Height)
	if animsCount > 0 {
		fmt.Printf("Model animations: %d\n", animsCount)
	}

	// Main game loop
	for !rl.WindowShouldClose() {
		// Update camera
		rl.UpdateCamera(&camera, rl.CameraOrbital)

		// Update model animation
		if animsCount > 0 {
			animFrameCounter++
			rl.UpdateModelAnimation(model, modelAnimations[currentAnim], animFrameCounter)

			// Loop animation
			if animFrameCounter >= modelAnimations[currentAnim].FrameCount {
				animFrameCounter = 0
			}
		}

		// Draw
		rl.BeginDrawing()
		rl.ClearBackground(rl.NewColor(135, 206, 235, 255)) // Sky blue

		rl.BeginMode3D(camera)

		// Draw all voxels with ONE draw call!
		voxelWorld.Draw()

		// Draw animated model in the center
		rl.DrawModel(model, rl.NewVector3(0.0, 10.0, 0.0), 2.0, rl.White)

		// Draw grid
		rl.DrawGrid(100, 1.0)

		rl.EndMode3D()

		// Draw info
		rl.DrawText("Voxel World - Instanced Rendering", 10, 10, 20, rl.DarkGray)
		infoText := fmt.Sprintf("Voxels: %d | Draw Calls: 1 | Frame: %d", voxelWorld.ActiveInstances, animFrameCounter)
		rl.DrawText(infoText, 10, 40, 20, rl.DarkGray)
		rl.DrawText("Use mouse to rotate camera", 10, 70, 20, rl.DarkGray)

		rl.DrawFPS(10, screenHeight-30)

		rl.EndDrawing()
	}
}

// generateVoxelTerrainWithTextures gera um terreno de voxels com texturas aleatórias
func generateVoxelTerrainWithTextures(world *VoxelWorld, numVoxels int32) {
	// Criar um chunk principal
	chunk := NewChunk(rl.NewVector3(0, 0, 0), 100)

	// Gerar voxels em um padrão de terreno com ruído
	gridSize := int32(math.Sqrt(float64(numVoxels)))

	for i := int32(0); i < numVoxels; i++ {
		x := i % gridSize
		z := i / gridSize

		// Criar um padrão de altura usando função seno/cosseno
		fx := float32(x) * 0.1
		fz := float32(z) * 0.1
		height := float32(math.Sin(float64(fx))*3.0 + math.Cos(float64(fz))*3.0 + 5.0)

		// Adicionar alguma aleatoriedade
		height += rand.Float32() * 2.0

		// Escolher textura aleatória (0 = azul, 1 = verde)
		textureIndex := int32(rand.Intn(2))

		position := rl.NewVector3(float32(x)-float32(gridSize)/2, height, float32(z)-float32(gridSize)/2)
		chunk.AddVoxelWithTexture(position, rl.White, textureIndex)
	}

	world.AddChunk(chunk)
}

// generateVoxelTerrain gera um terreno de voxels interessante (sem texturas)
func generateVoxelTerrain(world *VoxelWorld, numVoxels int32) {
	// Criar um chunk principal
	chunk := NewChunk(rl.NewVector3(0, 0, 0), 100)

	// Gerar voxels em um padrão de terreno com ruído
	gridSize := int32(math.Sqrt(float64(numVoxels)))

	for i := int32(0); i < numVoxels; i++ {
		x := i % gridSize
		z := i / gridSize

		// Criar um padrão de altura usando função seno/cosseno
		fx := float32(x) * 0.1
		fz := float32(z) * 0.1
		height := float32(math.Sin(float64(fx))*3.0 + math.Cos(float64(fz))*3.0 + 5.0)

		// Adicionar alguma aleatoriedade
		height += rand.Float32() * 2.0

		// Escolher cor baseada na altura
		var color rl.Color
		if height < 3.0 {
			color = rl.NewColor(34, 139, 34, 255) // Verde escuro (baixo)
		} else if height < 6.0 {
			color = rl.NewColor(107, 142, 35, 255) // Verde oliva (médio)
		} else if height < 9.0 {
			color = rl.NewColor(139, 137, 137, 255) // Cinza (alto)
		} else {
			color = rl.White // Branco (picos)
		}

		position := rl.NewVector3(float32(x)-float32(gridSize)/2, height, float32(z)-float32(gridSize)/2)
		chunk.AddVoxel(position, color)
	}

	world.AddChunk(chunk)
}

// Funções alternativas de geração de terreno

// generateVoxelCube gera um cubo de voxels
func generateVoxelCube(world *VoxelWorld, size int32) {
	chunk := NewChunk(rl.NewVector3(0, 0, 0), size)

	for x := int32(0); x < size; x++ {
		for y := int32(0); y < size; y++ {
			for z := int32(0); z < size; z++ {
				// Apenas voxels na superfície do cubo
				if x == 0 || x == size-1 || y == 0 || y == size-1 || z == 0 || z == size-1 {
					color := rl.NewColor(
						uint8(rand.Intn(255)),
						uint8(rand.Intn(255)),
						uint8(rand.Intn(255)),
						255,
					)
					position := rl.NewVector3(float32(x)-float32(size)/2, float32(y), float32(z)-float32(size)/2)
					chunk.AddVoxel(position, color)
				}
			}
		}
	}

	world.AddChunk(chunk)
}

// generateVoxelSphere gera uma esfera de voxels
func generateVoxelSphere(world *VoxelWorld, radius float32, numVoxels int32) {
	chunk := NewChunk(rl.NewVector3(0, 0, 0), 100)

	for i := int32(0); i < numVoxels; i++ {
		// Gerar pontos aleatórios em uma esfera
		theta := rand.Float32() * 2.0 * math.Pi
		phi := rand.Float32() * math.Pi

		x := radius * float32(math.Sin(float64(phi))) * float32(math.Cos(float64(theta)))
		y := radius * float32(math.Sin(float64(phi))) * float32(math.Sin(float64(theta)))
		z := radius * float32(math.Cos(float64(phi)))

		color := rl.NewColor(
			uint8(rand.Intn(255)),
			uint8(rand.Intn(255)),
			uint8(rand.Intn(255)),
			255,
		)

		chunk.AddVoxel(rl.NewVector3(x, y+radius, z), color)
	}

	world.AddChunk(chunk)
}
