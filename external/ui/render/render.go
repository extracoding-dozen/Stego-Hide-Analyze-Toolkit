package render

import (
	"Stego-Hide-Analyze-Toolkit/internal/steganoanalisys/dto"
	"image"
	"image/color"
)

// RenderAnalysisMap converts a raw probability map (AnalysisMap) into a visual image.
//
// Logic:
// 1. Converts the source image to grayscale.
// 2. Checks each pixel's value in the analysis map against the threshold.
// 3. If value > threshold, the pixel is highlighted in Red.
// 4. Otherwise, it is drawn in grayscale.
func RenderAnalysisMap(srcImg image.Image, data dto.AnalysisMap, threshold float64) AnalysisResult {
	if srcImg == nil || data.Values == nil {
		return AnalysisResult{}
	}

	w, h := data.Width, data.Height
	resImg := image.NewRGBA(image.Rect(0, 0, w, h))
	var suspicious []image.Point

	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {

			c := srcImg.At(x, y)
			r, g, b, _ := c.RGBA()

			r8, g8, b8 := uint8(r>>8), uint8(g>>8), uint8(b>>8)

			gray := uint8(0.299*float64(r8) + 0.587*float64(g8) + 0.114*float64(b8))

			val := data.Values[x][y]
			isSuspicious := val > threshold

			if isSuspicious {
				// Highlight suspicious pixels with a red tint mixed with the original gray
				resImg.Set(x, y, color.RGBA{
					R: uint8(float64(gray)*0.3 + 255*0.7),
					G: uint8(float64(gray) * 0.3),
					B: uint8(float64(gray) * 0.3),
					A: 255,
				})
				suspicious = append(suspicious, image.Point{X: x, Y: y})
			} else {
				// Normal pixels remain grayscale
				resImg.Set(x, y, color.RGBA{R: gray, G: gray, B: gray, A: 255})
			}
		}
	}

	return AnalysisResult{
		ResultImage:      resImg,
		SuspiciousPixels: suspicious,
		SuspiciousCount:  len(suspicious),
	}
}
