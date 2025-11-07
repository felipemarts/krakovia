package main

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// ChunkManager gerencia o carregamento e descarregamento de chunks
type ChunkManager struct {
	Chunks              map[int64]*Chunk
	RenderDistance      int32 // Distância de renderização em chunks
	UnloadDistance      int32 // Distância para descarregar chunks
	LastPlayerChunk     ChunkCoord
	UpdateCooldown      float32 // Tempo desde a última atualização de chunks
	UpdateCooldownLimit float32 // Tempo mínimo entre atualizações (em segundos)
}

// NewChunkManager cria um novo gerenciador de chunks
func NewChunkManager(renderDistance int32) *ChunkManager {
	return &ChunkManager{
		Chunks:              make(map[int64]*Chunk),
		RenderDistance:      renderDistance,
		UnloadDistance:      renderDistance + 2, // Descarrega um pouco além da distância de renderização
		UpdateCooldown:      0,
		UpdateCooldownLimit: 0.05, // Atualizar chunks no máximo a cada 0.05 segundos (20 vezes por segundo)
	}
}

// Update atualiza os chunks baseado na posição do jogador
func (cm *ChunkManager) Update(playerPos rl.Vector3, dt float32) {
	// Incrementar cooldown
	cm.UpdateCooldown += dt

	// Obter chunk atual do jogador
	currentChunk := GetChunkCoordFromFloat(playerPos.X, playerPos.Y, playerPos.Z)

	// Se o cooldown passou, tentar carregar chunks gradualmente
	if cm.UpdateCooldown >= cm.UpdateCooldownLimit {
		cm.LoadChunksAroundPlayer(playerPos)

		// Se o jogador mudou de chunk, descarregar chunks distantes
		if currentChunk != cm.LastPlayerChunk {
			cm.UnloadDistantChunks(playerPos)
			cm.LastPlayerChunk = currentChunk
		}

		cm.UpdateCooldown = 0
	}
}

// LoadChunksAroundPlayer carrega chunks ao redor do jogador
func (cm *ChunkManager) LoadChunksAroundPlayer(playerPos rl.Vector3) {
	playerChunk := GetChunkCoordFromFloat(playerPos.X, playerPos.Y, playerPos.Z)

	// Limitar o número de chunks carregados por frame para evitar lag
	chunksLoadedThisFrame := 0
	maxChunksPerFrame := 4 // Carregar no máximo 4 chunks por frame para carregamento mais rápido

	// Carregar chunks em um raio ao redor do jogador, priorizando os mais próximos
	for distance := int32(0); distance <= cm.RenderDistance; distance++ {
		for x := playerChunk.X - distance; x <= playerChunk.X+distance; x++ {
			for z := playerChunk.Z - distance; z <= playerChunk.Z+distance; z++ {
				// Verificar se está dentro do raio circular
				dx := float32(x - playerChunk.X)
				dz := float32(z - playerChunk.Z)
				dist := float32(math.Sqrt(float64(dx*dx + dz*dz)))

				if dist <= float32(cm.RenderDistance) {
					// Carregar apenas chunks no nível do solo (y=0) por enquanto
					coord := ChunkCoord{X: x, Y: 0, Z: z}
					key := coord.Key()

					// Se o chunk não existe, criar e gerar
					if _, exists := cm.Chunks[key]; !exists {
						chunk := NewChunk(x, 0, z)
						chunk.GenerateTerrain()
						cm.Chunks[key] = chunk

						chunksLoadedThisFrame++
						if chunksLoadedThisFrame >= maxChunksPerFrame {
							return // Parar de carregar neste frame
						}
					}
				}
			}
		}
	}
}

// UnloadDistantChunks descarrega chunks distantes do jogador
func (cm *ChunkManager) UnloadDistantChunks(playerPos rl.Vector3) {
	playerChunk := GetChunkCoordFromFloat(playerPos.X, playerPos.Y, playerPos.Z)

	// Lista de chunks para remover
	toRemove := make([]int64, 0)

	for key, chunk := range cm.Chunks {
		// Calcular distância do chunk ao jogador
		dx := float32(chunk.Coord.X - playerChunk.X)
		dz := float32(chunk.Coord.Z - playerChunk.Z)
		distance := float32(math.Sqrt(float64(dx*dx + dz*dz)))

		// Se está além da distância de descarregamento, marcar para remoção
		if distance > float32(cm.UnloadDistance) {
			toRemove = append(toRemove, key)
		}
	}

	// Remover chunks marcados
	for _, key := range toRemove {
		delete(cm.Chunks, key)
	}
}

// GetBlock retorna o tipo de bloco nas coordenadas mundiais
func (cm *ChunkManager) GetBlock(x, y, z int32) BlockType {
	// Obter coordenadas do chunk
	chunkCoord := GetChunkCoord(x, y, z)
	key := chunkCoord.Key()

	// Verificar se o chunk existe
	chunk, exists := cm.Chunks[key]
	if !exists {
		return BlockAir
	}

	// Converter para coordenadas locais do chunk
	// Usar módulo para garantir coordenadas locais corretas
	localX := ((x % ChunkSize) + ChunkSize) % ChunkSize
	localY := ((y % ChunkHeight) + ChunkHeight) % ChunkHeight
	localZ := ((z % ChunkSize) + ChunkSize) % ChunkSize

	return chunk.GetBlock(localX, localY, localZ)
}

// SetBlock define o tipo de bloco nas coordenadas mundiais
func (cm *ChunkManager) SetBlock(x, y, z int32, block BlockType) {
	// Obter coordenadas do chunk
	chunkCoord := GetChunkCoord(x, y, z)
	key := chunkCoord.Key()

	// Verificar se o chunk existe
	chunk, exists := cm.Chunks[key]
	if !exists {
		// Se não existe, criar o chunk
		chunk = NewChunk(chunkCoord.X, chunkCoord.Y, chunkCoord.Z)
		chunk.GenerateTerrain()
		cm.Chunks[key] = chunk
	}

	// Converter para coordenadas locais do chunk
	// Usar módulo para garantir coordenadas locais corretas
	localX := ((x % ChunkSize) + ChunkSize) % ChunkSize
	localY := ((y % ChunkHeight) + ChunkHeight) % ChunkHeight
	localZ := ((z % ChunkSize) + ChunkSize) % ChunkSize

	chunk.SetBlock(localX, localY, localZ, block)
}

// Render renderiza todos os chunks carregados
func (cm *ChunkManager) Render(grassMesh, dirtMesh, stoneMesh rl.Mesh, material rl.Material, playerPos rl.Vector3) {
	// Renderizar apenas chunks próximos ao jogador para melhor performance
	playerChunk := GetChunkCoordFromFloat(playerPos.X, playerPos.Y, playerPos.Z)

	for _, chunk := range cm.Chunks {
		// Calcular distância do chunk ao jogador
		dx := float32(chunk.Coord.X - playerChunk.X)
		dz := float32(chunk.Coord.Z - playerChunk.Z)
		distSq := dx*dx + dz*dz

		// Renderizar apenas chunks dentro da distância de renderização
		if distSq <= float32(cm.RenderDistance*cm.RenderDistance) {
			chunk.Render(grassMesh, dirtMesh, stoneMesh, material)
		}
	}
}

// GetTotalBlocks retorna o número total de blocos carregados (para debug)
func (cm *ChunkManager) GetTotalBlocks() int {
	total := 0
	for _, chunk := range cm.Chunks {
		total += len(chunk.GrassTransforms) + len(chunk.DirtTransforms) + len(chunk.StoneTransforms)
	}
	return total
}

// GetLoadedChunksCount retorna o número de chunks carregados
func (cm *ChunkManager) GetLoadedChunksCount() int {
	return len(cm.Chunks)
}
