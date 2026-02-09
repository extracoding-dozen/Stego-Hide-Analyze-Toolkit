package entropy_attack

import (
	"github.com/extracoding-dozen/Stego-Hide-Analyze-Toolkit/internal/steganoanalisys/dto"
	"github.com/extracoding-dozen/Stego-Hide-Analyze-Toolkit/internal/utils"
	"image"
	"math"
)

// EntropyAttack implements a spatial entropy analysis.
// It detects high-entropy regions in the LSB plane by examining the relationship
// between neighboring pixel LSBs.
type EntropyAttack struct {
	// Radius defines the neighborhood size for local entropy calculation.
	Radius int
}

// Name returns the display name of the attack.
func (a *EntropyAttack) Name() string {
	return "Локальная Энтропия (Spatial LSB)"
}

// GetParameters returns the configurable options, specifically the radius.
func (a *EntropyAttack) GetParameters() []dto.Parameter {
	return []dto.Parameter{
		{
			Key:     "radius",
			Name:    "Радиус соседей",
			Min:     1,
			Max:     10,
			Def:     2,
			Step:    1,
			IntMode: true,
		},
	}
}

// SetParameter updates the Radius parameter.
func (a *EntropyAttack) SetParameter(key string, val float64) {
	if key == "radius" {
		a.Radius = int(val)
	}
}

// Compute executes the entropy analysis on the provided image.
// If Radius is 0, it defaults to 2.
func (a *EntropyAttack) Compute(img image.Image) (dto.AnalysisMap, error) {
	if a.Radius == 0 {
		a.Radius = 2
	}
	raw := a.ComputeEntropyMap(img, a.Radius)
	return dto.AnalysisMap{Values: raw.Values, Width: raw.Width, Height: raw.Height}, nil
}

// ThresholdInfo returns visual thresholds: 0.85 to 1.0, with a default of 0.96.
// Higher values indicate higher randomness (entropy), typical of encrypted data.
func (a *EntropyAttack) ThresholdInfo() (min, max, def float64) {
	return 0.85, 1.0, 0.96
}

// ComputeEntropyMap calculates the local entropy for every pixel in the image.
// The result is a heatmap where each point represents the entropy of its neighborhood.
func (a *EntropyAttack) ComputeEntropyMap(img image.Image, radius int) dto.AnalysisMap {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	gray := utils.GetGrayMatrix(img)

	values := make([][]float64, w)

	for x := 0; x < w; x++ {
		values[x] = make([]float64, h)
		for y := 0; y < h; y++ {
			values[x][y] = a.calculateSpatialEntropy(gray, x, y, w, h, radius)
		}
	}
	return dto.AnalysisMap{Values: values, Width: w, Height: h}
}

// calculateSpatialEntropy computes the entropy based on LSB pairs within the radius.
// It looks at the LSB of the current pixel and its neighbor to form a 2-bit value (0-3),
// accumulates frequencies, and calculates Shannon entropy.
func (a *EntropyAttack) calculateSpatialEntropy(gray [][]uint8, x, y, w, h, r int) float64 {

	counts := make([]float64, 4)
	total := 0.0

	for i := -r; i <= r; i++ {
		for j := -r; j <= r; j++ {
			nx, ny := x+i, y+j

			if nx >= 0 && nx < w-1 && ny >= 0 && ny < h {

				bitCurrent := gray[nx][ny] & 1
				bitNext := gray[nx+1][ny] & 1

				pairVal := (bitCurrent << 1) | bitNext

				counts[pairVal]++
				total++
			}
		}
	}

	if total == 0 {
		return 0.0
	}

	entropy := 0.0
	for k := 0; k < 4; k++ {
		p := counts[k] / total
		if p > 0 {
			entropy -= p * math.Log2(p)
		}
	}

	// Normalize entropy (max entropy for 4 states is 2.0 bits)
	return entropy / 2.0
}
