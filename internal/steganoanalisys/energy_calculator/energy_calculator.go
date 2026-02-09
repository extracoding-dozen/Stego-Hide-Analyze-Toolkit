package enegry_calculator

import (
	"image"
	"image/color"
	"math"
)

// EnergyCalculator is responsible for computing the "energy" or residual error
// of an image. It helps in detecting high-frequency noise or irregularities
// often introduced by steganography.
type EnergyCalculator struct {
	radius int
}

// NewEnergyCalculator creates a new calculator instance with a specified
// neighborhood radius for prediction.
func NewEnergyCalculator(radius int) *EnergyCalculator {
	return &EnergyCalculator{radius: radius}
}

// Calculate computes the difference between actual pixel values and predicted values.
//
// The prediction is based on the average intensity of neighboring pixels within the
// configured radius.
//
// Returns:
//   - A 2D matrix of difference values (residuals).
//   - The maximum difference found (useful for normalization).
func (e *EnergyCalculator) Calculate(img image.Image) ([][]float64, float64) {
	bounds := img.Bounds()
	width, height := bounds.Max.X, bounds.Max.Y

	// Pre-calculate grayscale values to avoid repeated conversions
	grayMatrix := make([][]float64, width)
	for x := 0; x < width; x++ {
		grayMatrix[x] = make([]float64, height)
		for y := 0; y < height; y++ {
			grayMatrix[x][y] = e.getLuminance(img.At(x, y))
		}
	}

	result := make([][]float64, width)
	maxVal := 0.0

	for x := 0; x < width; x++ {
		result[x] = make([]float64, height)
		for y := 0; y < height; y++ {
			sum := 0.0
			count := 0.0

			// Iterate over neighbors within the radius
			for i := -e.radius; i <= e.radius; i++ {
				for j := -e.radius; j <= e.radius; j++ {
					// Skip the center pixel itself
					if i == 0 && j == 0 {
						continue
					}

					nx, ny := x+i, y+j

					// Check bounds
					if nx >= 0 && nx < width && ny >= 0 && ny < height {
						sum += grayMatrix[nx][ny]
						count++
					}
				}
			}

			if count > 0 {
				predicted := sum / count
				original := grayMatrix[x][y]
				diff := math.Abs(original - predicted)

				result[x][y] = diff
				if diff > maxVal {
					maxVal = diff
				}
			}
		}
	}
	return result, maxVal
}

// getLuminance converts a color to its grayscale intensity using ITU-R BT.601 coefficients.
func (e *EnergyCalculator) getLuminance(c color.Color) float64 {
	r, g, b, _ := c.RGBA()
	// Standard formula: Y = 0.299R + 0.587G + 0.114B
	return 0.299*float64(r>>8) + 0.587*float64(g>>8) + 0.114*float64(b>>8)
}
