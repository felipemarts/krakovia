package game

import (
	"testing"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// ========== TESTES DA SOLUÇÃO DE FPS ==========

// TestChunkLoading_FixValidation verifica se o limite de mesh updates funciona
func TestChunkLoading_FixValidation(t *testing.T) {
	// Desabilitar upload para GPU durante o teste
	DisableGPUUploadForTesting = true
	defer func() { DisableGPUUploadForTesting = false }()

	world := NewWorld()
	world.ChunkManager.RenderDistance = 5

	player := NewPlayer(rl.NewVector3(16, 15, 16))
	player.FlyMode = true
	input := &SimulatedInput{}

	t.Log("=== Teste da Solução: Limite de Mesh Updates ===")
	t.Log("")

	// Fase 1: Carregar muitos chunks
	t.Log("Fase 1: Carregando área grande...")
	for i := 0; i < 300; i++ {
		dt := float32(1.0 / 60.0)
		world.Update(player.Position, dt)
		player.Update(dt, world, input)
	}

	chunksLoaded := world.GetLoadedChunksCount()
	t.Logf("Chunks carregados: %d", chunksLoaded)

	// Contar quantos chunks precisam de update
	pendingMeshes := 0
	for _, chunk := range world.ChunkManager.Chunks {
		if chunk.NeedUpdateMeshes {
			pendingMeshes++
		}
	}

	t.Logf("Chunks aguardando mesh update: %d", pendingMeshes)

	if pendingMeshes == 0 {
		t.Log("Nenhum chunk pendente, movendo jogador para forçar updates...")
		input.Forward = true
		for i := 0; i < 60; i++ {
			dt := float32(1.0 / 60.0)
			world.Update(player.Position, dt)
			player.Update(dt, world, input)
		}

		pendingMeshes = 0
		for _, chunk := range world.ChunkManager.Chunks {
			if chunk.NeedUpdateMeshes {
				pendingMeshes++
			}
		}
		t.Logf("Chunks aguardando mesh update após movimento: %d", pendingMeshes)
	}

	// Fase 2: Testar o UpdatePendingMeshes com limite
	t.Log("\nFase 2: Testando limite de mesh updates por frame...")

	const maxPerFrame = 3
	frame := 1
	totalUpdated := 0

	for pendingMeshes > 0 {
		updated := world.ChunkManager.UpdatePendingMeshes(maxPerFrame, world.DynamicAtlas)
		totalUpdated += updated

		if updated > maxPerFrame {
			t.Errorf("❌ FALHA: Frame %d atualizou %d meshes (limite: %d)", frame, updated, maxPerFrame)
		} else if updated > 0 {
			t.Logf("Frame %d: %d meshes atualizadas (dentro do limite)", frame, updated)
		}

		// Recontar pendentes
		pendingMeshes = 0
		for _, chunk := range world.ChunkManager.Chunks {
			if chunk.NeedUpdateMeshes {
				pendingMeshes++
			}
		}

		frame++

		// Segurança: parar após 1000 frames
		if frame > 1000 {
			t.Fatalf("Muitos frames necessários, algo está errado")
		}
	}

	t.Logf("\n=== Resultados ===")
	t.Logf("Total de meshes atualizadas: %d", totalUpdated)
	t.Logf("Frames necessários: %d", frame-1)
	t.Logf("Meshes por frame (média): %.2f", float64(totalUpdated)/float64(frame-1))

	if frame-1 > 0 {
		avgPerFrame := float64(totalUpdated) / float64(frame-1)
		if avgPerFrame > float64(maxPerFrame) {
			t.Errorf("❌ Média de meshes por frame (%.2f) excede o limite (%d)", avgPerFrame, maxPerFrame)
		} else {
			t.Logf("✓ Limite de %d meshes por frame está sendo respeitado", maxPerFrame)
		}
	}
}

// TestChunkLoading_FixStressTest testa a solução sob stress
func TestChunkLoading_FixStressTest(t *testing.T) {
	// Desabilitar upload para GPU durante o teste
	DisableGPUUploadForTesting = true
	defer func() { DisableGPUUploadForTesting = false }()

	world := NewWorld()
	world.ChunkManager.RenderDistance = 5
	world.ChunkManager.UnloadDistance = 7

	player := NewPlayer(rl.NewVector3(16, 15, 16))
	player.FlyMode = true
	input := &SimulatedInput{Forward: true}

	const totalFrames = 600
	maxMeshesInFrame := 0
	totalMeshUpdates := 0

	t.Log("=== Teste de Stress da Solução ===")

	for i := 0; i < totalFrames; i++ {
		dt := float32(1.0 / 60.0)

		world.Update(player.Position, dt)
		player.Update(dt, world, input)

		// Simular o que o Render() faz
		const maxMeshUpdatesPerFrame = 3
		meshesUpdated := world.ChunkManager.UpdatePendingMeshes(maxMeshUpdatesPerFrame, world.DynamicAtlas)

		if meshesUpdated > maxMeshesInFrame {
			maxMeshesInFrame = meshesUpdated
		}

		totalMeshUpdates += meshesUpdated

		if meshesUpdated > maxMeshUpdatesPerFrame {
			t.Errorf("❌ Frame %d: %d meshes atualizadas (limite: %d)",
				i, meshesUpdated, maxMeshUpdatesPerFrame)
		}
	}

	t.Logf("\n=== Resultados ===")
	t.Logf("Total de frames: %d", totalFrames)
	t.Logf("Total de mesh updates: %d", totalMeshUpdates)
	t.Logf("Máximo em 1 frame: %d", maxMeshesInFrame)
	t.Logf("Chunks finais: %d", world.GetLoadedChunksCount())

	if maxMeshesInFrame <= 3 {
		t.Log("✓ Solução funcionando: nunca excedeu 3 meshes por frame")
	} else {
		t.Errorf("❌ Solução falhou: máximo foi %d meshes em 1 frame", maxMeshesInFrame)
	}
}

// TestChunkLoading_CompareBeforeAfterFix compara antes/depois da solução
func TestChunkLoading_CompareBeforeAfterFix(t *testing.T) {
	// Desabilitar upload para GPU durante o teste
	DisableGPUUploadForTesting = true
	defer func() { DisableGPUUploadForTesting = false }()

	t.Log("=== Comparação: Antes vs Depois da Solução ===")
	t.Log("")

	world := NewWorld()
	world.ChunkManager.RenderDistance = 5

	player := NewPlayer(rl.NewVector3(16, 15, 16))
	player.FlyMode = true
	input := &SimulatedInput{}

	// Carregar área grande
	for i := 0; i < 300; i++ {
		dt := float32(1.0 / 60.0)
		world.Update(player.Position, dt)
		player.Update(dt, world, input)
	}

	// Contar pendentes
	totalPending := 0
	for _, chunk := range world.ChunkManager.Chunks {
		if chunk.NeedUpdateMeshes {
			totalPending++
		}
	}

	t.Log("CENÁRIO:")
	t.Logf("  Chunks carregados: %d", world.GetLoadedChunksCount())
	t.Logf("  Chunks com mesh update pendente: %d", totalPending)
	t.Log("")

	// Simular ANTES da solução (processar tudo de uma vez)
	t.Log("ANTES da solução (sem limite):")
	t.Logf("  Em 1 frame, processaria: %d mesh updates", totalPending)
	t.Logf("  Tempo estimado (5ms por upload): %dms", totalPending*5)
	if totalPending*5 > 16 {
		t.Logf("  ❌ FPS DROP: %dms > 16.6ms (60 FPS)", totalPending*5)
	}
	t.Log("")

	// Simular DEPOIS da solução (com limite de 3)
	const maxPerFrame = 3
	framesNeeded := (totalPending + maxPerFrame - 1) / maxPerFrame // Ceil division

	t.Log("DEPOIS da solução (limite de 3 por frame):")
	t.Logf("  Frames necessários: %d", framesNeeded)
	t.Logf("  Meshes por frame: máximo %d", maxPerFrame)
	t.Logf("  Tempo estimado por frame: %dms", maxPerFrame*5)
	if maxPerFrame*5 <= 16 {
		t.Logf("  ✓ SEM FPS DROP: %dms < 16.6ms (60 FPS)", maxPerFrame*5)
	}
	t.Log("")

	t.Log("RESULTADO:")
	improvement := float64(totalPending) / float64(maxPerFrame)
	t.Logf("  Distribuído ao longo de %d frames (vs 1 frame antes)", framesNeeded)
	t.Logf("  Melhoria: %.1fx menos trabalho por frame", improvement)
	t.Logf("  ✓ FPS mantido em 60 enquanto meshes são processadas gradualmente")
}

// TestChunkLoading_FixRealWorldScenario testa cenário real de jogo
func TestChunkLoading_FixRealWorldScenario(t *testing.T) {
	// Desabilitar upload para GPU durante o teste
	DisableGPUUploadForTesting = true
	defer func() { DisableGPUUploadForTesting = false }()

	world := NewWorld()
	world.ChunkManager.RenderDistance = 5

	player := NewPlayer(rl.NewVector3(16, 15, 16))
	player.FlyMode = true
	input := &SimulatedInput{Forward: true}

	const totalFrames = 1200 // 20 segundos
	const maxMeshUpdatesPerFrame = 3

	meshUpdatesPerFrame := make([]int, totalFrames)
	maxInAnyFrame := 0

	t.Log("=== Teste de Cenário Real (20 segundos de jogo) ===")

	for i := 0; i < totalFrames; i++ {
		dt := float32(1.0 / 60.0)

		world.Update(player.Position, dt)
		player.Update(dt, world, input)

		// Simular Render()
		meshesUpdated := world.ChunkManager.UpdatePendingMeshes(maxMeshUpdatesPerFrame, world.DynamicAtlas)
		meshUpdatesPerFrame[i] = meshesUpdated

		if meshesUpdated > maxInAnyFrame {
			maxInAnyFrame = meshesUpdated
		}

		// Log a cada 5 segundos
		if i > 0 && i%300 == 0 {
			pending := 0
			for _, chunk := range world.ChunkManager.Chunks {
				if chunk.NeedUpdateMeshes {
					pending++
				}
			}
			t.Logf("%.0fs: chunks=%d, pending meshes=%d",
				float64(i)/60.0, world.GetLoadedChunksCount(), pending)
		}
	}

	// Análise final
	framesWithUpdates := 0
	totalUpdates := 0
	for _, updates := range meshUpdatesPerFrame {
		if updates > 0 {
			framesWithUpdates++
			totalUpdates += updates
		}
	}

	t.Logf("\n=== Resultados de 20 Segundos ===")
	t.Logf("Total de frames: %d", totalFrames)
	t.Logf("Frames com mesh updates: %d", framesWithUpdates)
	t.Logf("Total de mesh updates: %d", totalUpdates)
	t.Logf("Máximo em 1 frame: %d", maxInAnyFrame)
	t.Logf("Chunks finais: %d", world.GetLoadedChunksCount())

	if maxInAnyFrame <= maxMeshUpdatesPerFrame {
		t.Logf("✓ SUCESSO: Limite de %d meshes/frame sempre respeitado", maxMeshUpdatesPerFrame)
	} else {
		t.Errorf("❌ FALHA: Excedeu limite (%d meshes em 1 frame)", maxInAnyFrame)
	}

	// Verificar chunks pendentes no final
	finalPending := 0
	for _, chunk := range world.ChunkManager.Chunks {
		if chunk.NeedUpdateMeshes {
			finalPending++
		}
	}

	if finalPending > 0 {
		t.Logf("⚠ Aviso: %d chunks ainda aguardando mesh update", finalPending)
	} else {
		t.Log("✓ Todas as meshes foram processadas")
	}
}
