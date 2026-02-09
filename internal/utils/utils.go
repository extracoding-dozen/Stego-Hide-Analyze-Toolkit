package utils

import (
	"golang.org/x/image/bmp"
	_ "golang.org/x/image/bmp" // Register BMP format
	"image"
	"image/jpeg"
	_ "image/jpeg" // Register JPEG format
	"image/png"
	_ "image/png" // Register PNG format
	"os"
	"strings"
)

// LoadImage opens a file from the disk and decodes it into an image.Image.
// It supports JPEG, PNG, and BMP formats via side-effect imports.
func LoadImage(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	return img, err
}

// SaveImage encodes and saves an image to the specified path.
// The format is determined by the file extension (.png, .jpg, .bmp).
//
// Quality settings:
//   - JPEG: Quality 90
//   - BMP/PNG: Default settings
func SaveImage(img image.Image, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	lowerPath := strings.ToLower(path)
	switch {
	case strings.HasSuffix(lowerPath, ".png"):
		return png.Encode(f, img)
	case strings.HasSuffix(lowerPath, ".jpg") || strings.HasSuffix(lowerPath, ".jpeg"):
		return jpeg.Encode(f, img, &jpeg.Options{Quality: 90})
	case strings.HasSuffix(lowerPath, ".bmp"):
		return bmp.Encode(f, img)
	default:
		// Default to BMP if extension is unknown
		return bmp.Encode(f, img)
	}

	// Ensure data is written to disk
	if err := f.Sync(); err != nil {
		return err
	}
	return nil
}

// GetGrayMatrix converts an arbitrary image into a 2D grid of 8-bit grayscale values.
// This is used as the raw input for most steganalysis algorithms.
func GetGrayMatrix(img image.Image) [][]uint8 {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	m := make([][]uint8, w)
	for x := 0; x < w; x++ {
		m[x] = make([]uint8, h)
		for y := 0; y < h; y++ {
			c := img.At(x, y)
			r, g, b, _ := c.RGBA()
			// Convert 16-bit color components to 8-bit grayscale
			m[x][y] = uint8((0.299*float64(r>>8) + 0.587*float64(g>>8) + 0.114*float64(b>>8)))
		}
	}
	return m
}
