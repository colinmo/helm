package main

import (
	"fmt"
	"image/color"
	"path"
	"time"

	fyne "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func dateSinceNowInString(oldDate time.Time) string {
	bob := time.Since(oldDate.Local())
	mep := bob.Seconds()
	if mep < 0 {
		mep = mep + 60*60*10
	}
	metric := ""
	switch {
	case mep >= 31540000:
		metric = "year"
		mep = mep / 31540000
	case mep >= 2628333.332282:
		metric = "month"
		mep = mep / 2628333.332282
	case mep >= 604800:
		metric = "week"
		mep = mep / 604800
	case mep >= 86400:
		metric = "day"
		mep = mep / 86400
	case mep >= 3600:
		metric = "hour"
		mep = mep / 3600
	case mep >= 60:
		metric = "minute"
		mep = mep / 60
	default:
		metric = "second"
	}
	if mep >= 2 {
		metric = metric + "s"
	}
	return fmt.Sprintf("%d %s", int(mep), metric)
}

func setupPriorityIcons() map[string]*widget.Icon {
	priorityIcons := map[string]*widget.Icon{}
	priorityIcons["1"] = widget.NewIcon(
		fyne.NewStaticResource("priority1.svg", []byte(`<?xml version="1.0"?>
		<svg version="1.1" width="30" height="30"
			xmlns="http://www.w3.org/2000/svg">
			<circle cx="15" cy="15" r="14" fill="black" stroke="black" />
			<circle cx="15" cy="15" r="13.5" fill="red" stroke="red" />
			<circle cx="15" cy="15" r="10.5" fill="black" stroke="black" />
			<circle cx="15" cy="15" r="10" fill="white" stroke="white" />
		</svg>`)))
	priorityIcons["2"] = widget.NewIcon(
		fyne.NewStaticResource("priority2.svg", []byte(`<?xml version="1.0"?>
		<svg version="1.1" width="30" height="30"
			xmlns="http://www.w3.org/2000/svg">
			<path stroke-width="4" stroke="black" fill="none" stroke-linecap="round" d="M 11,3.2 A 15,13 1 0 0 11,27" />
			<path stroke-width="4" stroke="black" fill="none" stroke-linecap="round" d="M 19,3 A 15,13 0 0 1 18.5,27" />
			<path stroke-width="3" stroke="orange" fill="none" stroke-linecap="round" d="M 11,3.2 A 15,13 1 0 0 11,27" />
			<path stroke-width="3" stroke="orange" fill="none" stroke-linecap="round" d="M 19,3 A 15,13 0 0 1 18.5,27" />
		</svg>`)))
	priorityIcons["3"] = widget.NewIcon(
		fyne.NewStaticResource("priority3.svg", []byte(`<?xml version="1.0"?>
		<svg version="1.1" width="30" height="30"
			xmlns="http://www.w3.org/2000/svg">
			<path stroke-width="4" stroke="black" fill="none" stroke-linecap="round" d="M 6,6.2 A 13.5,15 0 0 1 24,6.2" />
			<path stroke-width="3" stroke="yellow" fill="none" stroke-linecap="round" d="M 6,6.2 A 13.5,15 0 0 1 24,6.2" />
			<g transform="rotate(120, 15, 15)">
				<path stroke-width="4" stroke="black" fill="none" stroke-linecap="round" d="M 6,6.2 A 13.5,15 0 0 1 24,6.2" />
				<path stroke-width="3" stroke="yellow" fill="none" stroke-linecap="round" d="M 6,6.2 A 13.5,15 0 0 1 24,6.2" />
			</g>
			<g transform="rotate(-120, 15, 15)">
				<path stroke-width="4" stroke="black" fill="none" stroke-linecap="round" d="M 6,6.2 A 13.5,15 0 0 1 24,6.2" />
				<path stroke-width="3" stroke="yellow" fill="none" stroke-linecap="round" d="M 6,6.2 A 13.5,15 0 0 1 24,6.2" />
			</g>
		</svg>`)))
	priorityIcons["4"] = widget.NewIcon(
		fyne.NewStaticResource("priority4.svg", []byte(`<?xml version="1.0"?>
		<svg version="1.1" width="30" height="30"
			xmlns="http://www.w3.org/2000/svg">
			<path stroke-width="4" stroke="black" fill="none" stroke-linecap="round" d="M 11,3.2 A 15,13 1 0 0 2.5,11.5" />
			<path stroke-width="3" stroke="lime" fill="none" stroke-linecap="round" d="M 11,3.2 A 15,13 1 0 0 2.5,11.5" />
			<path stroke-width="4" stroke="black" fill="none" stroke-linecap="round" d="M 2.5,18 A 15,13 1 0 0 11,26.8" />
			<path stroke-width="3" stroke="lime" fill="none" stroke-linecap="round" d="M 2.5,18 A 15,13 1 0 0 11,26.8" />
			<path stroke-width="4" stroke="black" fill="none" stroke-linecap="round" d="M 19,3.2 A 15,13 0 0 1 27.5,11.5" />
			<path stroke-width="3" stroke="lime" fill="none" stroke-linecap="round" d="M 19,3.2 A 15,13 0 0 1 27.5,11.5" />
			<path stroke-width="4" stroke="black" fill="none" stroke-linecap="round" d="M 27.5,18.5 A 15,13 0 0 1 18.5,26.8" />
			<path stroke-width="3" stroke="lime" fill="none" stroke-linecap="round" d="M 27.5,18.5 A 15,13 0 0 1 18.5,26.8" />
		</svg>`)))
	priorityIcons["5"] = widget.NewIcon(
		fyne.NewStaticResource("priority5.svg", []byte(`<?xml version="1.0"?>
		<svg version="1.1" width="30" height="30"
			xmlns="http://www.w3.org/2000/svg">
			<path stroke-width="4" stroke="black" fill="black" stroke-linecap="round" d="M 4.5,8 A 15,15 0 0 1 11,3" />
			<path stroke-width="3" stroke="cyan" fill="cyan" stroke-linecap="round" d="M 4.5,8 A 15,15 0 0 1 11,3" />
			<path stroke-width="4" stroke="black" fill="black" stroke-linecap="round" d="M 19,3.2 A 15,15 0 0 1 25.5,8" />
			<path stroke-width="3" stroke="cyan" fill="cyan" stroke-linecap="round" d="M 19,3.2 A 15,15 0 0 1 25.5,8" />
			<path stroke-width="4" stroke="black" fill="black" stroke-linecap="round" d="M 27.3,13 A 15,15 0 0 1 25.5,22" />
			<path stroke-width="3" stroke="cyan" fill="cyan" stroke-linecap="round" d="M 27.3,13 A 15,15 0 0 1 25.5,22" />
			<path stroke-width="4" stroke="black" fill="black" stroke-linecap="round" d="M 21,26 A 14,15 0 0 1 9,26" />
			<path stroke-width="3" stroke="cyan" fill="cyan" stroke-linecap="round" d="M 21,26 A 14,15 0 0 1 9,26" />
			<path stroke-width="4" stroke="black" fill="black" stroke-linecap="round" d="M 4.5,22 A 15,15 0 0 1 2.7,13" />
			<path stroke-width="3" stroke="cyan" fill="cyan" stroke-linecap="round" d="M 4.5,22 A 15,15 0 0 1 2.7,13" />
		</svg>`)))
	return priorityIcons
}

func getPriorityIconFor(index string, priorityIcons map[string]*widget.Icon) *widget.Icon {
	if icon, ok := priorityIcons[index]; ok {
		return icon
	}
	return widget.NewIcon(theme.CancelIcon())
}

func setupJiraTaskTypeIcons() map[string]*widget.Icon {
	taskIcons := map[string]*widget.Icon{}
	baseOpacity := "0.3"
	taskIcons["Epic"] = widget.NewIcon(
		fyne.NewStaticResource("epic.svg", []byte(`<?xml version="1.0" encoding="UTF-8" standalone="no"?>
		<svg width="16px" height="16px" viewBox="0 0 16 16" version="1.1" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" xmlns:sketch="http://www.bohemiancoding.com/sketch/ns">
			<g id="Page-1" stroke="none" stroke-width="1" fill="none" fill-rule="evenodd" sketch:type="MSPage" style="opacity: `+baseOpacity+`">
				<g id="epic" sketch:type="MSArtboardGroup">
					<g id="Epic" sketch:type="MSLayerGroup" transform="translate(1.000000, 1.000000)">
						<rect id="Rectangle-36" fill="#904EE2" sketch:type="MSShapeGroup" x="0" y="0" width="14" height="14" rx="2"></rect>
						<g id="Page-1" transform="translate(4.000000, 3.000000)" fill="#FFFFFF" sketch:type="MSShapeGroup">
							<path d="M5.9233,3.7566 L5.9213,3.7526 C5.9673,3.6776 6.0003,3.5946 6.0003,3.4996 C6.0003,3.2236 5.7763,2.9996 5.5003,2.9996 L3.0003,2.9996 L3.0003,0.4996 C3.0003,0.2236 2.7763,-0.0004 2.5003,-0.0004 C2.3283,-0.0004 2.1853,0.0916 2.0953,0.2226 C2.0673,0.2636 2.0443,0.3056 2.0293,0.3526 L0.0813,4.2366 L0.0833,4.2396 C0.0353,4.3166 0.0003,4.4026 0.0003,4.4996 C0.0003,4.7766 0.2243,4.9996 0.5003,4.9996 L3.0003,4.9996 L3.0003,7.4996 C3.0003,7.7766 3.2243,7.9996 3.5003,7.9996 C3.6793,7.9996 3.8293,7.9006 3.9183,7.7586 L3.9213,7.7596 L3.9343,7.7336 C3.9453,7.7126 3.9573,7.6936 3.9653,7.6716 L5.9233,3.7566 Z" id="Fill-1"></path>
						</g>
					</g>
				</g>
			</g>
		</svg>`)))
	taskIcons["Story"] = widget.NewIcon(
		fyne.NewStaticResource("story.svg", []byte(`<?xml version="1.0" encoding="UTF-8" standalone="no"?>
		<svg width="16px" height="16px" viewBox="0 0 16 16" version="1.1" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" xmlns:sketch="http://www.bohemiancoding.com/sketch/ns">
			<!-- Generator: Sketch 3.5.2 (25235) - http://www.bohemiancoding.com/sketch -->
			<title>story</title>
			<desc>Created with Sketch.</desc>
			<defs></defs>
			<g id="Page-1" stroke="none" stroke-width="1" fill="none" fill-rule="evenodd" sketch:type="MSPage" style="opacity: `+baseOpacity+`">
				<g id="story" sketch:type="MSArtboardGroup">
					<g id="Story" sketch:type="MSLayerGroup" transform="translate(1.000000, 1.000000)">
						<rect id="Rectangle-36" fill="#63BA3C" sketch:type="MSShapeGroup" x="0" y="0" width="14" height="14" rx="2"></rect>
						<path d="M9,3 L5,3 C4.448,3 4,3.448 4,4 L4,10.5 C4,10.776 4.224,11 4.5,11 C4.675,11 4.821,10.905 4.91,10.769 L4.914,10.77 L6.84,8.54 C6.92,8.434 7.08,8.434 7.16,8.54 L9.086,10.77 L9.09,10.769 C9.179,10.905 9.325,11 9.5,11 C9.776,11 10,10.776 10,10.5 L10,4 C10,3.448 9.552,3 9,3" id="Page-1" fill="#FFFFFF" sketch:type="MSShapeGroup"></path>
					</g>
				</g>
			</g>
		</svg>`)))
	taskIcons["Bug"] = widget.NewIcon(
		fyne.NewStaticResource("bug.svg", []byte(`<?xml version="1.0" encoding="UTF-8" standalone="no"?>
		<svg width="16px" height="16px" viewBox="0 0 16 16" version="1.1" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" xmlns:sketch="http://www.bohemiancoding.com/sketch/ns">
			<!-- Generator: Sketch 3.5.2 (25235) - http://www.bohemiancoding.com/sketch -->
			<title>bug</title>
			<desc>Created with Sketch.</desc>
			<defs></defs>
			<g id="Page-1" stroke="none" stroke-width="1" fill="none" fill-rule="evenodd" sketch:type="MSPage" style="opacity: `+baseOpacity+`">
				<g id="bug" sketch:type="MSArtboardGroup">
					<g id="Bug" sketch:type="MSLayerGroup" transform="translate(1.000000, 1.000000)">
						<rect id="Rectangle-36" fill="#E5493A" sketch:type="MSShapeGroup" x="0" y="0" width="14" height="14" rx="2"></rect>
						<path d="M10,7 C10,8.657 8.657,10 7,10 C5.343,10 4,8.657 4,7 C4,5.343 5.343,4 7,4 C8.657,4 10,5.343 10,7" id="Fill-2" fill="#FFFFFF" sketch:type="MSShapeGroup"></path>
					</g>
				</g>
			</g>
		</svg>`)))
	taskIcons["Task"] = widget.NewIcon(
		fyne.NewStaticResource("task.svg", []byte(`<?xml version="1.0" encoding="UTF-8" standalone="no"?>
			<svg width="16px" height="16px" viewBox="0 0 16 16" version="1.1" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" xmlns:sketch="http://www.bohemiancoding.com/sketch/ns">
				<!-- Generator: Sketch 3.5.2 (25235) - http://www.bohemiancoding.com/sketch -->
				<title>task</title>
				<desc>Created with Sketch.</desc>
				<defs></defs>
				<g id="Page-1" stroke="none" stroke-width="1" fill="none" fill-rule="evenodd" sketch:type="MSPage" style="opacity: `+baseOpacity+`">
					<g id="task" sketch:type="MSArtboardGroup">
						<g id="Task" sketch:type="MSLayerGroup" transform="translate(1.000000, 1.000000)">
							<rect id="Rectangle-36" fill="#4BADE8" sketch:type="MSShapeGroup" x="0" y="0" width="14" height="14" rx="2"></rect>
							<g id="Page-1" transform="translate(4.000000, 4.500000)" stroke="#FFFFFF" stroke-width="2" stroke-linecap="round" sketch:type="MSShapeGroup">
								<path d="M2,5 L6,0" id="Stroke-1"></path>
								<path d="M2,5 L0,3" id="Stroke-3"></path>
							</g>
						</g>
					</g>
				</g>
			</svg>`)))
	taskIcons["Initiative"] = widget.NewIcon(
		fyne.NewStaticResource("init.svg", []byte(`<?xml version="1.0" encoding="UTF-8" standalone="no"?>
		<svg width="16px" height="16px" viewBox="0 0 16 16" version="1.1" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" xmlns:sketch="http://www.bohemiancoding.com/sketch/ns">
			<!-- Generator: Sketch 3.5.2 (25235) - http://www.bohemiancoding.com/sketch -->
			<title>suggestion</title>
			<desc>Created with Sketch.</desc>
			<defs></defs>
			<g id="Page-1" stroke="none" stroke-width="1" fill="none" fill-rule="evenodd" sketch:type="MSPage" style="opacity: `+baseOpacity+`">
				<g id="suggestion" sketch:type="MSArtboardGroup">
					<g id="Suggestion" sketch:type="MSLayerGroup" transform="translate(1.000000, 1.000000)">
						<rect id="Rectangle-36" fill="#FF9C23" sketch:type="MSShapeGroup" x="0" y="0" width="14" height="14" rx="2"></rect>
						<g id="Page-1" transform="translate(4.000000, 2.000000)" fill="#FFFFFF" sketch:type="MSShapeGroup">
							<path d="M2.4916,9 C1.9396,9 1.4916,8.552 1.4916,8 L1.4916,6 L4.4916,6 L4.4916,8 C4.4916,8.552 4.0436,9 3.4916,9 L2.4916,9 Z" id="Fill-4"></path>
							<path d="M4.49,7.5528 L3.49,7.5528 L3.49,6.7398 C3.492,6.6268 3.501,6.5218 3.521,6.4208 C3.587,6.0748 3.776,5.8148 3.963,5.5768 L4.205,5.2798 C4.308,5.1538 4.412,5.0288 4.51,4.8988 C4.921,4.3588 5.062,3.7818 4.941,3.1348 C4.772,2.2308 3.937,1.5468 2.999,1.5428 C2.046,1.5468 1.211,2.2308 1.042,3.1348 C0.922,3.7818 1.063,4.3588 1.473,4.8988 C1.573,5.0298 1.677,5.1568 1.782,5.2838 L2.02,5.5778 C2.236,5.8438 2.434,6.0898 2.498,6.4218 C2.517,6.5228 2.526,6.6288 2.528,6.7328 L2.528,7.5528 L1.528,7.5528 L1.528,6.7398 C1.527,6.7008 1.524,6.6558 1.516,6.6108 C1.499,6.5238 1.392,6.3918 1.299,6.2758 L1.01,5.9188 C0.897,5.7818 0.785,5.6458 0.677,5.5048 C0.094,4.7378 -0.114,3.8788 0.059,2.9508 C0.318,1.5608 1.547,0.5488 2.98,0.5428 C4.436,0.5488 5.665,1.5608 5.924,2.9508 C6.097,3.8788 5.889,4.7378 5.306,5.5048 C5.2,5.6438 5.089,5.7798 4.977,5.9148 L4.748,6.1978 C4.632,6.3438 4.526,6.4848 4.503,6.6098 C4.494,6.6568 4.491,6.7028 4.49,6.7488 L4.49,7.5528 Z" id="Fill-1"></path>
						</g>
					</g>
				</g>
			</g>
		</svg>`)))
	return taskIcons
}

func getJiraTypeIconFor(index string, taskIcons map[string]*widget.Icon) *widget.Icon {
	if icon, ok := taskIcons[index]; ok {
		return icon
	}
	return widget.NewIcon(theme.CancelIcon())

}

func buildMonthSelect(dateToShow time.Time, owningDialog *dialog.Dialog) *fyne.Container {
	// Calculate the days shown
	startOfMonth, _ := time.Parse("2006-January-03", fmt.Sprintf("%s-%s", dateToShow.Format("2006-January"), "01"))
	startOfMonthDisplay := startOfMonth
	startOffset := int(startOfMonth.Weekday())
	if startOffset != 6 {
		startOfMonthDisplay = startOfMonthDisplay.AddDate(0, 0, -1*int(startOfMonth.Weekday()))
	} else {
		startOffset = 0
	}
	totalDays := startOffset + startOfMonth.AddDate(0, 1, -1).Day()
	remainder := totalDays % 7
	if remainder > 0 {
		totalDays += 7 - totalDays%7
	}

	days := []fyne.CanvasObject{
		widget.NewLabelWithStyle("S", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("M", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("T", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("W", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("T", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("F", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("S", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
	}
	thisDay := startOfMonthDisplay
	todayString := time.Now().Format("01/02/2006")
	for i := 0; i < totalDays; i++ {
		mike := thisDay
		bg := canvas.NewRectangle(color.NRGBA{R: 220, G: 220, B: 220, A: 0})
		if thisDay.Format("01/02/2006") == todayString {
			bg = canvas.NewRectangle(color.NRGBA{R: 100, G: 200, B: 150, A: 255})
		}
		days = append(days, container.NewStack(bg, widget.NewButton(fmt.Sprintf("%d", thisDay.Day()), func() {
			x, _ := AppStatus.CurrentZettleDKB.Get()
			saveZettle(markdownInput.Text, x)

			AppStatus.CurrentZettleDBDate = mike
			AppStatus.CurrentZettleDKB.Set(zettleFileName(AppStatus.CurrentZettleDBDate))
			x, _ = AppStatus.CurrentZettleDKB.Get()
			markdownInput.Text = getFileContentsAndCreateIfMissing(path.Join(appPreferences.ZettlekastenHome, x))
			markdownInput.Refresh()
			(*owningDialog).Hide()
		})))
		thisDay = thisDay.AddDate(0, 0, 1)
	}
	return container.NewGridWithColumns(7,
		days...)
}

func createDatePicker(dateToShow time.Time, owningDialog *dialog.Dialog) fyne.CanvasObject {
	var calendarWidget *fyne.Container
	var monthSelect *widget.Label
	var monthDisplay *fyne.Container
	var backMonth *widget.Button
	var forwardMonth *widget.Button

	monthSelect = widget.NewLabel(dateToShow.Format("January 2006"))

	monthDisplay = buildMonthSelect(dateToShow, owningDialog)

	backMonth = widget.NewButtonWithIcon("", theme.NavigateBackIcon(), func() {
		dateToShow = dateToShow.AddDate(0, -1, 0)
		monthSelect = widget.NewLabel(dateToShow.Format("January 2006"))
		monthDisplay = buildMonthSelect(dateToShow, owningDialog)
		calendarWidget.RemoveAll()
		calendarWidget.Add(container.NewBorder(
			container.NewHBox(
				backMonth,
				layout.NewSpacer(),
				monthSelect,
				layout.NewSpacer(),
				forwardMonth,
			),
			nil,
			nil,
			nil,
			monthDisplay))
		calendarWidget.Refresh()
	})
	forwardMonth = widget.NewButtonWithIcon("", theme.NavigateNextIcon(), func() {
		dateToShow = dateToShow.AddDate(0, 1, 0)
		monthSelect = widget.NewLabel(dateToShow.Format("January 2006"))
		monthDisplay = buildMonthSelect(dateToShow, owningDialog)
		calendarWidget.RemoveAll()
		calendarWidget.Add(container.NewBorder(
			container.NewHBox(
				backMonth,
				layout.NewSpacer(),
				monthSelect,
				layout.NewSpacer(),
				forwardMonth,
			),
			nil,
			nil,
			nil,
			monthDisplay))
		calendarWidget.Refresh()
	})
	// Build the UI
	// Note: RemoveAll/Add required so the above back/Forward months look the same
	calendarWidget = container.NewHBox(widget.NewLabel("Loading"))
	calendarWidget.RemoveAll()
	calendarWidget.Add(container.NewBorder(
		container.NewHBox(
			backMonth,
			layout.NewSpacer(),
			monthSelect,
			layout.NewSpacer(),
			forwardMonth,
		),
		nil,
		nil,
		nil,
		monthDisplay))
	return calendarWidget
}
