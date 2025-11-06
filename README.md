# Krakovia - Minecraft em Go com Raylib

Um jogo estilo Minecraft desenvolvido em Go usando a biblioteca Raylib.

## Recursos

- Movimentação com WASD
- Controle de câmera com mouse
- Sistema de pulo e gravidade
- Colocação de blocos (botão direito do mouse)
- Remoção de blocos (botão esquerdo do mouse)
- Detecção de colisão com blocos
- Geração procedural de terreno
- Três tipos de blocos: Grama, Terra e Pedra

## Requisitos

- Go 1.21 ou superior
- GCC (para compilar raylib)
  - Windows: MinGW-w64 ou TDM-GCC
  - Linux: `sudo apt install build-essential`
  - macOS: Xcode Command Line Tools

## Instalação

1. Clone o repositório:
```bash
git clone <url-do-repositorio>
cd krakovia
```

2. Baixe as dependências:
```bash
go mod download
```

## Como Jogar

1. Execute o jogo:
```bash
go run main.go
```

2. Controles:
   - **W/A/S/D** - Movimentação
   - **Espaço** - Pular
   - **Mouse** - Olhar ao redor
   - **Botão Esquerdo do Mouse** - Remover bloco
   - **Botão Direito do Mouse** - Colocar bloco
   - **ESC** - Sair

## Estrutura do Código

- `main.go` - Arquivo principal contendo toda a lógica do jogo
  - **Player** - Sistema de jogador com física e controles
  - **World** - Sistema de mundo voxel com geração de terreno
  - **BlockType** - Tipos de blocos disponíveis

## Recursos Implementados

### Sistema de Física
- Gravidade realista
- Detecção de colisão em 3D
- Pulo com verificação de chão

### Sistema de Mundo
- Armazenamento eficiente de blocos usando hashmap
- Geração procedural de terreno com variação de altura
- Diferentes camadas: superfície (grama), subsolo (terra), profundo (pedra)

### Sistema de Interação
- Raycasting para detectar blocos
- Visualização do bloco selecionado
- Colocação e remoção de blocos

## Melhorias Futuras

- Sistema de chunks para mundos maiores
- Mais tipos de blocos
- Texturas
- Iluminação
- Água
- Geração de mundo com Perlin Noise
- Inventário
- Diferentes ferramentas
- Sons

## Compilação

Para compilar o executável:

```bash
go build -o krakovia.exe main.go
```

## Licença

MIT
