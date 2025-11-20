package game

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"sort"
	"strings"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// TextureDefinition representa uma textura salva pelo jogador
type TextureDefinition struct {
	Name     string `json:"name"`
	Filename string `json:"filename"`
}

// TextureManager gerencia todas as texturas customizadas do jogador
type TextureManager struct {
	Textures    map[string]*TextureDefinition
	TexturesDir string
	DataFile    string
}

// NewTextureManager cria um novo gerenciador de texturas
func NewTextureManager() *TextureManager {
	tm := &TextureManager{
		Textures:    make(map[string]*TextureDefinition),
		TexturesDir: "assets/custom_textures",
		DataFile:    "data/textures.json",
	}

	// Criar diretórios
	os.MkdirAll(tm.TexturesDir, 0755)
	os.MkdirAll("data", 0755)

	// Carregar texturas existentes
	tm.LoadTextures()

	return tm
}

// SaveTexture salva uma textura com um nome
func (tm *TextureManager) SaveTexture(name string, img image.Image) error {
	// Validar nome
	if name == "" {
		return fmt.Errorf("nome não pode ser vazio")
	}

	// Criar filename seguro
	safeName := strings.ReplaceAll(strings.ToLower(name), " ", "_")
	filename := fmt.Sprintf("%s.png", safeName)
	filepath := filepath.Join(tm.TexturesDir, filename)

	// Salvar imagem
	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("erro ao criar arquivo: %w", err)
	}
	defer file.Close()

	err = png.Encode(file, img)
	if err != nil {
		return fmt.Errorf("erro ao salvar PNG: %w", err)
	}

	// Adicionar ao mapa
	tm.Textures[name] = &TextureDefinition{
		Name:     name,
		Filename: filename,
	}

	// Salvar índice
	return tm.SaveIndex()
}

// SaveIndex salva o índice de texturas
func (tm *TextureManager) SaveIndex() error {
	data, err := json.MarshalIndent(tm.Textures, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(tm.DataFile, data, 0644)
}

// LoadTextures carrega as texturas do disco
func (tm *TextureManager) LoadTextures() error {
	data, err := os.ReadFile(tm.DataFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	return json.Unmarshal(data, &tm.Textures)
}

// GetTexturePath retorna o caminho completo de uma textura
func (tm *TextureManager) GetTexturePath(name string) string {
	tex, exists := tm.Textures[name]
	if !exists {
		return ""
	}
	return filepath.Join(tm.TexturesDir, tex.Filename)
}

// ListTextures retorna lista de nomes de texturas ordenada alfabeticamente
func (tm *TextureManager) ListTextures() []string {
	names := make([]string, 0, len(tm.Textures))
	for name := range tm.Textures {
		names = append(names, name)
	}

	// Ordenar alfabeticamente para ordem consistente
	sort.Strings(names)

	return names
}

// DeleteTexture remove uma textura
func (tm *TextureManager) DeleteTexture(name string) error {
	tex, exists := tm.Textures[name]
	if !exists {
		return fmt.Errorf("textura não encontrada")
	}

	// Remover arquivo
	filepath := filepath.Join(tm.TexturesDir, tex.Filename)
	os.Remove(filepath)

	// Remover do mapa
	delete(tm.Textures, name)

	return tm.SaveIndex()
}

// LoadTextureImage carrega a imagem de uma textura
func (tm *TextureManager) LoadTextureImage(name string) (image.Image, error) {
	path := tm.GetTexturePath(name)
	if path == "" {
		return nil, fmt.Errorf("textura não encontrada")
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return png.Decode(file)
}

// TexturePainter representa o editor de pintura de texturas
type TexturePainter struct {
	// Canvas 32x32
	Canvas [32][32]rl.Color

	// Ferramentas
	CurrentColor    rl.Color
	BrushSize       int
	Tool            PaintTool

	// Paleta de cores
	Palette         []rl.Color
	SelectedPalette int

	// Estado
	IsDrawing       bool
	LastX, LastY    int

	// Zoom e posição
	PixelSize       int32
	CanvasX, CanvasY int32

	// Histórico para undo
	History         [][32][32]rl.Color
	HistoryIndex    int
}

// PaintTool representa uma ferramenta de pintura
type PaintTool int

const (
	ToolPencil PaintTool = iota
	ToolEraser
	ToolFill
	ToolPicker
)

// NewTexturePainter cria um novo editor de pintura
func NewTexturePainter() *TexturePainter {
	tp := &TexturePainter{
		CurrentColor:    rl.White,
		BrushSize:       1,
		Tool:            ToolPencil,
		PixelSize:       12,
		CanvasX:         50,
		CanvasY:         120,
		SelectedPalette: 0,
		History:         make([][32][32]rl.Color, 0),
		HistoryIndex:    -1,
	}

	// Inicializar canvas com transparente
	tp.ClearCanvas()

	// Criar paleta de cores padrão
	tp.Palette = []rl.Color{
		rl.White,
		rl.Black,
		rl.Red,
		rl.Green,
		rl.Blue,
		rl.Yellow,
		rl.Orange,
		rl.Purple,
		rl.Pink,
		rl.Brown,
		rl.Gray,
		rl.DarkGray,
		rl.Lime,
		rl.SkyBlue,
		rl.Violet,
		rl.Beige,
		// Tons de pele/terra
		rl.NewColor(139, 90, 43, 255),
		rl.NewColor(160, 82, 45, 255),
		rl.NewColor(210, 180, 140, 255),
		rl.NewColor(244, 164, 96, 255),
		// Tons de grama/natureza
		rl.NewColor(34, 139, 34, 255),
		rl.NewColor(0, 100, 0, 255),
		rl.NewColor(107, 142, 35, 255),
		rl.NewColor(85, 107, 47, 255),
		// Tons de pedra/metal
		rl.NewColor(112, 128, 144, 255),
		rl.NewColor(119, 136, 153, 255),
		rl.NewColor(192, 192, 192, 255),
		rl.NewColor(169, 169, 169, 255),
		// Tons de água/gelo
		rl.NewColor(0, 191, 255, 255),
		rl.NewColor(30, 144, 255, 255),
		rl.NewColor(173, 216, 230, 255),
		rl.NewColor(240, 248, 255, 255),
	}

	return tp
}

// ClearCanvas limpa o canvas
func (tp *TexturePainter) ClearCanvas() {
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			tp.Canvas[y][x] = rl.NewColor(0, 0, 0, 0)
		}
	}
	tp.SaveHistory()
}

// SaveHistory salva o estado atual no histórico
func (tp *TexturePainter) SaveHistory() {
	// Limitar histórico
	if tp.HistoryIndex < len(tp.History)-1 {
		tp.History = tp.History[:tp.HistoryIndex+1]
	}

	if len(tp.History) > 50 {
		tp.History = tp.History[1:]
	}

	var state [32][32]rl.Color
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			state[y][x] = tp.Canvas[y][x]
		}
	}

	tp.History = append(tp.History, state)
	tp.HistoryIndex = len(tp.History) - 1
}

// Undo desfaz a última ação
func (tp *TexturePainter) Undo() {
	if tp.HistoryIndex > 0 {
		tp.HistoryIndex--
		tp.Canvas = tp.History[tp.HistoryIndex]
	}
}

// Redo refaz a última ação
func (tp *TexturePainter) Redo() {
	if tp.HistoryIndex < len(tp.History)-1 {
		tp.HistoryIndex++
		tp.Canvas = tp.History[tp.HistoryIndex]
	}
}

// Update atualiza o painter
func (tp *TexturePainter) Update() {
	mousePos := rl.GetMousePosition()

	// Calcular posição do mouse no canvas
	canvasEndX := tp.CanvasX + 32*tp.PixelSize
	canvasEndY := tp.CanvasY + 32*tp.PixelSize

	inCanvas := mousePos.X >= float32(tp.CanvasX) && mousePos.X < float32(canvasEndX) &&
		mousePos.Y >= float32(tp.CanvasY) && mousePos.Y < float32(canvasEndY)

	if inCanvas {
		pixelX := int((mousePos.X - float32(tp.CanvasX)) / float32(tp.PixelSize))
		pixelY := int((mousePos.Y - float32(tp.CanvasY)) / float32(tp.PixelSize))

		if pixelX >= 0 && pixelX < 32 && pixelY >= 0 && pixelY < 32 {
			// Começar a desenhar
			if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
				tp.IsDrawing = true
				tp.LastX = pixelX
				tp.LastY = pixelY
				tp.ApplyTool(pixelX, pixelY)
			}

			// Continuar desenhando
			if rl.IsMouseButtonDown(rl.MouseLeftButton) && tp.IsDrawing {
				// Desenhar linha do último ponto ao atual
				tp.DrawLine(tp.LastX, tp.LastY, pixelX, pixelY)
				tp.LastX = pixelX
				tp.LastY = pixelY
			}

			// Parar de desenhar
			if rl.IsMouseButtonReleased(rl.MouseLeftButton) && tp.IsDrawing {
				tp.IsDrawing = false
				tp.SaveHistory()
			}
		}
	}

	// Atalhos de teclado
	if rl.IsKeyDown(rl.KeyLeftControl) || rl.IsKeyDown(rl.KeyRightControl) {
		if rl.IsKeyPressed(rl.KeyZ) {
			tp.Undo()
		}
		if rl.IsKeyPressed(rl.KeyY) {
			tp.Redo()
		}
	}

	// Mudar tamanho do pincel
	wheel := rl.GetMouseWheelMove()
	if wheel != 0 && inCanvas {
		tp.BrushSize += int(wheel)
		if tp.BrushSize < 1 {
			tp.BrushSize = 1
		}
		if tp.BrushSize > 8 {
			tp.BrushSize = 8
		}
	}
}

// ApplyTool aplica a ferramenta atual em um pixel
func (tp *TexturePainter) ApplyTool(x, y int) {
	switch tp.Tool {
	case ToolPencil:
		tp.DrawPixel(x, y, tp.CurrentColor)
	case ToolEraser:
		tp.DrawPixel(x, y, rl.NewColor(0, 0, 0, 0))
	case ToolFill:
		tp.FloodFill(x, y, tp.CurrentColor)
		tp.SaveHistory()
	case ToolPicker:
		tp.CurrentColor = tp.Canvas[y][x]
	}
}

// DrawPixel desenha um pixel com o tamanho do pincel
func (tp *TexturePainter) DrawPixel(x, y int, color rl.Color) {
	half := tp.BrushSize / 2
	for dy := -half; dy <= half; dy++ {
		for dx := -half; dx <= half; dx++ {
			px, py := x+dx, y+dy
			if px >= 0 && px < 32 && py >= 0 && py < 32 {
				tp.Canvas[py][px] = color
			}
		}
	}
}

// DrawLine desenha uma linha entre dois pontos
func (tp *TexturePainter) DrawLine(x0, y0, x1, y1 int) {
	dx := abs(x1 - x0)
	dy := -abs(y1 - y0)
	sx := 1
	if x0 > x1 {
		sx = -1
	}
	sy := 1
	if y0 > y1 {
		sy = -1
	}
	err := dx + dy

	for {
		tp.ApplyTool(x0, y0)

		if x0 == x1 && y0 == y1 {
			break
		}

		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

// FloodFill preenche uma área com uma cor
func (tp *TexturePainter) FloodFill(x, y int, newColor rl.Color) {
	if x < 0 || x >= 32 || y < 0 || y >= 32 {
		return
	}

	oldColor := tp.Canvas[y][x]
	if colorEquals(oldColor, newColor) {
		return
	}

	// BFS flood fill
	queue := []struct{ x, y int }{{x, y}}
	visited := make(map[int]bool)

	for len(queue) > 0 {
		p := queue[0]
		queue = queue[1:]

		key := p.y*32 + p.x
		if visited[key] {
			continue
		}
		visited[key] = true

		if p.x < 0 || p.x >= 32 || p.y < 0 || p.y >= 32 {
			continue
		}

		if !colorEquals(tp.Canvas[p.y][p.x], oldColor) {
			continue
		}

		tp.Canvas[p.y][p.x] = newColor

		queue = append(queue, struct{ x, y int }{p.x + 1, p.y})
		queue = append(queue, struct{ x, y int }{p.x - 1, p.y})
		queue = append(queue, struct{ x, y int }{p.x, p.y + 1})
		queue = append(queue, struct{ x, y int }{p.x, p.y - 1})
	}
}

// Render desenha o painter
func (tp *TexturePainter) Render() {
	// Fundo do canvas
	rl.DrawRectangle(tp.CanvasX-2, tp.CanvasY-2, 32*tp.PixelSize+4, 32*tp.PixelSize+4, rl.DarkGray)

	// Desenhar pixels
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			color := tp.Canvas[y][x]
			px := tp.CanvasX + int32(x)*tp.PixelSize
			py := tp.CanvasY + int32(y)*tp.PixelSize

			// Fundo xadrez para transparência
			if color.A < 255 {
				checkColor := rl.LightGray
				if (x+y)%2 == 0 {
					checkColor = rl.White
				}
				rl.DrawRectangle(px, py, tp.PixelSize, tp.PixelSize, checkColor)
			}

			if color.A > 0 {
				rl.DrawRectangle(px, py, tp.PixelSize, tp.PixelSize, color)
			}
		}
	}

	// Grid
	for i := int32(0); i <= 32; i++ {
		rl.DrawLine(tp.CanvasX+i*tp.PixelSize, tp.CanvasY, tp.CanvasX+i*tp.PixelSize, tp.CanvasY+32*tp.PixelSize, rl.NewColor(100, 100, 100, 100))
		rl.DrawLine(tp.CanvasX, tp.CanvasY+i*tp.PixelSize, tp.CanvasX+32*tp.PixelSize, tp.CanvasY+i*tp.PixelSize, rl.NewColor(100, 100, 100, 100))
	}

	// Paleta de cores
	paletteX := tp.CanvasX + 32*tp.PixelSize + 30
	paletteY := tp.CanvasY
	rl.DrawText("Cores:", paletteX, paletteY-25, 16, rl.White)

	colorSize := int32(20)
	colorsPerRow := 8
	for i, col := range tp.Palette {
		cx := paletteX + int32(i%colorsPerRow)*colorSize
		cy := paletteY + int32(i/colorsPerRow)*colorSize

		rl.DrawRectangle(cx, cy, colorSize-1, colorSize-1, col)

		if i == tp.SelectedPalette {
			rl.DrawRectangleLines(cx-1, cy-1, colorSize+1, colorSize+1, rl.Yellow)
		}

		// Detectar clique
		rect := rl.NewRectangle(float32(cx), float32(cy), float32(colorSize-1), float32(colorSize-1))
		if rl.CheckCollisionPointRec(rl.GetMousePosition(), rect) && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			tp.SelectedPalette = i
			tp.CurrentColor = col
		}
	}

	// Cor atual
	currentY := paletteY + int32(len(tp.Palette)/colorsPerRow+1)*colorSize + 10
	rl.DrawText("Atual:", paletteX, currentY, 14, rl.White)
	rl.DrawRectangle(paletteX+50, currentY-2, 30, 20, tp.CurrentColor)

	// Ferramentas
	toolY := currentY + 40
	rl.DrawText("Ferramentas:", paletteX, toolY, 14, rl.White)
	toolY += 20

	tools := []struct {
		name string
		tool PaintTool
	}{
		{"Lápis", ToolPencil},
		{"Borracha", ToolEraser},
		{"Balde", ToolFill},
		{"Conta-gotas", ToolPicker},
	}

	for i, t := range tools {
		btnY := toolY + int32(i)*25
		color := rl.NewColor(70, 70, 70, 255)
		if tp.Tool == t.tool {
			color = rl.NewColor(100, 100, 150, 255)
		}

		rect := rl.NewRectangle(float32(paletteX), float32(btnY), 100, 22)
		rl.DrawRectangleRec(rect, color)
		rl.DrawText(t.name, paletteX+5, btnY+3, 14, rl.White)

		if rl.CheckCollisionPointRec(rl.GetMousePosition(), rect) && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			tp.Tool = t.tool
		}
	}

	// Tamanho do pincel
	brushY := toolY + int32(len(tools))*25 + 20
	rl.DrawText(fmt.Sprintf("Pincel: %d", tp.BrushSize), paletteX, brushY, 14, rl.White)
	rl.DrawText("(scroll para mudar)", paletteX, brushY+15, 10, rl.Gray)

	// Atalhos
	rl.DrawText("Ctrl+Z: Desfazer | Ctrl+Y: Refazer", tp.CanvasX, tp.CanvasY+32*tp.PixelSize+10, 12, rl.Gray)
}

// ToImage converte o canvas para image.Image
func (tp *TexturePainter) ToImage() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 32, 32))

	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			c := tp.Canvas[y][x]
			img.Set(x, y, color.RGBA{c.R, c.G, c.B, c.A})
		}
	}

	return img
}

// LoadFromImage carrega uma imagem no canvas
func (tp *TexturePainter) LoadFromImage(img image.Image) {
	bounds := img.Bounds()
	for y := 0; y < 32 && y < bounds.Dy(); y++ {
		for x := 0; x < 32 && x < bounds.Dx(); x++ {
			r, g, b, a := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			tp.Canvas[y][x] = rl.NewColor(uint8(r>>8), uint8(g>>8), uint8(b>>8), uint8(a>>8))
		}
	}
	tp.SaveHistory()
}

func colorEquals(a, b rl.Color) bool {
	return a.R == b.R && a.G == b.G && a.B == b.B && a.A == b.A
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
