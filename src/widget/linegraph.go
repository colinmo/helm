package widget

import (
	"bytes"
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

	min, max       int
	points         []int
	xlabel, ylabel string
	id             string

	xlegend, ylegend, maxlegend *widget.Label
	c                           *fyne.Container
}

var finalImageWidth = float32(300.0)
var finalImageHeight = float32(160.0)
var linegraphBaseImage *fyne.StaticResource

func NewLinegraphWidget(
	min, max int,
	points []int,
	xlabel, ylabel string,
) (newLinegraph *Linegraph) {
	newLinegraph = &Linegraph{
		min:    min,
		max:    max,
		points: points,
		xlabel: xlabel,
		ylabel: ylabel,
		id:     uuid.New().String(),
		c:      container.NewWithoutLayout(),
	}
	newLinegraph.ExtendBaseWidget(newLinegraph)
	if linegraphBaseImage == nil {
		linegraphBaseImage = fyne.NewStaticResource("linegraph-background.svg", []byte(`<?xml version="1.0"?>
	<svg version="1.1" width="600" height="320" viewBox="0,0,600,320"
		xmlns="http://www.w3.org/2000/svg">
	</svg>`))
	}
	return
}

/**
 * SVG in Fyne cannot have text
 */
func (item *Linegraph) CreateRenderer() fyne.WidgetRenderer {
	item.xlegend = widget.NewLabel(item.xlabel)
	item.ylegend = widget.NewLabel(item.ylabel)
	item.maxlegend = widget.NewLabel(fmt.Sprintf("%d", item.max))

	item.c = container.NewWithoutLayout(
		item.BasicImage(),
		item.xlegend,
	)
	item.c.Resize(fyne.NewSize(finalImageWidth, finalImageHeight))
	item.c.Refresh()
	return widget.NewSimpleRenderer(item.c)
}

func (item *Linegraph) GraphSVG() []byte {
	return []byte(`<?xml version="1.0"?>
	<svg version="1.1" width="600" height="320" viewBox="0,0,600,320"
		xmlns="http://www.w3.org/2000/svg">
		<rect x="0" y="0" width="600" height="320" stroke="none" fill="white" />
		<line x1="10" y1="10" x2="10" y2="310" stroke="black" stroke-width="1px" />
		<line x1="10" y1="310" x2="590" y2="310" stroke="black" stroke-width="1px" />
		<path d="` + item.ValuesToPointString() + `" fill="none" stroke="blue" stroke-width="1px"/>
	</svg>`)
}

func (item *Linegraph) BasicImage() *canvas.Image {
	graph := canvas.NewImageFromReader(bytes.NewReader(item.GraphSVG()), uuid.New().String()+".svg")
	graph.FillMode = canvas.ImageFillOriginal
	graph.Move(fyne.NewPos(0, 0))
	graph.Resize(fyne.NewSize(finalImageWidth, finalImageHeight))
	graph.Refresh()
	return graph
}

func (item *Linegraph) Refresh() {
	item.UpdateXLegend()
	if len(item.c.Objects) > 0 {
		item.c.Objects[0] = item.BasicImage()
		item.c.Refresh()
		canvas.Refresh(item)
	}
}
func (item *Linegraph) UpdateItems(newvalues []int) {
	item.points = newvalues
	item.Refresh()
}

func (item *Linegraph) UpdateXLegend() {
	l := len(item.points)
	if l > 0 {
		item.xlegend.Text = fmt.Sprintf("%d/%d\n", item.points[len(item.points)-1], item.max)
	}
}

func (item *Linegraph) UpdateMax(newmax int) {
	item.max = newmax
	item.Refresh()
}
func (item *Linegraph) UpdateItemsAndMax(newvalues []int, newmax int) {
	item.points = newvalues
	item.max = newmax
	item.Refresh()
}

func (item *Linegraph) ValuesToPointString() (pointString string) {
	pointString = "M"
	dotString := ""
	prefix := ""
	thisLen := len(item.points) - 1
	if thisLen == 0 {
		thisLen = 2
	}
	xstep := 580 / thisLen
	if item.max != 0 {
		for i, x := range item.points {
			point := item.ValueToPoint(x)
			pointString = fmt.Sprintf("%s%s%d,%d ", pointString, prefix, 10+i*xstep, point)
			prefix = "L"
			dotString = fmt.Sprintf("%s M%d,%dm-2,-2l0,4l4,0l0,-4l-4,0", dotString, 10+i*xstep, point)
		}
	}
	pointString = fmt.Sprintf("%s %s", pointString, dotString)
	return
}

func (item *Linegraph) ValueToPoint(val int) int {
	border := 10
	height := 300

	if item.max == 0 {
		return 0
	}
	point := int(border + height - int((float32(val)/float32(item.max))*float32(height)))

	return point
}
