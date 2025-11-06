# Como Usar Texturas nos Voxels

## Sistema Implementado

O sistema de voxels agora suporta texturas! Cada voxel pode ter sua própria textura usando um **texture atlas**.

## O que é Texture Atlas?

Um texture atlas é uma imagem grande que contém várias texturas pequenas organizadas em uma grade. Por exemplo, um atlas 4x4 contém 16 texturas diferentes.

```
+-----+-----+-----+-----+
| Gra | Ped | Are | Ter |  <- Linha 0
| sso | ra  | ia  | ra  |
+-----+-----+-----+-----+
| Mad | Águ | Lav | Gel |  <- Linha 1
| eira| a   | a   | o   |
+-----+-----+-----+-----+
| ... | ... | ... | ... |  <- Linha 2
+-----+-----+-----+-----+
| ... | ... | ... | ... |  <- Linha 3
+-----+-----+-----+-----+
```

## Como Usar

### 1. Criar um VoxelWorld com Textura

```go
// Sem textura (usa apenas cores)
voxelWorld := NewVoxelWorld(10000, "", 4)

// Com textura atlas (4x4 = 16 texturas)
voxelWorld := NewVoxelWorld(10000, "assets/voxel_atlas.png", 4)
```

### 2. Adicionar Voxels com Texturas Específicas

```go
chunk := NewChunk(rl.NewVector3(0, 0, 0), 100)

// Adicionar voxel com textura índice 0 (grama)
chunk.AddVoxelWithTexture(
    rl.NewVector3(0, 0, 0),
    rl.White,  // Cor (multiplica com a textura)
    0,         // Índice da textura no atlas
)

// Adicionar voxel com textura índice 1 (pedra)
chunk.AddVoxelWithTexture(
    rl.NewVector3(1, 0, 0),
    rl.White,
    1,
)

// Adicionar voxel com textura índice 5 (madeira)
chunk.AddVoxelWithTexture(
    rl.NewVector3(2, 0, 0),
    rl.White,
    5,
)
```

### 3. Cálculo do Índice de Textura

Para um atlas NxN, o índice de cada textura é calculado assim:

```
índice = linha * N + coluna

Exemplo para atlas 4x4:
- Textura na posição (0, 0) = 0 * 4 + 0 = 0
- Textura na posição (0, 1) = 0 * 4 + 1 = 1
- Textura na posição (1, 0) = 1 * 4 + 0 = 4
- Textura na posição (2, 3) = 2 * 4 + 3 = 11
```

## Criando um Texture Atlas

### Opção 1: Manualmente com Editor de Imagem

1. Crie uma imagem quadrada (ex: 256x256, 512x512, 1024x1024)
2. Divida em uma grade (ex: 4x4, 8x8)
3. Desenhe cada textura em uma célula da grade
4. Salve como PNG

### Opção 2: Programaticamente

```go
// Exemplo de como criar um atlas simples proceduralmente
func CreateSimpleAtlas(atlasSize int32, textureSize int32) rl.Image {
    totalSize := atlasSize * textureSize
    img := rl.GenImageColor(int(totalSize), int(totalSize), rl.Black)

    // Preencher cada textura com uma cor diferente
    for row := int32(0); row < atlasSize; row++ {
        for col := int32(0); col < atlasSize; col++ {
            color := rl.NewColor(
                uint8((row * 255) / atlasSize),
                uint8((col * 255) / atlasSize),
                128,
                255,
            )

            x := col * textureSize
            y := row * textureSize

            rl.ImageDrawRectangle(&img, int(x), int(y), int(textureSize), int(textureSize), color)
        }
    }

    return img
}
```

## Exemplo Completo: Terreno com Diferentes Texturas

```go
func generateTerrainWithTextures(world *VoxelWorld, size int32) {
    chunk := NewChunk(rl.NewVector3(0, 0, 0), size)

    for x := int32(0); x < size; x++ {
        for z := int32(0); z < size; z++ {
            height := calculateHeight(x, z) // Sua função de altura

            // Escolher textura baseada na altura
            var textureIndex int32
            if height < 5.0 {
                textureIndex = 0 // Grama (baixo)
            } else if height < 10.0 {
                textureIndex = 4 // Pedra (médio)
            } else {
                textureIndex = 8 // Neve (alto)
            }

            position := rl.NewVector3(float32(x), height, float32(z))
            chunk.AddVoxelWithTexture(position, rl.White, textureIndex)
        }
    }

    world.AddChunk(chunk)
}
```

## Performance

- ✅ 10.000 voxels com texturas = **1 draw call**
- ✅ Todas as texturas carregadas uma vez em memória
- ✅ Sem overhead por textura adicional
- ✅ GPU faz todo o trabalho de instancing

## Dicas

1. **Tamanho do Atlas**: Use potências de 2 (4x4, 8x8, 16x16)
2. **Resolução**: 16x16 ou 32x32 pixels por textura é comum para estilo voxel/Minecraft
3. **Filtro**: O código usa `FilterPoint` para estilo pixelado
4. **Cores**: Você pode multiplicar a textura por uma cor usando o parâmetro `color`
