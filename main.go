package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
)

type Color struct {
    Red, Green, Blue int
}

type OctreeNode struct {
    Color        Color
    PixelCount   int
    PaletteIndex int
    Children     [8]*OctreeNode
}

const MaxDepth = 8

type OctreeQuantizer struct {
    Levels map[int][]*OctreeNode
    Root   *OctreeNode
}

func NewColor(red, green, blue int) Color {
    return Color{Red: red, Green: green, Blue: blue}
}

func NewOctreeNode(level int, parent *OctreeQuantizer) *OctreeNode {
    node := &OctreeNode{
        Color: Color{0, 0, 0},
    }
    if level < MaxDepth-1 {
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
    if level >= MaxDepth {
        node.Color.Red += color.Red
        node.Color.Green += color.Green
        node.Color.Blue += color.Blue
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
        return Color{0, 0, 0}
    }
    return Color{
        Red:   node.Color.Red / node.PixelCount,
        Green: node.Color.Green / node.PixelCount,
        Blue:  node.Color.Blue / node.PixelCount,
    }
}

func NewOctreeQuantizer() *OctreeQuantizer {
    quantizer := &OctreeQuantizer{
        Levels: make(map[int][]*OctreeNode),
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

func (quantizer *OctreeQuantizer) MakePalette(colorCount int) []Color {
    var palette []Color
    paletteIndex := 0
    leafCount := len(quantizer.GetLeaves())
    for level := MaxDepth - 1; level >= 0; level-- {
        if nodes, exists := quantizer.Levels[level]; exists {
            for _, node := range nodes {
                leafCount -= node.RemoveLeaves()
                if leafCount <= colorCount {
                    break
                }
            }
            if leafCount <= colorCount {
                break
            }
            quantizer.Levels[level] = nil
        }
    }
    for _, node := range quantizer.GetLeaves() {
        if paletteIndex >= colorCount {
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

func main() {
    // Load the image
    file, err := os.Open("input.png")
    if err != nil {
        fmt.Println("Error opening image:", err)
        return
    }
    defer file.Close()

    img, err := png.Decode(file)
	if err != nil {
        fmt.Println("Error decoding image:", err)
        return
    }

    // Create an OctreeQuantizer
    quantizer := NewOctreeQuantizer()

    // Add colors from the image to the quantizer
    bounds := img.Bounds()
    for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
        for x := bounds.Min.X; x < bounds.Max.X; x++ {
            r, g, b, _ := img.At(x, y).RGBA()
            color := Color{
                Red:   int(r >> 8),
                Green: int(g >> 8),
                Blue:  int(b >> 8),
            }
            quantizer.AddColor(color)
        }
    }

    // Generate the color palette
    colorCount := 256 // Number of colors in the palette
    palette := quantizer.MakePalette(colorCount)

    // Create a new image with the quantized colors
    quantizedImg := image.NewRGBA(bounds)
    for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
        for x := bounds.Min.X; x < bounds.Max.X; x++ {
            r, g, b, _ := img.At(x, y).RGBA()
            color_palette := Color{
                Red:   int(r >> 8),
                Green: int(g >> 8),
                Blue:  int(b >> 8),
            }
            paletteIndex := quantizer.GetPaletteIndex(color_palette)
            quantizedColor := palette[paletteIndex]
            quantizedImg.Set(x, y, color.RGBA{
                R: uint8(quantizedColor.Red),
                G: uint8(quantizedColor.Green),
                B: uint8(quantizedColor.Blue),
                A: 255,
            })
        }
    }

    // Save the quantized image
    outputFile, err := os.Create("output.png")
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

    fmt.Println("Quantized image saved as output.png")

// Create a palette image with larger pixels
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
                A: 255,
            })
        }
    }
}
    // Save the palette image
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