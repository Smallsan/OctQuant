package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
	// "time"
)


// OctreeNode represents a node in the octree palette.
type OctreeNode struct {
	isLeaf       bool
	colorCount   int
	redTotal     int
	greenTotal   int
	blueTotal    int
	children     [8]*OctreeNode
	paletteIndex int
}

// Octree represents an octree data structure used for color quantization.
type Octree struct {
	root       *OctreeNode    // root is a pointer to the root node of the octree.
	colorDepth int            // colorDepth represents the number of bits used to represent each color component.
	leafCount  int            // leafCount represents the number of leaf nodes in the octree.
	reducible  [8]*OctreeNode // reducible is an array of pointers to reducible nodes in the octree.
	palette    []color.Color  // palette is a slice that stores the color palette generated from the octree.
}



// NewOctree creates a new Octree with the specified color depth.
// The color depth determines the number of bits used to represent each color channel.
// Higher color depth allows for more accurate color representation but requires more memory.
func NewOctree(colorDepth int) *Octree {
	return &Octree{
		root:       &OctreeNode{},
		colorDepth: colorDepth,
	}
}

// BuildTree builds an octree from the given image.
// It iterates over each pixel in the image and inserts the color into the octree.
// It also prints the color at position (0, x) if y is 0 and x is less than 10.
func BuildTree(img image.Image, octree *Octree) {
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

// InsertColor inserts a color into the Octree.
// It takes a color.Color as input and inserts it into the Octree data structure.
// The color is converted to RGBA format and then inserted into the appropriate node in the Octree.
// If the Octree exceeds the maximum number of colors, it will be reduced.
func (o *Octree) InsertColor(c color.Color) {
    const maxDepth = 8
    const maxColors = 256

	// Converts the color to RGBA format.
    rgba := color.RGBAModel.Convert(c).(color.RGBA)
    r, g, b := rgba.R, rgba.G, rgba.B

	// Initializes the current node to the root of the Octree.
    currentNode := o.root
	// Initializes the bit mask to the highest bit of the color depth.
    bitMask := 1 << (maxDepth - 1)

	// Iterates over the color depth and the maximum depth to insert the color into the Octree.
	for level := 0; level < o.colorDepth && level < maxDepth; level++ {
		// Calculates the index of the child node based on the color components and the bit mask.
		index := ((int(r) & bitMask) >> (maxDepth - 3 - level)) |
				 ((int(g) & bitMask) >> (maxDepth - 2 - level)) |
				 ((int(b) & bitMask) >> (maxDepth - 1 - level))
	
				 // If the child node is nil, it creates a new node and adds it to the reducible list if it is not a leaf node.
		if currentNode.children[index] == nil {
			currentNode.children[index] = &OctreeNode{}
			// If the current level is less than the maximum depth, it adds the child node to the reducible list.
			if level < maxDepth-1 {
                o.reducible[level+1] = currentNode.children[index]
            }
			// If the current level is equal to the maximum depth minus one, it increments the leaf count.
            if level == maxDepth-1 {
                o.leafCount++
            }
        }
// Updates the current node to the child node at the calculated index and shifts the bit mask to the right.
        currentNode = currentNode.children[index]
		// Updates the bit mask by shifting it to the right.
        bitMask >>= 1
    }

	// If the current node is not a leaf node, it marks it as a leaf node and increments the leaf count.
  if !currentNode.isLeaf {
        currentNode.isLeaf = true
    }
    currentNode.colorCount++
    currentNode.redTotal += int(r)
    currentNode.greenTotal += int(g)
    currentNode.blueTotal += int(b)

	// If the leaf count exceeds the maximum number of colors, it reduces the Octree.
    if o.leafCount > maxColors {
        o.Reduce()
    }
}

// Reduce reduces the octree by merging nodes until the leaf count is less than or equal to 256.
// It iterates through the reducible nodes in reverse order and merges them if they are not nil.
// After merging, it updates the total color values and color count of the merged node.
// Finally, it marks the merged node as a leaf and increments the leaf count.
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

// BuildPalette builds the color palette for the Octree.
// It initializes the palette slice and calls the buildPaletteRecursive method to populate it.
func (o *Octree) BuildPalette() {
	o.palette = make([]color.Color, 0, 256)
	o.buildPaletteRecursive(o.root)
}

// buildPaletteRecursive recursively builds the color palette for the Octree.
// It traverses the Octree and calculates the average color for each leaf node.
// If the palette already contains 256 colors, the function stops adding new colors.
// The calculated average color is added to the palette and the palette index is assigned to the node.
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

// GetPaletteIndex returns the palette index for a given color.
// It traverses the octree to find the leaf node corresponding to the color,
// and returns the palette index stored in that node.
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


// ConvertToPaletted converts the given image to a paletted image using the Octree's palette.
// It returns a pointer to the converted paletted image.
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
    octree := NewOctree(4)
	BuildTree(img, octree)

    // fmt.Printf("Octree insertion took: %v\n", time.Since(octreeStart))

    // buildPaletteStart := time.Now()
    octree.BuildPalette()
    // fmt.Printf("Building palette took: %v\n", time.Since(buildPaletteStart))
/*
    fmt.Printf("Palette built with %d colors\n", len(octree.palette))
    for i, color := range octree.palette {
        fmt.Printf("Palette[%d] = %v\n", i, color)
    }
		*/

    // convertStart := time.Now()
    palettedImage := octree.ConvertToPaletted(img)
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