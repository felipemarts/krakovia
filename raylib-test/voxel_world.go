package main

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

// Voxel representa um único voxel no mundo
type Voxel struct {
	Position     rl.Vector3
	Color        rl.Color
	Active       bool
	TextureIndex int32 // Índice da textura no atlas (0-based)
}

// Chunk representa um grupo de voxels
type Chunk struct {
	Position rl.Vector3
	Voxels   []Voxel
	Size     int32
}

// VoxelWorld gerencia o mundo de voxels e renderização instanciada
type VoxelWorld struct {
	Chunks          []*Chunk
	InstanceMesh    rl.Mesh
	InstanceBuffer  uint32
	Shader          rl.Shader
	Material        rl.Material
	MaxInstances    int32
	ActiveInstances int32
	Transforms      []rl.Matrix
	Colors          []rl.Color
	TextureIndices  []float32 // Índices de textura para cada instância
	AtlasTexture    rl.Texture2D
	Texture0        rl.Texture2D // Textura azul
	Texture1        rl.Texture2D // Textura verde
	Model0          rl.Model     // Modelo com textura 0
	Model1          rl.Model     // Modelo com textura 1
	AtlasSize       int32        // Número de texturas por lado no atlas (ex: 4x4 = 16 texturas)
}

// NewChunk cria um novo chunk
func NewChunk(position rl.Vector3, size int32) *Chunk {
	return &Chunk{
		Position: position,
		Voxels:   make([]Voxel, 0),
		Size:     size,
	}
}

// AddVoxel adiciona um voxel ao chunk
func (c *Chunk) AddVoxel(position rl.Vector3, color rl.Color) {
	c.Voxels = append(c.Voxels, Voxel{
		Position:     position,
		Color:        color,
		Active:       true,
		TextureIndex: 0,
	})
}

// AddVoxelWithTexture adiciona um voxel com textura específica ao chunk
func (c *Chunk) AddVoxelWithTexture(position rl.Vector3, color rl.Color, textureIndex int32) {
	c.Voxels = append(c.Voxels, Voxel{
		Position:     position,
		Color:        color,
		Active:       true,
		TextureIndex: textureIndex,
	})
}

// NewVoxelWorld cria um novo mundo de voxels com suporte a instancing
func NewVoxelWorld(maxInstances int32, atlasTexturePath string, atlasSize int32) *VoxelWorld {
	world := &VoxelWorld{
		Chunks:         make([]*Chunk, 0),
		MaxInstances:   maxInstances,
		Transforms:     make([]rl.Matrix, maxInstances),
		Colors:         make([]rl.Color, maxInstances),
		TextureIndices: make([]float32, maxInstances),
		AtlasSize:      atlasSize,
	}

	// Criar mesh de um cubo (voxel básico) com coordenadas de textura
	world.InstanceMesh = rl.GenMeshCube(1.0, 1.0, 1.0)

	// Carregar textura atlas e extrair texturas individuais
	if atlasTexturePath != "" {
		// Carregar atlas completo
		atlasImg := rl.LoadImage(atlasTexturePath)

		// Criar sub-imagens para cada textura
		// Textura 0: metade esquerda (azul)
		tex0Img := rl.ImageFromImage(*atlasImg, rl.NewRectangle(0, 0, 32, 32))
		world.Texture0 = rl.LoadTextureFromImage(&tex0Img)
		rl.SetTextureFilter(world.Texture0, rl.FilterPoint)
		rl.UnloadImage(&tex0Img)

		// Textura 1: metade direita (verde)
		tex1Img := rl.ImageFromImage(*atlasImg, rl.NewRectangle(32, 0, 32, 32))
		world.Texture1 = rl.LoadTextureFromImage(&tex1Img)
		rl.SetTextureFilter(world.Texture1, rl.FilterPoint)
		rl.UnloadImage(&tex1Img)

		// Carregar atlas completo também
		world.AtlasTexture = rl.LoadTextureFromImage(atlasImg)
		rl.SetTextureFilter(world.AtlasTexture, rl.FilterPoint)

		rl.UnloadImage(atlasImg)
	}

	// Criar material padrão
	world.Material = rl.LoadMaterialDefault()

	// Criar modelos com texturas se disponíveis
	if world.Texture0.ID != 0 && world.Texture1.ID != 0 {
		world.Model0 = rl.LoadModelFromMesh(world.InstanceMesh)
		rl.SetMaterialTexture(world.Model0.Materials, rl.MapDiffuse, world.Texture0)

		world.Model1 = rl.LoadModelFromMesh(world.InstanceMesh)
		rl.SetMaterialTexture(world.Model1.Materials, rl.MapDiffuse, world.Texture1)
	}

	return world
}

// AddChunk adiciona um chunk ao mundo
func (w *VoxelWorld) AddChunk(chunk *Chunk) {
	w.Chunks = append(w.Chunks, chunk)
}

// UpdateInstanceData atualiza os dados de instancing com base nos voxels ativos
func (w *VoxelWorld) UpdateInstanceData() {
	w.ActiveInstances = 0

	for _, chunk := range w.Chunks {
		for _, voxel := range chunk.Voxels {
			if !voxel.Active || w.ActiveInstances >= w.MaxInstances {
				continue
			}

			// Criar matriz de transformação para este voxel
			position := rl.Vector3Add(chunk.Position, voxel.Position)
			w.Transforms[w.ActiveInstances] = rl.MatrixTranslate(position.X, position.Y, position.Z)
			w.Colors[w.ActiveInstances] = voxel.Color
			w.TextureIndices[w.ActiveInstances] = float32(voxel.TextureIndex)
			w.ActiveInstances++
		}
	}
}

// Draw renderiza todos os voxels
func (w *VoxelWorld) Draw() {
	if w.ActiveInstances == 0 {
		return
	}

	// Desenhar todos os voxels com texturas usando modelo pré-criado
	if w.Model0.MeshCount > 0 && w.Model1.MeshCount > 0 {
		for i := int32(0); i < w.ActiveInstances; i++ {
			// Extrair posição da matriz de transformação
			mat := w.Transforms[i]
			pos := rl.NewVector3(mat.M12, mat.M13, mat.M14)

			textureIndex := int32(w.TextureIndices[i])

			// Usar a textura apropriada
			if textureIndex == 0 {
				rl.DrawModel(w.Model0, pos, 1.0, rl.White)
			} else {
				rl.DrawModel(w.Model1, pos, 1.0, rl.White)
			}
		}
	} else {
		// Sem textura, desenhar só com cores originais
		for i := int32(0); i < w.ActiveInstances; i++ {
			mat := w.Transforms[i]
			pos := rl.NewVector3(mat.M12, mat.M13, mat.M14)
			rl.DrawCube(pos, 1.0, 1.0, 1.0, w.Colors[i])
		}
	}
}

// Unload libera os recursos
func (w *VoxelWorld) Unload() {
	rl.UnloadMesh(&w.InstanceMesh)
	rl.UnloadMaterial(w.Material)
	if w.AtlasTexture.ID != 0 {
		rl.UnloadTexture(w.AtlasTexture)
	}
	if w.Texture0.ID != 0 {
		rl.UnloadTexture(w.Texture0)
	}
	if w.Texture1.ID != 0 {
		rl.UnloadTexture(w.Texture1)
	}
	if w.Model0.MeshCount > 0 {
		rl.UnloadModel(w.Model0)
	}
	if w.Model1.MeshCount > 0 {
		rl.UnloadModel(w.Model1)
	}
}

// Shaders para instancing com suporte a texture atlas
const vertexShaderCode = `
#version 330

// Input vertex attributes
in vec3 vertexPosition;
in vec2 vertexTexCoord;
in vec3 vertexNormal;
in vec4 vertexColor;

// Input uniform values
uniform mat4 mvp;
uniform mat4 matModel;
uniform mat4 matNormal;

// Output vertex attributes (to fragment shader)
out vec2 fragTexCoord;
out vec4 fragColor;
out vec3 fragNormal;
out vec3 fragPosition;
flat out int fragInstanceID;

void main()
{
    // Send vertex attributes to fragment shader
    fragTexCoord = vertexTexCoord;
    fragColor = vertexColor;
    fragNormal = normalize(vec3(matNormal*vec4(vertexNormal, 1.0)));
    fragPosition = vec3(matModel*vec4(vertexPosition, 1.0));
    fragInstanceID = gl_InstanceID;

    // Calculate final vertex position
    gl_Position = mvp*vec4(vertexPosition, 1.0);
}
`

const fragmentShaderCode = `
#version 330

// Input vertex attributes (from vertex shader)
in vec2 fragTexCoord;
in vec4 fragColor;
in vec3 fragNormal;
in vec3 fragPosition;
flat in int fragInstanceID;

// Input uniform values
uniform sampler2D texture0;
uniform vec4 colDiffuse;
uniform float atlasColumns; // Número de texturas na horizontal (ex: 2.0 para atlas 2x1)

// Output fragment color
out vec4 finalColor;

// Função hash simples para variar as texturas baseado no instance ID
float hash(int id) {
    int x = id * 1597334677;
    x = ((x >> 16) ^ x) * 0x45d9f3b;
    x = ((x >> 16) ^ x) * 0x45d9f3b;
    x = (x >> 16) ^ x;
    return float(x & 1); // Retorna 0 ou 1
}

void main()
{
    // DEBUG: Colorir baseado no instance ID para verificar se shader está funcionando
    float id = float(fragInstanceID);
    vec3 debugColor = vec3(
        fract(id * 0.1),
        fract(id * 0.2),
        fract(id * 0.3)
    );

    // Escolher textura baseada no instance ID (alternando entre texturas)
    float textureIndex = hash(fragInstanceID);

    // Calcular coordenadas UV no atlas
    float textureWidth = 1.0 / atlasColumns;
    float offsetX = textureIndex * textureWidth;

    // Ajustar coordenadas UV para a textura específica no atlas
    vec2 atlasTexCoord = vec2(
        offsetX + (fragTexCoord.x * textureWidth),
        fragTexCoord.y
    );

    // Sample texture from atlas
    vec4 texelColor = texture(texture0, atlasTexCoord);

    // Basic lighting
    vec3 lightDir = normalize(vec3(0.5, 1.0, 0.3));
    float NdotL = max(dot(fragNormal, lightDir), 0.0);
    vec3 ambient = vec3(0.6, 0.6, 0.6);
    vec3 diffuse = vec3(NdotL * 0.4);
    vec3 lighting = ambient + diffuse;

    // Se textura tem cor válida, usar textura, senão usar debug color
    if (length(texelColor.rgb) > 0.1) {
        finalColor = texelColor * vec4(lighting, 1.0);
    } else {
        finalColor = vec4(debugColor * lighting, 1.0);
    }
}
`
