package ui

import (
	"fmt"
	"github.com/extracoding-dozen/Stego-Hide-Analyze-Toolkit/external/ui/detect_tab"
	calculator "github.com/extracoding-dozen/Stego-Hide-Analyze-Toolkit/internal/steganoanalisys/energy_calculator"
	drawer "github.com/extracoding-dozen/Stego-Hide-Analyze-Toolkit/internal/steganoanalisys/map_drawer"
	"github.com/extracoding-dozen/Stego-Hide-Analyze-Toolkit/internal/utils"
	"image"
	"image/color"
	"strings"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/sqweek/dialog"
)

const (
	minSensitivity float32 = 1.0
	maxSensitivity float32 = 255.0
)

// Controller is the main UI orchestrator.
// It manages the application window, global state, and navigation between tabs.
type Controller struct {
	Window *app.Window
	Theme  *material.Theme

	// CurrentTab tracks the active tab index (0: Analyze, 1: Convert, 2: Detect).
	CurrentTab    int
	BtnTabAnalyze widget.Clickable
	BtnTabConvert widget.Clickable
	BtnTabDetect  widget.Clickable

	// --- Analyze Tab State ---
	OpenBtn             widget.Clickable
	ListWidget          widget.List
	OriginalImg         image.Image
	ResultMaps          map[int]image.Image
	LegendImg           image.Image
	EnergyMaps          map[int][][]float64
	SensitivitySlider   widget.Float
	AnalysisSensitivity float32

	// --- Convert Tab State ---
	ConvOpenBtn   widget.Clickable
	ConvSaveBtn   widget.Clickable
	ConvSourceImg image.Image
	ConvPath      string

	// Global UI flags
	IsLoading bool
	ErrorMsg  string

	// DetectTab delegates logic for the Detection tab
	DetectTab *detect_tab.DetectTabController
}

// NewController initializes the main application controller and window.
func NewController() *Controller {
	w := new(app.Window)
	w.Option(app.Title("Stego Tool"), app.Size(unit.Dp(1000), unit.Dp(800)))
	th := material.NewTheme()

	c := &Controller{
		Window:     w,
		Theme:      material.NewTheme(),
		ResultMaps: make(map[int]image.Image),
		EnergyMaps: make(map[int][][]float64),
		ListWidget: widget.List{
			List: layout.List{Axis: layout.Vertical},
		},
		DetectTab: detect_tab.NewDetectTabController(w, th),
	}

	c.AnalysisSensitivity = 50.0

	c.SensitivitySlider.Value = (c.AnalysisSensitivity - minSensitivity) / (maxSensitivity - minSensitivity)

	c.LegendImg = generateGradientLegend()
	return c
}

// layout handles the main rendering loop for the current frame.
func (c *Controller) layout(gtx layout.Context) layout.Dimensions {
	inset := layout.UniformInset(unit.Dp(10))
	return inset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(c.layoutTabs),
			layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {

				switch c.CurrentTab {
				case 0:
					return c.layoutAnalyzeTab(gtx)
				case 1:
					return c.layoutConvertTab(gtx)
				case 2:
					return c.DetectTab.Layout(gtx)
				}
				return layout.Dimensions{}
			}),
		)
	})
}

// layoutAnalyzeTab renders the "Fitness Map" (Analysis) tab.
// This tab calculates energy maps to see if an image is suitable for embedding.
func (c *Controller) layoutAnalyzeTab(gtx layout.Context) layout.Dimensions {
	if c.OpenBtn.Clicked(gtx) && !c.IsLoading {
		c.startAnalysis()
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(c.layoutStatusAndError),
		layout.Rigid(layout.Spacer{Height: unit.Dp(20)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if c.OriginalImg == nil {
				return layout.Dimensions{}
			}
			return c.layoutSensitivitySlider(gtx)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(20)}.Layout),
		layout.Rigid(c.layoutLegend),
		layout.Rigid(layout.Spacer{Height: unit.Dp(20)}.Layout),
		layout.Flexed(1, c.layoutResults),
	)
}

// layoutSensitivitySlider controls the visualization sensitivity of the energy map.
func (c *Controller) layoutSensitivitySlider(gtx layout.Context) layout.Dimensions {
	oldValue := c.SensitivitySlider.Value

	slider := material.Slider(c.Theme, &c.SensitivitySlider)

	dims := layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {

			label := fmt.Sprintf("Чувствительность (порог): %.0f", c.AnalysisSensitivity)
			return material.Label(c.Theme, unit.Sp(14), label).Layout(gtx)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(5)}.Layout),
		layout.Rigid(slider.Layout),
	)

	if oldValue != c.SensitivitySlider.Value {

		c.AnalysisSensitivity = minSensitivity + c.SensitivitySlider.Value*(maxSensitivity-minSensitivity)

		c.redrawAnalysisMaps()
	}
	return dims
}

// redrawAnalysisMaps updates the heatmap images based on the new sensitivity value.
func (c *Controller) redrawAnalysisMaps() {
	if c.EnergyMaps == nil || c.OriginalImg == nil {
		return
	}
	drawer := drawer.NewMapDrawer()
	bounds := c.OriginalImg.Bounds()
	for radius, energyMap := range c.EnergyMaps {

		resImg := drawer.DrawWithSensitivity(energyMap, float64(c.AnalysisSensitivity), bounds.Dx(), bounds.Dy())
		c.ResultMaps[radius] = resImg
	}
}

// startAnalysis loads the image and calculates energy maps for radii 1, 2, and 3.
func (c *Controller) startAnalysis() {
	c.IsLoading = true
	c.ErrorMsg = ""
	go func() {
		defer c.Window.Invalidate()
		path, err := dialog.File().Filter("BMP Image", "bmp").Load()
		if err != nil {
			c.IsLoading = false
			return
		}

		img, err := utils.LoadImage(path)
		if err != nil {
			c.IsLoading = false
			c.ErrorMsg = err.Error()
			return
		}

		c.OriginalImg = img
		radii := []int{1, 2, 3}

		for _, r := range radii {
			calc := calculator.NewEnergyCalculator(r)
			energyMap, _ := calc.Calculate(img)
			c.EnergyMaps[r] = energyMap
		}

		c.redrawAnalysisMaps()

		c.IsLoading = false
	}()
}

// Run starts the main event loop for the Gio application.
func (c *Controller) Run() error {
	var ops op.Ops
	for {
		switch e := c.Window.Event().(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			c.layout(gtx)
			e.Frame(gtx.Ops)
		}
	}
}

// layoutTabs renders the navigation buttons at the top.
func (c *Controller) layoutTabs(gtx layout.Context) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
		c.tabButton(gtx, &c.BtnTabAnalyze, "1. Карта пригодности", 0),
		layout.Rigid(layout.Spacer{Width: unit.Dp(5)}.Layout),
		c.tabButton(gtx, &c.BtnTabConvert, "2. Конвертер", 1),
		layout.Rigid(layout.Spacer{Width: unit.Dp(5)}.Layout),
		c.tabButton(gtx, &c.BtnTabDetect, "3. Стегоанализ (Поиск)", 2),
	)
}

func (c *Controller) tabButton(gtx layout.Context, btn *widget.Clickable, txt string, idx int) layout.FlexChild {
	return layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		b := material.Button(c.Theme, btn, txt)
		if c.CurrentTab == idx {
			b.Background = color.NRGBA{R: 50, G: 100, B: 200, A: 255}
		} else {
			b.Background = color.NRGBA{R: 150, G: 150, B: 150, A: 255}
		}
		if btn.Clicked(gtx) {
			c.CurrentTab = idx
		}
		return b.Layout(gtx)
	})
}

// layoutStatusAndError renders global status buttons and error messages.
func (c *Controller) layoutStatusAndError(gtx layout.Context) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			btn := material.Button(c.Theme, &c.OpenBtn, "Загрузить BMP для анализа")
			if c.IsLoading {
				btn.Text = "Обработка..."
				btn.Background = color.NRGBA{R: 100, G: 100, B: 100, A: 255}
			}
			return btn.Layout(gtx)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(20)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if c.ErrorMsg != "" {
				lbl := material.Label(c.Theme, unit.Sp(14), c.ErrorMsg)
				lbl.Color = color.NRGBA{R: 200, G: 0, B: 0, A: 255}
				return lbl.Layout(gtx)
			}
			return layout.Dimensions{}
		}),
	)
}

func (c *Controller) layoutLegend(gtx layout.Context) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {

			label := fmt.Sprintf("Карта пригодности (Чувствительность = %.0f):", c.AnalysisSensitivity)
			return material.Label(c.Theme, unit.Sp(14), label).Layout(gtx)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(5)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(material.Label(c.Theme, unit.Sp(12), "Гладко").Layout),
				layout.Rigid(layout.Spacer{Width: unit.Dp(5)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					imgOp := paint.NewImageOp(c.LegendImg)
					imgOp.Add(gtx.Ops)
					dims := image.Pt(gtx.Dp(unit.Dp(300)), gtx.Dp(unit.Dp(20)))
					gtx.Constraints = layout.Exact(dims)
					paint.PaintOp{}.Add(gtx.Ops)
					return layout.Dimensions{Size: dims}
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(5)}.Layout),
				layout.Rigid(material.Label(c.Theme, unit.Sp(12), "Шум").Layout),
			)
		}),
	)
}

// layoutResults renders the list of analysis maps (Radius 1, 2, 3).
func (c *Controller) layoutResults(gtx layout.Context) layout.Dimensions {
	if c.OriginalImg == nil {
		return material.Label(c.Theme, unit.Sp(16), "Выберите файл...").Layout(gtx)
	}

	return material.List(c.Theme, &c.ListWidget).Layout(gtx, 1, func(gtx layout.Context, index int) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return c.drawLabeledImage(gtx, "Оригинал", c.OriginalImg)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(20)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceBetween}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return c.drawLabeledImage(gtx, "R=1 (Детали)", c.ResultMaps[1])
					}),
					layout.Rigid(layout.Spacer{Width: unit.Dp(5)}.Layout),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return c.drawLabeledImage(gtx, "R=2", c.ResultMaps[2])
					}),
					layout.Rigid(layout.Spacer{Width: unit.Dp(5)}.Layout),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return c.drawLabeledImage(gtx, "R=3 (Крупно)", c.ResultMaps[3])
					}),
				)
			}),
		)
	})
}

// layoutConvertTab renders the file converter interface.
func (c *Controller) layoutConvertTab(gtx layout.Context) layout.Dimensions {

	if c.ConvOpenBtn.Clicked(gtx) {
		c.startOpenConvert()
	}
	if c.ConvSaveBtn.Clicked(gtx) {
		c.startSaveConvert()
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			btn := material.Button(c.Theme, &c.ConvOpenBtn, "1. Открыть файл (JPG/PNG/BMP)")
			return btn.Layout(gtx)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if c.ConvSourceImg == nil {
				return layout.Dimensions{}
			}
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(material.Label(c.Theme, unit.Sp(14), "Файл загружен: "+c.ConvPath).Layout),
				layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {

					return c.drawLabeledImage(gtx, "Превью", c.ConvSourceImg)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(20)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					btn := material.Button(c.Theme, &c.ConvSaveBtn, "2. Сохранить как...")
					btn.Background = color.NRGBA{R: 0, G: 150, B: 0, A: 255}
					return btn.Layout(gtx)
				}),
			)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if c.ErrorMsg != "" {
				layout.Spacer{Height: unit.Dp(10)}.Layout(gtx)
				lbl := material.Label(c.Theme, unit.Sp(14), c.ErrorMsg)
				lbl.Color = color.NRGBA{R: 200, G: 0, B: 0, A: 255}
				return lbl.Layout(gtx)
			}
			return layout.Dimensions{}
		}),
	)
}

func (c *Controller) startOpenConvert() {
	c.ErrorMsg = ""
	go func() {
		defer c.Window.Invalidate()

		path, err := dialog.File().Filter("Images", "jpg", "jpeg", "png", "bmp").Load()
		if err != nil {
			return
		}

		img, err := utils.LoadImage(path)
		if err != nil {
			c.ErrorMsg = "Ошибка открытия: " + err.Error()
			return
		}
		c.ConvSourceImg = img
		c.ConvPath = path
	}()
}

func (c *Controller) startSaveConvert() {
	if c.ConvSourceImg == nil {
		return
	}

	defaultName := c.ConvPath
	if idx := strings.LastIndex(defaultName, "."); idx != -1 {
		defaultName = defaultName[:idx]
	}

	go func() {
		defer c.Window.Invalidate()

		targetPath, err := dialog.File().
			Title("Сохранить как...").
			SetStartFile(defaultName).
			Filter("BMP Image", "bmp").
			Filter("PNG Image", "png").
			Filter("JPEG Image", "jpg").
			Save()

		if err != nil {
			return
		}

		err = utils.SaveImage(c.ConvSourceImg, targetPath+".bmp")
		if err != nil {
			c.ErrorMsg = "Ошибка сохранения: " + err.Error()
		} else {
			c.ErrorMsg = "Файл успешно сохранен!"
		}
	}()
}

func (c *Controller) drawLabeledImage(gtx layout.Context, label string, img image.Image) layout.Dimensions {
	if img == nil {
		return layout.Dimensions{}
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(material.Label(c.Theme, unit.Sp(14), label).Layout),
		layout.Rigid(layout.Spacer{Height: unit.Dp(5)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			maxHeight := gtx.Dp(unit.Dp(300))
			if gtx.Constraints.Max.Y > maxHeight {
				gtx.Constraints.Max.Y = maxHeight
			}
			wImg := widget.Image{Src: paint.NewImageOp(img), Fit: widget.Contain}
			return wImg.Layout(gtx)
		}),
	)
}

// generateGradientLegend creates a simple 1px high rainbow gradient for the UI legend.
func generateGradientLegend() image.Image {
	width := 256
	height := 1
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	drawer := drawer.NewMapDrawer()
	for x := 0; x < width; x++ {
		normalized := float64(x) / float64(width-1)
		col := drawer.ValueToHeatmapColor(normalized)
		img.Set(x, 0, col)
	}
	return img
}
