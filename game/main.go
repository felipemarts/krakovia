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

	// Inicializar editor de blocos
	blockEditor := game.NewBlockEditorUI(world.CustomBlocks)

	// Configurar callback para atualizar atlas quando um bloco for salvo
	blockEditor.OnBlockSaved = func(block *game.CustomBlockDefinition) {
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

	// Inicializar hotbar
	hotbar := game.NewBlockHotbar(world.CustomBlocks)

	// Inicializar inventário
	inventory := game.NewInventoryUI(world.CustomBlocks, hotbar)

	// Input real do Raylib
	input := &game.RaylibInput{}

	// Desabilitar ESC para fechar janela (vamos controlar manualmente)
	rl.SetExitKey(0)

	// Loop principal do jogo
	for !rl.WindowShouldClose() {
		dt := rl.GetFrameTime()

		// ESC fecha o jogo apenas se o editor e inventário não estiverem abertos
		if rl.IsKeyPressed(rl.KeyEscape) {
			if inventory.IsOpen {
				inventory.Close()
			} else if !blockEditor.IsOpen() {
				break
			}
		}

		// E: Abrir/fechar inventário
		if rl.IsKeyPressed(rl.KeyE) && !blockEditor.IsOpen() {
			inventory.Toggle()
		}

		// Comandos de debug para atlas dinâmico
		if rl.IsKeyPressed(rl.KeyF1) {
			// F1: Imprimir estatísticas do atlas
			if world.DynamicAtlas != nil {
				world.DynamicAtlas.PrintStats()
			}
		}

		if rl.IsKeyPressed(rl.KeyF2) {
			// F2: Salvar atlas atual em arquivo
			if world.DynamicAtlas != nil {
				err := world.DynamicAtlas.SaveAtlasDebug("debug_atlas.png")
				if err != nil {
					fmt.Printf("Erro ao salvar atlas: %v\n", err)
				} else {
					fmt.Println("Atlas salvo em debug_atlas.png")
				}
			}
		}

		if rl.IsKeyPressed(rl.KeyF3) {
			// F3: Imprimir blocos visíveis
			if world.VisibleBlocks != nil {
				blocks := world.VisibleBlocks.GetRequiredBlocks()
				fmt.Printf("Blocos visíveis: %d tipos\n", len(blocks))
				for _, bt := range blocks {
					fmt.Printf("  - BlockType %d (count: %d)\n",
						bt, world.VisibleBlocks.BlockUsageCount[bt])
				}
			}
		}

		// B: Abrir/fechar editor de blocos
		if rl.IsKeyPressed(rl.KeyB) {
			blockEditor.Toggle()
		}

		// Atualizar editor de blocos
		blockEditor.Update(dt)

		// Atualizar inventário
		inventory.Update()

		// Atualizar mundo (carrega/descarrega chunks baseado na posição do jogador)
		world.Update(player.Position, dt)

		// Atualizar jogador e hotbar (apenas se o editor e inventário não estiverem abertos)
		if !blockEditor.IsOpen() && !inventory.IsOpen {
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

		// Renderizar hotbar (se o editor e inventário não estiverem abertos)
		if !blockEditor.IsOpen() && !inventory.IsOpen {
			hotbar.Render()
		}

		// Renderizar inventário
		inventory.Render()

		// Renderizar editor de blocos (por cima de tudo)
		blockEditor.Render()

		rl.EndDrawing()
	}
}

// renderUI desenha a interface do usuário
func renderUI(player *game.Player, world *game.World) {
	rl.DrawText("WASD - Mover | Espaço - Pular | Mouse - Olhar | P - Fly Mode | K - Collision Body | O - NoClip", 10, 10, 20, rl.Black)
	rl.DrawText("Click Esquerdo - Remover | Click Direito - Colocar | V - Alternar Câmera", 10, 35, 20, rl.Black)
	rl.DrawText("B - Editor de Blocos | E - Inventário | F1/F2/F3 - Debug", 10, 60, 20, rl.DarkGray)

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
