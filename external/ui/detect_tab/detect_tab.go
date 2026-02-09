package detect_tab

import (
	"fmt"
	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/extracoding-dozen/Stego-Hide-Analyze-Toolkit/external/ui/render"
	"github.com/extracoding-dozen/Stego-Hide-Analyze-Toolkit/internal/steganoanalisys/attack"
	"github.com/extracoding-dozen/Stego-Hide-Analyze-Toolkit/internal/steganoanalisys/attack/chi_square"
	entropy_attack "github.com/extracoding-dozen/Stego-Hide-Analyze-Toolkit/internal/steganoanalisys/attack/entropy"
	"github.com/extracoding-dozen/Stego-Hide-Analyze-Toolkit/internal/steganoanalisys/dto"
	"github.com/extracoding-dozen/Stego-Hide-Analyze-Toolkit/internal/utils"
	"github.com/sqweek/dialog"
	"image"
	"image/color"
	"math"
)

// DetectTabController manages the state and UI logic for the Steganography Detection tab.
// It allows users to load images, select analysis algorithms (attacks), tune parameters,
// and visualize the results.
type DetectTabController struct {
	// Window is a reference to the main application window.
	Window *app.Window
	// Theme holds the Material Design styling configuration.
	Theme *material.Theme

	// Attacks is the registry of available steganography analysis algorithms.
	Attacks []attack.Attack
	// CurrentAttack is the currently selected algorithm.
	CurrentAttack attack.Attack

	// AlgoSelector controls the dropdown/radio selection for algorithms.
	AlgoSelector widget.Enum

	// ParamWidgets maps parameter keys (e.g., "block_size") to their specific UI sliders.
	// This allows the UI to dynamically adapt to different algorithms.
	ParamWidgets map[string]*widget.Float

	BtnOpen         widget.Clickable
	BtnRun          widget.Clickable
	ThresholdSlider widget.Float
	LogList         widget.List

	SourceImg  image.Image
	ResultImg  image.Image
	ResultData render.AnalysisResult

	// CachedMap stores the raw probability data to allow re-rendering
	// with different thresholds without re-running the heavy computation.
	CachedMap    dto.AnalysisMap
	IsCalculated bool

	ThresholdVal float64
	MinT, MaxT   float64

	IsLoading bool
	StatusMsg string
}

// NewDetectTabController initializes the controller with default attacks (Entropy, Chi-Square).
func NewDetectTabController(w *app.Window, th *material.Theme) *DetectTabController {
	c := &DetectTabController{
		Window:       w,
		Theme:        th,
		LogList:      widget.List{List: layout.List{Axis: layout.Vertical}},
		ParamWidgets: make(map[string]*widget.Float),

		Attacks: []attack.Attack{
			&entropy_attack.EntropyAttack{Radius: 2},
			&chi_square.ChiSquareAttack{BlockSize: 32},
		},
	}

	c.CurrentAttack = c.Attacks[0]
	c.AlgoSelector.Value = c.Attacks[0].Name()
	c.applyThresholdDefaults()
	c.setupCurrentAttackParams()
	return c
}

// setupCurrentAttackParams initializes UI widgets for the parameters of the selected attack.
func (c *DetectTabController) setupCurrentAttackParams() {

	min, max, def := c.CurrentAttack.ThresholdInfo()
	c.MinT, c.MaxT = min, max
	c.ThresholdVal = def
	c.ThresholdSlider.Value = float32((def - min) / (max - min))

	params := c.CurrentAttack.GetParameters()
	for _, p := range params {
		if _, exists := c.ParamWidgets[p.Key]; !exists {
			// Create a new slider if one doesn't exist for this parameter key
			w := &widget.Float{}
			w.Value = float32((p.Def - p.Min) / (p.Max - p.Min))
			c.ParamWidgets[p.Key] = w
		}
	}
}

// Layout renders the entire tab interface: controls, parameters, and the workspace.
func (c *DetectTabController) Layout(gtx layout.Context) layout.Dimensions {
	if c.BtnOpen.Clicked(gtx) {
		c.openFile()
	}
	if c.BtnRun.Clicked(gtx) {
		c.runAnalysis()
	}

	oldAlgoName := c.AlgoSelector.Value

	dims := layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(c.layoutControls),
		layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),

		layout.Rigid(c.layoutAlgorithmParams),

		layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
		layout.Rigid(c.layoutThresholdSlider),
		layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
		layout.Flexed(1, c.layoutWorkspace),
	)

	// Handle algorithm switching
	if c.AlgoSelector.Value != oldAlgoName {
		newAlgoName := c.AlgoSelector.Value
		for _, atk := range c.Attacks {
			if atk.Name() == newAlgoName {
				c.CurrentAttack = atk
				c.IsCalculated = false
				c.ResultImg = nil

				c.setupCurrentAttackParams()

				c.Window.Invalidate()
				break
			}
		}
	}

	return dims
}

// layoutThresholdSlider renders the sensitivity slider.
// Updates to this slider trigger a re-render of the result image (updateRender) immediately.
func (c *DetectTabController) layoutThresholdSlider(gtx layout.Context) layout.Dimensions {

	if !c.IsCalculated {
		return layout.Dimensions{}
	}

	oldSliderVal := c.ThresholdSlider.Value

	slider := material.Slider(c.Theme, &c.ThresholdSlider)

	dims := layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {

			val := c.MinT + float64(c.ThresholdSlider.Value)*(c.MaxT-c.MinT)
			label := fmt.Sprintf("Порог чувствительности: %.3f", val)

			return material.Label(c.Theme, unit.Sp(14), label).Layout(gtx)
		}),
		layout.Rigid(slider.Layout),
	)

	if c.ThresholdSlider.Value != oldSliderVal {

		c.ThresholdVal = c.MinT + float64(c.ThresholdSlider.Value)*(c.MaxT-c.MinT)

		c.updateRender()

		c.Window.Invalidate()
	}

	return dims
}

// layoutAlgorithmParams renders sliders for the specific parameters of the active algorithm.
func (c *DetectTabController) layoutAlgorithmParams(gtx layout.Context) layout.Dimensions {
	params := c.CurrentAttack.GetParameters()
	if len(params) == 0 {
		return layout.Dimensions{}
	}

	var children []layout.FlexChild

	children = append(children, layout.Rigid(material.Label(c.Theme, unit.Sp(12), "Параметры алгоритма:").Layout))

	for _, p := range params {
		p := p

		w, ok := c.ParamWidgets[p.Key]
		if !ok {
			continue
		}

		oldSliderVal := w.Value

		slider := material.Slider(c.Theme, w)

		oldRealVal := c.calcRealValue(p, float64(oldSliderVal))

		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {

			dims := slider.Layout(gtx)

			if w.Value != oldSliderVal {
				newRealVal := c.calcRealValue(p, float64(w.Value))

				if newRealVal != oldRealVal {
					c.CurrentAttack.SetParameter(p.Key, newRealVal)
					// Invalidate results when parameters change
					c.IsCalculated = false
					c.ResultImg = nil

					c.Window.Invalidate()
				}
			}

			labelTxt := fmt.Sprintf("%s: %.0f", p.Name, oldRealVal)
			if !p.IntMode {
				labelTxt = fmt.Sprintf("%s: %.2f", p.Name, oldRealVal)
			}

			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(material.Label(c.Theme, unit.Sp(12), labelTxt).Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return dims
				}),
			)
		}))

		children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(5)}.Layout))
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
}

// calcRealValue maps the slider's 0.0-1.0 range to the actual parameter range (Min-Max).
// It also handles stepping and integer rounding if required.
func (c *DetectTabController) calcRealValue(p dto.Parameter, sliderVal float64) float64 {
	val := p.Min + sliderVal*(p.Max-p.Min)

	if p.IntMode {

		if p.Step > 0 {
			val = math.Round(val/p.Step) * p.Step
		} else {
			val = math.Round(val)
		}
	}

	if val < p.Min {
		return p.Min
	}
	if val > p.Max {
		return p.Max
	}

	return val
}

// applyThresholdDefaults resets the threshold slider to the algorithm's default.
func (c *DetectTabController) applyThresholdDefaults() {
	min, max, def := c.CurrentAttack.ThresholdInfo()
	c.MinT, c.MaxT = min, max
	c.ThresholdVal = def

	c.ThresholdSlider.Value = float32((def - min) / (max - min))
}

// layoutControls renders the top button bar (Open, Algo Selection, Run).
func (c *DetectTabController) layoutControls(gtx layout.Context) layout.Dimensions {

	var widgets []layout.FlexChild

	widgets = append(widgets, layout.Rigid(material.Button(c.Theme, &c.BtnOpen, "Открыть").Layout))
	widgets = append(widgets, layout.Rigid(layout.Spacer{Width: unit.Dp(20)}.Layout))

	for _, atk := range c.Attacks {
		name := atk.Name()

		rb := material.RadioButton(c.Theme, &c.AlgoSelector, name, name)
		widgets = append(widgets, layout.Rigid(rb.Layout))
		widgets = append(widgets, layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout))
	}

	widgets = append(widgets, layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout))

	if c.SourceImg != nil {
		txt := "АНАЛИЗ"
		if c.IsLoading {
			txt = "..."
		}
		btn := material.Button(c.Theme, &c.BtnRun, txt)
		if c.IsLoading {
			btn.Background = color.NRGBA{100, 100, 100, 255}
		}
		widgets = append(widgets, layout.Rigid(btn.Layout))
	}

	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx, widgets...)
}

// runAnalysis executes the selected attack in a separate goroutine.
func (c *DetectTabController) runAnalysis() {
	if c.SourceImg == nil {
		return
	}
	c.IsLoading = true
	c.StatusMsg = "Вычисление..."

	attack := c.CurrentAttack

	go func() {
		defer c.Window.Invalidate()

		rawData, err := attack.Compute(c.SourceImg)

		if err == nil {
			c.CachedMap = rawData
			c.IsCalculated = true
			c.StatusMsg = "Готово"
			c.updateRender()
		} else {
			c.StatusMsg = "Ошибка: " + err.Error()
		}
		c.IsLoading = false
	}()
}

// updateRender generates the result image based on CachedMap and the current ThresholdVal.
func (c *DetectTabController) updateRender() {
	if !c.IsCalculated {
		return
	}

	res := render.RenderAnalysisMap(c.SourceImg, c.CachedMap, c.ThresholdVal)

	c.ResultImg = res.ResultImage
	c.ResultData = res

	c.Window.Invalidate()
}

// layoutWorkspace renders the split view: Image Comparison (Left) and Log List (Right).
func (c *DetectTabController) layoutWorkspace(gtx layout.Context) layout.Dimensions {
	if c.SourceImg == nil {
		return material.Label(c.Theme, unit.Sp(16), "Загрузите изображение").Layout(gtx)
	}
	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
		layout.Flexed(2, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return c.drawImg(gtx, "Исходник", c.SourceImg)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return c.drawImg(gtx, "Детекция", c.ResultImg)
				}),
			)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
		layout.Flexed(1, c.layoutLog),
	)
}

// layoutLog renders the list of suspicious pixels found.
func (c *DetectTabController) layoutLog(gtx layout.Context) layout.Dimensions {

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			count := len(c.ResultData.SuspiciousPixels)
			txt := fmt.Sprintf("Найдено точек: %d", count)
			if count > 1000 {
				txt += " (показаны первые 1000)"
			}
			return material.Label(c.Theme, unit.Sp(14), txt).Layout(gtx)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(5)}.Layout),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {

			return widget.Border{Color: color.NRGBA{A: 255}, Width: unit.Dp(1)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {

				displayCount := len(c.ResultData.SuspiciousPixels)
				if displayCount > 1000 {
					displayCount = 1000
				}

				return material.List(c.Theme, &c.LogList).Layout(gtx, displayCount, func(gtx layout.Context, i int) layout.Dimensions {
					pt := c.ResultData.SuspiciousPixels[i]
					return material.Label(c.Theme, unit.Sp(12), fmt.Sprintf("[%d] X:%d Y:%d", i+1, pt.X, pt.Y)).Layout(gtx)
				})
			})
		}),
	)
}

func (c *DetectTabController) openFile() {
	go func() {
		path, err := dialog.File().Filter("Images", "png", "bmp", "jpg").Load()
		if err != nil {
			return
		}
		img, err := utils.LoadImage(path)
		if err == nil {
			c.SourceImg = img
			c.ResultImg = nil
			c.IsCalculated = false
			c.StatusMsg = "Файл загружен"
			c.Window.Invalidate()
		}
	}()
}
func (c *DetectTabController) drawImg(gtx layout.Context, lbl string, img image.Image) layout.Dimensions {
	if img == nil {
		return layout.Dimensions{}
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(material.Label(c.Theme, unit.Sp(14), lbl).Layout),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return widget.Image{Src: paint.NewImageOp(img), Fit: widget.Contain}.Layout(gtx)
		}),
	)
}
