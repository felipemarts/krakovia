package main

import (
	"fmt"
	"testing"
)

func TestChunkBlockPositions(t *testing.T) {
	// Testar posições de blocos em chunks negativos
	tests := []struct {
		name string
		chunkX, chunkZ int32
		expectedFirstBlockX, expectedFirstBlockZ float32
	}{
		{"Chunk (0,0)", 0, 0, 0.5, 0.5},
		{"Chunk (1,0)", 1, 0, 32.5, 0.5},
		{"Chunk (-1,0)", -1, 0, -31.5, 0.5},
		{"Chunk (-2,0)", -2, 0, -63.5, 0.5},
		{"Chunk (0,1)", 0, 1, 0.5, 32.5},
		{"Chunk (0,-1)", 0, -1, 0.5, -31.5},
		{"Chunk (-1,-1)", -1, -1, -31.5, -31.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunk := NewChunk(tt.chunkX, 0, tt.chunkZ)

			// Adicionar um bloco manualmente no canto (0,10,0) do chunk
			chunk.Blocks[0][10][0] = BlockGrass
			chunk.UpdateMeshes()

			if len(chunk.GrassTransforms) != 1 {
				t.Errorf("Esperava 1 transform, obteve %d", len(chunk.GrassTransforms))
				return
			}

			// Obter a transformação
			transform := chunk.GrassTransforms[0]

			// Extrair a posição da matriz (elementos m12, m13, m14)
			posX := transform.M12
			posZ := transform.M14

			// Verificar se a posição está correta
			if posX != tt.expectedFirstBlockX || posZ != tt.expectedFirstBlockZ {
				t.Errorf("Chunk (%d,0,%d): Esperava bloco em (%f, %f), obteve (%f, %f)",
					tt.chunkX, tt.chunkZ,
					tt.expectedFirstBlockX, tt.expectedFirstBlockZ,
					posX, posZ)
			}
		})
	}
}

func TestChunkWorldPosition(t *testing.T) {
	// Testar cálculo de posição mundial
	tests := []struct {
		chunkX, chunkZ int32
		localX, localZ int32
		expectedWorldX, expectedWorldZ float32
	}{
		// Chunk (0,0)
		{0, 0, 0, 0, 0.5, 0.5},
		{0, 0, 31, 31, 31.5, 31.5},

		// Chunk (-1,0)
		{-1, 0, 0, 0, -31.5, 0.5},
		{-1, 0, 31, 0, -0.5, 0.5},

		// Chunk (-2,-1)
		{-2, -1, 0, 0, -63.5, -31.5},
		{-2, -1, 31, 31, -32.5, -0.5},
	}

	for _, tt := range tests {
		name := fmt.Sprintf("Chunk(%d,%d)_Local(%d,%d)", tt.chunkX, tt.chunkZ, tt.localX, tt.localZ)
		t.Run(name, func(t *testing.T) {
			// Calcular posição mundial
			worldX := tt.chunkX * ChunkSize + tt.localX
			worldZ := tt.chunkZ * ChunkSize + tt.localZ

			// Adicionar 0.5 para centralizar
			actualX := float32(worldX) + 0.5
			actualZ := float32(worldZ) + 0.5

			if actualX != tt.expectedWorldX || actualZ != tt.expectedWorldZ {
				t.Errorf("Esperava (%f, %f), obteve (%f, %f)",
					tt.expectedWorldX, tt.expectedWorldZ,
					actualX, actualZ)
			}
		})
	}
}