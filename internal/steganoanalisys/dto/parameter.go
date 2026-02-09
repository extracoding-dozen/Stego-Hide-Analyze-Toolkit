package dto

// Parameter represents a configurable setting for a steganography attack.
// It is used to generate UI controls (sliders, inputs) dynamically.
type Parameter struct {
	// Key is the unique identifier for the parameter (e.g., "block_size").
	Key string

	// Name is the human-readable label displayed in the UI.
	Name string

	// Min represents the minimum allowed value.
	Min float64

	// Max represents the maximum allowed value.
	Max float64

	// Def is the default value.
	Def float64

	// Step determines the increment/decrement step size for the UI control.
	Step float64

	// IntMode indicates if the value should be treated as an integer.
	// If true, the UI should restrict input to whole numbers.
	IntMode bool
}
