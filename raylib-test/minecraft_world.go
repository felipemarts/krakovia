package main

import (
	"fmt"
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// MinecraftWorld gerencia todos os chunks
type MinecraftWorld struct {
	Chunks         map[string]*MinecraftChunk
	PlayerChunkPos rl.Vector3
	AtlasTexture   rl.Texture2D
	Texture0       rl.Texture2D
	Texture1       rl.Texture2D
}

// NewMinecraftWorld cria um novo mundo
func NewMinecraftWorld() *MinecraftWorld {
	world := &MinecraftWorld{
		Chunks: make(map[string]*MinecraftChunk),
	}

	// Carregar texturas se disponíveis
	world.loadTextures()

	return world
}

func (w *MinecraftWorld) loadTextures() {
	// Tentar carregar as texturas existentes
	atlasImg := rl.LoadImage("voxel_atlas.png")
	if atlasImg == nil {
		return
	}

	// Carregar atlas completo
	w.AtlasTexture = rl.LoadTextureFromImage(atlasImg)
	rl.SetTextureFilter(w.AtlasTexture, rl.FilterPoint)

	// Carregar texturas individuais também
	tex0Img := rl.ImageFromImage(*atlasImg, rl.NewRectangle(0, 0, 32, 32))
	w.Texture0 = rl.LoadTextureFromImage(&tex0Img)
	rl.SetTextureFilter(w.Texture0, rl.FilterPoint)
	rl.UnloadImage(&tex0Img)

	tex1Img := rl.ImageFromImage(*atlasImg, rl.NewRectangle(32, 0, 32, 32))
	w.Texture1 = rl.LoadTextureFromImage(&tex1Img)
	rl.SetTextureFilter(w.Texture1, rl.FilterPoint)
	rl.UnloadImage(&tex1Img)

	rl.UnloadImage(atlasImg)
}

// GetChunkKey retorna a chave única para um chunk
func GetChunkKey(x, y, z int32) string {
	return fmt.Sprintf("%d_%d_%d", x, y, z)
}

// WorldToChunkPos converte posição mundial para posição de chunk
func WorldToChunkPos(worldX, worldY, worldZ float32) (int32, int32, int32) {
	return int32(math.Floor(float64(worldX) / ChunkSize)),
		int32(math.Floor(float64(worldY) / ChunkSize)),
		int32(math.Floor(float64(worldZ) / ChunkSize))
}

// GetChunk retorna um chunk (cria se não existir)
func (w *MinecraftWorld) GetChunk(chunkX, chunkY, chunkZ int32) *MinecraftChunk {
	key := GetChunkKey(chunkX, chunkY, chunkZ)
	chunk, exists := w.Chunks[key]
	if !exists {
		chunk = NewMinecraftChunk(chunkX, chunkY, chunkZ)
		chunk.GenerateTerrain()
		w.Chunks[key] = chunk
	}
	return chunk
}

// GetBlock retorna o tipo de bloco em uma posição mundial
func (w *MinecraftWorld) GetBlock(worldX, worldY, worldZ float32) BlockType {
	chunkX, chunkY, chunkZ := WorldToChunkPos(worldX, worldY, worldZ)
	chunk := w.GetChunk(chunkX, chunkY, chunkZ)

	// Converter para posição local no chunk
	localX := int32(worldX) - chunkX*ChunkSize
	localY := int32(worldY) - chunkY*ChunkSize
	localZ := int32(worldZ) - chunkZ*ChunkSize

	// Normalizar para range 0-31
	if localX < 0 {
		localX += ChunkSize
	}
	if localY < 0 {
		localY += ChunkSize
	}
	if localZ < 0 {
		localZ += ChunkSize
	}

	return chunk.GetBlock(localX, localY, localZ)
}

// SetBlock define um bloco em uma posição mundial
func (w *MinecraftWorld) SetBlock(worldX, worldY, worldZ float32, blockType BlockType) {
	chunkX, chunkY, chunkZ := WorldToChunkPos(worldX, worldY, worldZ)
	chunk := w.GetChunk(chunkX, chunkY, chunkZ)

	localX := int32(worldX) - chunkX*ChunkSize
	localY := int32(worldY) - chunkY*ChunkSize
	localZ := int32(worldZ) - chunkZ*ChunkSize

	if localX < 0 {
		localX += ChunkSize
	}
	if localY < 0 {
		localY += ChunkSize
	}
	if localZ < 0 {
		localZ += ChunkSize
	}

	chunk.SetBlock(localX, localY, localZ, blockType)

	// Marcar chunks vizinhos como dirty se o bloco está na borda
	if localX == 0 {
		neighborChunk := w.GetChunk(chunkX-1, chunkY, chunkZ)
		if neighborChunk != nil {
			neighborChunk.IsDirty = true
		}
	} else if localX == ChunkSize-1 {
		neighborChunk := w.GetChunk(chunkX+1, chunkY, chunkZ)
		if neighborChunk != nil {
			neighborChunk.IsDirty = true
		}
	}
	if localY == 0 {
		neighborChunk := w.GetChunk(chunkX, chunkY-1, chunkZ)
		if neighborChunk != nil {
			neighborChunk.IsDirty = true
		}
	} else if localY == ChunkSize-1 {
		neighborChunk := w.GetChunk(chunkX, chunkY+1, chunkZ)
		if neighborChunk != nil {
			neighborChunk.IsDirty = true
		}
	}
	if localZ == 0 {
		neighborChunk := w.GetChunk(chunkX, chunkY, chunkZ-1)
		if neighborChunk != nil {
			neighborChunk.IsDirty = true
		}
	} else if localZ == ChunkSize-1 {
		neighborChunk := w.GetChunk(chunkX, chunkY, chunkZ+1)
		if neighborChunk != nil {
			neighborChunk.IsDirty = true
		}
	}
}

// UpdateChunks atualiza chunks ao redor do player
func (w *MinecraftWorld) UpdateChunks(playerPos rl.Vector3) {
	playerChunkX, _, playerChunkZ := WorldToChunkPos(playerPos.X, playerPos.Y, playerPos.Z)

	// Carregar chunks ao redor do player
	for cx := playerChunkX - RenderDist; cx <= playerChunkX+RenderDist; cx++ {
		for cy := int32(0); cy < 2; cy++ { // Apenas 2 chunks de altura (0 e 1)
			for cz := playerChunkZ - RenderDist; cz <= playerChunkZ+RenderDist; cz++ {
				chunk := w.GetChunk(cx, cy, cz)
				if chunk.IsDirty {
					chunk.BuildMesh(w)
				}
			}
		}
	}

	// Descarregar chunks distantes
	toUnload := make([]string, 0)
	for key, chunk := range w.Chunks {
		dx := chunk.Position.X - float32(playerChunkX)
		dz := chunk.Position.Z - float32(playerChunkZ)
		dist := math.Sqrt(float64(dx*dx + dz*dz))

		if dist > float64(RenderDist+2) {
			chunk.Unload()
			toUnload = append(toUnload, key)
		}
	}

	for _, key := range toUnload {
		delete(w.Chunks, key)
	}
}

// Draw renderiza todos os chunks ativos
func (w *MinecraftWorld) Draw() {
	for _, chunk := range w.Chunks {
		if chunk.IsActive {
			chunk.Draw()
		}
	}
}

// Unload libera todos os recursos
func (w *MinecraftWorld) Unload() {
	for _, chunk := range w.Chunks {
		chunk.Unload()
	}
	if w.Texture0.ID != 0 {
		rl.UnloadTexture(w.Texture0)
	}
	if w.Texture1.ID != 0 {
		rl.UnloadTexture(w.Texture1)
	}
}
