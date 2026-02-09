package chi_square

import (
	"github.com/extracoding-dozen/Stego-Hide-Analyze-Toolkit/internal/steganoanalisys/dto"
	"github.com/extracoding-dozen/Stego-Hide-Analyze-Toolkit/internal/utils"
	"image"
	"math"
)

// ChiSquareAttack implements the Chi-Square statistical attack.
// It splits the image into blocks and analyzes the distribution of pixel values
// to detect LSB (Least Significant Bit) sequential embedding.
type ChiSquareAttack struct {
	// BlockSize defines the side length of the square block used for analysis.
	BlockSize int
}

// Name returns the human-readable name of the attack.
func (a *ChiSquareAttack) Name() string {
	return "Хи-Квадрат (Blocks)"
}

// GetParameters returns the configuration options (e.g., Block Size).
func (a *ChiSquareAttack) GetParameters() []dto.Parameter {
	return []dto.Parameter{
		{
			Key:     "block_size",
			Name:    "Размер блока (px)",
			Min:     8,
			Max:     128,
			Def:     32,
			Step:    8,
			IntMode: true,
		},
	}
}

// SetParameter updates the algorithm configuration.
func (a *ChiSquareAttack) SetParameter(key string, val float64) {
	if key == "block_size" {
		a.BlockSize = int(val)
	}
}

// Compute performs the Chi-Square analysis on the given image.
// If BlockSize is not set, it defaults to 32.
func (a *ChiSquareAttack) Compute(img image.Image) (dto.AnalysisMap, error) {
	if a.BlockSize == 0 {
		a.BlockSize = 32
	}
	raw := a.ComputeChiSquareMap(img, a.BlockSize)
	return dto.AnalysisMap{Values: raw.Values, Width: raw.Width, Height: raw.Height}, nil
}

// ThresholdInfo returns visual thresholds: 0.5 to 1.0, with a default of 0.95.
// Values closer to 1.0 indicate a high probability of hidden data.
func (a *ChiSquareAttack) ThresholdInfo() (min, max, def float64) {
	return 0.5, 1.0, 0.95
}

// ComputeChiSquareMap iterates over the image in blocks and calculates the
// probability of embedding for each block using the Chi-Square test.
func (a *ChiSquareAttack) ComputeChiSquareMap(img image.Image, blockSize int) dto.AnalysisMap {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	gray := utils.GetGrayMatrix(img)

	values := make([][]float64, w)
	for i := range values {
		values[i] = make([]float64, h)
	}

	for bx := 0; bx < w; bx += blockSize {
		for by := 0; by < h; by += blockSize {
			maxX, maxY := bx+blockSize, by+blockSize
			if maxX > w {
				maxX = w
			}
			if maxY > h {
				maxY = h
			}

			// Skip blocks that are too uniform or empty
			if !a.isBlockSuitable(gray, bx, by, maxX, maxY) {
				a.fillBlock(values, bx, by, maxX, maxY, 0.0)
				continue
			}

			prob := a.calculateBlockChiSquare(gray, bx, by, maxX, maxY)

			a.fillBlock(values, bx, by, maxX, maxY, prob)
		}
	}
	return dto.AnalysisMap{Values: values, Width: w, Height: h}
}

// isBlockMonotone checks if a block consists of a single repeated value.
func (a *ChiSquareAttack) isBlockMonotone(gray [][]uint8, x1, y1, x2, y2 int) bool {
	if x1 >= x2 || y1 >= y2 {
		return true
	}

	firstVal := gray[x1][y1]
	diffFound := false

	for x := x1; x < x2; x++ {
		for y := y1; y < y2; y++ {
			if gray[x][y] != firstVal {
				diffFound = true
				break
			}
		}
		if diffFound {
			break
		}
	}
	return !diffFound
}

// fillBlock fills a rectangular region in the values matrix with a specific float value.
func (a *ChiSquareAttack) fillBlock(values [][]float64, x1, y1, x2, y2 int, val float64) {
	for x := x1; x < x2; x++ {
		for y := y1; y < y2; y++ {
			values[x][y] = val
		}
	}
}

// calculateBlockChiSquare computes the Chi-Square statistic for a specific block.
// It pairs values (2k, 2k+1) and compares the expected frequency versus observed.
func (a *ChiSquareAttack) calculateBlockChiSquare(gray [][]uint8, x1, y1, x2, y2 int) float64 {
	hist := make([]int, 256)

	for x := x1; x < x2; x++ {
		for y := y1; y < y2; y++ {
			val := gray[x][y]

			if val == 0 || val == 255 {
				continue
			}
			hist[val]++
		}
	}

	x2Stat := 0.0
	k := 0

	for i := 0; i < 254; i += 2 {
		n_even := float64(hist[i])
		n_odd := float64(hist[i+1])
		sum := n_even + n_odd

		if sum <= 4 {
			continue
		}

		expected := sum / 2.0
		term := math.Pow(n_even-expected, 2)/expected + math.Pow(n_odd-expected, 2)/expected
		x2Stat += term
		k++
	}

	if k < 2 {
		return 0.0
	}

	return chiSquareProb(x2Stat, k)
}

// chiSquareProb returns the probability that the observed Chi-Square statistic
// occurred by chance, given the degrees of freedom (dof).
func chiSquareProb(x2 float64, dof int) float64 {
	if dof <= 0 {
		return 0.0
	}

	return gammp(float64(dof)/2.0, x2/2.0)
}

// gammp computes the incomplete gamma function P(a, x).
func gammp(a, x float64) float64 {
	if x < 0.0 || a <= 0.0 {
		return 0.0
	}
	if x < a+1.0 {
		return gser(a, x)
	}
	return 1.0 - gcf(a, x)
}

// gser returns the incomplete gamma function P(a, x) evaluated by its series representation.
func gser(a, x float64) float64 {
	const itmax = 100
	const eps = 3.0e-7

	gln := logGamma(a)
	if x <= 0.0 {
		return 0.0
	}

	ap := a
	sum := 1.0 / a
	del := sum

	for n := 1; n <= itmax; n++ {
		ap += 1.0
		del *= x / ap
		sum += del
		if math.Abs(del) < math.Abs(sum)*eps {
			return sum * math.Exp(-x+a*math.Log(x)-gln)
		}
	}
	return 0.0
}

// gcf returns the incomplete gamma function Q(a, x) evaluated by its continued fraction representation.
func gcf(a, x float64) float64 {
	const itmax = 100
	const eps = 3.0e-7

	gln := logGamma(a)
	b := x + 1.0 - a
	c := 1.0 / 1.0e-30
	d := 1.0 / b
	h := d

	for i := 1; i <= itmax; i++ {
		an := -float64(i) * (float64(i) - a)
		b += 2.0
		d = an*d + b
		if math.Abs(d) < 1.0e-30 {
			d = 1.0e-30
		}
		c = b + an/c
		if math.Abs(c) < 1.0e-30 {
			c = 1.0e-30
		}
		d = 1.0 / d
		del := d * c
		h *= del
		if math.Abs(del-1.0) < eps {
			break
		}
	}
	return math.Exp(-x+a*math.Log(x)-gln) * h
}

// logGamma computes the natural logarithm of the gamma function.
func logGamma(x float64) float64 {
	tmp := (x-0.5)*math.Log(x+4.5) - (x + 4.5)
	ser := 1.000000000190015

	coef := []float64{
		76.18009172947146, -86.50532032941677,
		24.01409824083091, -1.231739572450155,
		0.1208650973866179e-2, -0.5395239384953e-5,
	}
	for _, c := range coef {
		x += 1.0
		ser += c / x
	}
	return tmp + math.Log(ser*math.Sqrt(2*math.Pi))
}

// isBlockSuitable checks if a block has enough variance to be statistically significant.
// Blocks with very low contrast (< 5 intensity difference) are ignored.
func (a *ChiSquareAttack) isBlockSuitable(gray [][]uint8, x1, y1, x2, y2 int) bool {
	if x1 >= x2 || y1 >= y2 {
		return false
	}

	minVal, maxVal := uint8(255), uint8(0)

	for x := x1; x < x2; x++ {
		for y := y1; y < y2; y++ {
			val := gray[x][y]
			if val < minVal {
				minVal = val
			}
			if val > maxVal {
				maxVal = val
			}
		}
	}

	if (maxVal - minVal) < 5 {
		return false
	}

	return true
}
