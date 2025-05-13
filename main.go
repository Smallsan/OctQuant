package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type QuantizerConfig struct {
	MaxColors int
	MaxDepth  int
}

// NewConfig creates a configuration with the specified max depth
// and calculates a reasonable default for MaxColors based on the depth
func NewConfig(maxDepth int) QuantizerConfig {
	// A reasonable default is to use 2^maxDepth colors
	// This is less than the theoretical maximum (8^maxDepth)
	// but provides good balance between quality and palette size
	maxColors := 1 << maxDepth // 2^maxDepth
	if maxColors > 256 {
		maxColors = 256 // Cap at 256 for most image formats
	}

	return QuantizerConfig{
		MaxColors: maxColors,
		MaxDepth:  maxDepth,
	}
}

// DefaultConfig returns a default configuration with depth 8
func DefaultConfig() QuantizerConfig {
	return NewConfig(8)
}

// Custom configuration with explicit control over both parameters
func CustomConfig(maxDepth int, maxColors int) QuantizerConfig {
	return QuantizerConfig{
		MaxColors: maxColors,
		MaxDepth:  maxDepth,
	}
}

type Color struct {
	Red, Green, Blue, Alpha int
}

type OctreeNode struct {
	Color        Color
	PixelCount   int
	PaletteIndex int
	Children     [8]*OctreeNode
}

type OctreeQuantizer struct {
	Levels map[int][]*OctreeNode
	Root   *OctreeNode
	Config QuantizerConfig
}

func NewColor(red, green, blue, alpha int) Color {
	return Color{Red: red, Green: green, Blue: blue, Alpha: alpha}
}

func NewOctreeNode(level int, parent *OctreeQuantizer) *OctreeNode {
	node := &OctreeNode{
		Color: Color{0, 0, 0, 0},
	}
	if level < parent.Config.MaxDepth-1 {
		parent.AddLevelNode(level, node)
	}
	return node
}

func (node *OctreeNode) IsLeaf() bool {
	return node.PixelCount > 0
}

func (node *OctreeNode) GetLeafNodes() []*OctreeNode {
	var leafNodes []*OctreeNode
	for _, child := range node.Children {
		if child != nil {
			if child.IsLeaf() {
				leafNodes = append(leafNodes, child)
			} else {
				leafNodes = append(leafNodes, child.GetLeafNodes()...)
			}
		}
	}
	return leafNodes
}

func (node *OctreeNode) GetNodesPixelCount() int {
	sumCount := node.PixelCount
	for _, child := range node.Children {
		if child != nil {
			sumCount += child.PixelCount
		}
	}
	return sumCount
}

func (node *OctreeNode) AddColor(color Color, level int, parent *OctreeQuantizer) {
	if level >= parent.Config.MaxDepth {
		node.Color.Red += color.Red
		node.Color.Green += color.Green
		node.Color.Blue += color.Blue
		node.Color.Alpha += color.Alpha
		node.PixelCount++
		return
	}
	index := node.GetColorIndexForLevel(color, level)
	if node.Children[index] == nil {
		node.Children[index] = NewOctreeNode(level, parent)
	}
	node.Children[index].AddColor(color, level+1, parent)
}

func (node *OctreeNode) GetPaletteIndex(color Color, level int) int {
	if node.IsLeaf() {
		return node.PaletteIndex
	}
	index := node.GetColorIndexForLevel(color, level)
	if node.Children[index] != nil {
		return node.Children[index].GetPaletteIndex(color, level+1)
	}
	for _, child := range node.Children {
		if child != nil {
			return child.GetPaletteIndex(color, level+1)
		}
	}
	return 0
}

func (node *OctreeNode) RemoveLeaves() int {
	result := 0
	for i := range node.Children {
		if node.Children[i] != nil {
			node.Color.Red += node.Children[i].Color.Red
			node.Color.Green += node.Children[i].Color.Green
			node.Color.Blue += node.Children[i].Color.Blue
			node.Color.Alpha += node.Children[i].Color.Alpha
			node.PixelCount += node.Children[i].PixelCount
			result++
		}
	}
	return result - 1
}

func (node *OctreeNode) GetColorIndexForLevel(color Color, level int) int {
	index := 0
	mask := 0x80 >> level
	if color.Red&mask != 0 {
		index |= 4
	}
	if color.Green&mask != 0 {
		index |= 2
	}
	if color.Blue&mask != 0 {
		index |= 1
	}
	return index
}

func (node *OctreeNode) GetColor() Color {
	if node.PixelCount == 0 {
		return Color{0, 0, 0, 0}
	}
	return Color{
		Red:   node.Color.Red / node.PixelCount,
		Green: node.Color.Green / node.PixelCount,
		Blue:  node.Color.Blue / node.PixelCount,
		Alpha: node.Color.Alpha / node.PixelCount,
	}
}

func NewOctreeQuantizer(config QuantizerConfig) *OctreeQuantizer {
	quantizer := &OctreeQuantizer{
		Levels: make(map[int][]*OctreeNode),
		Config: config,
	}
	quantizer.Root = NewOctreeNode(0, quantizer)
	return quantizer
}

func (quantizer *OctreeQuantizer) GetLeaves() []*OctreeNode {
	return quantizer.Root.GetLeafNodes()
}

func (quantizer *OctreeQuantizer) AddLevelNode(level int, node *OctreeNode) {
	quantizer.Levels[level] = append(quantizer.Levels[level], node)
}

func (quantizer *OctreeQuantizer) AddColor(color Color) {
	quantizer.Root.AddColor(color, 0, quantizer)
}

func (quantizer *OctreeQuantizer) MakePalette() []Color {
	var palette []Color
	paletteIndex := 0
	leafCount := len(quantizer.GetLeaves())
	for level := quantizer.Config.MaxDepth - 1; level >= 0; level-- {
		if nodes, exists := quantizer.Levels[level]; exists {
			for _, node := range nodes {
				leafCount -= node.RemoveLeaves()
				if leafCount <= quantizer.Config.MaxColors {
					break
				}
			}
			if leafCount <= quantizer.Config.MaxColors {
				break
			}
			quantizer.Levels[level] = nil
		}
	}
	for _, node := range quantizer.GetLeaves() {
		if paletteIndex >= quantizer.Config.MaxColors {
			break
		}
		if node.IsLeaf() {
			palette = append(palette, node.GetColor())
			node.PaletteIndex = paletteIndex
			paletteIndex++
		}
	}
	return palette
}

func (quantizer *OctreeQuantizer) GetPaletteIndex(color Color) int {
	return quantizer.Root.GetPaletteIndex(color, 0)
}
func convertToColorPalette(palette []Color) color.Palette {
	var colorPalette color.Palette
	for _, c := range palette {
		colorPalette = append(colorPalette, color.RGBA{
			R: uint8(c.Red),
			G: uint8(c.Green),
			B: uint8(c.Blue),
			A: uint8(c.Alpha),
		})
	}
	return colorPalette
}

func outputPalette(palette color.Palette) {
	pixelSize := 10
	paletteImg := image.NewRGBA(image.Rect(0, 0, 8*pixelSize, ((len(palette)+7)/8)*pixelSize))
	for i, c := range palette {
		x := (i % 8) * pixelSize
		y := (i / 8) * pixelSize
		for dx := 0; dx < pixelSize; dx++ {
			for dy := 0; dx < pixelSize; dy++ {
				r, g, b, a := c.RGBA()
				paletteImg.Set(x+dx, y+dy, color.RGBA{
					R: uint8(r >> 8),
					G: uint8(g >> 8),
					B: uint8(b >> 8),
					A: uint8(a >> 8),
				})
			}
		}
	}
	paletteFile, err := os.Create("palette.png")
	if err != nil {
		fmt.Println("Error creating palette image:", err)
		return
	}
	defer paletteFile.Close()

	err = png.Encode(paletteFile, paletteImg)
	if err != nil {
		fmt.Println("Error encoding palette image:", err)
		return
	}

	fmt.Println("Palette image saved as palette.png")
}

// Function to process GIFs with specified depth and color count
func forGifs(depth int, maxColors int) {
	start := time.Now()

	inputPath := "sd.gif"
	file, err := os.Open(inputPath)
	if err != nil {
		fmt.Println("Error opening GIF file:", err)
		return
	}
	defer file.Close()

	// Extract directory and filename for output path
	dir := filepath.Dir(inputPath)
	filename := filepath.Base(inputPath)
	filenameWithoutExt := filename[:len(filename)-len(filepath.Ext(filename))]

	outputPath := filepath.Join(dir, filenameWithoutExt+"_quantized.gif")
	palettePath := filepath.Join(dir, filenameWithoutExt+"_palette.png")

	gifImg, err := gif.DecodeAll(file)
	if err != nil {
		fmt.Println("Error decoding GIF:", err)
		return
	}

	config := CustomConfig(depth, maxColors)
	quantizer := NewOctreeQuantizer(config)

	for _, frame := range gifImg.Image {
		bounds := frame.Bounds()
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				r, g, b, a := frame.At(x, y).RGBA()
				color := NewColor(int(r>>8), int(g>>8), int(b>>8), int(a>>8))
				quantizer.AddColor(color)
			}
		}
	}

	palette := quantizer.MakePalette()

	colorPalette := convertToColorPalette(palette)

	colorPalette = append(colorPalette, color.RGBA{0, 0, 0, 0})
	transparentIndex := len(colorPalette) - 1

	bounds := gifImg.Image[0].Bounds()
	baseImage := image.NewRGBA(bounds)

	newGif := &gif.GIF{BackgroundIndex: uint8(transparentIndex)}
	for i, frame := range gifImg.Image {
		draw.Draw(baseImage, bounds, frame, image.Point{}, draw.Over)

		quantizedFrame := image.NewPaletted(bounds, colorPalette)
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				r, g, b, a := baseImage.At(x, y).RGBA()
				color := NewColor(int(r>>8), int(g>>8), int(b>>8), int(a>>8))
				if a == 0 {
					quantizedFrame.SetColorIndex(x, y, uint8(transparentIndex))
				} else {
					index := quantizer.GetPaletteIndex(color)
					quantizedFrame.SetColorIndex(x, y, uint8(index))
				}
			}
		}

		newGif.Disposal = append(newGif.Disposal, gif.DisposalNone)
		newGif.Image = append(newGif.Image, quantizedFrame)
		newGif.Delay = append(newGif.Delay, gifImg.Delay[i])
		newGif.Config.ColorModel = colorPalette
		newGif.Config.Width = bounds.Dx()
		newGif.Config.Height = bounds.Dy()
	}

	outputFile, err := os.Create(outputPath)
	if err != nil {
		fmt.Println("Error creating output GIF file:", err)
		return
	}
	defer outputFile.Close()

	err = gif.EncodeAll(outputFile, newGif)
	if err != nil {
		fmt.Println("Error encoding GIF:", err)
		return
	}

	fmt.Printf("Quantized GIF saved as %s\n", outputPath)

	// Create palette file for GIF
	pixelSize := 10
	paletteImg := image.NewRGBA(image.Rect(0, 0, 8*pixelSize, ((len(colorPalette)+7)/8)*pixelSize))
	for i, c := range colorPalette {
		x := (i % 8) * pixelSize
		y := (i / 8) * pixelSize
		for dx := 0; dx < pixelSize; dx++ {
			for dy := 0; dy < pixelSize; dy++ {
				r, g, b, a := c.RGBA()
				paletteImg.Set(x+dx, y+dy, color.RGBA{
					R: uint8(r >> 8),
					G: uint8(g >> 8),
					B: uint8(b >> 8),
					A: uint8(a >> 8),
				})
			}
		}
	}
	paletteFile, err := os.Create(palettePath)
	if err != nil {
		fmt.Println("Error creating palette image:", err)
		return
	}
	defer paletteFile.Close()

	err = png.Encode(paletteFile, paletteImg)
	if err != nil {
		fmt.Println("Error encoding palette image:", err)
		return
	}

	fmt.Printf("Palette image saved as %s\n", palettePath)

	elapsed := time.Since(start)
	fmt.Printf("Processing took %s\n", elapsed)
}

// Function to process images with specified depth and color count
func forImages(depth int, maxColors int) {
	inputPath := "input.png"
	file, err := os.Open(inputPath)
	if err != nil {
		fmt.Println("Error opening image:", err)
		return
	}
	defer file.Close()

	// Extract directory and filename for output path
	dir := filepath.Dir(inputPath)
	filename := filepath.Base(inputPath)
	filenameWithoutExt := filename[:len(filename)-len(filepath.Ext(filename))]

	outputPath := filepath.Join(dir, filenameWithoutExt+"_quantized.png")
	palettePath := filepath.Join(dir, filenameWithoutExt+"_palette.png")

	img, err := png.Decode(file)
	if err != nil {
		fmt.Println("Error decoding image:", err)
		return
	}

	// Use custom config with both parameters
	config := CustomConfig(depth, maxColors)
	quantizer := NewOctreeQuantizer(config)

	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			color := Color{
				Red:   int(r >> 8),
				Green: int(g >> 8),
				Blue:  int(b >> 8),
				Alpha: int(a >> 8),
			}
			quantizer.AddColor(color)
		}
	}

	palette := quantizer.MakePalette()
	colorCount := len(palette)

	quantizedImg := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			color_palette := Color{
				Red:   int(r >> 8),
				Green: int(g >> 8),
				Blue:  int(b >> 8),
				Alpha: int(a >> 8),
			}
			paletteIndex := quantizer.GetPaletteIndex(color_palette)
			quantizedColor := palette[paletteIndex]
			quantizedImg.Set(x, y, color.RGBA{
				R: uint8(quantizedColor.Red),
				G: uint8(quantizedColor.Green),
				B: uint8(quantizedColor.Blue),
				A: uint8(quantizedColor.Alpha),
			})
		}
	}

	outputFile, err := os.Create(outputPath)
	if err != nil {
		fmt.Println("Error creating output image:", err)
		return
	}
	defer outputFile.Close()

	err = png.Encode(outputFile, quantizedImg)
	if err != nil {
		fmt.Println("Error encoding output image:", err)
		return
	}

	fmt.Printf("Quantized image saved as %s\n", outputPath)

	// Create palette file for image
	pixelSize := 10
	paletteImg := image.NewRGBA(image.Rect(0, 0, 8*pixelSize, ((colorCount+7)/8)*pixelSize))
	for i, c := range palette {
		x := (i % 8) * pixelSize
		y := (i / 8) * pixelSize
		for dx := 0; dx < pixelSize; dx++ {
			for dy := 0; dy < pixelSize; dy++ {
				paletteImg.Set(x+dx, y+dy, color.RGBA{
					R: uint8(c.Red),
					G: uint8(c.Green),
					B: uint8(c.Blue),
					A: uint8(c.Alpha),
				})
			}
		}
	}
	paletteFile, err := os.Create(palettePath)
	if err != nil {
		fmt.Println("Error creating palette image:", err)
		return
	}
	defer paletteFile.Close()

	err = png.Encode(paletteFile, paletteImg)
	if err != nil {
		fmt.Println("Error encoding palette image:", err)
		return
	}

	fmt.Printf("Palette image saved as %s\n", palettePath)
}

func main() {
	// Default values
	depth := 8
	maxColors := 256
	inputFile := "input.png" // Default input file

	// Parse command line arguments
	if len(os.Args) > 1 {
		// Check if first argument is a file path
		if _, err := os.Stat(os.Args[1]); err == nil {
			inputFile = os.Args[1]

			// Shift arguments if first arg is a file
			if len(os.Args) > 2 {
				if d, err := strconv.Atoi(os.Args[2]); err == nil {
					if d >= 1 && d <= 8 {
						depth = d
					}
				}

				if len(os.Args) > 3 {
					if c, err := strconv.Atoi(os.Args[3]); err == nil {
						if c >= 2 && c <= 256 {
							maxColors = c
						}
					}
				}
			}
		} else {
			// First argument is depth
			if d, err := strconv.Atoi(os.Args[1]); err == nil {
				if d >= 1 && d <= 8 {
					depth = d
				} else {
					fmt.Println("Depth must be between 1 and 8, using default of 8")
				}
			}

			// Second argument is max colors (if provided)
			if len(os.Args) > 2 {
				if c, err := strconv.Atoi(os.Args[2]); err == nil {
					if c >= 2 && c <= 256 {
						maxColors = c
					} else {
						fmt.Println("MaxColors must be between 2 and 256, using default based on depth")
					}
				}
			} else {
				// If only depth is provided, use 2^depth as default for maxColors
				suggestedMaxColors := 1 << depth // 2^depth
				if suggestedMaxColors <= 256 {
					maxColors = suggestedMaxColors
				}
			}
		}
	}

	fmt.Printf("Processing file: %s with depth: %d, maxColors: %d\n", inputFile, depth, maxColors)

	// Determine if we're processing a GIF or other image format
	ext := filepath.Ext(inputFile)
	if ext == ".gif" {
		forGifs(depth, maxColors)
	} else {
		forImages(depth, maxColors)
	}
}
