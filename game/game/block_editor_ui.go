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

// BlockEditorState representa os estados do editor de blocos
type BlockEditorState int

const (
	EditorStateClosed BlockEditorState = iota
	EditorStateMain                    // Menu principal do editor
	EditorStateTextureManager          // Gerenciador de texturas
	EditorStateTexturePaint            // Pintando nova textura
	EditorStateTextureUpload           // Upload de textura
	EditorStateBlockList               // Lista de blocos
	EditorStateBlockCreate             // Criando novo bloco
	EditorStateBlockEdit               // Editando bloco existente
	EditorStateSelectTexture           // Selecionando textura para face
	EditorStateFileBrowser             // Navegando arquivos
)

// BlockEditorUI representa a interface de criação de blocos
type BlockEditorUI struct {
	// Estado atual
	State          BlockEditorState
	CustomBlockMgr *CustomBlockManager
	TextureMgr     *TextureManager
	Painter        *TexturePainter

	// Bloco sendo editado
	CurrentBlock     *CustomBlockDefinition
	CurrentBlockName string

	// Face selecionada
	SelectedFace BlockFace

	// Texturas carregadas para preview (Raylib textures)
	FaceTextures       [6]rl.Texture2D
	FaceTexturesLoaded [6]bool
	FaceTextureNames   [6]string // Nome da textura selecionada para cada face

	// UI state
	ScrollOffset     int
	HoverButton      int
	SelectedBlockIdx int
	TextureListScroll int

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

	// Cache de texturas carregadas
	TextureCache map[string]rl.Texture2D
}

// NewBlockEditorUI cria uma nova interface do editor de blocos
func NewBlockEditorUI(cbm *CustomBlockManager) *BlockEditorUI {
	editor := &BlockEditorUI{
		State:          EditorStateClosed,
		CustomBlockMgr: cbm,
		TextureMgr:     NewTextureManager(),
		Painter:        NewTexturePainter(),
		SelectedFace:   FaceFront,
		CurrentDir:     ".",
		TextureCache:   make(map[string]rl.Texture2D),
	}

	return editor
}

// loadTextureCache carrega todas as texturas para o cache
func (e *BlockEditorUI) loadTextureCache() {
	// Limpar cache anterior
	for _, tex := range e.TextureCache {
		if tex.ID != 0 {
			rl.UnloadTexture(tex)
		}
	}
	e.TextureCache = make(map[string]rl.Texture2D)

	// Carregar texturas do TextureManager
	for _, name := range e.TextureMgr.ListTextures() {
		texPath := e.TextureMgr.GetTexturePath(name)
		if texPath != "" {
			tex := rl.LoadTexture(texPath)
			if tex.ID != 0 {
				e.TextureCache[name] = tex
			}
		}
	}
}

// getTexture retorna uma textura do cache pelo nome
func (e *BlockEditorUI) getTexture(name string) rl.Texture2D {
	if tex, exists := e.TextureCache[name]; exists {
		return tex
	}
	return rl.Texture2D{}
}

// Toggle abre o editor (apenas abre, não fecha - use ESC para fechar)
func (e *BlockEditorUI) Toggle() {
	if e.State == EditorStateClosed {
		e.State = EditorStateMain
		rl.EnableCursor()
		e.loadTextureCache()
	}
	// Não fecha com B - use ESC para fechar/voltar
}

// Close fecha o editor
func (e *BlockEditorUI) Close() {
	e.State = EditorStateClosed
	e.CurrentBlock = nil
	e.Message = ""

	// Descarregar texturas de preview
	for i := 0; i < 6; i++ {
		if e.FaceTexturesLoaded[i] {
			rl.UnloadTexture(e.FaceTextures[i])
			e.FaceTexturesLoaded[i] = false
		}
	}

	// Limpar cache de texturas
	for _, tex := range e.TextureCache {
		if tex.ID != 0 {
			rl.UnloadTexture(tex)
		}
	}
	e.TextureCache = make(map[string]rl.Texture2D)

	rl.DisableCursor()
}

// IsOpen retorna se o editor está aberto
func (e *BlockEditorUI) IsOpen() bool {
	return e.State != EditorStateClosed
}

// Update atualiza a lógica do editor
func (e *BlockEditorUI) Update(dt float32) {
	if e.State == EditorStateClosed {
		return
	}

	// Atualizar timer de mensagem
	if e.MessageTimer > 0 {
		e.MessageTimer -= dt
		if e.MessageTimer <= 0 {
			e.Message = ""
		}
	}

	// Atualizar painter se estiver nesse estado
	if e.State == EditorStateTexturePaint {
		e.Painter.Update()
	}

	// Processar input
	e.handleInput()
}

// handleInput processa entrada do usuário
func (e *BlockEditorUI) handleInput() {
	// ESC para voltar/fechar
	if rl.IsKeyPressed(rl.KeyEscape) {
		switch e.State {
		case EditorStateMain:
			e.Close()
		case EditorStateTextureManager, EditorStateBlockList:
			e.State = EditorStateMain
		case EditorStateTexturePaint, EditorStateTextureUpload:
			e.State = EditorStateTextureManager
			e.IsTyping = false
		case EditorStateBlockCreate, EditorStateBlockEdit:
			e.State = EditorStateBlockList
			e.IsTyping = false
		case EditorStateSelectTexture:
			e.State = EditorStateBlockEdit
		case EditorStateFileBrowser:
			e.State = EditorStateTextureUpload
		}
		return
	}

	// Processar input de texto se estiver digitando
	if e.IsTyping {
		char := rl.GetCharPressed()
		for char > 0 {
			if char >= 32 && char <= 126 && len(e.InputBuffer) < 32 {
				e.InputBuffer += string(rune(char))
			}
			char = rl.GetCharPressed()
		}

		if rl.IsKeyPressed(rl.KeyBackspace) && len(e.InputBuffer) > 0 {
			e.InputBuffer = e.InputBuffer[:len(e.InputBuffer)-1]
		}

		if rl.IsKeyPressed(rl.KeyEnter) {
			e.IsTyping = false
			e.confirmInput()
		}
	}
}

// confirmInput confirma o input atual baseado no estado
func (e *BlockEditorUI) confirmInput() {
	switch e.State {
	case EditorStateTexturePaint:
		if e.InputBuffer != "" {
			// Salvar textura pintada
			img := e.Painter.ToImage()
			err := e.TextureMgr.SaveTexture(e.InputBuffer, img)
			if err != nil {
				e.showMessage(fmt.Sprintf("Erro: %s", err))
			} else {
				e.showMessage(fmt.Sprintf("Textura '%s' salva!", e.InputBuffer))
				e.loadTextureCache() // Recarregar cache após salvar
				e.State = EditorStateTextureManager
			}
		}
	case EditorStateBlockCreate:
		if e.InputBuffer != "" {
			e.createNewBlock(e.InputBuffer)
		}
	}
}

// Render desenha a UI do editor
func (e *BlockEditorUI) Render() {
	if e.State == EditorStateClosed {
		return
	}

	// Fundo semi-transparente
	rl.DrawRectangle(0, 0, ScreenWidth, ScreenHeight, rl.NewColor(0, 0, 0, 200))

	switch e.State {
	case EditorStateMain:
		e.renderMainMenu()
	case EditorStateTextureManager:
		e.renderTextureManager()
	case EditorStateTexturePaint:
		e.renderTexturePaint()
	case EditorStateTextureUpload:
		e.renderTextureUpload()
	case EditorStateBlockList:
		e.renderBlockList()
	case EditorStateBlockCreate:
		e.renderBlockCreate()
	case EditorStateBlockEdit:
		e.renderBlockEdit()
	case EditorStateSelectTexture:
		e.renderSelectTexture()
	case EditorStateFileBrowser:
		e.renderFileBrowser()
	}

	// Mensagem de feedback
	if e.Message != "" {
		msgWidth := rl.MeasureText(e.Message, 20)
		rl.DrawRectangle((ScreenWidth-msgWidth)/2-10, ScreenHeight-60, msgWidth+20, 30, rl.NewColor(0, 100, 0, 200))
		rl.DrawText(e.Message, (ScreenWidth-msgWidth)/2, ScreenHeight-55, 20, rl.White)
	}

	// Instruções
	rl.DrawText("ESC - Voltar | B - Fechar Editor", 10, ScreenHeight-30, 16, rl.Gray)
}

// renderMainMenu desenha o menu principal
func (e *BlockEditorUI) renderMainMenu() {
	title := "Editor de Blocos"
	titleWidth := rl.MeasureText(title, 30)
	rl.DrawText(title, (ScreenWidth-titleWidth)/2, 50, 30, rl.White)

	centerX := int32(ScreenWidth / 2)
	startY := int32(150)
	buttonWidth := int32(300)
	buttonHeight := int32(60)

	// Botão: Gerenciar Texturas
	if e.drawButton("Gerenciar Texturas", centerX-buttonWidth/2, startY, buttonWidth, buttonHeight) {
		e.State = EditorStateTextureManager
	}
	rl.DrawText("Criar ou fazer upload de texturas", centerX-130, startY+buttonHeight+5, 14, rl.Gray)

	startY += buttonHeight + 40

	// Botão: Gerenciar Blocos
	if e.drawButton("Gerenciar Blocos", centerX-buttonWidth/2, startY, buttonWidth, buttonHeight) {
		e.State = EditorStateBlockList
	}
	rl.DrawText("Criar blocos e definir texturas para cada face", centerX-170, startY+buttonHeight+5, 14, rl.Gray)
}

// renderTextureManager desenha o gerenciador de texturas
func (e *BlockEditorUI) renderTextureManager() {
	rl.DrawText("Gerenciador de Texturas", 50, 50, 26, rl.Yellow)

	startY := int32(100)
	leftX := int32(50)

	// Botões de ação
	if e.drawButton("Criar Nova (Paint)", leftX, startY, 200, 40) {
		e.State = EditorStateTexturePaint
		e.Painter.ClearCanvas()
		e.InputBuffer = ""
	}

	if e.drawButton("Upload de Arquivo", leftX+220, startY, 200, 40) {
		e.State = EditorStateTextureUpload
		e.InputBuffer = ""
		// Abrir no diretório home
		homeDir, _ := os.UserHomeDir()
		if homeDir != "" {
			e.CurrentDir = homeDir
		}
	}

	// Lista de texturas existentes
	startY += 60
	rl.DrawText("Texturas Salvas:", leftX, startY, 18, rl.White)
	startY += 30

	textures := e.TextureMgr.ListTextures()
	if len(textures) == 0 {
		rl.DrawText("Nenhuma textura criada ainda", leftX, startY, 16, rl.Gray)
	} else {
		buttonHeight := int32(35)
		maxVisible := 10

		for i, name := range textures {
			if i < e.TextureListScroll {
				continue
			}
			if i >= e.TextureListScroll+maxVisible {
				break
			}

			y := startY + int32(i-e.TextureListScroll)*(buttonHeight+5)

			// Mostrar preview da textura usando cache
			tex := e.getTexture(name)
			if tex.ID != 0 {
				rl.DrawTexturePro(tex, rl.NewRectangle(0, 0, float32(tex.Width), float32(tex.Height)),
					rl.NewRectangle(float32(leftX), float32(y), 32, 32),
					rl.NewVector2(0, 0), 0, rl.White)
			}

			// Nome da textura
			rl.DrawText(name, leftX+40, y+8, 16, rl.White)

			// Botão deletar
			if e.drawButtonColor("X", leftX+300, y, 30, buttonHeight, rl.Red) {
				e.TextureMgr.DeleteTexture(name)
				e.loadTextureCache() // Recarregar cache após deletar
				e.showMessage(fmt.Sprintf("Textura '%s' deletada", name))
			}
		}
	}
}

// renderTexturePaint desenha o editor de pintura
func (e *BlockEditorUI) renderTexturePaint() {
	rl.DrawText("Criar Textura - Paint", 50, 30, 26, rl.Yellow)

	// Desenhar o painter
	e.Painter.Render()

	// Campo de nome
	nameY := int32(ScreenHeight - 120)
	rl.DrawText("Nome da textura:", 50, nameY, 18, rl.White)

	inputRect := rl.NewRectangle(50, float32(nameY+25), 300, 35)
	rl.DrawRectangleRec(inputRect, rl.NewColor(50, 50, 50, 255))
	rl.DrawRectangleLinesEx(inputRect, 2, rl.White)

	displayText := e.InputBuffer
	if e.IsTyping && int(rl.GetTime()*2)%2 == 0 {
		displayText += "_"
	}
	rl.DrawText(displayText, 55, nameY+32, 18, rl.White)

	// Detectar clique no campo
	if rl.CheckCollisionPointRec(rl.GetMousePosition(), inputRect) && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		e.IsTyping = true
	}

	// Botões
	if e.drawButton("Salvar", 370, nameY+25, 100, 35) && e.InputBuffer != "" {
		img := e.Painter.ToImage()
		err := e.TextureMgr.SaveTexture(e.InputBuffer, img)
		if err != nil {
			e.showMessage(fmt.Sprintf("Erro: %s", err))
		} else {
			e.showMessage(fmt.Sprintf("Textura '%s' salva!", e.InputBuffer))
			e.loadTextureCache() // Recarregar cache após salvar
			e.State = EditorStateTextureManager
		}
	}

	if e.drawButton("Limpar", 480, nameY+25, 100, 35) {
		e.Painter.ClearCanvas()
	}
}

// renderTextureUpload desenha a tela de upload de textura
func (e *BlockEditorUI) renderTextureUpload() {
	rl.DrawText("Upload de Textura", 50, 50, 26, rl.Yellow)

	startY := int32(100)
	leftX := int32(50)

	// Campo de nome
	rl.DrawText("Nome da textura:", leftX, startY, 18, rl.White)
	startY += 25

	inputRect := rl.NewRectangle(float32(leftX), float32(startY), 300, 35)
	rl.DrawRectangleRec(inputRect, rl.NewColor(50, 50, 50, 255))
	rl.DrawRectangleLinesEx(inputRect, 2, rl.White)

	displayText := e.InputBuffer
	if e.IsTyping && int(rl.GetTime()*2)%2 == 0 {
		displayText += "_"
	}
	rl.DrawText(displayText, leftX+5, startY+8, 18, rl.White)

	if rl.CheckCollisionPointRec(rl.GetMousePosition(), inputRect) && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		e.IsTyping = true
	}

	startY += 50

	// Botão para selecionar arquivo
	if e.drawButton("Selecionar Arquivo PNG...", leftX, startY, 300, 40) {
		if e.InputBuffer == "" {
			e.showMessage("Digite um nome primeiro!")
		} else {
			e.loadDirectoryContents()
			e.State = EditorStateFileBrowser
		}
	}

	rl.DrawText("O arquivo deve ser PNG 32x32 pixels", leftX, startY+45, 14, rl.Gray)
}

// renderFileBrowser desenha o navegador de arquivos
func (e *BlockEditorUI) renderFileBrowser() {
	startY := int32(50)
	leftX := int32(50)

	rl.DrawText("Selecionar Arquivo PNG", leftX, startY, 24, rl.Yellow)
	startY += 30

	// Diretório atual
	rl.DrawText(fmt.Sprintf("Pasta: %s", e.CurrentDir), leftX, startY, 14, rl.Gray)
	startY += 25

	// Lista de arquivos
	buttonHeight := int32(30)
	maxVisible := 12

	for i, item := range e.FilePaths {
		if i < e.FileListScroll {
			continue
		}
		if i >= e.FileListScroll+maxVisible {
			break
		}

		isDir := strings.HasSuffix(item, "/") || item == ".."
		btnColor := rl.NewColor(70, 70, 70, 255)
		if isDir {
			btnColor = rl.NewColor(70, 70, 100, 255)
		}

		if e.drawButtonColor(item, leftX, startY, 500, buttonHeight, btnColor) {
			if item == ".." {
				e.CurrentDir = filepath.Dir(e.CurrentDir)
				e.loadDirectoryContents()
			} else if isDir {
				e.CurrentDir = filepath.Join(e.CurrentDir, strings.TrimSuffix(item, "/"))
				e.loadDirectoryContents()
			} else {
				// Carregar arquivo
				fullPath := filepath.Join(e.CurrentDir, item)
				e.uploadTextureFromFile(fullPath)
			}
		}
		startY += buttonHeight + 3
	}

	// Scroll
	if len(e.FilePaths) > maxVisible {
		if rl.IsKeyPressed(rl.KeyDown) && e.FileListScroll < len(e.FilePaths)-maxVisible {
			e.FileListScroll++
		}
		if rl.IsKeyPressed(rl.KeyUp) && e.FileListScroll > 0 {
			e.FileListScroll--
		}
	}
}

// uploadTextureFromFile faz upload de uma textura de arquivo
func (e *BlockEditorUI) uploadTextureFromFile(path string) {
	file, err := os.Open(path)
	if err != nil {
		e.showMessage(fmt.Sprintf("Erro: %s", err))
		return
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		e.showMessage(fmt.Sprintf("Erro ao decodificar: %s", err))
		return
	}

	// Verificar tamanho
	bounds := img.Bounds()
	if bounds.Dx() != 32 || bounds.Dy() != 32 {
		e.showMessage(fmt.Sprintf("Imagem deve ser 32x32, recebida: %dx%d", bounds.Dx(), bounds.Dy()))
		return
	}

	// Salvar textura
	err = e.TextureMgr.SaveTexture(e.InputBuffer, img)
	if err != nil {
		e.showMessage(fmt.Sprintf("Erro: %s", err))
		return
	}

	e.showMessage(fmt.Sprintf("Textura '%s' salva!", e.InputBuffer))
	e.loadTextureCache() // Recarregar cache após salvar
	e.State = EditorStateTextureManager
}

// renderBlockList desenha a lista de blocos
func (e *BlockEditorUI) renderBlockList() {
	rl.DrawText("Gerenciador de Blocos", 50, 50, 26, rl.Yellow)

	startY := int32(100)
	leftX := int32(50)

	// Botão criar novo
	if e.drawButton("Criar Novo Bloco", leftX, startY, 200, 40) {
		e.State = EditorStateBlockCreate
		e.InputBuffer = ""
		e.IsTyping = true
	}

	// Lista de blocos
	startY += 60
	rl.DrawText("Blocos Existentes:", leftX, startY, 18, rl.White)
	startY += 30

	blocks := e.CustomBlockMgr.ListBlocks()
	if len(blocks) == 0 {
		rl.DrawText("Nenhum bloco criado ainda", leftX, startY, 16, rl.Gray)
	} else {
		for i, block := range blocks {
			btnText := fmt.Sprintf("%d. %s", block.ID, block.Name)
			if e.drawButton(btnText, leftX, startY, 400, 40) {
				e.CurrentBlock = block
				e.loadBlockTextures()
				e.State = EditorStateBlockEdit
			}
			startY += 45

			if i >= 8 {
				rl.DrawText("...", leftX, startY, 16, rl.Gray)
				break
			}
		}
	}
}

// renderBlockCreate desenha a tela de criação de bloco
func (e *BlockEditorUI) renderBlockCreate() {
	centerX := int32(ScreenWidth / 2)
	startY := int32(150)

	rl.DrawText("Criar Novo Bloco", centerX-100, 50, 26, rl.Yellow)

	rl.DrawText("Nome do Bloco:", centerX-150, startY, 20, rl.White)
	startY += 30

	inputRect := rl.NewRectangle(float32(centerX-150), float32(startY), 300, 40)
	rl.DrawRectangleRec(inputRect, rl.NewColor(50, 50, 50, 255))
	rl.DrawRectangleLinesEx(inputRect, 2, rl.White)

	displayText := e.InputBuffer
	if e.IsTyping && int(rl.GetTime()*2)%2 == 0 {
		displayText += "_"
	}
	rl.DrawText(displayText, centerX-140, startY+10, 20, rl.White)

	startY += 60

	if e.drawButton("Criar", centerX-110, startY, 100, 40) && e.InputBuffer != "" {
		e.createNewBlock(e.InputBuffer)
	}

	if e.drawButton("Cancelar", centerX+10, startY, 100, 40) {
		e.State = EditorStateBlockList
		e.IsTyping = false
	}
}

// renderBlockEdit desenha o editor de bloco
func (e *BlockEditorUI) renderBlockEdit() {
	if e.CurrentBlock == nil {
		e.State = EditorStateBlockList
		return
	}

	rl.DrawText(fmt.Sprintf("Editando: %s (ID: %d)", e.CurrentBlock.Name, e.CurrentBlock.ID), 50, 50, 24, rl.Yellow)

	startY := int32(100)
	leftX := int32(50)

	rl.DrawText("Clique em cada face para selecionar uma textura:", leftX, startY, 16, rl.White)
	startY += 30

	// Grid de faces
	faceSize := int32(80)
	faceSpacing := int32(10)

	// Layout em cruz
	// Linha 1: Top
	e.drawFaceButton(FaceTop, leftX+faceSize+faceSpacing, startY, faceSize)

	// Linha 2: Left, Front, Right, Back
	startY += faceSize + faceSpacing
	e.drawFaceButton(FaceLeft, leftX, startY, faceSize)
	e.drawFaceButton(FaceFront, leftX+faceSize+faceSpacing, startY, faceSize)
	e.drawFaceButton(FaceRight, leftX+(faceSize+faceSpacing)*2, startY, faceSize)
	e.drawFaceButton(FaceBack, leftX+(faceSize+faceSpacing)*3, startY, faceSize)

	// Linha 3: Bottom
	startY += faceSize + faceSpacing
	e.drawFaceButton(FaceBottom, leftX+faceSize+faceSpacing, startY, faceSize)

	// Botões de ação
	actionY := int32(420)

	rl.DrawText("Dica: Use R no jogo para rotacionar ao colocar o bloco", leftX, actionY, 14, rl.Gray)

	actionY += 30
	if e.drawButton("Salvar Bloco", leftX, actionY, 150, 40) {
		e.saveCurrentBlock()
	}

	if e.drawButton("Adicionar ao Hotbar", leftX+160, actionY, 170, 40) {
		// TODO: Adicionar ao hotbar
		e.showMessage("Bloco disponível no hotbar!")
	}

	if e.drawButtonColor("Deletar", leftX+340, actionY, 100, 40, rl.Red) {
		e.deleteCurrentBlock()
	}
}

// renderSelectTexture desenha o seletor de textura
func (e *BlockEditorUI) renderSelectTexture() {
	rl.DrawText(fmt.Sprintf("Selecionar textura para: %s", FaceNames[e.SelectedFace]), 50, 50, 24, rl.Yellow)

	startY := int32(100)
	leftX := int32(50)

	textures := e.TextureMgr.ListTextures()
	if len(textures) == 0 {
		rl.DrawText("Nenhuma textura disponível!", leftX, startY, 18, rl.Red)
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
			tex := e.getTexture(name)
			if tex.ID != 0 {
				rect := rl.NewRectangle(float32(x), float32(y), float32(texSize), float32(texSize))
				rl.DrawTexturePro(tex, rl.NewRectangle(0, 0, float32(tex.Width), float32(tex.Height)), rect, rl.NewVector2(0, 0), 0, rl.White)

				// Borda
				isSelected := e.FaceTextureNames[e.SelectedFace] == name
				borderColor := rl.White
				if isSelected {
					borderColor = rl.Yellow
				}
				rl.DrawRectangleLinesEx(rect, 2, borderColor)

				// Nome
				rl.DrawText(name, x, y+texSize+2, 10, rl.White)

				// Detectar clique
				if rl.CheckCollisionPointRec(rl.GetMousePosition(), rect) && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
					e.setFaceTexture(name)
				}
			}
		}
	}

	// Botão cancelar
	if e.drawButton("Cancelar", leftX, int32(ScreenHeight-100), 150, 40) {
		e.State = EditorStateBlockEdit
	}
}

// setFaceTexture define a textura para a face selecionada
func (e *BlockEditorUI) setFaceTexture(textureName string) {
	if e.CurrentBlock == nil {
		return
	}

	// Carregar imagem da textura
	img, err := e.TextureMgr.LoadTextureImage(textureName)
	if err != nil {
		e.showMessage(fmt.Sprintf("Erro: %s", err))
		return
	}

	// Definir no bloco
	err = e.CustomBlockMgr.SetFaceTexture(e.CurrentBlock.ID, e.SelectedFace, img)
	if err != nil {
		e.showMessage(fmt.Sprintf("Erro: %s", err))
		return
	}

	// Salvar nome da textura
	e.FaceTextureNames[e.SelectedFace] = textureName

	// Recarregar preview
	e.loadFaceTexture(e.SelectedFace)

	e.showMessage(fmt.Sprintf("Textura '%s' definida para %s", textureName, FaceNames[e.SelectedFace]))
	e.State = EditorStateBlockEdit
}

// drawFaceButton desenha um botão para uma face
func (e *BlockEditorUI) drawFaceButton(face BlockFace, x, y, size int32) {
	rect := rl.NewRectangle(float32(x), float32(y), float32(size), float32(size))

	// Cor de fundo
	bgColor := rl.NewColor(60, 60, 60, 255)

	// Desenhar textura se carregada
	if e.FaceTexturesLoaded[face] {
		rl.DrawTexturePro(
			e.FaceTextures[face],
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
		e.SelectedFace = face
		e.State = EditorStateSelectTexture
	}
}

// loadDirectoryContents carrega o conteúdo do diretório atual
func (e *BlockEditorUI) loadDirectoryContents() {
	e.FilePaths = []string{}
	e.FileListScroll = 0

	entries, err := os.ReadDir(e.CurrentDir)
	if err != nil {
		e.showMessage(fmt.Sprintf("Erro: %s", err))
		return
	}

	if e.CurrentDir != "/" {
		e.FilePaths = append(e.FilePaths, "..")
	}

	// Diretórios primeiro
	for _, entry := range entries {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			e.FilePaths = append(e.FilePaths, entry.Name()+"/")
		}
	}

	// Arquivos PNG
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(strings.ToLower(entry.Name()), ".png") {
			e.FilePaths = append(e.FilePaths, entry.Name())
		}
	}
}

// loadBlockTextures carrega texturas do bloco atual
func (e *BlockEditorUI) loadBlockTextures() {
	if e.CurrentBlock == nil {
		return
	}

	for i := 0; i < 6; i++ {
		e.loadFaceTexture(BlockFace(i))
	}
}

// loadFaceTexture carrega textura de uma face
func (e *BlockEditorUI) loadFaceTexture(face BlockFace) {
	if e.CurrentBlock == nil {
		return
	}

	if e.FaceTexturesLoaded[face] {
		rl.UnloadTexture(e.FaceTextures[face])
		e.FaceTexturesLoaded[face] = false
	}

	texPath := e.CurrentBlock.FaceTextures[face]
	if texPath == "" {
		return
	}

	tex := rl.LoadTexture(texPath)
	if tex.ID != 0 {
		e.FaceTextures[face] = tex
		e.FaceTexturesLoaded[face] = true
		rl.SetTextureFilter(tex, rl.FilterPoint)
	}
}

// createNewBlock cria um novo bloco
func (e *BlockEditorUI) createNewBlock(name string) {
	block := e.CustomBlockMgr.CreateBlock(name)
	e.CurrentBlock = block
	e.loadBlockTextures()
	e.IsTyping = false
	e.State = EditorStateBlockEdit
	e.showMessage(fmt.Sprintf("Bloco '%s' criado!", name))
}

// saveCurrentBlock salva o bloco atual
func (e *BlockEditorUI) saveCurrentBlock() {
	if e.CurrentBlock == nil {
		return
	}

	err := e.CustomBlockMgr.SaveBlock(e.CurrentBlock.ID)
	if err != nil {
		e.showMessage(fmt.Sprintf("Erro: %s", err))
		return
	}

	e.CustomBlockMgr.BuildAtlas()

	// Notificar que o bloco foi salvo (para atualizar atlas global)
	if e.OnBlockSaved != nil {
		e.OnBlockSaved(e.CurrentBlock)
	}

	e.showMessage("Bloco salvo!")
}

// deleteCurrentBlock deleta o bloco atual
func (e *BlockEditorUI) deleteCurrentBlock() {
	if e.CurrentBlock == nil {
		return
	}

	e.CustomBlockMgr.DeleteBlock(e.CurrentBlock.ID)
	e.CurrentBlock = nil
	e.State = EditorStateBlockList
	e.showMessage("Bloco deletado!")
}

// showMessage mostra mensagem temporária
func (e *BlockEditorUI) showMessage(msg string) {
	e.Message = msg
	e.MessageTimer = 3.0
}

// drawButton desenha um botão
func (e *BlockEditorUI) drawButton(text string, x, y, width, height int32) bool {
	return e.drawButtonColor(text, x, y, width, height, rl.NewColor(70, 70, 70, 255))
}

// drawButtonColor desenha um botão com cor
func (e *BlockEditorUI) drawButtonColor(text string, x, y, width, height int32, color rl.Color) bool {
	rect := rl.NewRectangle(float32(x), float32(y), float32(width), float32(height))
	mousePos := rl.GetMousePosition()
	isHover := rl.CheckCollisionPointRec(mousePos, rect)

	bgColor := color
	if isHover {
		bgColor.R = uint8(min(int(bgColor.R)+30, 255))
		bgColor.G = uint8(min(int(bgColor.G)+30, 255))
		bgColor.B = uint8(min(int(bgColor.B)+30, 255))
	}

	rl.DrawRectangleRec(rect, bgColor)
	rl.DrawRectangleLinesEx(rect, 1, rl.White)

	textWidth := rl.MeasureText(text, 16)
	textX := x + (width-textWidth)/2
	textY := y + (height-16)/2
	rl.DrawText(text, textX, textY, 16, rl.White)

	return isHover && rl.IsMouseButtonPressed(rl.MouseLeftButton)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
