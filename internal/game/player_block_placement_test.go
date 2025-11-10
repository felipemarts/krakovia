package game

import (
	"testing"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// TestPlayerCannotPlaceBlockInOwnPosition verifica que o jogador não pode
// colocar blocos na própria posição (prevenindo que se enterre)
func TestPlayerCannotPlaceBlockInOwnPosition(t *testing.T) {
	tests := []struct {
		name           string
		playerPos      rl.Vector3
		blockPos       rl.Vector3
		shouldCollide  bool
		description    string
	}{
		{
			name:          "Bloco no mesmo local do jogador",
			playerPos:     rl.NewVector3(5.5, 10.0, 5.5),
			blockPos:      rl.NewVector3(5.5, 10.0, 5.5),
			shouldCollide: true,
			description:   "Bloco exatamente onde o jogador está deve colidir",
		},
		{
			name:          "Bloco na altura da cabeça do jogador",
			playerPos:     rl.NewVector3(5.5, 10.0, 5.5),
			blockPos:      rl.NewVector3(5.5, 11.0, 5.5),
			shouldCollide: true,
			description:   "Bloco na altura da cabeça (dentro da altura do jogador) deve colidir",
		},
		{
			name:          "Bloco logo acima do jogador",
			playerPos:     rl.NewVector3(5.5, 10.0, 5.5),
			blockPos:      rl.NewVector3(5.5, 12.0, 5.5),
			shouldCollide: false,
			description:   "Bloco acima da altura do jogador (1.8) não deve colidir",
		},
		{
			name:          "Bloco logo abaixo do jogador",
			playerPos:     rl.NewVector3(5.5, 10.0, 5.5),
			blockPos:      rl.NewVector3(5.5, 9.0, 5.5),
			shouldCollide: false,
			description:   "Bloco abaixo dos pés do jogador não deve colidir",
		},
		{
			name:          "Bloco adjacente horizontal",
			playerPos:     rl.NewVector3(5.5, 10.0, 5.5),
			blockPos:      rl.NewVector3(6.5, 10.0, 5.5),
			shouldCollide: false,
			description:   "Bloco a 1 bloco de distância horizontal não deve colidir",
		},
		{
			name:          "Bloco próximo mas dentro do raio",
			playerPos:     rl.NewVector3(5.5, 10.0, 5.5),
			blockPos:      rl.NewVector3(5.9, 10.0, 5.5),
			shouldCollide: true,
			description:   "Bloco próximo dentro do raio do jogador (0.3) deve colidir",
		},
		{
			name:          "Bloco na borda do cilindro",
			playerPos:     rl.NewVector3(5.0, 10.0, 5.0),
			blockPos:      rl.NewVector3(5.5, 10.5, 5.0),
			shouldCollide: true,
			description:   "Bloco na borda do cilindro do jogador deve colidir",
		},
		{
			name:          "Bloco longe do jogador",
			playerPos:     rl.NewVector3(5.5, 10.0, 5.5),
			blockPos:      rl.NewVector3(10.5, 10.0, 10.5),
			shouldCollide: false,
			description:   "Bloco longe do jogador não deve colidir",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			player := NewPlayer(tt.playerPos)

			result := player.wouldBlockCollideWithPlayer(tt.blockPos)

			if result != tt.shouldCollide {
				t.Errorf("%s: esperado colisão=%v, obteve=%v (playerPos=%v, blockPos=%v, raio=%.2f, altura=%.2f)",
					tt.description, tt.shouldCollide, result, tt.playerPos, tt.blockPos, player.Radius, player.Height)
			}
		})
	}
}

// TestPlayerBlockPlacementCollisionEdgeCases testa casos extremos de colisão
func TestPlayerBlockPlacementCollisionEdgeCases(t *testing.T) {
	t.Run("Jogador parcialmente dentro do bloco horizontalmente", func(t *testing.T) {
		player := NewPlayer(rl.NewVector3(5.4, 10.0, 5.4))
		blockPos := rl.NewVector3(5.5, 10.5, 5.5) // Centro do bloco próximo

		if !player.wouldBlockCollideWithPlayer(blockPos) {
			t.Error("Jogador parcialmente dentro do bloco deveria colidir")
		}
	})

	t.Run("Jogador com pés no limite superior do bloco", func(t *testing.T) {
		player := NewPlayer(rl.NewVector3(5.5, 11.0, 5.5))
		blockPos := rl.NewVector3(5.5, 10.0, 5.5) // Bloco abaixo

		// Os pés do jogador estão em y=11.0
		// O topo do bloco está em y=11.0
		// Não deve colidir (sem sobreposição)
		if player.wouldBlockCollideWithPlayer(blockPos) {
			t.Error("Jogador com pés exatamente no topo do bloco não deveria colidir")
		}
	})

	t.Run("Jogador com cabeça no limite inferior do bloco", func(t *testing.T) {
		player := NewPlayer(rl.NewVector3(5.5, 10.0, 5.5))
		// Cabeça do jogador está em y=10.0 + 1.8 = 11.8
		blockPos := rl.NewVector3(5.5, 11.8, 5.5) // Base do bloco em 11.8

		// Não deve colidir (sem sobreposição)
		if player.wouldBlockCollideWithPlayer(blockPos) {
			t.Error("Jogador com cabeça exatamente na base do bloco não deveria colidir")
		}
	})

	t.Run("Bloco diagonal próximo do jogador", func(t *testing.T) {
		player := NewPlayer(rl.NewVector3(5.5, 10.0, 5.5))
		// Bloco diagonal, dentro do raio
		blockPos := rl.NewVector3(5.7, 10.5, 5.7)

		// Distância horizontal: sqrt((5.7-5.5)^2 + (5.7-5.5)^2) = sqrt(0.08) ≈ 0.283
		// Raio jogador (0.3) + raio bloco (0.5) = 0.8
		// 0.283 < 0.8, então deve colidir
		if !player.wouldBlockCollideWithPlayer(blockPos) {
			t.Error("Bloco diagonal próximo deveria colidir")
		}
	})

	t.Run("Bloco diagonal no limite do raio", func(t *testing.T) {
		player := NewPlayer(rl.NewVector3(5.0, 10.0, 5.0))
		// Distância exata para estar fora do raio
		// raio_total = 0.3 + 0.5 = 0.8
		// Colocar em (5.6, 10.5, 5.6): distância = sqrt(0.36+0.36) = 0.849 > 0.8
		blockPos := rl.NewVector3(5.6, 10.5, 5.6)

		if player.wouldBlockCollideWithPlayer(blockPos) {
			t.Error("Bloco fora do raio não deveria colidir")
		}
	})
}

// TestPlayerPlacementIntegration testa a integração da verificação de colisão
// no processo de colocação de blocos
func TestPlayerPlacementIntegration(t *testing.T) {
	t.Run("Não deve colocar bloco na posição do jogador", func(t *testing.T) {
		// Simular cenário real onde jogador tenta colocar bloco
		player := NewPlayer(rl.NewVector3(5.5, 10.0, 5.5))

		// Simular que o jogador está olhando para um bloco e quer colocar
		// um novo bloco na própria posição (centro do bloco em 5.5, 10.0, 5.5)
		placePos := rl.NewVector3(5.5, 10.0, 5.5)

		// Verificação que deve impedir a colocação
		shouldNotPlace := player.wouldBlockCollideWithPlayer(placePos)

		if !shouldNotPlace {
			t.Error("Deveria detectar colisão e impedir colocação do bloco na posição do jogador")
		}
	})

	t.Run("Deve permitir colocar bloco longe do jogador", func(t *testing.T) {
		player := NewPlayer(rl.NewVector3(5.5, 10.0, 5.5))

		// Bloco a 2 blocos de distância (centro do bloco em 7.5, 10.0, 5.5)
		placePos := rl.NewVector3(7.5, 10.0, 5.5)

		// Não deve colidir
		shouldNotPlace := player.wouldBlockCollideWithPlayer(placePos)

		if shouldNotPlace {
			t.Error("Não deveria detectar colisão para bloco longe do jogador")
		}
	})

	t.Run("Deve permitir colocar bloco no chão abaixo do jogador", func(t *testing.T) {
		player := NewPlayer(rl.NewVector3(5.5, 10.0, 5.5))

		// Bloco exatamente abaixo dos pés (centro do bloco em 5.5, 9.0, 5.5)
		// Bloco ocupa y=9.0 a y=10.0, jogador começa em y=10.0
		placePos := rl.NewVector3(5.5, 9.0, 5.5)

		// Não deve colidir (bloco está abaixo dos pés)
		shouldNotPlace := player.wouldBlockCollideWithPlayer(placePos)

		if shouldNotPlace {
			t.Error("Deveria permitir colocar bloco abaixo dos pés do jogador")
		}
	})
}

// BenchmarkBlockCollisionCheck benchmark da verificação de colisão
func BenchmarkBlockCollisionCheck(b *testing.B) {
	player := NewPlayer(rl.NewVector3(5.5, 10.0, 5.5))
	blockPos := rl.NewVector3(5.5, 10.5, 5.5)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		player.wouldBlockCollideWithPlayer(blockPos)
	}
}
