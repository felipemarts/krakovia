# Testes do Krakovia

Este documento descreve a estratégia de testes implementada para o jogo Krakovia.

## Visão Geral

Os testes foram projetados para garantir que novas features não quebrem funcionalidades existentes. Utilizamos **testes unitários headless** (sem interface gráfica) que simulam o comportamento do jogo.

## Arquitetura de Testes

### 1. Abstração de Input

Criamos uma interface `Input` que permite simular entrada do usuário sem depender do Raylib:

```go
type Input interface {
    IsForwardPressed() bool
    IsBackPressed() bool
    IsJumpPressed() bool
    // ... outros métodos
}
```

- **`RaylibInput`**: Implementação real usada no jogo
- **`SimulatedInput`**: Implementação para testes que simula teclas e mouse

### 2. Mundo Plano para Testes

Função helper `createFlatWorld()` cria um terreno plano e previsível:
- Chão de grama em Y=10
- Camadas de dirt abaixo (Y=0-9)
- Tamanho: 32x64x32 blocos

### 3. Simulação de Frames

Função `simulateFrames()` executa múltiplos frames de atualização:
- Simula 60 FPS (dt = 1/60)
- Permite testar física e movimento ao longo do tempo

## Suíte de Testes

### Categoria 1: Movimentação do Player

#### `TestPlayerMovement_Forward`
- **O que testa**: Movimento para frente (tecla W)
- **Validação**: Player se move ~4.3 unidades por segundo em Z
- **Cenário**: Player caminha por 1 segundo

#### `TestPlayerMovement_Backward`
- **O que testa**: Movimento para trás (tecla S)
- **Validação**: Player se move na direção -Z
- **Cenário**: Player anda para trás por 1 segundo

#### `TestPlayerMovement_Strafe`
- **O que testa**: Movimento lateral (tecla A)
- **Validação**: Player se move lateralmente em X
- **Cenário**: Movimento lateral por 1 segundo

#### `TestPlayerMovement_Collision`
- **O que testa**: Sistema de colisão com blocos
- **Validação**: Player não atravessa paredes
- **Cenário**: Player tenta caminhar através de uma parede

### Categoria 2: Pulo e Física

#### `TestPlayerJump`
- **O que testa**: Mecânica de pulo
- **Validação**:
  - Velocidade Y positiva após pular
  - Player sobe e depois volta ao chão
  - Flag `IsOnGround` atualiza corretamente
- **Cenário**: Player pula e cai

#### `TestPlayerJump_CannotDoubleJump`
- **O que testa**: Prevenir pulo duplo
- **Validação**: Player não pode pular enquanto está no ar
- **Cenário**: Tentar pular duas vezes seguidas

#### `TestPlayerJump_HeadCollision`
- **O que testa**: Colisão com teto
- **Validação**: Velocidade Y é zerada ao colidir com bloco acima
- **Cenário**: Player pula em área com teto baixo

#### `TestPlayerPhysics_Gravity`
- **O que testa**: Gravidade
- **Validação**: Player cai quando está no ar
- **Cenário**: Player spawna no ar (Y=20) e cai até o chão

#### `TestPlayerPhysics_DiagonalMovement`
- **O que testa**: Normalização de movimento diagonal
- **Validação**: Movimento diagonal não é mais rápido que movimento reto
- **Cenário**: Player se move pressionando W+D simultaneamente

### Categoria 3: Mira e Raycast

#### `TestPlayerAiming_LookAtBlock`
- **O que testa**: Detecção de blocos via raycast
- **Validação**:
  - `LookingAtBlock` é true quando mira no chão
  - `TargetBlock` contém coordenadas corretas
- **Cenário**: Player olha para baixo (Pitch=1.5) e detecta o chão

#### `TestPlayerAiming_NoBlockInRange`
- **O que testa**: Limite de alcance do raycast
- **Validação**: `LookingAtBlock` é false quando não há blocos
- **Cenário**: Player no ar olhando para cima

#### `TestPlayerAiming_MaxDistance`
- **O que testa**: Distância máxima de detecção (8 blocos)
- **Validação**: Blocos além de 8 blocos não são detectados
- **Cenário**: Bloco colocado a 14 blocos de distância

### Categoria 4: Adicionar Blocos

#### `TestPlayerPlaceBlock`
- **O que testa**: Colocação de blocos
- **Validação**:
  - Bloco é colocado na posição `PlaceBlock`
  - Tipo do bloco é `BlockStone`
- **Cenário**: Player olha para baixo e clica botão direito

#### `TestPlayerPlaceBlock_CannotPlaceWithoutTarget`
- **O que testa**: Prevenir colocação sem target
- **Validação**: Blocos não são criados quando `LookingAtBlock` é false
- **Cenário**: Player no ar olhando para cima tenta colocar bloco

#### `TestPlayerPlaceBlock_MultipleBlocks`
- **O que testa**: Colocação de múltiplos blocos
- **Validação**: Sistema permite colocar vários blocos em sequência
- **Cenário**: Player coloca blocos, move-se, e repete

### Categoria 5: Remover Blocos

#### `TestPlayerRemoveBlock`
- **O que testa**: Remoção de blocos
- **Validação**:
  - Bloco é removido (torna-se `BlockAir`)
  - `TargetBlock` aponta para o bloco correto
- **Cenário**: Player olha para o chão e clica botão esquerdo

#### `TestPlayerRemoveBlock_CannotRemoveWithoutTarget`
- **O que testa**: Prevenir remoção sem target
- **Validação**: Blocos não são removidos quando não há target
- **Cenário**: Player olhando para área vazia tenta remover

#### `TestPlayerRemoveBlock_TerrainModification`
- **O que testa**: Modificação de terreno e queda
- **Validação**: Player cai quando remove bloco abaixo dele
- **Cenário**: Player remove chão sob seus pés

## Como Executar os Testes

### Executar todos os testes
```bash
go test
```

### Executar com saída detalhada
```bash
go test -v
```

### Executar teste específico
```bash
go test -v -run TestPlayerJump
```

### Ver cobertura de código
```bash
go test -cover
```

### Gerar relatório de cobertura HTML
```bash
go test -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Helpers de Teste

### `createFlatWorld() *World`
Cria um mundo plano para testes consistentes.

### `simulateFrames(player, world, input, frames)`
Simula N frames de atualização a 60 FPS.

### `approximatelyEqual(a, b, epsilon) bool`
Compara floats com margem de erro.

## Boas Práticas

### Ao Adicionar Novas Features

1. **Escreva o teste primeiro** (TDD)
   ```go
   func TestNovaFeature(t *testing.T) {
       // Setup
       world := createFlatWorld()
       player := NewPlayer(...)

       // Ação
       // ... simular comportamento

       // Validação
       if resultado != esperado {
           t.Errorf("...")
       }
   }
   ```

2. **Execute todos os testes** antes de commitar
   ```bash
   go test -v
   ```

3. **Mantenha testes independentes** - cada teste deve funcionar isoladamente

4. **Use nomes descritivos** - `TestPlayerJump_CannotDoubleJump` é melhor que `TestJump2`

### Debugging de Testes

Se um teste falhar:

1. **Execute apenas aquele teste**:
   ```bash
   go test -v -run TestNomeFalhou
   ```

2. **Adicione logs temporários**:
   ```go
   t.Logf("Player pos: (%.2f, %.2f, %.2f)", player.Position.X, player.Position.Y, player.Position.Z)
   ```

3. **Verifique valores intermediários** com `t.Logf()` antes de validar

## Cobertura Atual

- ✅ Movimentação (W/A/S/D)
- ✅ Pulo e física de gravidade
- ✅ Colisão com blocos
- ✅ Raycast e detecção de blocos
- ✅ Adicionar blocos
- ✅ Remover blocos
- ✅ Movimento diagonal normalizado

## Próximos Passos

Features que ainda não têm testes:

- [ ] Geração de terreno procedural
- [ ] Diferentes tipos de blocos
- [ ] Inventário do player
- [ ] Renderização (difícil de testar headless)
- [ ] Performance com muitos blocos

## Estrutura de Arquivos

```
krakovia/
├── main.go          # Código principal do jogo
├── input.go         # Abstração de input (Input interface)
├── main_test.go     # Todos os testes
├── TESTING.md       # Esta documentação
└── go.mod           # Dependências
```

## Exemplo Completo de Teste

```go
func TestPlayerWalkAndJump(t *testing.T) {
    // 1. Setup: criar mundo e player
    world := createFlatWorld()
    player := NewPlayer(rl.NewVector3(16, 12, 16))
    input := &SimulatedInput{}

    // Estabilizar (player cai e pousa no chão)
    simulateFrames(player, world, input, 60)

    // 2. Ação: caminhar para frente
    input.Forward = true
    simulateFrames(player, world, input, 60) // 1 segundo
    input.Forward = false

    posAntesPulo := player.Position.Y

    // 3. Ação: pular
    input.Jump = true
    player.Update(1.0/60.0, world, input)

    // Simular subida
    simulateFrames(player, world, input, 20)

    // 4. Validação: player está mais alto
    if player.Position.Y <= posAntesPulo {
        t.Error("Player deveria ter subido após pular")
    }

    // 5. Validação: player se moveu em Z
    if player.Position.Z <= 16 {
        t.Error("Player deveria ter se movido para frente")
    }
}
```

---

**Última atualização**: 2025-11-06
**Testes implementados**: 18
**Taxa de sucesso**: 100% ✅
