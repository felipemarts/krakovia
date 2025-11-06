package main

import (
	"fmt"

	rl "github.com/gen2brain/raylib-go/raylib"
)

func main() {
	// Inicializar janela
	screenWidth := int32(1920)
	screenHeight := int32(1080)

	rl.InitWindow(screenWidth, screenHeight, "Minecraft Clone - Go + Raylib")
	defer rl.CloseWindow()

	// Desabilitar cursor para FPS camera
	rl.DisableCursor()

	rl.SetTargetFPS(60)

	// Criar mundo
	world := NewMinecraftWorld()
	defer world.Unload()

	// Criar player na posição inicial (y=30 para spawnar acima do terreno)
	player := NewMinecraftPlayer(rl.NewVector3(0, 30, 0))

	fmt.Println("=== Minecraft Clone ===")
	fmt.Println("Controls:")
	fmt.Println("  WASD - Move")
	fmt.Println("  Mouse - Look around")
	fmt.Println("  Space - Jump")
	fmt.Println("  Left Click - Break block")
	fmt.Println("  Right Click - Place block")
	fmt.Println("  1-5 or Mouse Wheel - Select block type")
	fmt.Println("  ESC - Exit")
	fmt.Println()

	frameCount := 0
	updateChunksInterval := 10 // Atualizar chunks a cada N frames

	// Main game loop
	for !rl.WindowShouldClose() {
		dt := rl.GetFrameTime()

		// Atualizar player (física, input, câmera)
		player.Update(world, dt)

		// Atualizar chunks periodicamente
		if frameCount%updateChunksInterval == 0 {
			world.UpdateChunks(player.Position)
		}
		frameCount++

		// Interação com blocos (adicionar/remover)
		player.HandleBlockInteraction(world)

		// Desenhar
		rl.BeginDrawing()
		rl.ClearBackground(rl.NewColor(135, 206, 235, 255)) // Sky blue

		rl.BeginMode3D(player.Camera)

		// Desenhar mundo
		world.Draw()

		// Desenhar player (cápsula)
		player.Draw()

		// Desenhar crosshair (mira) no bloco apontado
		hit, hitPos, _ := player.Raycast(world)
		if hit {
			// Desenhar wireframe ao redor do bloco apontado
			rl.DrawCubeWires(
				rl.NewVector3(hitPos.X+0.5, hitPos.Y+0.5, hitPos.Z+0.5),
				1.01, 1.01, 1.01,
				rl.Black,
			)
		}

		rl.EndMode3D()

		// UI
		rl.DrawText("Minecraft Clone", 10, 10, 20, rl.DarkGray)
		posText := fmt.Sprintf("Position: (%.1f, %.1f, %.1f)", player.Position.X, player.Position.Y, player.Position.Z)
		rl.DrawText(posText, 10, 40, 20, rl.DarkGray)

		selectedText := fmt.Sprintf("Selected: %s [%d]", player.GetSelectedBlockName(), player.SelectedBlock)
		rl.DrawText(selectedText, 10, 70, 20, rl.DarkGray)

		chunksText := fmt.Sprintf("Chunks loaded: %d", len(world.Chunks))
		rl.DrawText(chunksText, 10, 100, 20, rl.DarkGray)

		groundText := "On ground: No"
		if player.IsOnGround {
			groundText = "On ground: Yes"
		}
		rl.DrawText(groundText, 10, 130, 20, rl.DarkGray)

		// Desenhar preview do bloco selecionado
		blockColor := GetBlockColor(player.SelectedBlock)
		rl.DrawRectangle(screenWidth-60, 10, 50, 50, blockColor)
		rl.DrawRectangleLines(screenWidth-60, 10, 50, 50, rl.Black)

		// Crosshair
		crosshairSize := int32(10)
		centerX := screenWidth / 2
		centerY := screenHeight / 2
		rl.DrawLine(centerX-crosshairSize, centerY, centerX+crosshairSize, centerY, rl.White)
		rl.DrawLine(centerX, centerY-crosshairSize, centerX, centerY+crosshairSize, rl.White)

		rl.DrawFPS(10, screenHeight-30)

		rl.EndDrawing()
	}
}
