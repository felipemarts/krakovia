package game

import (
	"fmt"
	"image"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// UnifiedTab representa as abas da interface unificada
type UnifiedTab int

const (
	TabInventory UnifiedTab = iota
	TabBlockEditor
	// TabSettings // Futuro
)

// UnifiedEditorState representa os sub-estados do editor de blocos
type UnifiedEditorState int

const (
	EditorSubStateMain UnifiedEditorState = iota
	EditorSubStateTextureManager
	EditorSubStateTexturePaint
	EditorSubStateTextureUpload
	EditorSubStateBlockList
	EditorSubStateBlockCreate
	EditorSubStateBlockEdit
	EditorSubStateSelectTexture
	EditorSubStateFileBrowser
)

// UnifiedInventoryUI representa a interface unificada de inventário e editor
type UnifiedInventoryUI struct {
	// Estado geral
	IsOpen    bool
	ActiveTab UnifiedTab

	// Componentes compartilhados
	CustomBlockMgr *CustomBlockManager
	Hotbar         *BlockHotbar
	TextureMgr     *TextureManager
	Painter        *TexturePainter

	// === Aba Inventário ===
	// Pesquisa
	SearchBuffer string
	IsSearching  bool

	// Blocos filtrados
	FilteredBlocks []BlockType

	// Scroll do inventário
	InventoryScroll int

	// Drag and drop
	IsDragging     bool
	DraggedBlock   BlockType
	DragStartX     float32
	DragStartY     float32

	// Slot de hotbar selecionado para drop
	HoverHotbarSlot int

	// Cache de texturas para inventário
	InventoryTextureCache map[uint16]rl.Texture2D

	// === Aba Editor de Blocos ===
	EditorState UnifiedEditorState

	// Bloco sendo editado
	CurrentBlock     *CustomBlockDefinition
	CurrentBlockName string

	// Face selecionada
	SelectedFace BlockFace

	// Texturas carregadas para preview (Raylib textures)
	FaceTextures       [6]rl.Texture2D
	FaceTexturesLoaded [6]bool
	FaceTextureNames   [6]string

	// UI state do editor
	EditorScrollOffset   int
	HoverButton          int
	SelectedBlockIdx     int
	TextureListScroll    int

	// Input state
	InputBuffer string
	IsTyping    bool

	// File dialog
	FilePaths      []string
	FileListScroll int
	CurrentDir     string

	// Messages
	Message      string
	MessageTimer float32

	// Callback para notificar quando um bloco é salvo
	OnBlockSaved func(block *CustomBlockDefinition)

	// Cache de texturas do editor
	EditorTextureCache map[string]rl.Texture2D
}

// NewUnifiedInventoryUI cria uma nova interface unificada
func NewUnifiedInventoryUI(cbm *CustomBlockManager, hotbar *BlockHotbar) *UnifiedInventoryUI {
	ui := &UnifiedInventoryUI{
		IsOpen:                false,
		ActiveTab:             TabInventory,
		CustomBlockMgr:        cbm,
		Hotbar:                hotbar,
		TextureMgr:            NewTextureManager(),
		Painter:               NewTexturePainter(),
		FilteredBlocks:        make([]BlockType, 0),
		HoverHotbarSlot:       -1,
		InventoryTextureCache: make(map[uint16]rl.Texture2D),
		EditorState:           EditorSubStateMain,
		SelectedFace:          FaceFront,
		CurrentDir:            ".",
		EditorTextureCache:    make(map[string]rl.Texture2D),
	}

	ui.refreshBlockList()
	return ui
}

// Toggle abre/fecha a interface unificada
func (ui *UnifiedInventoryUI) Toggle() {
	ui.IsOpen = !ui.IsOpen
	if ui.IsOpen {
		rl.EnableCursor()
		ui.loadInventoryTextures()
		ui.loadEditorTextureCache()
		ui.refreshBlockList()
	} else {
		ui.Close()
	}
}

// Close fecha a interface
func (ui *UnifiedInventoryUI) Close() {
	ui.IsOpen = false
	ui.IsDragging = false
	ui.IsSearching = false
	ui.IsTyping = false
	ui.EditorState = EditorSubStateMain
	ui.CurrentBlock = nil
	ui.Message = ""

	// Limpar cache de texturas do inventário
	for _, tex := range ui.InventoryTextureCache {
		if tex.ID != 0 {
			rl.UnloadTexture(tex)
		}
	}
	ui.InventoryTextureCache = make(map[uint16]rl.Texture2D)

	// Limpar cache de texturas do editor
	for _, tex := range ui.EditorTextureCache {
		if tex.ID != 0 {
			rl.UnloadTexture(tex)
		}
	}
	ui.EditorTextureCache = make(map[string]rl.Texture2D)

	// Descarregar texturas de preview das faces
	for i := 0; i < 6; i++ {
		if ui.FaceTexturesLoaded[i] {
			rl.UnloadTexture(ui.FaceTextures[i])
			ui.FaceTexturesLoaded[i] = false
		}
	}

	rl.DisableCursor()
}

// loadInventoryTextures carrega as texturas dos blocos customizados para o inventário
func (ui *UnifiedInventoryUI) loadInventoryTextures() {
	// Limpar cache anterior
	for _, tex := range ui.InventoryTextureCache {
		if tex.ID != 0 {
			rl.UnloadTexture(tex)
		}
	}
	ui.InventoryTextureCache = make(map[uint16]rl.Texture2D)

	// Carregar texturas de todos os blocos customizados
	customBlocks := ui.CustomBlockMgr.ListBlocks()
	for _, block := range customBlocks {
		if block.FaceTextures[FaceFront] != "" {
			tex := rl.LoadTexture(block.FaceTextures[FaceFront])
			if tex.ID != 0 {
				ui.InventoryTextureCache[block.ID] = tex
			}
		}
	}
}

// loadEditorTextureCache carrega todas as texturas para o cache do editor
func (ui *UnifiedInventoryUI) loadEditorTextureCache() {
	// Limpar cache anterior
	for _, tex := range ui.EditorTextureCache {
		if tex.ID != 0 {
			rl.UnloadTexture(tex)
		}
	}
	ui.EditorTextureCache = make(map[string]rl.Texture2D)

	// Carregar texturas do TextureManager
	for _, name := range ui.TextureMgr.ListTextures() {
		texPath := ui.TextureMgr.GetTexturePath(name)
		if texPath != "" {
			tex := rl.LoadTexture(texPath)
			if tex.ID != 0 {
				ui.EditorTextureCache[name] = tex
			}
		}
	}
}

// refreshBlockList atualiza a lista de blocos disponíveis
func (ui *UnifiedInventoryUI) refreshBlockList() {
	ui.FilteredBlocks = make([]BlockType, 0)

	// Adicionar todos os blocos customizados
	customBlocks := ui.CustomBlockMgr.ListBlocks()
	for _, block := range customBlocks {
		ui.FilteredBlocks = append(ui.FilteredBlocks, BlockType(block.ID))
	}

	// Aplicar filtro de pesquisa
	if ui.SearchBuffer != "" {
		filtered := make([]BlockType, 0)
		searchLower := strings.ToLower(ui.SearchBuffer)

		for _, blockType := range ui.FilteredBlocks {
			name := ui.getBlockName(blockType)
			if strings.Contains(strings.ToLower(name), searchLower) {
				filtered = append(filtered, blockType)
			}
		}

		ui.FilteredBlocks = filtered
	}

	// Limitar a 30 blocos para performance
	if len(ui.FilteredBlocks) > 30 {
		ui.FilteredBlocks = ui.FilteredBlocks[:30]
	}
}

// getBlockName retorna o nome de um bloco
func (ui *UnifiedInventoryUI) getBlockName(blockType BlockType) string {
	if blockType == NoBlock {
		return "Vazio"
	}

	block := ui.CustomBlockMgr.GetBlock(uint16(blockType))
	if block != nil {
		return block.Name
	}
	return fmt.Sprintf("Block %d", blockType)
}

// Update atualiza a lógica da interface
func (ui *UnifiedInventoryUI) Update(dt float32) {
	if !ui.IsOpen {
		return
	}

	// Atualizar timer de mensagem
	if ui.MessageTimer > 0 {
		ui.MessageTimer -= dt
		if ui.MessageTimer <= 0 {
			ui.Message = ""
		}
	}

	// ESC para voltar/fechar
	if rl.IsKeyPressed(rl.KeyEscape) {
		ui.handleEscape()
		return
	}

	// Alternar abas com Tab
	if rl.IsKeyPressed(rl.KeyTab) && !ui.IsTyping && !ui.IsSearching {
		if ui.EditorState == EditorSubStateMain || ui.ActiveTab == TabInventory {
			if ui.ActiveTab == TabInventory {
				ui.ActiveTab = TabBlockEditor
			} else {
				ui.ActiveTab = TabInventory
			}
		}
	}

	// Atualizar conforme a aba ativa
	switch ui.ActiveTab {
	case TabInventory:
		ui.updateInventory()
	case TabBlockEditor:
		ui.updateBlockEditor(dt)
	}
}

// handleEscape trata o pressionamento de ESC
func (ui *UnifiedInventoryUI) handleEscape() {
	switch ui.ActiveTab {
	case TabInventory:
		if ui.IsSearching {
			ui.IsSearching = false
		} else {
			ui.Close()
		}
	case TabBlockEditor:
		switch ui.EditorState {
		case EditorSubStateMain:
			ui.Close()
		case EditorSubStateTextureManager, EditorSubStateBlockList:
			ui.EditorState = EditorSubStateMain
		case EditorSubStateTexturePaint, EditorSubStateTextureUpload:
			ui.EditorState = EditorSubStateTextureManager
			ui.IsTyping = false
		case EditorSubStateBlockCreate, EditorSubStateBlockEdit:
			ui.EditorState = EditorSubStateBlockList
			ui.IsTyping = false
		case EditorSubStateSelectTexture:
			ui.EditorState = EditorSubStateBlockEdit
		case EditorSubStateFileBrowser:
			ui.EditorState = EditorSubStateTextureUpload
		}
	}
}

// updateInventory atualiza a lógica do inventário
func (ui *UnifiedInventoryUI) updateInventory() {
	// Processar input de pesquisa
	if ui.IsSearching {
		char := rl.GetCharPressed()
		for char > 0 {
			if char >= 32 && char <= 126 && len(ui.SearchBuffer) < 20 {
				ui.SearchBuffer += string(rune(char))
				ui.refreshBlockList()
			}
			char = rl.GetCharPressed()
		}

		if rl.IsKeyPressed(rl.KeyBackspace) && len(ui.SearchBuffer) > 0 {
			ui.SearchBuffer = ui.SearchBuffer[:len(ui.SearchBuffer)-1]
			ui.refreshBlockList()
		}

		if rl.IsKeyPressed(rl.KeyEnter) {
			ui.IsSearching = false
		}
	}

	// Scroll
	wheel := rl.GetMouseWheelMove()
	if wheel != 0 {
		ui.InventoryScroll -= int(wheel) * 2
		if ui.InventoryScroll < 0 {
			ui.InventoryScroll = 0
		}
		maxScroll := len(ui.FilteredBlocks)/8 - 4
		if maxScroll < 0 {
			maxScroll = 0
		}
		if ui.InventoryScroll > maxScroll {
			ui.InventoryScroll = maxScroll
		}
	}

	// Atualizar drag
	if ui.IsDragging {
		if rl.IsMouseButtonReleased(rl.MouseLeftButton) {
			// Soltar no hotbar
			if ui.HoverHotbarSlot >= 0 && ui.HoverHotbarSlot < 9 {
				ui.Hotbar.SetSlot(ui.HoverHotbarSlot, ui.DraggedBlock)
			}
			ui.IsDragging = false
		}
	}

	// Verificar hover no hotbar
	ui.updateHotbarHover()
}

// updateHotbarHover verifica se o mouse está sobre um slot do hotbar
func (ui *UnifiedInventoryUI) updateHotbarHover() {
	mousePos := rl.GetMousePosition()

	slotSize := int32(50)
	slotSpacing := int32(5)
	totalWidth := 9*slotSize + 8*slotSpacing
	startX := (ScreenWidth - totalWidth) / 2
	// Deve corresponder à posição em renderHotbar()
	startY := ScreenHeight - slotSize - 90

	ui.HoverHotbarSlot = -1

	for i := 0; i < 9; i++ {
		x := startX + int32(i)*(slotSize+slotSpacing)
		rect := rl.NewRectangle(float32(x), float32(startY), float32(slotSize), float32(slotSize))

		if rl.CheckCollisionPointRec(mousePos, rect) {
			ui.HoverHotbarSlot = i
			break
		}
	}
}

// updateBlockEditor atualiza a lógica do editor de blocos
func (ui *UnifiedInventoryUI) updateBlockEditor(dt float32) {
	// Atualizar painter se estiver nesse estado
	if ui.EditorState == EditorSubStateTexturePaint {
		ui.Painter.Update()
	}

	// Processar input de texto se estiver digitando
	if ui.IsTyping {
		char := rl.GetCharPressed()
		for char > 0 {
			if char >= 32 && char <= 126 && len(ui.InputBuffer) < 32 {
				ui.InputBuffer += string(rune(char))
			}
			char = rl.GetCharPressed()
		}

		if rl.IsKeyPressed(rl.KeyBackspace) && len(ui.InputBuffer) > 0 {
			ui.InputBuffer = ui.InputBuffer[:len(ui.InputBuffer)-1]
		}

		if rl.IsKeyPressed(rl.KeyEnter) {
			ui.IsTyping = false
			ui.confirmEditorInput()
		}
	}
}

// confirmEditorInput confirma o input atual do editor
func (ui *UnifiedInventoryUI) confirmEditorInput() {
	switch ui.EditorState {
	case EditorSubStateTexturePaint:
		if ui.InputBuffer != "" {
			// Salvar textura pintada
			img := ui.Painter.ToImage()
			err := ui.TextureMgr.SaveTexture(ui.InputBuffer, img)
			if err != nil {
				ui.showMessage(fmt.Sprintf("Erro: %s", err))
			} else {
				ui.showMessage(fmt.Sprintf("Textura '%s' salva!", ui.InputBuffer))
				ui.loadEditorTextureCache()
				ui.EditorState = EditorSubStateTextureManager
			}
		}
	case EditorSubStateBlockCreate:
		if ui.InputBuffer != "" {
			ui.createNewBlock(ui.InputBuffer)
		}
	}
}

// Render desenha a interface
func (ui *UnifiedInventoryUI) Render() {
	if !ui.IsOpen {
		return
	}

	// Fundo semi-transparente
	rl.DrawRectangle(0, 0, ScreenWidth, ScreenHeight, rl.NewColor(0, 0, 0, 200))

	// Desenhar abas
	ui.renderTabs()

	// Desenhar conteúdo da aba ativa
	switch ui.ActiveTab {
	case TabInventory:
		ui.renderInventoryContent()
	case TabBlockEditor:
		ui.renderBlockEditorContent()
	}

	// Mensagem de feedback
	if ui.Message != "" {
		msgWidth := rl.MeasureText(ui.Message, 20)
		rl.DrawRectangle((ScreenWidth-msgWidth)/2-10, ScreenHeight-60, msgWidth+20, 30, rl.NewColor(0, 100, 0, 200))
		rl.DrawText(ui.Message, (ScreenWidth-msgWidth)/2, ScreenHeight-55, 20, rl.White)
	}

	// Instruções
	rl.DrawText("ESC - Voltar/Fechar | Tab - Alternar abas", 10, ScreenHeight-30, 16, rl.Gray)
}

// renderTabs desenha as abas
func (ui *UnifiedInventoryUI) renderTabs() {
	tabWidth := int32(200)
	tabHeight := int32(40)
	startX := int32(50)
	startY := int32(10)

	tabs := []struct {
		name string
		tab  UnifiedTab
	}{
		{"Inventario", TabInventory},
		{"Editor de Blocos", TabBlockEditor},
	}

	for i, tab := range tabs {
		x := startX + int32(i)*(tabWidth+5)
		rect := rl.NewRectangle(float32(x), float32(startY), float32(tabWidth), float32(tabHeight))

		// Cor de fundo
		bgColor := rl.NewColor(50, 50, 50, 255)
		if ui.ActiveTab == tab.tab {
			bgColor = rl.NewColor(80, 80, 120, 255)
		}

		mousePos := rl.GetMousePosition()
		isHover := rl.CheckCollisionPointRec(mousePos, rect)
		if isHover {
			bgColor.R = uint8(minInt(int(bgColor.R)+20, 255))
			bgColor.G = uint8(minInt(int(bgColor.G)+20, 255))
			bgColor.B = uint8(minInt(int(bgColor.B)+20, 255))
		}

		rl.DrawRectangleRec(rect, bgColor)

		// Borda
		borderColor := rl.White
		if ui.ActiveTab == tab.tab {
			borderColor = rl.Yellow
		}
		rl.DrawRectangleLinesEx(rect, 2, borderColor)

		// Texto
		textWidth := rl.MeasureText(tab.name, 18)
		textX := x + (tabWidth-textWidth)/2
		textY := startY + (tabHeight-18)/2
		rl.DrawText(tab.name, textX, textY, 18, rl.White)

		// Detectar clique
		if isHover && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			// Só permite trocar de aba se não estiver em sub-estado do editor
			if ui.EditorState == EditorSubStateMain || ui.ActiveTab == TabInventory {
				ui.ActiveTab = tab.tab
			}
		}
	}
}

// renderInventoryContent desenha o conteúdo da aba de inventário
func (ui *UnifiedInventoryUI) renderInventoryContent() {
	// Título
	title := "Inventario"
	titleWidth := rl.MeasureText(title, 30)
	rl.DrawText(title, (ScreenWidth-titleWidth)/2, 60, 30, rl.White)

	// Barra de pesquisa
	searchX := int32(50)
	searchY := int32(110)
	searchWidth := int32(300)
	searchHeight := int32(35)

	searchRect := rl.NewRectangle(float32(searchX), float32(searchY), float32(searchWidth), float32(searchHeight))
	rl.DrawRectangleRec(searchRect, rl.NewColor(50, 50, 50, 255))

	borderColor := rl.White
	if ui.IsSearching {
		borderColor = rl.Yellow
	}
	rl.DrawRectangleLinesEx(searchRect, 2, borderColor)

	// Ícone de pesquisa
	rl.DrawText("Pesquisar:", searchX, searchY-20, 14, rl.Gray)

	displayText := ui.SearchBuffer
	if ui.IsSearching && int(rl.GetTime()*2)%2 == 0 {
		displayText += "_"
	}
	if displayText == "" && !ui.IsSearching {
		rl.DrawText("Clique para pesquisar...", searchX+10, searchY+8, 16, rl.Gray)
	} else {
		rl.DrawText(displayText, searchX+10, searchY+8, 16, rl.White)
	}

	// Detectar clique na barra de pesquisa
	if rl.CheckCollisionPointRec(rl.GetMousePosition(), searchRect) && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		ui.IsSearching = true
	}

	// Grade de blocos
	gridX := int32(50)
	gridY := int32(160)
	blockSize := int32(60)
	blockSpacing := int32(10)
	blocksPerRow := 8

	// Fundo da grade
	gridWidth := int32(blocksPerRow)*(blockSize+blockSpacing) - blockSpacing
	gridHeight := int32(300)
	rl.DrawRectangle(gridX-5, gridY-5, gridWidth+10, gridHeight+10, rl.NewColor(40, 40, 40, 255))

	// Desenhar blocos
	visibleRows := 4
	startIdx := ui.InventoryScroll * blocksPerRow
	endIdx := startIdx + visibleRows*blocksPerRow

	if endIdx > len(ui.FilteredBlocks) {
		endIdx = len(ui.FilteredBlocks)
	}

	for i := startIdx; i < endIdx; i++ {
		blockType := ui.FilteredBlocks[i]
		col := (i - startIdx) % blocksPerRow
		row := (i - startIdx) / blocksPerRow

		x := gridX + int32(col)*(blockSize+blockSpacing)
		y := gridY + int32(row)*(blockSize+blockSpacing)

		ui.drawInventoryBlockSlot(blockType, x, y, blockSize)
	}

	// Indicador de scroll
	if len(ui.FilteredBlocks) > visibleRows*blocksPerRow {
		totalRows := (len(ui.FilteredBlocks) + blocksPerRow - 1) / blocksPerRow
		scrollText := fmt.Sprintf("Scroll: %d/%d", ui.InventoryScroll+1, totalRows-visibleRows+1)
		rl.DrawText(scrollText, gridX, gridY+gridHeight+10, 14, rl.Gray)
	}

	// Mostrar quantidade de blocos
	totalBlocks := len(ui.CustomBlockMgr.ListBlocks())
	countText := fmt.Sprintf("%d/%d blocos", len(ui.FilteredBlocks), totalBlocks)
	rl.DrawText(countText, gridX+gridWidth-120, searchY+8, 16, rl.Gray)

	// Hotbar (na parte inferior)
	ui.renderHotbar()

	// Desenhar bloco sendo arrastado
	if ui.IsDragging {
		mousePos := rl.GetMousePosition()
		ui.drawInventoryBlockIcon(ui.DraggedBlock, int32(mousePos.X)-25, int32(mousePos.Y)-25, 50)
	}

	// Instruções específicas
	rl.DrawText("Clique e arraste blocos para o hotbar | Scroll para navegar", 50, ScreenHeight-50, 14, rl.Gray)
}

// drawInventoryBlockSlot desenha um slot de bloco na grade do inventário
func (ui *UnifiedInventoryUI) drawInventoryBlockSlot(blockType BlockType, x, y, size int32) {
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
	ui.drawInventoryBlockIcon(blockType, x+5, y+5, size-10)

	// Borda
	rl.DrawRectangleLinesEx(rect, 1, rl.White)

	// Nome do bloco (tooltip)
	if isHover {
		name := ui.getBlockName(blockType)
		nameWidth := rl.MeasureText(name, 12)
		tooltipX := x + (size-nameWidth)/2
		tooltipY := y + size + 2

		// Fundo do tooltip
		rl.DrawRectangle(tooltipX-2, tooltipY-1, nameWidth+4, 14, rl.NewColor(0, 0, 0, 200))
		rl.DrawText(name, tooltipX, tooltipY, 12, rl.White)
	}

	// Iniciar drag
	if isHover && rl.IsMouseButtonPressed(rl.MouseLeftButton) && !ui.IsDragging {
		ui.IsDragging = true
		ui.DraggedBlock = blockType
		ui.DragStartX = mousePos.X
		ui.DragStartY = mousePos.Y
	}
}

// drawInventoryBlockIcon desenha o ícone de um bloco no inventário
func (ui *UnifiedInventoryUI) drawInventoryBlockIcon(blockType BlockType, x, y, size int32) {
	if blockType == NoBlock {
		return
	}

	texID := uint16(blockType)

	if tex, exists := ui.InventoryTextureCache[texID]; exists && tex.ID != 0 {
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
func (ui *UnifiedInventoryUI) renderHotbar() {
	slotSize := int32(50)
	slotSpacing := int32(5)
	totalWidth := 9*slotSize + 8*slotSpacing
	startX := (ScreenWidth - totalWidth) / 2
	startY := ScreenHeight - slotSize - 90

	// Label
	rl.DrawText("Hotbar (arraste blocos aqui):", startX, startY-25, 16, rl.White)

	for i := 0; i < 9; i++ {
		x := startX + int32(i)*(slotSize+slotSpacing)
		y := startY

		// Cor de fundo
		bgColor := rl.NewColor(50, 50, 50, 200)
		if i == ui.HoverHotbarSlot && ui.IsDragging {
			bgColor = rl.NewColor(80, 120, 80, 220)
		}

		// Desenhar slot
		rl.DrawRectangle(x, y, slotSize, slotSize, bgColor)

		// Borda
		borderColor := rl.White
		if i == ui.Hotbar.SelectedSlot {
			borderColor = rl.Yellow
		}
		rl.DrawRectangleLines(x, y, slotSize, slotSize, borderColor)

		// Número do slot
		rl.DrawText(fmt.Sprintf("%d", i+1), x+3, y+3, 12, rl.Gray)

		// Desenhar bloco no slot
		blockType := ui.Hotbar.Slots[i]
		if blockType != NoBlock {
			ui.drawInventoryBlockIcon(blockType, x+5, y+5, slotSize-10)
		}
	}
}

// renderBlockEditorContent desenha o conteúdo da aba do editor de blocos
func (ui *UnifiedInventoryUI) renderBlockEditorContent() {
	switch ui.EditorState {
	case EditorSubStateMain:
		ui.renderEditorMainMenu()
	case EditorSubStateTextureManager:
		ui.renderEditorTextureManager()
	case EditorSubStateTexturePaint:
		ui.renderEditorTexturePaint()
	case EditorSubStateTextureUpload:
		ui.renderEditorTextureUpload()
	case EditorSubStateBlockList:
		ui.renderEditorBlockList()
	case EditorSubStateBlockCreate:
		ui.renderEditorBlockCreate()
	case EditorSubStateBlockEdit:
		ui.renderEditorBlockEdit()
	case EditorSubStateSelectTexture:
		ui.renderEditorSelectTexture()
	case EditorSubStateFileBrowser:
		ui.renderEditorFileBrowser()
	}
}

// renderEditorMainMenu desenha o menu principal do editor
func (ui *UnifiedInventoryUI) renderEditorMainMenu() {
	title := "Editor de Blocos"
	titleWidth := rl.MeasureText(title, 30)
	rl.DrawText(title, (ScreenWidth-titleWidth)/2, 60, 30, rl.White)

	centerX := int32(ScreenWidth / 2)
	startY := int32(150)
	buttonWidth := int32(300)
	buttonHeight := int32(60)

	// Botão: Gerenciar Texturas
	if ui.drawButton("Gerenciar Texturas", centerX-buttonWidth/2, startY, buttonWidth, buttonHeight) {
		ui.EditorState = EditorSubStateTextureManager
	}
	rl.DrawText("Criar ou fazer upload de texturas", centerX-130, startY+buttonHeight+5, 14, rl.Gray)

	startY += buttonHeight + 40

	// Botão: Gerenciar Blocos
	if ui.drawButton("Gerenciar Blocos", centerX-buttonWidth/2, startY, buttonWidth, buttonHeight) {
		ui.EditorState = EditorSubStateBlockList
	}
	rl.DrawText("Criar blocos e definir texturas para cada face", centerX-170, startY+buttonHeight+5, 14, rl.Gray)
}

// renderEditorTextureManager desenha o gerenciador de texturas
func (ui *UnifiedInventoryUI) renderEditorTextureManager() {
	rl.DrawText("Gerenciador de Texturas", 50, 60, 26, rl.Yellow)

	startY := int32(110)
	leftX := int32(50)

	// Botões de ação
	if ui.drawButton("Criar Nova (Paint)", leftX, startY, 200, 40) {
		ui.EditorState = EditorSubStateTexturePaint
		ui.Painter.ClearCanvas()
		ui.InputBuffer = ""
	}

	if ui.drawButton("Upload de Arquivo", leftX+220, startY, 200, 40) {
		ui.EditorState = EditorSubStateTextureUpload
		ui.InputBuffer = ""
		homeDir, _ := os.UserHomeDir()
		if homeDir != "" {
			ui.CurrentDir = homeDir
		}
	}

	// Lista de texturas existentes
	startY += 60
	rl.DrawText("Texturas Salvas:", leftX, startY, 18, rl.White)
	startY += 30

	textures := ui.TextureMgr.ListTextures()
	if len(textures) == 0 {
		rl.DrawText("Nenhuma textura criada ainda", leftX, startY, 16, rl.Gray)
	} else {
		buttonHeight := int32(35)
		maxVisible := 10

		for i, name := range textures {
			if i < ui.TextureListScroll {
				continue
			}
			if i >= ui.TextureListScroll+maxVisible {
				break
			}

			y := startY + int32(i-ui.TextureListScroll)*(buttonHeight+5)

			// Mostrar preview da textura usando cache
			if tex, exists := ui.EditorTextureCache[name]; exists && tex.ID != 0 {
				rl.DrawTexturePro(tex, rl.NewRectangle(0, 0, float32(tex.Width), float32(tex.Height)),
					rl.NewRectangle(float32(leftX), float32(y), 32, 32),
					rl.NewVector2(0, 0), 0, rl.White)
			}

			// Nome da textura
			rl.DrawText(name, leftX+40, y+8, 16, rl.White)

			// Botão deletar
			if ui.drawButtonColor("X", leftX+300, y, 30, buttonHeight, rl.Red) {
				ui.TextureMgr.DeleteTexture(name)
				ui.loadEditorTextureCache()
				ui.showMessage(fmt.Sprintf("Textura '%s' deletada", name))
			}
		}
	}
}

// renderEditorTexturePaint desenha o editor de pintura
func (ui *UnifiedInventoryUI) renderEditorTexturePaint() {
	rl.DrawText("Criar Textura - Paint", 50, 60, 26, rl.Yellow)

	// Desenhar o painter
	ui.Painter.Render()

	// Campo de nome
	nameY := int32(ScreenHeight - 120)
	rl.DrawText("Nome da textura:", 50, nameY, 18, rl.White)

	inputRect := rl.NewRectangle(50, float32(nameY+25), 300, 35)
	rl.DrawRectangleRec(inputRect, rl.NewColor(50, 50, 50, 255))
	rl.DrawRectangleLinesEx(inputRect, 2, rl.White)

	displayText := ui.InputBuffer
	if ui.IsTyping && int(rl.GetTime()*2)%2 == 0 {
		displayText += "_"
	}
	rl.DrawText(displayText, 55, nameY+32, 18, rl.White)

	// Detectar clique no campo
	if rl.CheckCollisionPointRec(rl.GetMousePosition(), inputRect) && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		ui.IsTyping = true
	}

	// Botões
	if ui.drawButton("Salvar", 370, nameY+25, 100, 35) && ui.InputBuffer != "" {
		img := ui.Painter.ToImage()
		err := ui.TextureMgr.SaveTexture(ui.InputBuffer, img)
		if err != nil {
			ui.showMessage(fmt.Sprintf("Erro: %s", err))
		} else {
			ui.showMessage(fmt.Sprintf("Textura '%s' salva!", ui.InputBuffer))
			ui.loadEditorTextureCache()
			ui.EditorState = EditorSubStateTextureManager
		}
	}

	if ui.drawButton("Limpar", 480, nameY+25, 100, 35) {
		ui.Painter.ClearCanvas()
	}
}

// renderEditorTextureUpload desenha a tela de upload de textura
func (ui *UnifiedInventoryUI) renderEditorTextureUpload() {
	rl.DrawText("Upload de Textura", 50, 60, 26, rl.Yellow)

	startY := int32(110)
	leftX := int32(50)

	// Campo de nome
	rl.DrawText("Nome da textura:", leftX, startY, 18, rl.White)
	startY += 25

	inputRect := rl.NewRectangle(float32(leftX), float32(startY), 300, 35)
	rl.DrawRectangleRec(inputRect, rl.NewColor(50, 50, 50, 255))
	rl.DrawRectangleLinesEx(inputRect, 2, rl.White)

	displayText := ui.InputBuffer
	if ui.IsTyping && int(rl.GetTime()*2)%2 == 0 {
		displayText += "_"
	}
	rl.DrawText(displayText, leftX+5, startY+8, 18, rl.White)

	if rl.CheckCollisionPointRec(rl.GetMousePosition(), inputRect) && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		ui.IsTyping = true
	}

	startY += 50

	// Botão para selecionar arquivo
	if ui.drawButton("Selecionar Arquivo PNG...", leftX, startY, 300, 40) {
		if ui.InputBuffer == "" {
			ui.showMessage("Digite um nome primeiro!")
		} else {
			ui.loadDirectoryContents()
			ui.EditorState = EditorSubStateFileBrowser
		}
	}

	rl.DrawText("O arquivo deve ser PNG 32x32 pixels", leftX, startY+45, 14, rl.Gray)
}

// renderEditorFileBrowser desenha o navegador de arquivos
func (ui *UnifiedInventoryUI) renderEditorFileBrowser() {
	startY := int32(60)
	leftX := int32(50)

	rl.DrawText("Selecionar Arquivo PNG", leftX, startY, 24, rl.Yellow)
	startY += 30

	// Diretório atual
	rl.DrawText(fmt.Sprintf("Pasta: %s", ui.CurrentDir), leftX, startY, 14, rl.Gray)
	startY += 25

	// Lista de arquivos
	buttonHeight := int32(30)
	maxVisible := 12

	for i, item := range ui.FilePaths {
		if i < ui.FileListScroll {
			continue
		}
		if i >= ui.FileListScroll+maxVisible {
			break
		}

		isDir := strings.HasSuffix(item, "/") || item == ".."
		btnColor := rl.NewColor(70, 70, 70, 255)
		if isDir {
			btnColor = rl.NewColor(70, 70, 100, 255)
		}

		if ui.drawButtonColor(item, leftX, startY, 500, buttonHeight, btnColor) {
			if item == ".." {
				ui.CurrentDir = filepath.Dir(ui.CurrentDir)
				ui.loadDirectoryContents()
			} else if isDir {
				ui.CurrentDir = filepath.Join(ui.CurrentDir, strings.TrimSuffix(item, "/"))
				ui.loadDirectoryContents()
			} else {
				// Carregar arquivo
				fullPath := filepath.Join(ui.CurrentDir, item)
				ui.uploadTextureFromFile(fullPath)
			}
		}
		startY += buttonHeight + 3
	}

	// Scroll
	if len(ui.FilePaths) > maxVisible {
		if rl.IsKeyPressed(rl.KeyDown) && ui.FileListScroll < len(ui.FilePaths)-maxVisible {
			ui.FileListScroll++
		}
		if rl.IsKeyPressed(rl.KeyUp) && ui.FileListScroll > 0 {
			ui.FileListScroll--
		}
	}
}

// uploadTextureFromFile faz upload de uma textura de arquivo
func (ui *UnifiedInventoryUI) uploadTextureFromFile(path string) {
	file, err := os.Open(path)
	if err != nil {
		ui.showMessage(fmt.Sprintf("Erro: %s", err))
		return
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		ui.showMessage(fmt.Sprintf("Erro ao decodificar: %s", err))
		return
	}

	// Verificar tamanho
	bounds := img.Bounds()
	if bounds.Dx() != 32 || bounds.Dy() != 32 {
		ui.showMessage(fmt.Sprintf("Imagem deve ser 32x32, recebida: %dx%d", bounds.Dx(), bounds.Dy()))
		return
	}

	// Salvar textura
	err = ui.TextureMgr.SaveTexture(ui.InputBuffer, img)
	if err != nil {
		ui.showMessage(fmt.Sprintf("Erro: %s", err))
		return
	}

	ui.showMessage(fmt.Sprintf("Textura '%s' salva!", ui.InputBuffer))
	ui.loadEditorTextureCache()
	ui.EditorState = EditorSubStateTextureManager
}

// renderEditorBlockList desenha a lista de blocos
func (ui *UnifiedInventoryUI) renderEditorBlockList() {
	rl.DrawText("Gerenciador de Blocos", 50, 60, 26, rl.Yellow)

	startY := int32(110)
	leftX := int32(50)

	// Botão criar novo
	if ui.drawButton("Criar Novo Bloco", leftX, startY, 200, 40) {
		ui.EditorState = EditorSubStateBlockCreate
		ui.InputBuffer = ""
		ui.IsTyping = true
	}

	// Lista de blocos
	startY += 60
	rl.DrawText("Blocos Existentes:", leftX, startY, 18, rl.White)
	startY += 30

	blocks := ui.CustomBlockMgr.ListBlocks()
	if len(blocks) == 0 {
		rl.DrawText("Nenhum bloco criado ainda", leftX, startY, 16, rl.Gray)
	} else {
		for i, block := range blocks {
			btnText := fmt.Sprintf("%d. %s", block.ID, block.Name)
			if ui.drawButton(btnText, leftX, startY, 400, 40) {
				ui.CurrentBlock = block
				ui.loadBlockTextures()
				ui.EditorState = EditorSubStateBlockEdit
			}
			startY += 45

			if i >= 8 {
				rl.DrawText("...", leftX, startY, 16, rl.Gray)
				break
			}
		}
	}
}

// renderEditorBlockCreate desenha a tela de criação de bloco
func (ui *UnifiedInventoryUI) renderEditorBlockCreate() {
	centerX := int32(ScreenWidth / 2)
	startY := int32(150)

	rl.DrawText("Criar Novo Bloco", centerX-100, 60, 26, rl.Yellow)

	rl.DrawText("Nome do Bloco:", centerX-150, startY, 20, rl.White)
	startY += 30

	inputRect := rl.NewRectangle(float32(centerX-150), float32(startY), 300, 40)
	rl.DrawRectangleRec(inputRect, rl.NewColor(50, 50, 50, 255))
	rl.DrawRectangleLinesEx(inputRect, 2, rl.White)

	displayText := ui.InputBuffer
	if ui.IsTyping && int(rl.GetTime()*2)%2 == 0 {
		displayText += "_"
	}
	rl.DrawText(displayText, centerX-140, startY+10, 20, rl.White)

	startY += 60

	if ui.drawButton("Criar", centerX-110, startY, 100, 40) && ui.InputBuffer != "" {
		ui.createNewBlock(ui.InputBuffer)
	}

	if ui.drawButton("Cancelar", centerX+10, startY, 100, 40) {
		ui.EditorState = EditorSubStateBlockList
		ui.IsTyping = false
	}
}

// renderEditorBlockEdit desenha o editor de bloco
func (ui *UnifiedInventoryUI) renderEditorBlockEdit() {
	if ui.CurrentBlock == nil {
		ui.EditorState = EditorSubStateBlockList
		return
	}

	rl.DrawText(fmt.Sprintf("Editando: %s (ID: %d)", ui.CurrentBlock.Name, ui.CurrentBlock.ID), 50, 60, 24, rl.Yellow)

	startY := int32(110)
	leftX := int32(50)

	rl.DrawText("Clique em cada face para selecionar uma textura:", leftX, startY, 16, rl.White)
	startY += 30

	// Grid de faces
	faceSize := int32(80)
	faceSpacing := int32(10)

	// Layout em cruz
	// Linha 1: Top
	ui.drawEditorFaceButton(FaceTop, leftX+faceSize+faceSpacing, startY, faceSize)

	// Linha 2: Left, Front, Right, Back
	startY += faceSize + faceSpacing
	ui.drawEditorFaceButton(FaceLeft, leftX, startY, faceSize)
	ui.drawEditorFaceButton(FaceFront, leftX+faceSize+faceSpacing, startY, faceSize)
	ui.drawEditorFaceButton(FaceRight, leftX+(faceSize+faceSpacing)*2, startY, faceSize)
	ui.drawEditorFaceButton(FaceBack, leftX+(faceSize+faceSpacing)*3, startY, faceSize)

	// Linha 3: Bottom
	startY += faceSize + faceSpacing
	ui.drawEditorFaceButton(FaceBottom, leftX+faceSize+faceSpacing, startY, faceSize)

	// Botões de ação
	actionY := int32(420)

	rl.DrawText("Dica: Use R no jogo para rotacionar ao colocar o bloco", leftX, actionY, 14, rl.Gray)

	actionY += 30
	if ui.drawButton("Salvar Bloco", leftX, actionY, 150, 40) {
		ui.saveCurrentBlock()
	}

	if ui.drawButtonColor("Deletar", leftX+160, actionY, 100, 40, rl.Red) {
		ui.deleteCurrentBlock()
	}
}

// renderEditorSelectTexture desenha o seletor de textura
func (ui *UnifiedInventoryUI) renderEditorSelectTexture() {
	rl.DrawText(fmt.Sprintf("Selecionar textura para: %s", FaceNames[ui.SelectedFace]), 50, 60, 24, rl.Yellow)

	startY := int32(110)
	leftX := int32(50)

	textures := ui.TextureMgr.ListTextures()
	if len(textures) == 0 {
		rl.DrawText("Nenhuma textura disponivel!", leftX, startY, 18, rl.Red)
		rl.DrawText("Crie texturas primeiro no Gerenciador de Texturas", leftX, startY+25, 14, rl.Gray)
	} else {
		// Grade de texturas
		texSize := int32(64)
		texSpacing := int32(10)
		texPerRow := 8

		for i, name := range textures {
			col := i % texPerRow
			row := i / texPerRow

			x := leftX + int32(col)*(texSize+texSpacing)
			y := startY + int32(row)*(texSize+texSpacing+20)

			// Preview da textura usando cache
			if tex, exists := ui.EditorTextureCache[name]; exists && tex.ID != 0 {
				rect := rl.NewRectangle(float32(x), float32(y), float32(texSize), float32(texSize))
				rl.DrawTexturePro(tex, rl.NewRectangle(0, 0, float32(tex.Width), float32(tex.Height)), rect, rl.NewVector2(0, 0), 0, rl.White)

				// Borda
				isSelected := ui.FaceTextureNames[ui.SelectedFace] == name
				borderColor := rl.White
				if isSelected {
					borderColor = rl.Yellow
				}
				rl.DrawRectangleLinesEx(rect, 2, borderColor)

				// Nome
				rl.DrawText(name, x, y+texSize+2, 10, rl.White)

				// Detectar clique
				if rl.CheckCollisionPointRec(rl.GetMousePosition(), rect) && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
					ui.setFaceTexture(name)
				}
			}
		}
	}

	// Botão cancelar
	if ui.drawButton("Cancelar", leftX, int32(ScreenHeight-100), 150, 40) {
		ui.EditorState = EditorSubStateBlockEdit
	}
}

// setFaceTexture define a textura para a face selecionada
func (ui *UnifiedInventoryUI) setFaceTexture(textureName string) {
	if ui.CurrentBlock == nil {
		return
	}

	// Carregar imagem da textura
	img, err := ui.TextureMgr.LoadTextureImage(textureName)
	if err != nil {
		ui.showMessage(fmt.Sprintf("Erro: %s", err))
		return
	}

	// Definir no bloco
	err = ui.CustomBlockMgr.SetFaceTexture(ui.CurrentBlock.ID, ui.SelectedFace, img)
	if err != nil {
		ui.showMessage(fmt.Sprintf("Erro: %s", err))
		return
	}

	// Salvar nome da textura
	ui.FaceTextureNames[ui.SelectedFace] = textureName

	// Recarregar preview
	ui.loadFaceTexture(ui.SelectedFace)

	ui.showMessage(fmt.Sprintf("Textura '%s' definida para %s", textureName, FaceNames[ui.SelectedFace]))
	ui.EditorState = EditorSubStateBlockEdit
}

// drawEditorFaceButton desenha um botão para uma face
func (ui *UnifiedInventoryUI) drawEditorFaceButton(face BlockFace, x, y, size int32) {
	rect := rl.NewRectangle(float32(x), float32(y), float32(size), float32(size))

	// Cor de fundo
	bgColor := rl.NewColor(60, 60, 60, 255)

	// Desenhar textura se carregada
	if ui.FaceTexturesLoaded[face] {
		rl.DrawTexturePro(
			ui.FaceTextures[face],
			rl.NewRectangle(0, 0, 32, 32),
			rect,
			rl.NewVector2(0, 0),
			0,
			rl.White,
		)
	} else {
		rl.DrawRectangleRec(rect, bgColor)
		rl.DrawText("+", x+size/2-10, y+size/2-15, 30, rl.Gray)
	}

	// Borda
	rl.DrawRectangleLinesEx(rect, 2, rl.White)

	// Nome da face
	faceName := FaceNames[face]
	nameWidth := rl.MeasureText(faceName, 12)
	rl.DrawText(faceName, x+(size-nameWidth)/2, y+size+2, 12, rl.White)

	// Detectar clique
	if rl.CheckCollisionPointRec(rl.GetMousePosition(), rect) && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		ui.SelectedFace = face
		ui.EditorState = EditorSubStateSelectTexture
	}
}

// loadDirectoryContents carrega o conteúdo do diretório atual
func (ui *UnifiedInventoryUI) loadDirectoryContents() {
	ui.FilePaths = []string{}
	ui.FileListScroll = 0

	entries, err := os.ReadDir(ui.CurrentDir)
	if err != nil {
		ui.showMessage(fmt.Sprintf("Erro: %s", err))
		return
	}

	if ui.CurrentDir != "/" {
		ui.FilePaths = append(ui.FilePaths, "..")
	}

	// Diretórios primeiro
	for _, entry := range entries {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			ui.FilePaths = append(ui.FilePaths, entry.Name()+"/")
		}
	}

	// Arquivos PNG
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(strings.ToLower(entry.Name()), ".png") {
			ui.FilePaths = append(ui.FilePaths, entry.Name())
		}
	}
}

// loadBlockTextures carrega texturas do bloco atual
func (ui *UnifiedInventoryUI) loadBlockTextures() {
	if ui.CurrentBlock == nil {
		return
	}

	for i := 0; i < 6; i++ {
		ui.loadFaceTexture(BlockFace(i))
	}
}

// loadFaceTexture carrega textura de uma face
func (ui *UnifiedInventoryUI) loadFaceTexture(face BlockFace) {
	if ui.CurrentBlock == nil {
		return
	}

	if ui.FaceTexturesLoaded[face] {
		rl.UnloadTexture(ui.FaceTextures[face])
		ui.FaceTexturesLoaded[face] = false
	}

	texPath := ui.CurrentBlock.FaceTextures[face]
	if texPath == "" {
		return
	}

	tex := rl.LoadTexture(texPath)
	if tex.ID != 0 {
		ui.FaceTextures[face] = tex
		ui.FaceTexturesLoaded[face] = true
		rl.SetTextureFilter(tex, rl.FilterPoint)
	}
}

// createNewBlock cria um novo bloco
func (ui *UnifiedInventoryUI) createNewBlock(name string) {
	block := ui.CustomBlockMgr.CreateBlock(name)
	ui.CurrentBlock = block
	ui.loadBlockTextures()
	ui.IsTyping = false
	ui.EditorState = EditorSubStateBlockEdit
	ui.showMessage(fmt.Sprintf("Bloco '%s' criado!", name))
	// Atualizar lista do inventário
	ui.refreshBlockList()
	ui.loadInventoryTextures()
}

// saveCurrentBlock salva o bloco atual
func (ui *UnifiedInventoryUI) saveCurrentBlock() {
	if ui.CurrentBlock == nil {
		return
	}

	err := ui.CustomBlockMgr.SaveBlock(ui.CurrentBlock.ID)
	if err != nil {
		ui.showMessage(fmt.Sprintf("Erro: %s", err))
		return
	}

	ui.CustomBlockMgr.BuildAtlas()

	// Notificar que o bloco foi salvo (para atualizar atlas global)
	if ui.OnBlockSaved != nil {
		ui.OnBlockSaved(ui.CurrentBlock)
	}

	// Atualizar cache do inventário
	ui.loadInventoryTextures()
	ui.refreshBlockList()

	ui.showMessage("Bloco salvo!")
}

// deleteCurrentBlock deleta o bloco atual
func (ui *UnifiedInventoryUI) deleteCurrentBlock() {
	if ui.CurrentBlock == nil {
		return
	}

	ui.CustomBlockMgr.DeleteBlock(ui.CurrentBlock.ID)
	ui.CurrentBlock = nil
	ui.EditorState = EditorSubStateBlockList
	ui.showMessage("Bloco deletado!")
	// Atualizar lista do inventário
	ui.refreshBlockList()
	ui.loadInventoryTextures()
}

// showMessage mostra mensagem temporária
func (ui *UnifiedInventoryUI) showMessage(msg string) {
	ui.Message = msg
	ui.MessageTimer = 3.0
}

// drawButton desenha um botão
func (ui *UnifiedInventoryUI) drawButton(text string, x, y, width, height int32) bool {
	return ui.drawButtonColor(text, x, y, width, height, rl.NewColor(70, 70, 70, 255))
}

// drawButtonColor desenha um botão com cor
func (ui *UnifiedInventoryUI) drawButtonColor(text string, x, y, width, height int32, color rl.Color) bool {
	rect := rl.NewRectangle(float32(x), float32(y), float32(width), float32(height))
	mousePos := rl.GetMousePosition()
	isHover := rl.CheckCollisionPointRec(mousePos, rect)

	bgColor := color
	if isHover {
		bgColor.R = uint8(minInt(int(bgColor.R)+30, 255))
		bgColor.G = uint8(minInt(int(bgColor.G)+30, 255))
		bgColor.B = uint8(minInt(int(bgColor.B)+30, 255))
	}

	rl.DrawRectangleRec(rect, bgColor)
	rl.DrawRectangleLinesEx(rect, 1, rl.White)

	textWidth := rl.MeasureText(text, 16)
	textX := x + (width-textWidth)/2
	textY := y + (height-16)/2
	rl.DrawText(text, textX, textY, 16, rl.White)

	return isHover && rl.IsMouseButtonPressed(rl.MouseLeftButton)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
