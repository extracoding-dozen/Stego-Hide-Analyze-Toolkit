package attack

import (
	"image"
	"stego-hide-analize-toolkit/internal/steganoanalisys/dto"
)

// Attack defines the interface for all steganography analysis algorithms.
// Implementations of this interface process images to detect hidden data.
type Attack interface {
	// Name returns the display name of the attack algorithm.
	Name() string

	// Compute executes the analysis algorithm on the provided image.
	// It returns an AnalysisMap containing the results or an error if processing fails.
	Compute(img image.Image) (dto.AnalysisMap, error)

	// ThresholdInfo returns the range and default threshold for visualization.
	// min, max: define the absolute range of values produced by the algorithm.
	// def: suggests a default threshold value for highlighting suspicious areas.
	ThresholdInfo() (min, max, def float64)

	// GetParameters returns a list of configurable parameters for the algorithm.
	GetParameters() []dto.Parameter

	// SetParameter updates a specific configuration value by its key.
	SetParameter(key string, val float64)
}
