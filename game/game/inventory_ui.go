package game

import (
	"fmt"
	"strings"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// InventoryUI representa a interface de inventário
type InventoryUI struct {
	// Estado
	IsOpen         bool
	CustomBlockMgr *CustomBlockManager
	Hotbar         *BlockHotbar

	// Pesquisa
	SearchBuffer   string
	IsSearching    bool

	// Blocos filtrados
	FilteredBlocks []BlockType

	// Scroll
	ScrollOffset   int

	// Drag and drop
	IsDragging     bool
	DraggedBlock   BlockType
	DragStartX     float32
	DragStartY     float32

	// Slot de hotbar selecionado para drop
	HoverHotbarSlot int

	// Cache de texturas carregadas
	TextureCache map[uint16]rl.Texture2D
}

// NewInventoryUI cria uma nova interface de inventário
func NewInventoryUI(cbm *CustomBlockManager, hotbar *BlockHotbar) *InventoryUI {
	inv := &InventoryUI{
		IsOpen:          false,
		CustomBlockMgr:  cbm,
		Hotbar:          hotbar,
		FilteredBlocks:  make([]BlockType, 0),
		HoverHotbarSlot: -1,
		TextureCache:    make(map[uint16]rl.Texture2D),
	}

	inv.refreshBlockList()
	return inv
}

// Toggle abre/fecha o inventário
func (inv *InventoryUI) Toggle() {
	inv.IsOpen = !inv.IsOpen
	if inv.IsOpen {
		rl.EnableCursor()
		inv.loadTextures()
		inv.refreshBlockList()
	} else {
		rl.DisableCursor()
		inv.IsDragging = false
		inv.IsSearching = false
	}
}

// loadTextures carrega as texturas dos blocos customizados para o cache
func (inv *InventoryUI) loadTextures() {
	// Limpar cache anterior
	for _, tex := range inv.TextureCache {
		if tex.ID != 0 {
			rl.UnloadTexture(tex)
		}
	}
	inv.TextureCache = make(map[uint16]rl.Texture2D)

	// Carregar texturas de todos os blocos customizados (incluindo o default que é ID 256)
	customBlocks := inv.CustomBlockMgr.ListBlocks()
	for _, block := range customBlocks {
		if block.FaceTextures[FaceFront] != "" {
			tex := rl.LoadTexture(block.FaceTextures[FaceFront])
			if tex.ID != 0 {
				inv.TextureCache[block.ID] = tex
			}
		}
	}
}

// refreshBlockList atualiza a lista de blocos disponíveis
func (inv *InventoryUI) refreshBlockList() {
	inv.FilteredBlocks = make([]BlockType, 0)

	// Adicionar todos os blocos customizados (incluindo o default com ID 256)
	customBlocks := inv.CustomBlockMgr.ListBlocks()
	for _, block := range customBlocks {
		inv.FilteredBlocks = append(inv.FilteredBlocks, BlockType(block.ID))
	}

	// Aplicar filtro de pesquisa
	if inv.SearchBuffer != "" {
		filtered := make([]BlockType, 0)
		searchLower := strings.ToLower(inv.SearchBuffer)

		for _, blockType := range inv.FilteredBlocks {
			name := inv.getBlockName(blockType)
			if strings.Contains(strings.ToLower(name), searchLower) {
				filtered = append(filtered, blockType)
			}
		}

		inv.FilteredBlocks = filtered
	}

	// Limitar a 30 blocos para performance
	if len(inv.FilteredBlocks) > 30 {
		inv.FilteredBlocks = inv.FilteredBlocks[:30]
	}
}

// getBlockName retorna o nome de um bloco
func (inv *InventoryUI) getBlockName(blockType BlockType) string {
	if blockType == NoBlock {
		return "Vazio"
	}

	// Todos os blocos são customizados agora (ID >= 256)
	block := inv.CustomBlockMgr.GetBlock(uint16(blockType))
	if block != nil {
		return block.Name
	}
	return fmt.Sprintf("Block %d", blockType)
}

// Update atualiza a lógica do inventário
func (inv *InventoryUI) Update() {
	if !inv.IsOpen {
		return
	}

	// Processar input de pesquisa
	if inv.IsSearching {
		char := rl.GetCharPressed()
		for char > 0 {
			if char >= 32 && char <= 126 && len(inv.SearchBuffer) < 20 {
				inv.SearchBuffer += string(rune(char))
				inv.refreshBlockList()
			}
			char = rl.GetCharPressed()
		}

		if rl.IsKeyPressed(rl.KeyBackspace) && len(inv.SearchBuffer) > 0 {
			inv.SearchBuffer = inv.SearchBuffer[:len(inv.SearchBuffer)-1]
			inv.refreshBlockList()
		}

		if rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyEscape) {
			inv.IsSearching = false
		}
	}

	// Scroll
	wheel := rl.GetMouseWheelMove()
	if wheel != 0 {
		inv.ScrollOffset -= int(wheel) * 2
		if inv.ScrollOffset < 0 {
			inv.ScrollOffset = 0
		}
		maxScroll := len(inv.FilteredBlocks)/8 - 4
		if maxScroll < 0 {
			maxScroll = 0
		}
		if inv.ScrollOffset > maxScroll {
			inv.ScrollOffset = maxScroll
		}
	}

	// Atualizar drag
	if inv.IsDragging {
		if rl.IsMouseButtonReleased(rl.MouseLeftButton) {
			// Soltar no hotbar
			if inv.HoverHotbarSlot >= 0 && inv.HoverHotbarSlot < 9 {
				inv.Hotbar.SetSlot(inv.HoverHotbarSlot, inv.DraggedBlock)
			}
			inv.IsDragging = false
		}
	}

	// Verificar hover no hotbar
	inv.updateHotbarHover()
}

// updateHotbarHover verifica se o mouse está sobre um slot do hotbar
func (inv *InventoryUI) updateHotbarHover() {
	mousePos := rl.GetMousePosition()

	slotSize := int32(50)
	slotSpacing := int32(5)
	totalWidth := 9*slotSize + 8*slotSpacing
	startX := (ScreenWidth - totalWidth) / 2
	startY := ScreenHeight - slotSize - 20

	inv.HoverHotbarSlot = -1

	for i := 0; i < 9; i++ {
		x := startX + int32(i)*(slotSize+slotSpacing)
		rect := rl.NewRectangle(float32(x), float32(startY), float32(slotSize), float32(slotSize))

		if rl.CheckCollisionPointRec(mousePos, rect) {
			inv.HoverHotbarSlot = i
			break
		}
	}
}

// Render desenha o inventário
func (inv *InventoryUI) Render() {
	if !inv.IsOpen {
		return
	}

	// Fundo semi-transparente
	rl.DrawRectangle(0, 0, ScreenWidth, ScreenHeight, rl.NewColor(0, 0, 0, 180))

	// Título
	title := "Inventário"
	titleWidth := rl.MeasureText(title, 30)
	rl.DrawText(title, (ScreenWidth-titleWidth)/2, 30, 30, rl.White)

	// Barra de pesquisa
	searchX := int32(50)
	searchY := int32(80)
	searchWidth := int32(300)
	searchHeight := int32(35)

	searchRect := rl.NewRectangle(float32(searchX), float32(searchY), float32(searchWidth), float32(searchHeight))
	rl.DrawRectangleRec(searchRect, rl.NewColor(50, 50, 50, 255))

	borderColor := rl.White
	if inv.IsSearching {
		borderColor = rl.Yellow
	}
	rl.DrawRectangleLinesEx(searchRect, 2, borderColor)

	// Ícone de pesquisa
	rl.DrawText("Pesquisar:", searchX, searchY-20, 14, rl.Gray)

	displayText := inv.SearchBuffer
	if inv.IsSearching && int(rl.GetTime()*2)%2 == 0 {
		displayText += "_"
	}
	if displayText == "" && !inv.IsSearching {
		rl.DrawText("Clique para pesquisar...", searchX+10, searchY+8, 16, rl.Gray)
	} else {
		rl.DrawText(displayText, searchX+10, searchY+8, 16, rl.White)
	}

	// Detectar clique na barra de pesquisa
	if rl.CheckCollisionPointRec(rl.GetMousePosition(), searchRect) && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		inv.IsSearching = true
	}

	// Grade de blocos
	gridX := int32(50)
	gridY := int32(130)
	blockSize := int32(60)
	blockSpacing := int32(10)
	blocksPerRow := 8

	// Fundo da grade
	gridWidth := int32(blocksPerRow)*(blockSize+blockSpacing) - blockSpacing
	gridHeight := int32(300)
	rl.DrawRectangle(gridX-5, gridY-5, gridWidth+10, gridHeight+10, rl.NewColor(40, 40, 40, 255))

	// Desenhar blocos
	visibleRows := 4
	startIdx := inv.ScrollOffset * blocksPerRow
	endIdx := startIdx + visibleRows*blocksPerRow

	if endIdx > len(inv.FilteredBlocks) {
		endIdx = len(inv.FilteredBlocks)
	}

	for i := startIdx; i < endIdx; i++ {
		blockType := inv.FilteredBlocks[i]
		col := (i - startIdx) % blocksPerRow
		row := (i - startIdx) / blocksPerRow

		x := gridX + int32(col)*(blockSize+blockSpacing)
		y := gridY + int32(row)*(blockSize+blockSpacing)

		inv.drawBlockSlot(blockType, x, y, blockSize)
	}

	// Indicador de scroll
	if len(inv.FilteredBlocks) > visibleRows*blocksPerRow {
		totalRows := (len(inv.FilteredBlocks) + blocksPerRow - 1) / blocksPerRow
		scrollText := fmt.Sprintf("Scroll: %d/%d", inv.ScrollOffset+1, totalRows-visibleRows+1)
		rl.DrawText(scrollText, gridX, gridY+gridHeight+10, 14, rl.Gray)
	}

	// Mostrar quantidade de blocos
	totalBlocks := len(inv.CustomBlockMgr.ListBlocks())
	countText := fmt.Sprintf("%d/%d blocos", len(inv.FilteredBlocks), totalBlocks)
	rl.DrawText(countText, gridX+gridWidth-120, searchY+8, 16, rl.Gray)

	// Hotbar (na parte inferior)
	inv.renderHotbar()

	// Desenhar bloco sendo arrastado
	if inv.IsDragging {
		mousePos := rl.GetMousePosition()
		inv.drawBlockIcon(inv.DraggedBlock, int32(mousePos.X)-25, int32(mousePos.Y)-25, 50)
	}

	// Instruções
	rl.DrawText("Clique e arraste blocos para o hotbar | ESC para fechar | Scroll para navegar", 50, ScreenHeight-30, 14, rl.Gray)
}

// drawBlockSlot desenha um slot de bloco na grade
func (inv *InventoryUI) drawBlockSlot(blockType BlockType, x, y, size int32) {
	rect := rl.NewRectangle(float32(x), float32(y), float32(size), float32(size))
	mousePos := rl.GetMousePosition()
	isHover := rl.CheckCollisionPointRec(mousePos, rect)

	// Cor de fundo
	bgColor := rl.NewColor(60, 60, 60, 255)
	if isHover {
		bgColor = rl.NewColor(80, 80, 100, 255)
	}
	rl.DrawRectangleRec(rect, bgColor)

	// Desenhar ícone do bloco
	inv.drawBlockIcon(blockType, x+5, y+5, size-10)

	// Borda
	rl.DrawRectangleLinesEx(rect, 1, rl.White)

	// Nome do bloco (tooltip)
	if isHover {
		name := inv.getBlockName(blockType)
		nameWidth := rl.MeasureText(name, 12)
		tooltipX := x + (size-nameWidth)/2
		tooltipY := y + size + 2

		// Fundo do tooltip
		rl.DrawRectangle(tooltipX-2, tooltipY-1, nameWidth+4, 14, rl.NewColor(0, 0, 0, 200))
		rl.DrawText(name, tooltipX, tooltipY, 12, rl.White)
	}

	// Iniciar drag
	if isHover && rl.IsMouseButtonPressed(rl.MouseLeftButton) && !inv.IsDragging {
		inv.IsDragging = true
		inv.DraggedBlock = blockType
		inv.DragStartX = mousePos.X
		inv.DragStartY = mousePos.Y
	}
}

// drawBlockIcon desenha o ícone de um bloco
func (inv *InventoryUI) drawBlockIcon(blockType BlockType, x, y, size int32) {
	if blockType == NoBlock {
		return
	}

	// Todos os blocos são customizados agora (ID >= 256)
	texID := uint16(blockType)

	if tex, exists := inv.TextureCache[texID]; exists && tex.ID != 0 {
		rl.DrawTexturePro(tex,
			rl.NewRectangle(0, 0, float32(tex.Width), float32(tex.Height)),
			rl.NewRectangle(float32(x), float32(y), float32(size), float32(size)),
			rl.NewVector2(0, 0), 0, rl.White)
		return
	}

	// Fallback: cor cinza
	rl.DrawRectangle(x, y, size, size, rl.Gray)
}

// renderHotbar desenha o hotbar dentro do inventário
func (inv *InventoryUI) renderHotbar() {
	slotSize := int32(50)
	slotSpacing := int32(5)
	totalWidth := 9*slotSize + 8*slotSpacing
	startX := (ScreenWidth - totalWidth) / 2
	startY := ScreenHeight - slotSize - 60

	// Label
	rl.DrawText("Hotbar (arraste blocos aqui):", startX, startY-25, 16, rl.White)

	for i := 0; i < 9; i++ {
		x := startX + int32(i)*(slotSize+slotSpacing)
		y := startY

		// Cor de fundo
		bgColor := rl.NewColor(50, 50, 50, 200)
		if i == inv.HoverHotbarSlot && inv.IsDragging {
			bgColor = rl.NewColor(80, 120, 80, 220)
		}

		// Desenhar slot
		rl.DrawRectangle(x, y, slotSize, slotSize, bgColor)

		// Borda
		borderColor := rl.White
		if i == inv.Hotbar.SelectedSlot {
			borderColor = rl.Yellow
		}
		rl.DrawRectangleLines(x, y, slotSize, slotSize, borderColor)

		// Número do slot
		rl.DrawText(fmt.Sprintf("%d", i+1), x+3, y+3, 12, rl.Gray)

		// Desenhar bloco no slot
		blockType := inv.Hotbar.Slots[i]
		if blockType != NoBlock {
			inv.drawBlockIcon(blockType, x+5, y+5, slotSize-10)
		}
	}
}

// Close fecha o inventário
func (inv *InventoryUI) Close() {
	if inv.IsOpen {
		inv.IsOpen = false
		rl.DisableCursor()
		inv.IsDragging = false
		inv.IsSearching = false
		// Limpar cache de texturas
		for _, tex := range inv.TextureCache {
			if tex.ID != 0 {
				rl.UnloadTexture(tex)
			}
		}
		inv.TextureCache = make(map[uint16]rl.Texture2D)
	}
}
