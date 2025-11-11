package main

import (
	"fmt"
	"image"
	"image/png"
	"os"
)

func main() {
	// Carregar atlas
	atlasFile, err := os.Open("assets/texture_atlas.png")
	if err != nil {
		panic(err)
	}
	defer atlasFile.Close()

	atlasImg, _, err := image.Decode(atlasFile)
	if err != nil {
		panic(err)
	}

	// Parâmetros do atlas
	const gridSize = 8
	const tileSize = 32

	// Criar diretório de saída
	os.MkdirAll("assets/textures", 0755)

	// Extrair cada tile
	tileCount := 0
	for row := 0; row < gridSize; row++ {
		for col := 0; col < gridSize; col++ {
			// Criar imagem 32x32
			tileImg := image.NewRGBA(image.Rect(0, 0, tileSize, tileSize))

			// Copiar pixels do atlas
			for y := 0; y < tileSize; y++ {
				for x := 0; x < tileSize; x++ {
					srcX := col*tileSize + x
					srcY := row*tileSize + y
					color := atlasImg.At(srcX, srcY)
					tileImg.Set(x, y, color)
				}
			}

			// Salvar arquivo
			filename := fmt.Sprintf("assets/textures/tile_%d_%d.png", row, col)
			outFile, err := os.Create(filename)
			if err != nil {
				panic(err)
			}

			png.Encode(outFile, tileImg)
			outFile.Close()

			tileCount++
			fmt.Printf("Extraído: %s\n", filename)
		}
	}

	fmt.Printf("\nTotal: %d texturas extraídas\n", tileCount)
}
