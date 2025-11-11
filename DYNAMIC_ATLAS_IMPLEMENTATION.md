# Implementação de Sistema de Atlas Dinâmico

## Objetivo
Criar um sistema de gerenciamento de texturas dinâmico que monta e desmonta o atlas automaticamente conforme o jogador se move pelo mundo, permitindo suportar centenas de tipos de blocos diferentes mesmo com atlas limitado.

---

## Fases de Implementação

### **FASE 1: Preparação do Terreno de Teste**
**Objetivo:** Criar um mundo com blocos de tipos variados para testar o sistema.

#### 1.1. Expandir Tipos de Blocos
```go
// internal/game/blocks.go

type BlockType uint8

const (
    BlockAir BlockType = iota
    BlockGrass
    BlockDirt
    BlockStone
    BlockWood
    BlockLeaves
    BlockSand
    BlockGravel
    BlockCobblestone
    BlockPlanks
    BlockBricks
    BlockGlass
    BlockIronOre
    BlockGoldOre
    BlockDiamondOre
    BlockCoal
    // ... adicionar até ter ~20-30 tipos diferentes
)
```

#### 1.2. Geração Determinística de Terreno
**Arquivo:** `internal/game/terrain_generator.go` (novo)

```go
package game

import (
    "math"
)

type TerrainGenerator struct {
    Seed int64
}

func NewTerrainGenerator(seed int64) *TerrainGenerator {
    return &TerrainGenerator{Seed: seed}
}

// Hash determinístico para gerar tipos de blocos baseado em posição
func (tg *TerrainGenerator) hash3D(x, y, z int32) uint64 {
    h := uint64(tg.Seed)
    h ^= uint64(x) * 0x45d9f3b
    h ^= uint64(y) * 0x45d9f3b * 3
    h ^= uint64(z) * 0x45d9f3b * 7
    h = (h ^ (h >> 16)) * 0x45d9f3b
    h = (h ^ (h >> 16)) * 0x45d9f3b
    h = h ^ (h >> 16)
    return h
}

// Gera tipo de bloco baseado na posição (determinístico)
func (tg *TerrainGenerator) GetBlockTypeAt(x, y, z int32) BlockType {
    // Camada de ar
    if y > 8 {
        return BlockAir
    }

    // Camada de superfície (y=8)
    if y == 8 {
        // Usar hash para escolher tipo de bloco de superfície
        h := tg.hash3D(x, y, z)
        surfaceTypes := []BlockType{
            BlockGrass, BlockSand, BlockGravel, BlockStone,
        }
        return surfaceTypes[h%uint64(len(surfaceTypes))]
    }

    // Camadas intermediárias (y=4-7)
    if y >= 4 && y < 8 {
        h := tg.hash3D(x, y, z)
        midTypes := []BlockType{
            BlockDirt, BlockCobblestone, BlockGravel,
            BlockCoal, BlockIronOre,
        }
        return midTypes[h%uint64(len(midTypes))]
    }

    // Camadas profundas (y=0-3)
    h := tg.hash3D(x, y, z)
    deepTypes := []BlockType{
        BlockStone, BlockCobblestone, BlockIronOre,
        BlockGoldOre, BlockDiamondOre,
    }
    return deepTypes[h%uint64(len(deepTypes))]
}
```

#### 1.3. Integração com Chunk
**Modificar:** `internal/game/chunk.go`

```go
// Adicionar ao método Generate ou criar novo método
func (c *Chunk) GenerateWithTerrainGenerator(tg *TerrainGenerator) {
    worldX := c.Coord.X * ChunkSize
    worldY := c.Coord.Y * ChunkHeight
    worldZ := c.Coord.Z * ChunkSize

    for x := int32(0); x < ChunkSize; x++ {
        for y := int32(0); y < ChunkHeight; y++ {
            for z := int32(0); z < ChunkSize; z++ {
                wx := worldX + x
                wy := worldY + y
                wz := worldZ + z

                blockType := tg.GetBlockTypeAt(wx, wy, wz)
                c.Blocks[x][y][z] = blockType
            }
        }
    }

    c.NeedUpdateMeshes = true
}
```

#### 1.4. Inicializar TerrainGenerator
**Modificar:** `internal/game/game.go` ou onde World é criado

```go
// Adicionar campo em World
type World struct {
    // ... campos existentes
    TerrainGenerator *TerrainGenerator
}

// No InitWorld ou similar
world.TerrainGenerator = NewTerrainGenerator(12345) // Seed fixo para testes

// Ao criar chunks
chunk.GenerateWithTerrainGenerator(world.TerrainGenerator)
```

**Resultado esperado:** Mundo gerado com ~15-20 tipos de blocos diferentes distribuídos deterministicamente.

---

### **FASE 2: Extração de Texturas Individuais**
**Objetivo:** Dividir o atlas atual em arquivos individuais para simular texturas baixadas da API.

#### 2.1. Script de Extração
**Arquivo:** `tools/extract_textures.go` (novo)

```go
package main

import (
    "fmt"
    "image"
    "image/png"
    "os"
)

func main() {
    // Carregar atlas
    atlasFile, err := os.Open("assets/texture_atlas.png")
    if err != nil {
        panic(err)
    }
    defer atlasFile.Close()

    atlasImg, _, err := image.Decode(atlasFile)
    if err != nil {
        panic(err)
    }

    // Parâmetros do atlas
    const gridSize = 8
    const tileSize = 32

    // Criar diretório de saída
    os.MkdirAll("assets/textures", 0755)

    // Extrair cada tile
    tileCount := 0
    for row := 0; row < gridSize; row++ {
        for col := 0; col < gridSize; col++ {
            // Criar imagem 32x32
            tileImg := image.NewRGBA(image.Rect(0, 0, tileSize, tileSize))

            // Copiar pixels do atlas
            for y := 0; y < tileSize; y++ {
                for x := 0; x < tileSize; x++ {
                    srcX := col*tileSize + x
                    srcY := row*tileSize + y
                    color := atlasImg.At(srcX, srcY)
                    tileImg.Set(x, y, color)
                }
            }

            // Salvar arquivo
            filename := fmt.Sprintf("assets/textures/tile_%d_%d.png", row, col)
            outFile, err := os.Create(filename)
            if err != nil {
                panic(err)
            }

            png.Encode(outFile, tileImg)
            outFile.Close()

            tileCount++
            fmt.Printf("Extraído: %s\n", filename)
        }
    }

    fmt.Printf("\nTotal: %d texturas extraídas\n", tileCount)
}
```

#### 2.2. Executar Extração
```bash
cd c:\Users\Felipe\Documents\Devel\krakovia
go run tools/extract_textures.go
```

#### 2.3. Mapeamento de Blocos para Texturas
**Arquivo:** `internal/game/texture_mapping.go` (novo)

```go
package game

// Mapear cada BlockType para o arquivo de textura correspondente
var BlockTextureFiles = map[BlockType]string{
    BlockGrass:        "textures/tile_1_1.png",
    BlockDirt:         "textures/tile_1_0.png",
    BlockStone:        "textures/tile_1_2.png",
    BlockWood:         "textures/tile_2_0.png",
    BlockLeaves:       "textures/tile_2_1.png",
    BlockSand:         "textures/tile_2_2.png",
    BlockGravel:       "textures/tile_3_0.png",
    BlockCobblestone:  "textures/tile_3_1.png",
    BlockPlanks:       "textures/tile_3_2.png",
    BlockBricks:       "textures/tile_4_0.png",
    BlockGlass:        "textures/tile_4_1.png",
    BlockIronOre:      "textures/tile_4_2.png",
    BlockGoldOre:      "textures/tile_5_0.png",
    BlockDiamondOre:   "textures/tile_5_1.png",
    BlockCoal:         "textures/tile_5_2.png",
    // ... adicionar todos os tipos
}

// Textura padrão (posição 0,0)
const DefaultTextureFile = "textures/tile_0_0.png"
```

**Resultado esperado:** Diretório `assets/textures/` com 64 arquivos PNG individuais.

---

### **FASE 3: Sistema de Atlas Dinâmico**
**Objetivo:** Criar gerenciador que constrói atlas em tempo real baseado nos blocos visíveis.

#### 3.1. Estrutura do Gerenciador
**Arquivo:** `internal/game/dynamic_atlas.go` (novo)

```go
package game

import (
    "image"
    "image/color"
    "image/png"
    "os"
    "sync"

    rl "github.com/gen2brain/raylib-go/raylib"
)

type DynamicAtlasManager struct {
    mu sync.RWMutex

    // Configuração
    AtlasGridSize   int32   // Ex: 4 para atlas 4x4
    TileSize        int32   // Ex: 32 pixels
    AtlasPixelSize  int32   // AtlasGridSize * TileSize

    // Cache de texturas carregadas
    TextureCache    map[BlockType]image.Image  // BlockType → imagem 32x32

    // Mapeamento de slots
    BlockToSlot     map[BlockType]int32        // BlockType → posição no atlas (0-15 para 4x4)
    SlotToBlock     map[int32]BlockType        // posição → BlockType
    UsedSlots       map[int32]bool             // quais slots estão ocupados
    NextSlot        int32                      // próximo slot disponível

    // Atlas atual
    AtlasImage      *image.RGBA                // Imagem do atlas montado
    AtlasTexture    rl.Texture2D               // Textura no GPU
    AtlasDirty      bool                       // Precisa rebuild?

    // Estatísticas
    LoadedTextures  int
    RebuildCount    int
}

func NewDynamicAtlasManager(gridSize, tileSize int32) *DynamicAtlasManager {
    dam := &DynamicAtlasManager{
        AtlasGridSize:  gridSize,
        TileSize:       tileSize,
        AtlasPixelSize: gridSize * tileSize,
        TextureCache:   make(map[BlockType]image.Image),
        BlockToSlot:    make(map[BlockType]int32),
        SlotToBlock:    make(map[int32]BlockType),
        UsedSlots:      make(map[int32]bool),
        NextSlot:       1, // Slot 0 reservado para default
    }

    // Criar atlas vazio
    dam.AtlasImage = image.NewRGBA(image.Rect(0, 0, int(dam.AtlasPixelSize), int(dam.AtlasPixelSize)))

    // Carregar textura default no slot 0
    dam.LoadTexture(BlockAir, DefaultTextureFile)
    dam.BlockToSlot[BlockAir] = 0
    dam.SlotToBlock[0] = BlockAir
    dam.UsedSlots[0] = true

    return dam
}

// Carrega uma textura individual do arquivo
func (dam *DynamicAtlasManager) LoadTexture(blockType BlockType, filePath string) error {
    dam.mu.Lock()
    defer dam.mu.Unlock()

    // Se já carregou, retorna
    if _, exists := dam.TextureCache[blockType]; exists {
        return nil
    }

    // Carregar arquivo
    file, err := os.Open("assets/" + filePath)
    if err != nil {
        return err
    }
    defer file.Close()

    img, _, err := image.Decode(file)
    if err != nil {
        return err
    }

    dam.TextureCache[blockType] = img
    dam.LoadedTextures++

    return nil
}

// Aloca um slot no atlas para um BlockType
func (dam *DynamicAtlasManager) AllocateSlot(blockType BlockType) int32 {
    dam.mu.Lock()
    defer dam.mu.Unlock()

    // Se já tem slot, retorna
    if slot, exists := dam.BlockToSlot[blockType]; exists {
        return slot
    }

    // Verificar se há slots disponíveis
    maxSlots := dam.AtlasGridSize * dam.AtlasGridSize
    if dam.NextSlot >= maxSlots {
        // Atlas cheio, retorna slot default
        return 0
    }

    // Alocar próximo slot
    slot := dam.NextSlot
    dam.NextSlot++

    dam.BlockToSlot[blockType] = slot
    dam.SlotToBlock[slot] = blockType
    dam.UsedSlots[slot] = true

    dam.AtlasDirty = true

    return slot
}

// Libera um slot (quando chunk é descarregado e textura não é mais necessária)
func (dam *DynamicAtlasManager) FreeSlot(blockType BlockType) {
    dam.mu.Lock()
    defer dam.mu.Unlock()

    slot, exists := dam.BlockToSlot[blockType]
    if !exists {
        return
    }

    delete(dam.BlockToSlot, blockType)
    delete(dam.SlotToBlock, slot)
    delete(dam.UsedSlots, slot)

    dam.AtlasDirty = true
}

// Reconstrói a imagem do atlas
func (dam *DynamicAtlasManager) RebuildAtlas() {
    dam.mu.Lock()
    defer dam.mu.Unlock()

    if !dam.AtlasDirty {
        return
    }

    // Limpar atlas
    for y := 0; y < int(dam.AtlasPixelSize); y++ {
        for x := 0; x < int(dam.AtlasPixelSize); x++ {
            dam.AtlasImage.Set(x, y, color.RGBA{0, 0, 0, 255})
        }
    }

    // Copiar cada textura para seu slot
    for blockType, slot := range dam.BlockToSlot {
        img, exists := dam.TextureCache[blockType]
        if !exists {
            continue
        }

        // Calcular posição no grid
        col := slot % dam.AtlasGridSize
        row := slot / dam.AtlasGridSize

        destX := int(col * dam.TileSize)
        destY := int(row * dam.TileSize)

        // Copiar pixels
        for y := 0; y < int(dam.TileSize); y++ {
            for x := 0; x < int(dam.TileSize); x++ {
                srcColor := img.At(x, y)
                dam.AtlasImage.Set(destX+x, destY+y, srcColor)
            }
        }
    }

    dam.AtlasDirty = false
    dam.RebuildCount++
}

// Upload do atlas para GPU
func (dam *DynamicAtlasManager) UploadToGPU() {
    dam.mu.RLock()
    defer dam.mu.RUnlock()

    // Descarregar textura antiga se existir
    if dam.AtlasTexture.ID != 0 {
        rl.UnloadTexture(dam.AtlasTexture)
    }

    // Converter image.RGBA para Raylib Image
    raylibImg := rl.Image{
        Data:    &dam.AtlasImage.Pix[0],
        Width:   dam.AtlasPixelSize,
        Height:  dam.AtlasPixelSize,
        Mipmaps: 1,
        Format:  rl.UncompressedR8g8b8a8,
    }

    // Upload para GPU
    dam.AtlasTexture = rl.LoadTextureFromImage(&raylibImg)
    rl.SetTextureFilter(dam.AtlasTexture, rl.FilterPoint)
}

// Retorna UVs para um BlockType
func (dam *DynamicAtlasManager) GetBlockUVs(blockType BlockType) (uMin, vMin, uMax, vMax float32) {
    dam.mu.RLock()
    defer dam.mu.RUnlock()

    slot, exists := dam.BlockToSlot[blockType]
    if !exists {
        slot = 0 // Default
    }

    col := slot % dam.AtlasGridSize
    row := slot / dam.AtlasGridSize

    tileUV := float32(1.0) / float32(dam.AtlasGridSize)

    uMin = float32(col) * tileUV
    vMin = float32(row) * tileUV
    uMax = uMin + tileUV
    vMax = vMin + tileUV

    return
}

// Debug: Salva atlas atual em arquivo
func (dam *DynamicAtlasManager) SaveAtlasDebug(filename string) error {
    dam.mu.RLock()
    defer dam.mu.RUnlock()

    file, err := os.Create(filename)
    if err != nil {
        return err
    }
    defer file.Close()

    return png.Encode(file, dam.AtlasImage)
}

// Debug: Imprime estatísticas
func (dam *DynamicAtlasManager) PrintStats() {
    dam.mu.RLock()
    defer dam.mu.RUnlock()

    fmt.Printf("=== Dynamic Atlas Stats ===\n")
    fmt.Printf("Grid Size: %dx%d (max %d textures)\n", dam.AtlasGridSize, dam.AtlasGridSize, dam.AtlasGridSize*dam.AtlasGridSize)
    fmt.Printf("Loaded Textures: %d\n", dam.LoadedTextures)
    fmt.Printf("Allocated Slots: %d\n", len(dam.UsedSlots))
    fmt.Printf("Rebuild Count: %d\n", dam.RebuildCount)
    fmt.Printf("Atlas Dirty: %v\n", dam.AtlasDirty)
    fmt.Printf("==========================\n")
}
```

#### 3.2. Sistema de Rastreamento de Blocos Visíveis
**Arquivo:** `internal/game/visible_blocks_tracker.go` (novo)

```go
package game

import "sync"

type VisibleBlocksTracker struct {
    mu sync.RWMutex

    // Conta quantos chunks visíveis usam cada tipo de bloco
    BlockUsageCount map[BlockType]int
}

func NewVisibleBlocksTracker() *VisibleBlocksTracker {
    return &VisibleBlocksTracker{
        BlockUsageCount: make(map[BlockType]int),
    }
}

// Registra blocos de um chunk como visíveis
func (vbt *VisibleBlocksTracker) RegisterChunk(chunk *Chunk) {
    vbt.mu.Lock()
    defer vbt.mu.Unlock()

    // Contar tipos únicos no chunk
    uniqueBlocks := make(map[BlockType]bool)

    for x := int32(0); x < ChunkSize; x++ {
        for y := int32(0); y < ChunkHeight; y++ {
            for z := int32(0); z < ChunkSize; z++ {
                blockType := chunk.Blocks[x][y][z]
                if blockType != BlockAir {
                    uniqueBlocks[blockType] = true
                }
            }
        }
    }

    // Incrementar contadores
    for blockType := range uniqueBlocks {
        vbt.BlockUsageCount[blockType]++
    }
}

// Remove blocos de um chunk dos visíveis
func (vbt *VisibleBlocksTracker) UnregisterChunk(chunk *Chunk) {
    vbt.mu.Lock()
    defer vbt.mu.Unlock()

    // Contar tipos únicos no chunk
    uniqueBlocks := make(map[BlockType]bool)

    for x := int32(0); x < ChunkSize; x++ {
        for y := int32(0); y < ChunkHeight; y++ {
            for z := int32(0); z < ChunkSize; z++ {
                blockType := chunk.Blocks[x][y][z]
                if blockType != BlockAir {
                    uniqueBlocks[blockType] = true
                }
            }
        }
    }

    // Decrementar contadores
    for blockType := range uniqueBlocks {
        if count, exists := vbt.BlockUsageCount[blockType]; exists {
            if count <= 1 {
                delete(vbt.BlockUsageCount, blockType)
            } else {
                vbt.BlockUsageCount[blockType] = count - 1
            }
        }
    }
}

// Retorna lista de blocos que devem estar no atlas
func (vbt *VisibleBlocksTracker) GetRequiredBlocks() []BlockType {
    vbt.mu.RLock()
    defer vbt.mu.RUnlock()

    blocks := make([]BlockType, 0, len(vbt.BlockUsageCount))
    for blockType := range vbt.BlockUsageCount {
        blocks = append(blocks, blockType)
    }

    return blocks
}
```

**Resultado esperado:** Sistema completo de gerenciamento de atlas com alocação/desalocação dinâmica.

---

### **FASE 4: Integração com Sistema de Chunks**
**Objetivo:** Fazer o atlas se atualizar automaticamente conforme chunks são carregados/descarregados.

#### 4.1. Adicionar DynamicAtlas ao World
**Modificar:** `internal/game/world.go`

```go
type World struct {
    // ... campos existentes

    // Substituir TextureAtlas estático por dinâmico
    DynamicAtlas      *DynamicAtlasManager
    VisibleBlocks     *VisibleBlocksTracker

    // Remover:
    // TextureAtlas   rl.Texture2D
}

func (w *World) InitWorldGraphics() {
    // Inicializar atlas dinâmico 4x4
    w.DynamicAtlas = NewDynamicAtlasManager(4, 32)
    w.VisibleBlocks = NewVisibleBlocksTracker()

    // Carregar texturas de todos os tipos conhecidos
    for blockType, texFile := range BlockTextureFiles {
        err := w.DynamicAtlas.LoadTexture(blockType, texFile)
        if err != nil {
            fmt.Printf("Erro ao carregar textura %s: %v\n", texFile, err)
        }
    }

    // Criar material
    w.Material = rl.LoadMaterialDefault()

    // Nota: textura será atualizada dinamicamente durante o jogo
}
```

#### 4.2. Modificar ChunkManager para Rastrear Blocos
**Modificar:** `internal/game/chunk_manager.go`

```go
// Adicionar no método que renderiza chunks
func (cm *ChunkManager) Render(world *World, playerPos rl.Vector3) {
    // 1. Determinar quais chunks estão visíveis
    playerChunk := GetChunkCoordFromFloat(playerPos.X, playerPos.Y, playerPos.Z)
    visibleChunks := make([]*Chunk, 0)

    for _, chunk := range cm.Chunks {
        dx := float32(chunk.Coord.X - playerChunk.X)
        dy := float32(chunk.Coord.Y - playerChunk.Y)
        dz := float32(chunk.Coord.Z - playerChunk.Z)
        distSq := dx*dx + dy*dy + dz*dz

        if distSq <= float32(cm.RenderDistance*cm.RenderDistance) {
            visibleChunks = append(visibleChunks, chunk)
        }
    }

    // 2. Atualizar rastreador de blocos visíveis
    world.VisibleBlocks.mu.Lock()
    world.VisibleBlocks.BlockUsageCount = make(map[BlockType]int)
    for _, chunk := range visibleChunks {
        world.VisibleBlocks.RegisterChunk(chunk)
    }
    world.VisibleBlocks.mu.Unlock()

    // 3. Atualizar atlas baseado nos blocos visíveis
    requiredBlocks := world.VisibleBlocks.GetRequiredBlocks()
    atlasChanged := false

    for _, blockType := range requiredBlocks {
        // Verificar se bloco já está no atlas
        world.DynamicAtlas.mu.RLock()
        _, exists := world.DynamicAtlas.BlockToSlot[blockType]
        world.DynamicAtlas.mu.RUnlock()

        if !exists {
            // Alocar slot para novo bloco
            world.DynamicAtlas.AllocateSlot(blockType)
            atlasChanged = true
        }
    }

    // 4. Rebuildar atlas se necessário
    if atlasChanged || world.DynamicAtlas.AtlasDirty {
        world.DynamicAtlas.RebuildAtlas()
        world.DynamicAtlas.UploadToGPU()

        // Atualizar material
        diffuseMap := world.Material.GetMap(rl.MapDiffuse)
        diffuseMap.Texture = world.DynamicAtlas.AtlasTexture

        // Marcar todos os chunks visíveis para regerarem mesh
        for _, chunk := range visibleChunks {
            chunk.NeedUpdateMeshes = true
        }
    }

    // 5. Atualizar meshes pendentes (limite de 3 por frame)
    const maxMeshUpdatesPerFrame = 3
    cm.UpdatePendingMeshes(maxMeshUpdatesPerFrame)

    // 6. Renderizar chunks
    for _, chunk := range visibleChunks {
        if chunk.ChunkMesh.Uploaded {
            rl.DrawMesh(chunk.ChunkMesh.Mesh, world.Material, rl.MatrixIdentity())
        }
    }
}
```

#### 4.3. Modificar GetBlockUVs para Usar Atlas Dinâmico
**Modificar:** `internal/game/rendering.go` e `internal/game/chunk_mesh.go`

```go
// Mudar todas as chamadas de GetBlockUVs para:
// GetBlockUVs(blockType) → world.DynamicAtlas.GetBlockUVs(blockType)

// Exemplo em chunk_mesh.go:
func (cm *ChunkMesh) AddQuad(x, y, z float32, face int, blockType BlockType, atlas *DynamicAtlasManager) {
    uMin, vMin, uMax, vMax := atlas.GetBlockUVs(blockType)

    // ... resto do código permanece igual
}

// Atualizar chamadas em chunk.go:
func (c *Chunk) UpdateMeshesWithNeighbors(getBlockFunc func(x, y, z int32) BlockType, atlas *DynamicAtlasManager) {
    c.ChunkMesh.Clear()

    // ... código existente ...

    for faceIndex, dir := range directions {
        neighborBlock := getBlockFunc(wx+dir.dx, wy+dir.dy, wz+dir.dz)

        if neighborBlock == BlockAir {
            c.ChunkMesh.AddQuad(float32(wx), float32(wy), float32(wz),
                              faceIndex, blockType, atlas)
        }
    }

    // ... resto do código
}
```

**Resultado esperado:** Sistema totalmente integrado onde o atlas se atualiza automaticamente.

---

### **FASE 5: Testes e Debug**
**Objetivo:** Validar funcionamento do sistema completo.

#### 5.1. Adicionar Comandos de Debug
**Modificar:** `internal/game/game.go` ou arquivo principal

```go
// No loop principal, adicionar teclas de debug
func (g *Game) Update() {
    // ... código existente ...

    // F1: Imprimir estatísticas do atlas
    if rl.IsKeyPressed(rl.KeyF1) {
        g.World.DynamicAtlas.PrintStats()
    }

    // F2: Salvar atlas atual em arquivo
    if rl.IsKeyPressed(rl.KeyF2) {
        err := g.World.DynamicAtlas.SaveAtlasDebug("debug_atlas.png")
        if err != nil {
            fmt.Printf("Erro ao salvar atlas: %v\n", err)
        } else {
            fmt.Println("Atlas salvo em debug_atlas.png")
        }
    }

    // F3: Imprimir blocos visíveis
    if rl.IsKeyPressed(rl.KeyF3) {
        blocks := g.World.VisibleBlocks.GetRequiredBlocks()
        fmt.Printf("Blocos visíveis: %d tipos\n", len(blocks))
        for _, bt := range blocks {
            fmt.Printf("  - BlockType %d (count: %d)\n",
                bt, g.World.VisibleBlocks.BlockUsageCount[bt])
        }
    }
}
```

#### 5.2. Cenários de Teste

**Teste 1: Carregamento Inicial**
- Iniciar jogo
- Verificar quantos blocos são carregados no atlas inicial
- Pressionar F1 para ver estatísticas
- Pressionar F2 para salvar atlas visual
- Esperado: Atlas 4x4 com 4-8 tipos de blocos

**Teste 2: Movimento pelo Mundo**
- Andar para diferentes direções
- Observar novos chunks sendo carregados
- Pressionar F1 periodicamente para ver mudanças no atlas
- Esperado: Atlas cresce até atingir 16 slots (4x4), novos blocos substituem slot 0 se cheio

**Teste 3: Limite do Atlas**
- Andar bastante até ter > 16 tipos de blocos visíveis
- Verificar se blocos extras usam textura default (slot 0)
- Pressionar F2 e verificar visualmente o atlas
- Esperado: Atlas fica cheio, blocos extras aparecem com textura default

**Teste 4: Descarregamento**
- Andar em uma direção, depois voltar
- Verificar se atlas muda ao descarregar chunks
- Esperado: Blocos que não são mais visíveis podem ser removidos do atlas

**Teste 5: Performance**
- Andar continuamente pelo mundo
- Monitorar FPS (mostrar no canto da tela)
- Esperado: FPS estável (~60), pequenos drops (~45-50) quando atlas rebuilda

#### 5.3. Métricas a Coletar
```go
// Adicionar ao World
type AtlasMetrics struct {
    TotalRebuilds     int
    AverageRebuildMs  float64
    TexturesLoaded    int
    MemoryUsageMB     float64
}

// Medir tempo de rebuild
func (dam *DynamicAtlasManager) RebuildAtlas() time.Duration {
    start := time.Now()

    // ... código de rebuild ...

    elapsed := time.Since(start)
    return elapsed
}
```

---

## Checklist de Implementação

### Fase 1: Terreno de Teste
- [ ] Expandir BlockType com 20+ tipos
- [ ] Criar TerrainGenerator com hash determinístico
- [ ] Integrar geração com Chunks
- [ ] Testar geração: mundo deve ter variedade de blocos

### Fase 2: Extração de Texturas
- [ ] Criar script extract_textures.go
- [ ] Executar e verificar 64 arquivos em assets/textures/
- [ ] Criar mapeamento BlockTextureFiles
- [ ] Testar carregamento de uma textura individual

### Fase 3: Atlas Dinâmico
- [ ] Implementar DynamicAtlasManager
- [ ] Implementar VisibleBlocksTracker
- [ ] Testar alocação/desalocação de slots
- [ ] Testar rebuild de atlas
- [ ] Testar salvamento de atlas debug

### Fase 4: Integração
- [ ] Modificar World para usar DynamicAtlas
- [ ] Modificar ChunkManager.Render
- [ ] Atualizar todas as chamadas GetBlockUVs
- [ ] Testar compilação
- [ ] Resolver erros de integração

### Fase 5: Testes
- [ ] Adicionar comandos de debug (F1, F2, F3)
- [ ] Executar teste 1: Carregamento inicial
- [ ] Executar teste 2: Movimento
- [ ] Executar teste 3: Limite do atlas
- [ ] Executar teste 4: Descarregamento
- [ ] Executar teste 5: Performance
- [ ] Documentar resultados

---

## Próximos Passos (Futuro)

### Otimizações Avançadas
1. **Cache em Disco:** Salvar texturas baixadas localmente
2. **Compressão:** Comprimir atlas para economizar VRAM
3. **Mipmap:** Gerar mipmaps para texturas distantes
4. **Streaming:** Baixar texturas em background threads

### Escalabilidade
1. **Atlas Múltiplos:** Suportar vários atlas 4x4 simultâneos
2. **Priorização:** Manter texturas mais usadas sempre carregadas
3. **LRU Cache:** Remover texturas menos recentemente usadas
4. **Texture Arrays:** Migrar para OpenGL Texture2DArray

### Multiplayer
1. **Protocolo de Rede:** Sincronizar texture_ids entre clientes
2. **CDN Integration:** Baixar texturas de servidor dedicado
3. **Validação:** Verificar integridade de texturas customizadas
4. **Moderação:** Sistema de aprovação de texturas

---

## Estimativa de Tempo

| Fase | Tempo Estimado |
|------|----------------|
| Fase 1: Terreno | 2-3 horas |
| Fase 2: Extração | 1 hora |
| Fase 3: Atlas Dinâmico | 4-5 horas |
| Fase 4: Integração | 3-4 horas |
| Fase 5: Testes | 2-3 horas |
| **Total** | **12-16 horas** |

---

## Notas Importantes

1. **Backup:** Fazer commit antes de cada fase
2. **Testes Incrementais:** Testar cada componente individualmente antes de integrar
3. **Debug Visual:** Salvar atlas em PNG após cada mudança para debug visual
4. **Performance:** Monitorar FPS continuamente durante desenvolvimento
5. **Fallback:** Sempre ter textura default para casos de erro

---

## Estrutura de Arquivos Após Implementação

```
krakovia/
├── assets/
│   ├── texture_atlas.png (original)
│   └── textures/
│       ├── tile_0_0.png (default)
│       ├── tile_0_1.png
│       ├── ...
│       └── tile_7_7.png (64 arquivos)
├── internal/game/
│   ├── blocks.go (expandido com 20+ tipos)
│   ├── terrain_generator.go (novo)
│   ├── texture_mapping.go (novo)
│   ├── dynamic_atlas.go (novo)
│   ├── visible_blocks_tracker.go (novo)
│   ├── world.go (modificado)
│   ├── chunk.go (modificado)
│   ├── chunk_mesh.go (modificado)
│   ├── chunk_manager.go (modificado)
│   └── rendering.go (modificado)
├── tools/
│   └── extract_textures.go (novo)
└── debug_atlas.png (gerado em runtime)
```

---

## Começar Implementação?

Para iniciar, execute:

```bash
# 1. Criar branch
git checkout -b feature/dynamic-atlas

# 2. Começar pela Fase 1
# Editar internal/game/blocks.go
# Criar internal/game/terrain_generator.go
# ...
```

Pronto para começar a implementação? Qual fase você quer que eu comece a codificar primeiro?
