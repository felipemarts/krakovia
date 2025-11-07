package main

import (
	"testing"
)

// TestBlockOcclusion testa a detecção de blocos ocultos em diferentes cenários
func TestBlockOcclusion(t *testing.T) {
	tests := []struct {
		name           string
		chunkCoords    []ChunkCoord // Coordenadas dos chunks a criar
		blockPos       struct{ x, y, z int32 } // Posição mundial do bloco a testar
		shouldBeHidden bool
		description    string
	}{
		{
			name:        "Bloco isolado no ar - deve ser visível",
			chunkCoords: []ChunkCoord{{X: 0, Y: 0, Z: 0}},
			blockPos:    struct{ x, y, z int32 }{x: 16, y: 16, z: 16}, // Centro do chunk
			shouldBeHidden: false,
			description:    "Um único bloco sem vizinhos deve ser visível",
		},
		{
			name:        "Bloco cercado dentro do mesmo chunk - deve ser oculto",
			chunkCoords: []ChunkCoord{{X: 0, Y: 0, Z: 0}},
			blockPos:    struct{ x, y, z int32 }{x: 16, y: 16, z: 16},
			shouldBeHidden: true,
			description:    "Bloco completamente cercado deve ser oculto",
		},
		{
			name: "Bloco na borda do chunk com vizinho no próximo chunk (X+) - coordenadas positivas",
			chunkCoords: []ChunkCoord{
				{X: 0, Y: 0, Z: 0},
				{X: 1, Y: 0, Z: 0},
			},
			blockPos:       struct{ x, y, z int32 }{x: 31, y: 16, z: 16}, // Última posição X do chunk 0
			shouldBeHidden: true,
			description:    "Bloco na borda deve verificar chunk vizinho",
		},
		{
			name: "Bloco na borda do chunk sem vizinho no próximo chunk (X+) - coordenadas positivas",
			chunkCoords: []ChunkCoord{
				{X: 0, Y: 0, Z: 0},
			},
			blockPos:       struct{ x, y, z int32 }{x: 31, y: 16, z: 16},
			shouldBeHidden: false,
			description:    "Bloco na borda sem chunk vizinho deve ser visível",
		},
		{
			name: "Bloco na borda do chunk (X-) - coordenadas negativas",
			chunkCoords: []ChunkCoord{
				{X: -1, Y: 0, Z: 0},
				{X: -2, Y: 0, Z: 0},
			},
			blockPos:       struct{ x, y, z int32 }{x: -32, y: 16, z: 16}, // Primeira posição X do chunk -1
			shouldBeHidden: true,
			description:    "Bloco na borda em coordenadas negativas deve verificar chunk vizinho",
		},
		{
			name: "Bloco na borda do chunk (Y+) - verificar chunk acima",
			chunkCoords: []ChunkCoord{
				{X: 0, Y: 0, Z: 0},
				{X: 0, Y: 1, Z: 0},
			},
			blockPos:       struct{ x, y, z int32 }{x: 16, y: 31, z: 16},
			shouldBeHidden: true,
			description:    "Bloco na borda superior deve verificar chunk acima",
		},
		{
			name: "Bloco na borda do chunk (Y-) - verificar chunk abaixo",
			chunkCoords: []ChunkCoord{
				{X: 0, Y: 0, Z: 0},
				{X: 0, Y: -1, Z: 0},
			},
			blockPos:       struct{ x, y, z int32 }{x: 16, y: 0, z: 16},
			shouldBeHidden: true,
			description:    "Bloco na borda inferior deve verificar chunk abaixo",
		},
		{
			name: "Bloco na borda do chunk (Z+) - coordenadas positivas",
			chunkCoords: []ChunkCoord{
				{X: 0, Y: 0, Z: 0},
				{X: 0, Y: 0, Z: 1},
			},
			blockPos:       struct{ x, y, z int32 }{x: 16, y: 16, z: 31},
			shouldBeHidden: true,
			description:    "Bloco na borda Z+ deve verificar chunk vizinho",
		},
		{
			name: "Bloco na borda do chunk (Z-) - coordenadas negativas",
			chunkCoords: []ChunkCoord{
				{X: 0, Y: 0, Z: -1},
				{X: 0, Y: 0, Z: -2},
			},
			blockPos:       struct{ x, y, z int32 }{x: 16, y: 16, z: -32},
			shouldBeHidden: true,
			description:    "Bloco na borda Z- em coordenadas negativas deve verificar chunk vizinho",
		},
		{
			name: "Bloco no meio de chunks negativos",
			chunkCoords: []ChunkCoord{
				{X: -1, Y: -1, Z: -1},
			},
			blockPos:       struct{ x, y, z int32 }{x: -16, y: -16, z: -16},
			shouldBeHidden: true,
			description:    "Bloco cercado em coordenadas todas negativas",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Criar ChunkManager
			cm := NewChunkManager(5)

			// Criar chunks necessários
			for _, coord := range tt.chunkCoords {
				chunk := NewChunk(coord.X, coord.Y, coord.Z)

				// Preencher o chunk inteiro com blocos de pedra
				for x := int32(0); x < ChunkSize; x++ {
					for y := int32(0); y < ChunkHeight; y++ {
						for z := int32(0); z < ChunkSize; z++ {
							chunk.Blocks[x][y][z] = BlockStone
						}
					}
				}

				cm.Chunks[coord.Key()] = chunk
			}

			// Para o primeiro teste (bloco isolado), limpar tudo ao redor
			if tt.name == "Bloco isolado no ar - deve ser visível" {
				chunk := cm.Chunks[ChunkCoord{X: 0, Y: 0, Z: 0}.Key()]
				// Limpar tudo
				for x := int32(0); x < ChunkSize; x++ {
					for y := int32(0); y < ChunkHeight; y++ {
						for z := int32(0); z < ChunkSize; z++ {
							chunk.Blocks[x][y][z] = BlockAir
						}
					}
				}
				// Colocar apenas um bloco no centro
				chunk.Blocks[16][16][16] = BlockStone
			}

			// Para o teste "Bloco na borda sem vizinho", remover o bloco vizinho no ar
			if tt.name == "Bloco na borda do chunk sem vizinho no próximo chunk (X+) - coordenadas positivas" {
				chunk := cm.Chunks[ChunkCoord{X: 0, Y: 0, Z: 0}.Key()]
				// Garantir que há um bloco na borda
				chunk.Blocks[31][16][16] = BlockStone
				// Mas todos os vizinhos internos também são pedra
				chunk.Blocks[30][16][16] = BlockStone // Vizinho X-
				chunk.Blocks[31][17][16] = BlockStone // Vizinho Y+
				chunk.Blocks[31][15][16] = BlockStone // Vizinho Y-
				chunk.Blocks[31][16][17] = BlockStone // Vizinho Z+
				chunk.Blocks[31][16][15] = BlockStone // Vizinho Z-
				// O vizinho X+ está no próximo chunk que não existe
			}

			// Testar se o bloco está oculto
			isHidden := cm.IsBlockHidden(tt.blockPos.x, tt.blockPos.y, tt.blockPos.z)

			// Debug: mostrar informações sobre os vizinhos
			t.Logf("Testando posição (%d, %d, %d)", tt.blockPos.x, tt.blockPos.y, tt.blockPos.z)
			t.Logf("Chunk esperado: %v", GetChunkCoord(tt.blockPos.x, tt.blockPos.y, tt.blockPos.z))

			// Verificar cada vizinho
			directions := []struct{ name string; dx, dy, dz int32 }{
				{"X+", 1, 0, 0},
				{"X-", -1, 0, 0},
				{"Y+", 0, 1, 0},
				{"Y-", 0, -1, 0},
				{"Z+", 0, 0, 1},
				{"Z-", 0, 0, -1},
			}

			for _, dir := range directions {
				nx, ny, nz := tt.blockPos.x+dir.dx, tt.blockPos.y+dir.dy, tt.blockPos.z+dir.dz
				neighborBlock := cm.GetBlock(nx, ny, nz)
				neighborChunk := GetChunkCoord(nx, ny, nz)
				t.Logf("  Vizinho %s: pos=(%d,%d,%d) chunk=%v tipo=%v",
					dir.name, nx, ny, nz, neighborChunk, neighborBlock)
			}

			if isHidden != tt.shouldBeHidden {
				t.Errorf("%s\nEsperado: hidden=%v, Obtido: hidden=%v\n%s",
					tt.name, tt.shouldBeHidden, isHidden, tt.description)
			}
		})
	}
}

// TestChunkCoordCalculation testa o cálculo de coordenadas de chunk
func TestChunkCoordCalculation(t *testing.T) {
	tests := []struct {
		worldX, worldY, worldZ int32
		expectedChunk          ChunkCoord
		description            string
	}{
		{0, 0, 0, ChunkCoord{0, 0, 0}, "Origem do mundo"},
		{16, 16, 16, ChunkCoord{0, 0, 0}, "Centro do primeiro chunk"},
		{31, 31, 31, ChunkCoord{0, 0, 0}, "Última posição do primeiro chunk"},
		{32, 32, 32, ChunkCoord{1, 1, 1}, "Primeira posição do próximo chunk"},
		{-1, -1, -1, ChunkCoord{-1, -1, -1}, "Primeira coordenada negativa"},
		{-16, -16, -16, ChunkCoord{-1, -1, -1}, "Centro do chunk negativo"},
		{-32, -32, -32, ChunkCoord{-1, -1, -1}, "Primeira posição do chunk (-1,-1,-1)"},
		{-33, -33, -33, ChunkCoord{-2, -2, -2}, "Chunk (-2,-2,-2)"},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			chunk := GetChunkCoord(tt.worldX, tt.worldY, tt.worldZ)
			if chunk != tt.expectedChunk {
				t.Errorf("GetChunkCoord(%d, %d, %d) = %v, esperado %v (%s)",
					tt.worldX, tt.worldY, tt.worldZ, chunk, tt.expectedChunk, tt.description)
			}
		})
	}
}

