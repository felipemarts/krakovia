# Krakovia

Sandbox voxel em Go que recria mecanicas basicas de jogos estilo Minecraft usando o binding `raylib-go`.

## Visao Geral
- Motor de chunks 32x32x32 com streaming dinamico via `ChunkManager`.
- Sistema completo de jogador em terceira pessoa com fisica, pulo, modo fly e deteccao precisa de colisao cilidrica.
- Interacao com blocos via raycasting, highlight visual e suporte a colocar/remover blocos.
- Renderizacao baseada em meshes combinadas por chunk e atlas de texturas localizado em `assets/texture_atlas.png`.
- Suite extensa de testes (stress, diagnostico, real scenario) para validar FPS, carregamento e colisao.

## Requisitos
- Go 1.21+ (o modulo usa 1.23.1).
- Toolchain C para compilar Raylib.
  - Windows: MinGW-w64/TDM-GCC.
  - Linux: `sudo apt install build-essential`.
  - macOS: Xcode Command Line Tools.

## Instalacao
```bash
git clone <url-do-repositorio>
cd krakovia
go mod download
```

## Como Executar
```bash
# build e execute direto
go run .

# ou gere um executavel
go build -o krakovia.exe .
```

Controles padrao:
- `W/A/S/D` movimentacao
- `Espaco` pular
- `Mouse` olhar
- Botao esquerdo: remover bloco
- Botao direito: colocar bloco
- `P`: alternar fly mode (`Shift` sobe, `Ctrl` desce)
- `V`: alternar entre primeira e terceira pessoa (com transição suave)
- `Esc`: sair

## Estrutura do Projeto
```
main.go             # ponto de entrada do jogo
internal/game       # logica principal (chunks, player, input, testes)
assets/             # recursos estaticos (texture_atlas.png)
docs/               # materiais auxiliares (SOLUTION_SUMMARY.md, TESTING.md)
go.mod / go.sum     # definicao do modulo Go
```

o pacote `internal/game` expõe `NewPlayer`, `NewWorld`, `RaylibInput` e constantes de configuracao (`ScreenWidth`, `ScreenHeight`, etc.), e `main.go` apenas orquestra a inicializacao e o loop principal.

## Testes
```bash
go test ./...
```

- Casos especificos e metodologia: `docs/TESTING.md`.
- Relatos de investigacoes anteriores: `docs/SOLUTION_SUMMARY.md`.

## Proximos Passos Sugeridos
1. Expandir o atlas e o gerador procedural para suportar novos blocos.
2. Adicionar pipeline de CI simples que rode `go test ./...` a cada commit.
3. Versionar binarios gerados (ex.: `krakovia.exe`) fora do repo final usando `.gitignore`.
