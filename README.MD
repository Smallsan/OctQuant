# Octree Color Quantization in Go
## Still trying to fix depth 4-8, I know there's a part I don't understand well, but I'm trying to fix it.

This Go program demonstrates the implementation of an octree color quantization algorithm. The algorithm is designed to reduce the number of colors in an image, making it suitable for formats that support a limited color palette, such as GIFs or 8-bit PNGs. The goal is to preserve the visual fidelity of the original image as much as possible while reducing the color count.

## Features

- **Octree Color Quantization**: Efficiently reduces the number of colors in an image.
- **Palette Generation**: Builds a color palette that closely matches the original image's colors.
- **Image Conversion**: Converts images to use the generated palette, resulting in smaller file sizes without significant loss of quality.

## How It Works

The program constructs an octree where each node represents a color or a range of colors. As colors are inserted into the tree, it grows dynamically, with each level of the tree representing a deeper color precision. When the tree exceeds a predefined number of colors, it is reduced by merging nodes, thus pruning the tree and reducing the color count.

## Usage

1. **Setting Up**: Ensure you have Go installed on your system.
2. **Running the Program**: Use the command `go run main.go` to execute the program. The input image should be named `rem.png` and placed in the same directory as the program.
3. **Output**: The program generates a new image named `rem_paletted.png`, which uses the reduced color palette.

## Implementation Details

- **OctreeNode**: Represents a node in the octree. Leaf nodes store color information, while non-leaf nodes are used for navigation.
- **Octree**: Manages the octree structure, including insertion of colors, reduction of the tree, and palette generation.
- **Main Function**: Handles image loading, octree construction, palette building, image conversion, and saving the converted image.

## Dependencies

- The program uses the standard Go library for image processing (`image`, `image/color`, `image/png`).

## Contributing

Contributions to improve the algorithm or extend the functionality are welcome. Please feel free to fork the repository, make your changes, and submit a pull request.

## License

This project is open-sourced under the MIT License. See the LICENSE file for more details.