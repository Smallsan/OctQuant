# Hexadecary Tree Color Quantization in Go
(Faster than golang's std brute force approach)

This Go program demonstrates the implementation of a hexadecary tree (16-ary tree) color quantization algorithm. The algorithm is designed to reduce the number of colors in an image more efficiently than the traditional octree method, making it suitable for formats that support a limited color palette, such as GIFs or 8-bit PNGs. The goal is to preserve the visual fidelity of the original image as much as possible while reducing the color count.

## Features

- **Hexadecary Tree Color Quantization**: More efficient color reduction in images compared to octree quantization.
- **Palette Generation**: Builds a color palette that closely matches the original image's colors.
- **Image Conversion**: Converts images to use the generated palette, resulting in smaller file sizes without significant loss of quality.

## How It Works

The program constructs a hexadecary tree where each node represents a color or a range of colors. As colors are inserted into the tree, it grows dynamically, with each level of the tree representing a deeper color precision. When the tree exceeds a predefined number of colors, it is reduced by merging nodes, thus pruning the tree and reducing the color count.

## Usage

1. **Setting Up**: Ensure you have Go installed on your system.
2. **Running the Program**: Use the command `go run main.go` to execute the program. The input image should be named `input.png` and placed in the same directory as the program.
3. **Output**: The program generates a new image named `output_paletted.png`, which uses the reduced color palette.

## Implementation Details

- **HexadecaryTreeNode**: Represents a node in the hexadecary tree. Leaf nodes store color information, while non-leaf nodes are used for navigation.
- **HexadecaryTree**: Manages the hexadecary tree structure, including insertion of colors, reduction of the tree, and palette generation.
- **Main Function**: Handles image loading, hexadecary tree construction, palette building, image conversion, and saving the converted image.


## Sampe Usage

```go
    file, err := os.Open("input.png")
    if err != nil {
        panic(err)
    }

    defer file.Close()
    img, err := png.Decode(file)
    if err != nil {
        panic(err)
    }

    octree := NewOctree(4)

    BuildTree(img, octree)

    octree.BuildPalette()

    palettedImage := octree.ConvertToPaletted(img)

    outFile, err := os.Create("output.png")
    if err != nil {
        panic(err)
    }
    defer outFile.Close()
    err = png.Encode(outFile, palettedImage)
    if err != nil {
        panic(err)
    }
```

## Image Examples and Performance

Below are examples showing the results of the octree color quantization algorithm at different tree depths, along with their performance metrics. The original image size is 171kb.

### Original Image

![Original Image](examples/original.png)

- **Size**: 171kb

### Depth 1

![Depth 1](examples/depth_1.png)

- **Total execution time**: 19.0033ms
- **Reduced size**: 10kb

### Depth 2

![Depth 2](examples/depth_2.png)

- **Total execution time**: 19.0033ms
- **Reduced size**: 17kb

### Depth 3

![Depth 3](examples/depth_3.png)

- **Total execution time**: 32.7629ms
- **Reduced size**: 27kb

### Depth 4

![Depth 4](examples/depth_4.png)

- **Total execution time**: 45.066ms
- **Reduced size**: 29kb

## Dependencies

- The program uses the standard Go library for image processing (`image`, `image/color`, `image/png`).

## Contributing

Contributions to improve the algorithm or extend the functionality are welcome. Please feel free to fork the repository, make your changes, and submit a pull request.

## License

This project is open-sourced under the MIT License. See the LICENSE file for more details.
