package main

import (
	"fmt"

	rl "github.com/gen2brain/raylib-go/raylib"

	"krakovia/game"
)

func main() {
	rl.SetTraceLogLevel(rl.LogWarning)

	rl.InitWindow(game.ScreenWidth, game.ScreenHeight, "Krakovia")
	defer rl.CloseWindow()

	rl.SetTargetFPS(60)
	rl.DisableCursor()

	// Inicializar jogador
	player := game.NewPlayer(rl.NewVector3(16, 16, 16))

	// Inicializar mundo
	world := game.NewWorld()

	// Inicializar gráficos do mundo (depois de InitWindow)
	world.InitWorldGraphics()

	// Inicializar hotbar
	hotbar := game.NewBlockHotbar(world.CustomBlocks)

	// Inicializar interface unificada (inventário + editor de blocos)
	unifiedUI := game.NewUnifiedInventoryUI(world.CustomBlocks, hotbar)

	// Configurar callback para atualizar atlas quando um bloco for salvo
	unifiedUI.OnBlockSaved = func(block *game.CustomBlockDefinition) {
		// Registrar cada face separadamente no atlas
		for faceIdx := 0; faceIdx < 6; faceIdx++ {
			if block.FaceImages[faceIdx] != nil {
				faceBlockType := game.EncodeCustomBlockFace(block.ID, game.BlockFace(faceIdx))
				world.DynamicAtlas.AddTextureImage(faceBlockType, block.FaceImages[faceIdx])
			}
		}
		// Também registrar a textura principal (para o inventário)
		if block.FaceImages[game.FaceFront] != nil {
			world.DynamicAtlas.AddTextureImage(game.BlockType(block.ID), block.FaceImages[game.FaceFront])
		}

		// Rebuildar e fazer upload do atlas
		world.DynamicAtlas.RebuildAtlas()
		world.DynamicAtlas.UploadToGPU()

		// Marcar todos os chunks para reconstruir meshes
		world.ChunkManager.MarkAllChunksDirty()
	}

	// Input real do Raylib
	input := &game.RaylibInput{}

	// Desabilitar ESC para fechar janela (vamos controlar manualmente)
	rl.SetExitKey(0)

	// Loop principal do jogo
	for !rl.WindowShouldClose() {
		dt := rl.GetFrameTime()

		// ESC fecha o jogo apenas se a interface unificada não estiver aberta
		if rl.IsKeyPressed(rl.KeyEscape) {
			if unifiedUI.IsOpen {
				// A interface unificada trata ESC internamente
			} else {
				break
			}
		}

		// E: Abrir interface unificada (inventário + editor de blocos)
		// Só abre se não estiver aberta (fecha apenas com ESC)
		if rl.IsKeyPressed(rl.KeyE) && !unifiedUI.IsOpen {
			unifiedUI.Toggle()
		}

		// Atualizar interface unificada
		unifiedUI.Update(dt)

		// Atualizar mundo (carrega/descarrega chunks baseado na posição do jogador)
		world.Update(player.Position, dt)

		// Atualizar jogador e hotbar (apenas se a interface unificada não estiver aberta)
		if !unifiedUI.IsOpen {
			hotbar.Update()
			player.SelectedBlock = hotbar.GetSelectedBlock()
			player.Update(dt, world, input)
		}

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

		// Renderizar hotbar (se a interface unificada não estiver aberta)
		if !unifiedUI.IsOpen {
			hotbar.Render()
		}

		// Renderizar interface unificada (inventário + editor de blocos)
		unifiedUI.Render()

		rl.EndDrawing()
	}
}

// renderUI desenha a interface do usuário
func renderUI(player *game.Player, world *game.World) {
	rl.DrawText("WASD - Mover | Espaco - Pular | Mouse - Olhar | P - Fly Mode | K - Collision Body | O - NoClip", 10, 10, 20, rl.Black)
	rl.DrawText("Click Esquerdo - Remover | Click Direito - Colocar | V - Alternar Camera", 10, 35, 20, rl.Black)
	rl.DrawText("E - Inventario/Editor de Blocos", 10, 60, 20, rl.DarkGray)

	yOffset := int32(85)

	// Mostrar status do modo fly
	if player.FlyMode {
		rl.DrawText("FLY MODE ATIVO | Shift - Subir | Ctrl - Descer", 10, yOffset, 20, rl.Red)
		yOffset += 25
	}

	// Mostrar status do NoClip
	if player.NoClip {
		rl.DrawText("NOCLIP ATIVO - Colisão desabilitada", 10, yOffset, 20, rl.Orange)
		yOffset += 25
	}

	rl.DrawText(fmt.Sprintf("Posição: (%.1f, %.1f, %.1f)", player.Position.X, player.Position.Y, player.Position.Z), 10, yOffset, 20, rl.Black)
	yOffset += 25

	// Mostrar chunk atual do jogador
	playerChunk := game.GetChunkCoordFromFloat(player.Position.X, player.Position.Y, player.Position.Z)
	rl.DrawText(fmt.Sprintf("Chunk: (%d, %d, %d)", playerChunk.X, playerChunk.Y, playerChunk.Z), 10, yOffset, 20, rl.Black)
	yOffset += 25

	totalBlocks := world.GetTotalBlocks()
	chunksLoaded := world.GetLoadedChunksCount()
	rl.DrawText(fmt.Sprintf("Blocos: %d | Chunks: %d", totalBlocks, chunksLoaded), 10, yOffset, 20, rl.Black)
	yOffset += 25

	// Mostrar animação atual do modelo
	animName, animIndex, animCount := player.GetAnimationDisplayInfo()
	if animCount > 0 {
		rl.DrawText(fmt.Sprintf("Animação: %s (%d/%d)", animName, animIndex, animCount), 10, yOffset, 20, rl.Purple)
	}

	rl.DrawText(fmt.Sprintf("FPS: %d", rl.GetFPS()), 10, game.ScreenHeight-30, 20, rl.Green)

	// Crosshair
	rl.DrawLine(game.ScreenWidth/2-10, game.ScreenHeight/2, game.ScreenWidth/2+10, game.ScreenHeight/2, rl.White)
	rl.DrawLine(game.ScreenWidth/2, game.ScreenHeight/2-10, game.ScreenWidth/2, game.ScreenHeight/2+10, rl.White)
}
