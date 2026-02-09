package map_drawer

import (
	"image"
	"image/color"
	"math"
)

// MapDrawer handles the visualization of analysis data (2D matrices) as heatmaps.
type MapDrawer struct {
}

// NewMapDrawer creates a new instance of MapDrawer.
func NewMapDrawer() *MapDrawer {
	return &MapDrawer{}
}

// Draw generates a heatmap image from an energy map.
// It normalizes values based on the provided global maximum (maxVal).
//
// Colors range from Blue (low value) to Red (high value).
func (md *MapDrawer) Draw(energyMap [][]float64, maxVal float64, width, height int) image.Image {
	heatmapImg := image.NewRGBA(image.Rect(0, 0, width, height))

	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {

			normalized := 0.0
			if maxVal > 0 {
				normalized = energyMap[x][y] / maxVal
			}

			col := md.ValueToHeatmapColor(normalized)
			heatmapImg.Set(x, y, col)
		}
	}
	return heatmapImg
}

// DrawWithSensitivity generates a heatmap with a custom sensitivity threshold.
// Values exceeding the sensitivity are clamped to the maximum intensity (Red).
//
// This is useful for highlighting weak signals by lowering the ceiling.
func (md *MapDrawer) DrawWithSensitivity(energyMap [][]float64, sensitivity float64, width, height int) image.Image {
	heatmapImg := image.NewRGBA(image.Rect(0, 0, width, height))
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			val := energyMap[x][y]

			normalized := val / sensitivity
			if normalized > 1.0 {
				normalized = 1.0
			}

			col := md.ValueToHeatmapColor(normalized)
			heatmapImg.Set(x, y, col)
		}
	}
	return heatmapImg
}

// ValueToHeatmapColor converts a normalized float value (0.0 - 1.0) into a color.
// Logic: 0.0 -> Blue (240 HSV), 1.0 -> Red (0 HSV).
func (md *MapDrawer) ValueToHeatmapColor(val float64) color.RGBA {
	// Invert val so 0.0 is Blue (240) and 1.0 is Red (0)
	hue := (1.0 - val) * 240.0
	return md.hsvToRGB(hue, 1.0, 1.0)
}

// hsvToRGB converts Hue, Saturation, Value to standard RGBA.
// Hue is expected in degrees (0-360), S and V are 0.0-1.0.
func (md *MapDrawer) hsvToRGB(h, s, v float64) color.RGBA {
	c := v * s
	x := c * (1 - math.Abs(math.Mod(h/60.0, 2)-1))
	m := v - c
	var r, g, b float64

	switch {
	case h < 60:
		r, g, b = c, x, 0
	case h < 120:
		r, g, b = x, c, 0
	case h < 180:
		r, g, b = 0, c, x
	case h < 240:
		r, g, b = 0, x, c
	case h < 300:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}
	return color.RGBA{
		R: uint8((r + m) * 255),
		G: uint8((g + m) * 255),
		B: uint8((b + m) * 255),
		A: 255,
	}
}
