package dto

// AnalysisMap holds the result of a steganography analysis.
// It contains a 2D grid of probability values or metric scores.
type AnalysisMap struct {
	// Values is a 2D matrix [width][height] where each float64 represents
	// the calculated score for that pixel or block (e.g., probability of embedding).
	Values [][]float64

	// Width of the analyzed map (matches the image width).
	Width int

	// Height of the analyzed map (matches the image height).
	Height int
}
