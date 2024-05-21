package widget

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/google/uuid"
)

/**
 * I want this to show a guage in the interface
 */

type Linegraph struct {
	widget.BaseWidget

	min, max       float64
	points         []float64
	xlabel, ylabel string
	id             string

	back, center                *canvas.Image
	xlegend, ylegend, maxlegend *widget.Label
}

var linegraphBaseImage *fyne.StaticResource

func NewLinegraphWidget(
	min, max float64,
	points []float64,
	xlabel, ylabel string,
) (newLinegraph *Linegraph) {
	newLinegraph = &Linegraph{
		min:    min,
		max:    max,
		points: points,
		xlabel: xlabel,
		ylabel: ylabel,
		id:     uuid.New().String(),
	}
	newLinegraph.ExtendBaseWidget(newLinegraph)
	if linegraphBaseImage == nil {
		linegraphBaseImage = fyne.NewStaticResource("linegraph-background.svg", []byte(`<?xml version="1.0"?>
	<svg version="1.1" width="600" height="320" viewBox="0,0,600,320"
		xmlns="http://www.w3.org/2000/svg">
		<rect x="0" y="0" width="600" height="320" stroke="none" fill="white" />
		<line x1="10" y1="10" x2="10" y2="310" stroke="black" stroke-width="1px" />
		<line x1="10" y1="310" x2="590" y2="310" stroke="black" stroke-width="1px" />
	</svg>`))
	}
	return
}

/**
 * SVG in Fyne cannot have text
 */
func (item *Linegraph) CreateRenderer() fyne.WidgetRenderer {
	item.back = canvas.NewImageFromResource(linegraphBaseImage)
	item.back.FillMode = canvas.ImageFillOriginal
	item.back.Move(fyne.NewPos(0, 0))
	item.back.Resize(fyne.NewSize(300, 600))
	item.back.Refresh()

	item.center = canvas.NewImageFromResource(
		fyne.NewStaticResource(item.id+".svg", []byte(`<?xml version="1.0"?>
		<svg version="1.1" width="600" height="320" viewBox="0,0,600,320"
			xmlns="http://www.w3.org/2000/svg">
			<path d="`+item.ValuesToPointString()+`" fill="none" stroke="red" stroke-width="1px"/>
		</svg>`),
		))
	item.center.FillMode = canvas.ImageFillOriginal
	item.center.Move(fyne.NewPos(0, 0))
	item.center.Resize(fyne.NewSize(300, 600))
	item.center.Refresh()

	item.xlegend = widget.NewLabel(item.xlabel)
	item.ylegend = widget.NewLabel(item.ylabel)
	item.maxlegend = widget.NewLabel(fmt.Sprintf("%f", item.max))

	/*
		c := container.NewBorder(
			item.maxlegend,
			item.xlegend,
			item.ylegend,
			nil,
			container.NewWithoutLayout(
				item.back,
				item.center,
			),
		)*/
	c := container.NewWithoutLayout(
		item.back,
		item.center,
	)
	c.Resize(fyne.NewSize(300, 600))
	c.Refresh()
	return widget.NewSimpleRenderer(c)
}

func (item *Linegraph) UpdateItems(newvalues []float64) {
	item.center = canvas.NewImageFromResource(
		fyne.NewStaticResource(item.id+".svg", []byte(`<?xml version="1.0"?>
		<svg version="1.1" width="600" height="320" viewBox="0,0,600,320"
			xmlns="http://www.w3.org/2000/svg">
			<path d="`+item.ValuesToPointString()+`" fill="none" stroke="red" stroke-width="1px"/>
		</svg>`),
		))

	item.Refresh()
}

func (item *Linegraph) ValuesToPointString() (pointString string) {
	pointString = "M"
	prefix := ""
	xstep := 580 / (len(item.points) - 1)
	for i, x := range item.points {
		pointString = fmt.Sprintf("%s%s%d,%f ", pointString, prefix, 10+i*xstep, 10+300-(x/item.max)*300)
		prefix = "L"
	}
	return
}
