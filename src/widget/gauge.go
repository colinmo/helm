package widget

import (
	"bufio"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/disintegration/imaging"
	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"github.com/google/uuid"
	"golang.org/x/image/font"
)

/**
 * I want this to show a guage in the interface
 */

type Gauge struct {
	widget.BaseWidget
	min, max, current float64
	id                string
}

var backgroundImage *fyne.StaticResource
var dial *fyne.StaticResource

func NewGaugeWidget(
	min, max, current float64,
) (newGauge *Gauge) {
	newGauge = &Gauge{
		min:     min,
		max:     max,
		current: current,
		id:      uuid.New().String(),
	}
	if backgroundImage == nil {
		fmt.Printf("Let's do this")
		// Put the static rendering images creation here?
		backgroundImage = fyne.NewStaticResource("gauge-background.svg", []byte(`<?xml version="1.0"?>
	<svg version="1.1" width="300" height="300" viewBox="0,0,300,300"
		xmlns="http://www.w3.org/2000/svg">
		<circle cx="150" cy="150" r="140" fill="black" stroke="black" />
		<circle cx="150" cy="150" r="135" fill="red" stroke="red" />
		<circle cx="150" cy="150" r="105" fill="black" stroke="black" />
		<circle cx="150" cy="150" r="100" fill="white" stroke="white" />
		<path d="M150,150 l0,-100" fill="none" stroke="yellow" stroke-width="3px" />
		<path d="M150,150 l0,-100" fill="none" stroke="blue" stroke-width="3px" transform="rotate(-30,150,150)" />
		<circle cx="150" cy="150" r="1" fill="black" stroke="green" />
	</svg>`))
	}
	return
}

/**
 * SVG in Fyne cannot have text
 * But Fyne itself cannot have rotations
 * So we use an SVG anyway and add text over the top
 *
 * Might be more efficient to have a single background
 * and single indicator line image and just reuse them
 * for each renderer.
 */
func (item *Gauge) CreateRenderer() fyne.WidgetRenderer {
	center := indicatorLabel("Mid", 150, 50, 0)
	min := indicatorLabel(fmt.Sprintf("%2.0f", item.min), 50, 150, -45)
	back := canvas.NewImageFromResource(backgroundImage)
	back.FillMode = canvas.ImageFillOriginal
	back.Move(fyne.NewPos(0, 0))
	back.Resize(fyne.NewSize(300, 300))
	back.Refresh()
	c := container.NewWithoutLayout(
		back,
		center,
		min,
	)

	return widget.NewSimpleRenderer(c)
}

func indicatorLabel(label string, x, y float32, r float64) fyne.CanvasObject {
	fontSize := 16.0
	rgba := image.NewRGBA(image.Rect(0, 0, 640, 480))
	bg := image.NewUniform(color.RGBA{0, 0, 0, 0})
	draw.Draw(rgba, rgba.Bounds(), bg, image.Point{0, 0}, draw.Src)
	fg := image.NewUniform(color.RGBA{255, 0, 255, 255})
	loadedFont, err := loadFont("cour.ttf")
	if err != nil {
		log.Fatal(err)
		return container.NewWithoutLayout()
	}
	c := freetype.NewContext()
	c.SetDPI(72)
	c.SetFont(loadedFont)
	c.SetFontSize(fontSize)
	c.SetClip(rgba.Bounds())
	c.SetDst(rgba)
	c.SetSrc(fg)
	c.SetHinting(font.HintingNone)
	pt := freetype.Pt(0, int(c.PointToFixed(fontSize)>>6))
	endX, _ := c.DrawString(label, pt)
	img := rgba.SubImage(image.Rect(pt.X.Floor(), 5, endX.X.Ceil(), endX.Y.Ceil()))

	//TEST
	outFile, err := os.Create(`C:\Users\relap\Downloads\out.png`)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	defer outFile.Close()
	b := bufio.NewWriter(outFile)
	err = png.Encode(b, img)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	err = b.Flush()
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	//test
	img3 := imaging.Rotate(img, r, color.Transparent)
	w := img3.Bounds().Max.X
	h := img3.Bounds().Max.Y
	img2 := canvas.NewImageFromImage(img3)
	img2.FillMode = canvas.ImageFillOriginal
	img2.Move(fyne.NewPos(x-float32(w)/2, y-float32(h)/2))
	img2.Resize(fyne.NewSize(float32(w), float32(h)))
	img2.Refresh()
	return img2
}

func loadFont(file string) (*truetype.Font, error) {
	// Read the font data.
	fontBytes, err := os.ReadFile(file)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	f, err := freetype.ParseFont(fontBytes)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return f, nil
}
