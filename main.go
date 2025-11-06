package main

import (
	"fmt"

	rl "github.com/gen2brain/raylib-go/raylib"
)

func main() {
	rl.InitWindow(screenWidth, screenHeight, "Krakovia - Minecraft em Go")
	defer rl.CloseWindow()

	rl.SetTargetFPS(60)
	rl.DisableCursor()

	// Inicializar jogador
	player := NewPlayer(rl.NewVector3(8, 100, 8))

	// Inicializar mundo
	world := NewWorld()

	// Inicializar gráficos do mundo (depois de InitWindow)
	world.InitWorldGraphics()

	// Input real do Raylib
	input := &RaylibInput{}

	// Loop principal do jogo
	for !rl.WindowShouldClose() {
		dt := rl.GetFrameTime()

		// Atualizar mundo (carrega/descarrega chunks baseado na posição do jogador)
		world.Update(player.Position, dt)

		// Atualizar jogador
		player.Update(dt, world, input)

		// Renderizar
		rl.BeginDrawing()
		rl.ClearBackground(rl.SkyBlue)

		rl.BeginMode3D(player.Camera)

		// Renderizar mundo
		world.Render(player.Position)

		// Renderizar jogador como cápsula
		player.RenderPlayer()

		// Desenhar highlight para indicar onde o bloco será removido
		if player.LookingAtBlock {
			// Centralizar o wireframe no meio do bloco
			centerPos := rl.NewVector3(
				player.TargetBlock.X+0.5,
				player.TargetBlock.Y+0.5,
				player.TargetBlock.Z+0.5,
			)
			rl.DrawCubeWiresV(centerPos, rl.NewVector3(1.01, 1.01, 1.01), rl.Red)
		}

		rl.EndMode3D()

		// UI
		renderUI(player, world)

		rl.EndDrawing()
	}
}

// renderUI desenha a interface do usuário
func renderUI(player *Player, world *World) {
	rl.DrawText("WASD - Mover | Espaço - Pular | Mouse - Olhar", 10, 10, 20, rl.Black)
	rl.DrawText("Click Esquerdo - Remover | Click Direito - Colocar", 10, 35, 20, rl.Black)
	rl.DrawText(fmt.Sprintf("Posição: (%.1f, %.1f, %.1f)", player.Position.X, player.Position.Y, player.Position.Z), 10, 60, 20, rl.Black)

	totalBlocks := world.GetTotalBlocks()
	chunksLoaded := world.GetLoadedChunksCount()
	rl.DrawText(fmt.Sprintf("Blocos: %d | Chunks: %d", totalBlocks, chunksLoaded), 10, 85, 20, rl.Black)
	rl.DrawText(fmt.Sprintf("FPS: %d", rl.GetFPS()), 10, screenHeight-30, 20, rl.Green)

	// Crosshair
	rl.DrawLine(screenWidth/2-10, screenHeight/2, screenWidth/2+10, screenHeight/2, rl.White)
	rl.DrawLine(screenWidth/2, screenHeight/2-10, screenWidth/2, screenHeight/2+10, rl.White)
}
