package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"time"
)

type OctreeNode struct {
	isLeaf       bool
	colorCount   int
	redTotal     int
	greenTotal   int
	blueTotal    int
	children     [8]*OctreeNode
	paletteIndex int
}

type Octree struct {
	root       *OctreeNode
	colorDepth int
	leafCount  int
	reducible  [8]*OctreeNode
	palette    []color.Color
}


func NewOctree(colorDepth int) *Octree {
	return &Octree{
		root:       &OctreeNode{},
		colorDepth: colorDepth,
	}
}

func (o *Octree) InsertColor(c color.Color) {
    const maxDepth = 8
    const maxColors = 256

    rgba := color.RGBAModel.Convert(c).(color.RGBA)
    r, g, b := rgba.R, rgba.G, rgba.B

    currentNode := o.root
    bitMask := 1 << (maxDepth - 1)

	for level := 0; level < o.colorDepth && level < maxDepth; level++ {
		index := ((int(r) & bitMask) >> (maxDepth - 3 - level)) |
				 ((int(g) & bitMask) >> (maxDepth - 2 - level)) |
				 ((int(b) & bitMask) >> (maxDepth - 1 - level))
	
		if currentNode.children[index] == nil {
			currentNode.children[index] = &OctreeNode{}
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

    if o.leafCount > maxColors {
        o.Reduce()
    }
}

func (o *Octree) Reduce() {
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
		redTotal, greenTotal, blueTotal, colorCount := 0, 0, 0, 0
		for i, child := range nodeToReduce.children {
			if child != nil {
				redTotal += child.redTotal
				greenTotal += child.greenTotal
				blueTotal += child.blueTotal
				colorCount += child.colorCount
				o.leafCount--
				nodeToReduce.children[i] = nil
			}
		}
		nodeToReduce.isLeaf = true
		nodeToReduce.redTotal = redTotal
		nodeToReduce.greenTotal = greenTotal
		nodeToReduce.blueTotal = blueTotal
		nodeToReduce.colorCount = colorCount
		o.leafCount++
	}
}

func (o *Octree) BuildPalette() {
	o.palette = make([]color.Color, 0, 256)
	o.buildPaletteRecursive(o.root)
}

func (o *Octree) buildPaletteRecursive(node *OctreeNode) {
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
			A: 255,
		}
		node.paletteIndex = len(o.palette)
		o.palette = append(o.palette, averageColor)
	} else {
		for _, child := range node.children {
			o.buildPaletteRecursive(child)
		}
	}
}

func (o *Octree) GetPaletteIndex(c color.Color) int {
	rgba := color.RGBAModel.Convert(c).(color.RGBA)
	r, g, b := rgba.R, rgba.G, rgba.B

	currentNode := o.root
	for level := 0; level < o.colorDepth; level++ {
		index := 0
		if r&(1<<(7-level)) != 0 {
			index |= 4
		}
		if g&(1<<(7-level)) != 0 {
			index |= 2
		}
		if b&(1<<(7-level)) != 0 {
			index |= 1
		}
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

func (o *Octree) ConvertToPaletted(img image.Image) *image.Paletted {
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
    start := time.Now()

    file, err := os.Open("rem.png")
    if err != nil {
        panic(err)
    }
    defer file.Close()
    img, err := png.Decode(file)
    if err != nil {
        panic(err)
    }
    fmt.Printf("Image loading took: %v\n", time.Since(start))

    octreeStart := time.Now()
    octree := NewOctree(4)
    bounds := img.Bounds()
    for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
        for x := bounds.Min.X; x < bounds.Max.X; x++ {
            color := img.At(x, y)
            octree.InsertColor(color)
            if y == 0 && x < 10 {
                fmt.Printf("Color at (0,%d): %v\n", x, color)
            }
        }
    }
    fmt.Printf("Octree insertion took: %v\n", time.Since(octreeStart))

    buildPaletteStart := time.Now()
    octree.BuildPalette()
    fmt.Printf("Building palette took: %v\n", time.Since(buildPaletteStart))

    fmt.Printf("Palette built with %d colors\n", len(octree.palette))
    for i, color := range octree.palette {
        fmt.Printf("Palette[%d] = %v\n", i, color)
    }

    convertStart := time.Now()
    palettedImage := octree.ConvertToPaletted(img)
    fmt.Printf("Converting to paletted image took: %v\n", time.Since(convertStart))

    saveStart := time.Now()
    outFile, err := os.Create("rem_paletted.png")
    if err != nil {
        panic(err)
    }
    defer outFile.Close()
    err = png.Encode(outFile, palettedImage)
    if err != nil {
        panic(err)
    }
    fmt.Printf("Saving paletted image took: %v\n", time.Since(saveStart))

    fmt.Printf("Total execution time: %v\n", time.Since(start))
}