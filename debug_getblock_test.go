package main

import (
	"testing"
)

// TestGetBlockCoordinateMapping testa se GetBlock está mapeando coordenadas corretamente
func TestGetBlockCoordinateMapping(t *testing.T) {
	cm := NewChunkManager(5)

	// Criar um chunk em (0, 0, 0)
	chunk := NewChunk(0, 0, 0)

	// Preencher completamente com pedra
	for x := int32(0); x < ChunkSize; x++ {
		for y := int32(0); y < ChunkHeight; y++ {
			for z := int32(0); z < ChunkSize; z++ {
				chunk.Blocks[x][y][z] = BlockStone
			}
		}
	}

	cm.Chunks[ChunkCoord{X: 0, Y: 0, Z: 0}.Key()] = chunk

	// Testar casos específicos
	tests := []struct {
		name        string
		worldX, worldY, worldZ int32
		expectedBlock BlockType
		expectedLocalX, expectedLocalY, expectedLocalZ int32
	}{
		{
			name: "Origem do chunk (0,0,0)",
			worldX: 0, worldY: 0, worldZ: 0,
			expectedBlock: BlockStone,
			expectedLocalX: 0, expectedLocalY: 0, expectedLocalZ: 0,
		},
		{
			name: "Centro do chunk (16,16,16)",
			worldX: 16, worldY: 16, worldZ: 16,
			expectedBlock: BlockStone,
			expectedLocalX: 16, expectedLocalY: 16, expectedLocalZ: 16,
		},
		{
			name: "Última posição do chunk (31,31,31)",
			worldX: 31, worldY: 31, worldZ: 31,
			expectedBlock: BlockStone,
			expectedLocalX: 31, expectedLocalY: 31, expectedLocalZ: 31,
		},
		{
			name: "Bloco problemático Y-1 de (31,0,0)",
			worldX: 31, worldY: -1, worldZ: 0,
			expectedBlock: BlockAir, // Chunk não existe
			expectedLocalX: -1, expectedLocalY: -1, expectedLocalZ: -1, // Inválido
		},
		{
			name: "Bloco problemático Z-1 de (31,0,0)",
			worldX: 31, worldY: 0, worldZ: -1,
			expectedBlock: BlockAir, // Chunk não existe
			expectedLocalX: -1, expectedLocalY: -1, expectedLocalZ: -1, // Inválido
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Obter o chunk esperado
			chunkCoord := GetChunkCoord(tt.worldX, tt.worldY, tt.worldZ)
			t.Logf("Posição mundial: (%d, %d, %d)", tt.worldX, tt.worldY, tt.worldZ)
			t.Logf("Chunk calculado: %v", chunkCoord)

			// Calcular coordenadas locais como GetBlock faz
			localX := ((tt.worldX % ChunkSize) + ChunkSize) % ChunkSize
			localY := ((tt.worldY % ChunkHeight) + ChunkHeight) % ChunkHeight
			localZ := ((tt.worldZ % ChunkSize) + ChunkSize) % ChunkSize
			t.Logf("Coordenadas locais calculadas: (%d, %d, %d)", localX, localY, localZ)

			// Obter o bloco
			block := cm.GetBlock(tt.worldX, tt.worldY, tt.worldZ)
			t.Logf("Bloco retornado: %v (esperado: %v)", block, tt.expectedBlock)

			if block != tt.expectedBlock {
				t.Errorf("Bloco incorreto! Esperado %v, obtido %v", tt.expectedBlock, block)
			}
		})
	}
}

// TestGetBlockAtBoundaries testa especificamente os blocos nas bordas Y=0 e Z=0
func TestGetBlockAtBoundaries(t *testing.T) {
	cm := NewChunkManager(5)

	// Criar um chunk em (0, 0, 0) totalmente preenchido
	chunk := NewChunk(0, 0, 0)
	for x := int32(0); x < ChunkSize; x++ {
		for y := int32(0); y < ChunkHeight; y++ {
			for z := int32(0); z < ChunkSize; z++ {
				chunk.Blocks[x][y][z] = BlockStone
			}
		}
	}
	cm.Chunks[ChunkCoord{X: 0, Y: 0, Z: 0}.Key()] = chunk

	t.Log("=== Testando blocos na borda Y=0 ===")
	for x := int32(0); x < 5; x++ {
		for z := int32(0); z < 5; z++ {
			block := cm.GetBlock(x, 0, z)
			if block != BlockStone {
				t.Errorf("Bloco (%d, 0, %d) deveria ser pedra, mas é %v", x, z, block)
			}
		}
	}

	t.Log("=== Testando blocos na borda Z=0 ===")
	for x := int32(0); x < 5; x++ {
		for y := int32(0); y < 5; y++ {
			block := cm.GetBlock(x, y, 0)
			if block != BlockStone {
				t.Errorf("Bloco (%d, %d, 0) deveria ser pedra, mas é %v", x, y, block)
			}
		}
	}

	t.Log("=== Testando blocos vizinhos de (31, 0, 0) ===")
	testPos := []struct{ name string; x, y, z int32 }{
		{"Próprio bloco (31,0,0)", 31, 0, 0},
		{"Vizinho Y- (31,-1,0)", 31, -1, 0},
		{"Vizinho Z- (31,0,-1)", 31, 0, -1},
		{"Vizinho Y+ (31,1,0)", 31, 1, 0},
		{"Vizinho Z+ (31,0,1)", 31, 0, 1},
	}

	for _, pos := range testPos {
		block := cm.GetBlock(pos.x, pos.y, pos.z)
		chunkCoord := GetChunkCoord(pos.x, pos.y, pos.z)
		localX := ((pos.x % ChunkSize) + ChunkSize) % ChunkSize
		localY := ((pos.y % ChunkHeight) + ChunkHeight) % ChunkHeight
		localZ := ((pos.z % ChunkSize) + ChunkSize) % ChunkSize

		t.Logf("%s: chunk=%v local=(%d,%d,%d) bloco=%v",
			pos.name, chunkCoord, localX, localY, localZ, block)
	}
}

// TestModuloOperationWithNegatives testa a operação de módulo com números negativos
func TestModuloOperationWithNegatives(t *testing.T) {
	tests := []struct {
		value    int32
		modulo   int32
		expected int32
		name     string
	}{
		{-1, 32, 31, "(-1 mod 32) deveria ser 31"},
		{-2, 32, 30, "(-2 mod 32) deveria ser 30"},
		{-32, 32, 0, "(-32 mod 32) deveria ser 0"},
		{-33, 32, 31, "(-33 mod 32) deveria ser 31"},
		{0, 32, 0, "(0 mod 32) deveria ser 0"},
		{31, 32, 31, "(31 mod 32) deveria ser 31"},
		{32, 32, 0, "(32 mod 32) deveria ser 0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Operação de módulo com correção para negativos (como usado no código)
			result := ((tt.value % tt.modulo) + tt.modulo) % tt.modulo
			t.Logf("((%d %% %d) + %d) %% %d = %d (esperado: %d)",
				tt.value, tt.modulo, tt.modulo, tt.modulo, result, tt.expected)

			if result != tt.expected {
				t.Errorf("Resultado incorreto! Esperado %d, obtido %d", tt.expected, result)
			}
		})
	}
}
