package main

import (
	"fmt"

	rl "github.com/gen2brain/raylib-go/raylib"
)

func main() {
	// Initialize window
	screenWidth := int32(1024)
	screenHeight := int32(768)

	rl.InitWindow(screenWidth, screenHeight, "Raylib GLB Animation Test")
	defer rl.CloseWindow()

	rl.SetTargetFPS(60)

	// Setup camera
	camera := rl.Camera3D{
		Position:   rl.NewVector3(5.0, 5.0, 5.0),
		Target:     rl.NewVector3(0.0, 1.0, 0.0),
		Up:         rl.NewVector3(0.0, 1.0, 0.0),
		Fovy:       45.0,
		Projection: rl.CameraPerspective,
	}

	// Load model
	model := rl.LoadModel("../model.glb")
	defer rl.UnloadModel(model)

	// Load model animations
	modelAnimations := rl.LoadModelAnimations("../model.glb")
	animsCount := int32(len(modelAnimations))
	defer rl.UnloadModelAnimations(modelAnimations)

	animFrameCounter := int32(0)
	currentAnim := int32(0)

	fmt.Printf("Model loaded successfully!\n")
	fmt.Printf("Number of animations: %d\n", animsCount)

	if animsCount > 0 {
		fmt.Printf("Animation 0 - Frames: %d, Bones: %d\n",
			modelAnimations[0].FrameCount,
			modelAnimations[0].BoneCount)
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
		rl.ClearBackground(rl.RayWhite)

		rl.BeginMode3D(camera)

		// Draw model
		rl.DrawModel(model, rl.NewVector3(0.0, 0.0, 0.0), 1.0, rl.White)

		// Draw grid
		rl.DrawGrid(10, 1.0)

		rl.EndMode3D()

		// Draw info
		rl.DrawText("Raylib GLB Animation Test", 10, 10, 20, rl.DarkGray)
		infoText := fmt.Sprintf("Animations: %d | Frame: %d", animsCount, animFrameCounter)
		rl.DrawText(infoText, 10, 40, 20, rl.DarkGray)
		rl.DrawText("Use mouse to rotate camera", 10, 70, 20, rl.DarkGray)

		rl.DrawFPS(10, screenHeight-30)

		rl.EndDrawing()
	}
}
