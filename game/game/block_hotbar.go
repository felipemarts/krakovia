package game

import (
	"fmt"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// BlockHotbar representa a barra de seleção de blocos do jogador
type BlockHotbar struct {
	// Slots do hotbar (BlockType)
	Slots [9]BlockType

	// Índice do slot selecionado
	SelectedSlot int

	// Referência ao gerenciador de blocos customizados
	CustomBlocks *CustomBlockManager

	// Orientação atual para colocação
	PlacementOrientation BlockOrientation
}

// NewBlockHotbar cria um novo hotbar
func NewBlockHotbar(cbm *CustomBlockManager) *BlockHotbar {
	hb := &BlockHotbar{
		SelectedSlot: 0,
		CustomBlocks: cbm,
	}

	// Slot 0 é sempre BlockGrass por padrão
	hb.Slots[0] = BlockGrass

	// Outros slots começam vazios (BlockAir)
	for i := 1; i < 9; i++ {
		hb.Slots[i] = BlockAir
	}

	return hb
}

// Update atualiza o hotbar (seleção de slot)
func (hb *BlockHotbar) Update() {
	// Teclas 1-9 para selecionar slot
	for i := 0; i < 9; i++ {
		if rl.IsKeyPressed(int32(rl.KeyOne + i)) {
			hb.SelectedSlot = i
		}
	}

	// Scroll do mouse para mudar slot
	wheel := rl.GetMouseWheelMove()
	if wheel != 0 {
		hb.SelectedSlot -= int(wheel)
		if hb.SelectedSlot < 0 {
			hb.SelectedSlot = 8
		} else if hb.SelectedSlot > 8 {
			hb.SelectedSlot = 0
		}
	}

	// R para rotacionar orientação
	if rl.IsKeyPressed(rl.KeyR) {
		hb.PlacementOrientation = (hb.PlacementOrientation + 1) % 4
	}
}

// GetSelectedBlock retorna o bloco selecionado atualmente
func (hb *BlockHotbar) GetSelectedBlock() BlockType {
	return hb.Slots[hb.SelectedSlot]
}

// SetSlot define um bloco em um slot específico
func (hb *BlockHotbar) SetSlot(slot int, blockType BlockType) {
	if slot >= 0 && slot < 9 {
		hb.Slots[slot] = blockType
	}
}

// AddCustomBlock adiciona um bloco customizado ao primeiro slot vazio
func (hb *BlockHotbar) AddCustomBlock(blockID uint16) bool {
	blockType := BlockType(blockID)

	// Procurar slot vazio
	for i := 0; i < 9; i++ {
		if hb.Slots[i] == BlockAir {
			hb.Slots[i] = blockType
			return true
		}
	}

	// Se não houver slot vazio, substituir o último
	hb.Slots[8] = blockType
	return true
}

// Render desenha o hotbar na tela
func (hb *BlockHotbar) Render() {
	// Posição do hotbar (centralizado na parte inferior)
	slotSize := int32(50)
	slotSpacing := int32(5)
	totalWidth := 9*slotSize + 8*slotSpacing
	startX := (ScreenWidth - totalWidth) / 2
	startY := ScreenHeight - slotSize - 20

	for i := 0; i < 9; i++ {
		x := startX + int32(i)*(slotSize+slotSpacing)
		y := startY

		// Cor de fundo
		bgColor := rl.NewColor(50, 50, 50, 200)
		if i == hb.SelectedSlot {
			bgColor = rl.NewColor(100, 100, 150, 220)
		}

		// Desenhar slot
		rl.DrawRectangle(x, y, slotSize, slotSize, bgColor)

		// Borda
		borderColor := rl.White
		if i == hb.SelectedSlot {
			borderColor = rl.Yellow
		}
		rl.DrawRectangleLines(x, y, slotSize, slotSize, borderColor)

		// Número do slot
		rl.DrawText(fmt.Sprintf("%d", i+1), x+3, y+3, 12, rl.Gray)

		// Desenhar representação do bloco
		blockType := hb.Slots[i]
		if blockType != BlockAir {
			// Cor do bloco
			var blockColor rl.Color
			var blockName string

			if IsCustomBlock(blockType) {
				// Bloco customizado
				blockID := GetCustomBlockID(blockType)
				block := hb.CustomBlocks.GetBlock(blockID)
				if block != nil {
					blockName = block.Name
					// Usar primeira letra como identificador
					if len(blockName) > 0 {
						rl.DrawText(string(blockName[0]), x+slotSize/2-5, y+slotSize/2-10, 20, rl.White)
					}
				}
				blockColor = rl.NewColor(100, 200, 100, 255)
			} else {
				// Bloco padrão
				switch blockType {
				case BlockGrass:
					blockColor = rl.NewColor(100, 200, 100, 255)
					blockName = "Grass"
				default:
					blockColor = rl.Gray
					blockName = "?"
				}
			}

			// Desenhar mini-bloco
			innerSize := slotSize - 10
			rl.DrawRectangle(x+5, y+5, innerSize, innerSize, blockColor)
		}
	}

	// Mostrar orientação atual
	orientations := []string{"N", "E", "S", "W"}
	orientationText := fmt.Sprintf("Orientação: %s (R para rotacionar)", orientations[hb.PlacementOrientation])
	textWidth := rl.MeasureText(orientationText, 14)
	rl.DrawText(orientationText, (ScreenWidth-textWidth)/2, startY-25, 14, rl.White)

	// Mostrar nome do bloco selecionado
	selectedBlock := hb.GetSelectedBlock()
	var selectedName string
	if selectedBlock == BlockAir {
		selectedName = "Vazio"
	} else if IsCustomBlock(selectedBlock) {
		blockID := GetCustomBlockID(selectedBlock)
		block := hb.CustomBlocks.GetBlock(blockID)
		if block != nil {
			selectedName = block.Name
		} else {
			selectedName = "Custom"
		}
	} else {
		switch selectedBlock {
		case BlockGrass:
			selectedName = "Grass"
		default:
			selectedName = "Unknown"
		}
	}

	nameText := fmt.Sprintf("Selecionado: %s", selectedName)
	nameWidth := rl.MeasureText(nameText, 16)
	rl.DrawText(nameText, (ScreenWidth-nameWidth)/2, startY-45, 16, rl.Yellow)
}
