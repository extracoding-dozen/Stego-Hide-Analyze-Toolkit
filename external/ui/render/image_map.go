package render

import "image"

// AnalysisResult aggregates the output of the visualization process.
type AnalysisResult struct {
	// ResultImage is the visual overlay (grayscale + red highlights).
	ResultImage image.Image

	// SuspiciousPixels is a list of coordinates where data might be hidden.
	SuspiciousPixels []image.Point

	// SuspiciousCount is the total number of flagged pixels.
	SuspiciousCount int
}
