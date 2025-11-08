package main

import (
	"fmt"
	"testing"
)

func TestChunkKey(t *testing.T) {
	// Testar se as chaves são únicas
	keys := make(map[int64]ChunkCoord)

	coords := []ChunkCoord{
		{X: 0, Y: 0, Z: 0},
		{X: 1, Y: 0, Z: 0},
		{X: -1, Y: 0, Z: 0},
		{X: 0, Y: 0, Z: 1},
		{X: 0, Y: 0, Z: -1},
		{X: -2, Y: 0, Z: 0},
		{X: 2, Y: 0, Z: 0},
		{X: -1, Y: 0, Z: -1},
		{X: 1, Y: 0, Z: 1},
		// Testes com Y negativo
		{X: 0, Y: -1, Z: 0},
		{X: 0, Y: -2, Z: 0},
		{X: -1, Y: -1, Z: -1},
		{X: 1, Y: -1, Z: 1},
		{X: -2, Y: -2, Z: -2},
		// Testes com Y positivo
		{X: 0, Y: 1, Z: 0},
		{X: 0, Y: 2, Z: 0},
		{X: 1, Y: 1, Z: 1},
	}

	for _, coord := range coords {
		key := coord.Key()
		if existing, exists := keys[key]; exists {
			t.Errorf("Colisão de chave! Chunk %v e %v têm a mesma chave: %d",
				existing, coord, key)
		}
		keys[key] = coord

		// Debug: mostrar a chave binária
		fmt.Printf("Chunk(%d,%d,%d) -> Key: %d (0x%016X)\n",
			coord.X, coord.Y, coord.Z, key, uint64(key))
	}
}
