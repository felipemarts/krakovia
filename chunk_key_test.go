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

func TestChunkKeyMask(t *testing.T) {
	// Testar se os valores negativos causam problemas com máscaras
	coord1 := ChunkCoord{X: -1, Y: 0, Z: 0}
	coord2 := ChunkCoord{X: 1048575, Y: 0, Z: 0} // 2^20 - 1

	key1 := coord1.Key()
	key2 := coord2.Key()

	fmt.Printf("Coord1(-1,0,0): %064b\n", uint64(key1))
	fmt.Printf("Coord2(1048575,0,0): %064b\n", uint64(key2))

	if key1 == key2 {
		t.Errorf("Chaves idênticas para coordenadas diferentes!")
	}
}