package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
)

type HexadecaryNode struct {
	isLeaf       bool
	colorCount   int
	redTotal     int
	greenTotal   int
	blueTotal    int
	alphaTotal   int
	children     [16]*HexadecaryNode
	paletteIndex int
}

type HexadecaryTree struct {
	root       *HexadecaryNode
	colorDepth int
	leafCount  int
	reducible  [16]*HexadecaryNode
	palette    []color.Color
}

func NewHexaTree(colorDepth int) *HexadecaryTree {
	return &HexadecaryTree{
		root:       &HexadecaryNode{},
		colorDepth: colorDepth,
	}
}

func BuildTree(img image.Image, octree *HexadecaryTree) {
	// Retrieves the bounds of an image in order to determine the range of pixels to iterate over.
	bounds := img.Bounds()
	// Iterates over each pixel within the bounds.
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			// Retrieves the color of the pixel at the specified position.
			color := img.At(x, y)
			// Inserts the color into the octree.
			octree.InsertColor(color)
			// Prints the first 10 colors of the first row. (Debugging)
			/* if y == 0 && x < 10 {
			       fmt.Printf("Color at (0,%d): %v\n", x, color)
			   }
			*/
		}
	}
}

func (o *HexadecaryTree) InsertColor(c color.Color) {
	const maxDepth = 8
	const maxColors = 256

	rgba := color.RGBAModel.Convert(c).(color.RGBA)
	r, g, b, a := rgba.R, rgba.G, rgba.B, rgba.A;

	currentNode := o.root
	bitMask := 1 << (maxDepth - 1)

	for level := 0; level < o.colorDepth && level < maxDepth; level++ {
		index := ((int(r) & bitMask) >> (maxDepth - 4 - level)) |
			((int(g) & bitMask) >> (maxDepth - 3 - level)) |
			((int(b) & bitMask) >> (maxDepth - 2 - level)) |
			((int(a) & bitMask) >> (maxDepth - 1 - level))

		if currentNode.children[index] == nil {
			currentNode.children[index] = &HexadecaryNode{}
			if level < maxDepth-1 {
				o.reducible[level+1] = currentNode.children[index]
			}
			if level == maxDepth-1 {
				o.leafCount++
			}
		}

		currentNode = currentNode.children[index]
		bitMask >>= 1
	}

	if !currentNode.isLeaf {
		currentNode.isLeaf = true
	}
	currentNode.colorCount++
	currentNode.redTotal += int(r)
	currentNode.greenTotal += int(g)
	currentNode.blueTotal += int(b)
	currentNode.alphaTotal += int(a)

	if o.leafCount > maxColors {
		o.Reduce()
	}
}

func (o *HexadecaryTree) Reduce() {
	for o.leafCount > 256 {
		var levelToReduce int
		for i := len(o.reducible) - 1; i >= 0; i-- {
			if o.reducible[i] != nil {
				levelToReduce = i
				break
			}
		}
		nodeToReduce := o.reducible[levelToReduce]
		if nodeToReduce == nil {
			return
		}
		o.reducible[levelToReduce] = nil
		redTotal, greenTotal, blueTotal, alphaTotal, colorCount := 0, 0, 0, 0, 0
		for i, child := range nodeToReduce.children {
			if child != nil {
				redTotal += child.redTotal
				greenTotal += child.greenTotal
				blueTotal += child.blueTotal
				alphaTotal += child.alphaTotal
				colorCount += child.colorCount
				o.leafCount--
				nodeToReduce.children[i] = nil
			}
		}
		nodeToReduce.isLeaf = true
		nodeToReduce.redTotal = redTotal
		nodeToReduce.greenTotal = greenTotal
		nodeToReduce.blueTotal = blueTotal
		nodeToReduce.alphaTotal = alphaTotal
		nodeToReduce.colorCount = colorCount
		o.leafCount++
	}
}

func (o *HexadecaryTree) BuildPalette() {
	o.palette = make([]color.Color, 0, 256)
	o.buildPaletteRecursive(o.root)
}

func (o *HexadecaryTree) buildPaletteRecursive(node *HexadecaryNode) {
	if node == nil {
		return
	}
	if node.isLeaf {
		if len(o.palette) >= 256 {
			return
		}
		averageColor := color.RGBA{
			R: uint8(node.redTotal / node.colorCount),
			G: uint8(node.greenTotal / node.colorCount),
			B: uint8(node.blueTotal / node.colorCount),
			A: uint8(node.alphaTotal / node.colorCount),
		}
		node.paletteIndex = len(o.palette)
		o.palette = append(o.palette, averageColor)
	} else {
		for _, child := range node.children {
			o.buildPaletteRecursive(child)
		}
	}
}

func (o *HexadecaryTree) GetPaletteIndex(c color.Color) int {
	rgba := color.RGBAModel.Convert(c).(color.RGBA)
	r, g, b, a := rgba.R, rgba.G, rgba.B, rgba.A;

	currentNode := o.root
	for level := 0; level < o.colorDepth; level++ {
		index := 0
		if r&(1<<(7-level)) != 0 {
			index |= 0b1000
		}
		if g&(1<<(7-level)) != 0 {
			index |= 0b0100
		}
		if b&(1<<(7-level)) != 0 {
			index |= 0b0010
		}
		if a&(1<<(7-level)) != 0 {
			index |= 0b0001
		}
		// insert A here somehow.
		if currentNode.children[index] == nil {
			break
		}
		currentNode = currentNode.children[index]
	}

	if currentNode.isLeaf {
		return currentNode.paletteIndex
	}

	return 0
}

func (o *HexadecaryTree) ConvertToPaletted(img image.Image) *image.Paletted {
	bounds := img.Bounds()
	palettedImage := image.NewPaletted(bounds, o.palette)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			colorIndex := o.GetPaletteIndex(img.At(x, y))
			palettedImage.SetColorIndex(x, y, uint8(colorIndex))
		}
	}
	return palettedImage
}

func main() {
    // start := time.Now()

    file, err := os.Open("rem.png")
    if err != nil {
        panic(err)
    }
    defer file.Close()
    img, err := png.Decode(file)
    if err != nil {
        panic(err)
    }
    // fmt.Printf("Image loading took: %v\n", time.Since(start))

    // octreeStart := time.Now()
    hexaTree := NewHexaTree(3)
	BuildTree(img, hexaTree)

    // fmt.Printf("hexaTree insertion took: %v\n", time.Since(hexaTreeStart))

    // buildPaletteStart := time.Now()
    hexaTree.BuildPalette()
    // fmt.Printf("Building palette took: %v\n", time.Since(buildPaletteStart))
/*
    fmt.Printf("Palette built with %d colors\n", len(hexaTree.palette))
    for i, color := range hexaTree.palette {
        fmt.Printf("Palette[%d] = %v\n", i, color)
    }
		*/

    // convertStart := time.Now()
    palettedImage := hexaTree.ConvertToPaletted(img)
    // fmt.Printf("Converting to paletted image took: %v\n", time.Since(convertStart))

    // saveStart := time.Now()
    outFile, err := os.Create("rem_paletted.png")
    if err != nil {
        panic(err)
    }
    defer outFile.Close()
    err = png.Encode(outFile, palettedImage)
    if err != nil {
        panic(err)
    }
    // fmt.Printf("Saving paletted image took: %v\n", time.Since(saveStart))

    // fmt.Printf("Total execution time: %v\n", time.Since(start))
}