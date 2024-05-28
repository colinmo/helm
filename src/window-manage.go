package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// TAPPABLE ICON
type tappableIcon struct {
	widget.Icon
	OnTapGo func(_ *fyne.PointEvent)
	MyId    int
}

func newTappableIcon(res fyne.Resource, tapped func(_ *fyne.PointEvent)) *tappableIcon {
	icon := &tappableIcon{}
	icon.ExtendBaseWidget(icon)
	icon.SetResource(res)
	icon.OnTapGo = tapped
	return icon
}

func (t *tappableIcon) Tapped(x *fyne.PointEvent) {
	t.OnTapGo(x)
}

func (t *tappableIcon) TappedSecondary(_ *fyne.PointEvent) {
}

// TAPPABLE LABEL
type tappableLabel struct {
	widget.Label
	OnTapGo func(_ *fyne.PointEvent)
}

func newTappableLabel(textLabel string, tapped func(_ *fyne.PointEvent)) *tappableLabel {
	label := &tappableLabel{}
	label.ExtendBaseWidget(label)
	label.SetText(textLabel)
	label.OnTapGo = tapped
	return label
}

func newTappableLabelWithStyle(
	textLabel string,
	align fyne.TextAlign,
	style fyne.TextStyle,
	tapped func(_ *fyne.PointEvent)) *tappableLabel {
	label := &tappableLabel{}
	label.ExtendBaseWidget(label)
	label.SetText(textLabel)
	label.Alignment = align
	label.TextStyle = style
	label.OnTapGo = tapped
	return label
}

func (t *tappableLabel) Tapped(x *fyne.PointEvent) {
	t.OnTapGo(x)
}

func (t *tappableLabel) TappedSecondary(_ *fyne.PointEvent) {
}
