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
	player := NewPlayer(rl.NewVector3(16, 16, 16))

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
	rl.DrawText("WASD - Mover | Espaço - Pular | Mouse - Olhar | P - Fly Mode", 10, 10, 20, rl.Black)
	rl.DrawText("Click Esquerdo - Remover | Click Direito - Colocar", 10, 35, 20, rl.Black)

	yOffset := int32(60)

	// Mostrar status do modo fly
	if player.FlyMode {
		rl.DrawText("FLY MODE ATIVO | Shift - Subir | Ctrl - Descer", 10, yOffset, 20, rl.Red)
		yOffset += 25
	}

	rl.DrawText(fmt.Sprintf("Posição: (%.1f, %.1f, %.1f)", player.Position.X, player.Position.Y, player.Position.Z), 10, yOffset, 20, rl.Black)
	yOffset += 25

	// Mostrar chunk atual do jogador
	playerChunk := GetChunkCoordFromFloat(player.Position.X, player.Position.Y, player.Position.Z)
	rl.DrawText(fmt.Sprintf("Chunk: (%d, %d, %d)", playerChunk.X, playerChunk.Y, playerChunk.Z), 10, yOffset, 20, rl.Black)
	yOffset += 25

	totalBlocks := world.GetTotalBlocks()
	chunksLoaded := world.GetLoadedChunksCount()
	rl.DrawText(fmt.Sprintf("Blocos: %d | Chunks: %d", totalBlocks, chunksLoaded), 10, yOffset, 20, rl.Black)
	rl.DrawText(fmt.Sprintf("FPS: %d", rl.GetFPS()), 10, screenHeight-30, 20, rl.Green)

	// Crosshair
	rl.DrawLine(screenWidth/2-10, screenHeight/2, screenWidth/2+10, screenHeight/2, rl.White)
	rl.DrawLine(screenWidth/2, screenHeight/2-10, screenWidth/2, screenHeight/2+10, rl.White)
}
