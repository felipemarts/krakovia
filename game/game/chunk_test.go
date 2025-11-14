package game

import (
	"testing"
)

func TestGetChunkCoord(t *testing.T) {
	tests := []struct {
		name     string
		worldX   int32
		worldY   int32
		worldZ   int32
		expectedX int32
		expectedY int32
		expectedZ int32
	}{
		// Testes positivos
		{"Origem", 0, 0, 0, 0, 0, 0},
		{"Positivo no chunk 0", 10, 10, 10, 0, 0, 0},
		{"Limite do chunk 0", 31, 31, 31, 0, 0, 0},
		{"Início do chunk 1", 32, 32, 32, 1, 1, 1},
		{"Meio do chunk 1", 50, 50, 50, 1, 1, 1},

		// Testes negativos - Casos críticos!
		{"Negativo -1", -1, -1, -1, -1, -1, -1},
		{"Negativo no chunk -1", -10, -10, -10, -1, -1, -1},
		{"Limite do chunk -1", -32, -32, -32, -1, -1, -1},
		{"Início do chunk -2", -33, -33, -33, -2, -2, -2},
		{"Meio do chunk -2", -50, -50, -50, -2, -2, -2},
		{"Posição do player (-42, 11, 30)", -42, 11, 30, -2, 0, 0},

		// Casos mistos
		{"Misto positivo e negativo", -10, 10, -10, -1, 0, -1},
		{"Grande negativo", -100, -100, -100, -4, -4, -4},
		{"Grande positivo", 100, 100, 100, 3, 3, 3},

		// Testes específicos para Y negativo
		{"Y negativo -1", 0, -1, 0, 0, -1, 0},
		{"Y negativo -32", 0, -32, 0, 0, -1, 0},
		{"Y negativo -33", 0, -33, 0, 0, -2, 0},
		{"Y negativo -64", 0, -64, 0, 0, -2, 0},
		{"Todos negativos pequenos", -5, -5, -5, -1, -1, -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetChunkCoord(tt.worldX, tt.worldY, tt.worldZ)
			if result.X != tt.expectedX || result.Y != tt.expectedY || result.Z != tt.expectedZ {
				t.Errorf("GetChunkCoord(%d, %d, %d) = (%d, %d, %d); want (%d, %d, %d)",
					tt.worldX, tt.worldY, tt.worldZ,
					result.X, result.Y, result.Z,
					tt.expectedX, tt.expectedY, tt.expectedZ)
			}
		})
	}
}

func TestGetChunkCoordFromFloat(t *testing.T) {
	tests := []struct {
		name     string
		worldX   float32
		worldY   float32
		worldZ   float32
		expectedX int32
		expectedY int32
		expectedZ int32
	}{
		// Testes positivos
		{"Origem", 0.0, 0.0, 0.0, 0, 0, 0},
		{"Positivo no chunk 0", 10.5, 10.5, 10.5, 0, 0, 0},
		{"Limite do chunk 0", 31.9, 31.9, 31.9, 0, 0, 0},
		{"Início do chunk 1", 32.0, 32.0, 32.0, 1, 1, 1},

		// Testes negativos
		{"Negativo -0.1", -0.1, -0.1, -0.1, -1, -1, -1},
		{"Negativo -1.0", -1.0, -1.0, -1.0, -1, -1, -1},
		{"Negativo no chunk -1", -10.5, -10.5, -10.5, -1, -1, -1},
		{"Limite do chunk -1", -31.9, -31.9, -31.9, -1, -1, -1},
		{"Início do chunk -2", -32.1, -32.1, -32.1, -2, -2, -2},
		{"Posição do player (-42.0, 11.0, 30.0)", -42.0, 11.0, 30.0, -2, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetChunkCoordFromFloat(tt.worldX, tt.worldY, tt.worldZ)
			if result.X != tt.expectedX || result.Y != tt.expectedY || result.Z != tt.expectedZ {
				t.Errorf("GetChunkCoordFromFloat(%f, %f, %f) = (%d, %d, %d); want (%d, %d, %d)",
					tt.worldX, tt.worldY, tt.worldZ,
					result.X, result.Y, result.Z,
					tt.expectedX, tt.expectedY, tt.expectedZ)
			}
		})
	}
}

func TestLocalCoordinates(t *testing.T) {
	// Testar conversão de coordenadas globais para locais
	tests := []struct {
		name        string
		worldX      int32
		expectedLocal int32
	}{
		// Positivos
		{"World 0", 0, 0},
		{"World 1", 1, 1},
		{"World 31", 31, 31},
		{"World 32", 32, 0},
		{"World 33", 33, 1},
		{"World 63", 63, 31},
		{"World 64", 64, 0},

		// Negativos
		{"World -1", -1, 31},
		{"World -2", -2, 30},
		{"World -31", -31, 1},
		{"World -32", -32, 0},
		{"World -33", -33, 31},
		{"World -64", -64, 0},
		{"World -65", -65, 31},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simular o cálculo de coordenada local
			localX := ((tt.worldX % ChunkSize) + ChunkSize) % ChunkSize
			if localX != tt.expectedLocal {
				t.Errorf("Local coordinate for world %d = %d; want %d",
					tt.worldX, localX, tt.expectedLocal)
			}
		})
	}
}
