package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"time"

	icon "vonexplaino.com/m/v2/helm/icon"

	fyne "fyne.io/fyne/v2"
	app "fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type AppStatus struct {
	CurrentZettleDBDate time.Time
	CurrentZettleDKB    binding.String
}

type AppPreferences struct {
	ZettlekastenHome string
	RouterUsername   string
	RouterPassword   string
}

var activeInternetTimeChan (chan time.Duration)
var thisApp fyne.App
var mainWindow fyne.Window
var preferencesWindow fyne.Window
var internetWindow fyne.Window
var markdownInput *widget.Entry
var appStatus AppStatus
var appPreferences AppPreferences

func main() {
	// Get initial statuses for things
	activeInternetTimeChan = make(chan time.Duration, 10)
	appStatus = AppStatus{
		CurrentZettleDBDate: time.Now().Local(),
		CurrentZettleDKB:    binding.NewString(),
	}
	appStatus.CurrentZettleDKB.Set(zettleFileName(time.Now().Local()))
	go waitingForInternetCommand()

	thisApp = app.NewWithID("com.vonexplaino.helm.preferences")
	thisApp.SetIcon(fyne.NewStaticResource("Systray", icon.Data))
	preferencesWindow = thisApp.NewWindow("Preferences")
	preferencesWindowSetup()
	internetWindow = thisApp.NewWindow("Internet Control")
	internetWindowSetup()
	mainWindow = thisApp.NewWindow("Markdown Daily Knowledgebase")
	markdownWindowSetup()
	if desk, ok := thisApp.(desktop.App); ok {
		m := fyne.NewMenu("MyApp",
			fyne.NewMenuItem("Todays Notes", func() {
				mainWindow.Show()
				// Reload from file
				x, _ := appStatus.CurrentZettleDKB.Get()
				markdownInput.Text = getFileContentsAndCreateIfMissing(path.Join(appPreferences.ZettlekastenHome, x))
				markdownInput.Refresh()
			}),
			fyne.NewMenuItem("Internet control", func() {
				internetWindow.Show()
			}),
			fyne.NewMenuItemSeparator(),
			fyne.NewMenuItem("Preferences", func() {
				preferencesWindow.Show()
			}),
		)
		desk.SetSystemTrayMenu(m)
	}
	// main window setup
	thisApp.Run()
}

func tellModemToAllowForPeriod(period time.Duration) {
	DisableParentalControls()
	activeInternetTimeChan <- period
}

func waitingForInternetCommand() {
	var activeInternetChangeTime time.Time
	var timerActive = false
	for {
		time.Sleep(time.Minute / 3)
		if len(activeInternetTimeChan) > 0 {
			// New message!
			change := <-activeInternetTimeChan
			if change < 0 {
				activeInternetChangeTime.Add(change * -1)
			} else {
				activeInternetChangeTime = time.Now().Add(change)
			}
			timerActive = true
		}
		if timerActive && time.Now().After(activeInternetChangeTime) {
			EnableParentalControls()
			timerActive = false
		}
	}
}

func saveZettle(content string, filename string) error {
	writeFileContents(path.Join(appPreferences.ZettlekastenHome, filename), content)
	return nil
}

func moveZettleDate(hours time.Duration) string {
	appStatus.CurrentZettleDBDate = appStatus.CurrentZettleDBDate.Add(time.Hour * hours)
	appStatus.CurrentZettleDKB.Set(zettleFileName(appStatus.CurrentZettleDBDate))
	x, _ := appStatus.CurrentZettleDKB.Get()
	return getFileContentsAndCreateIfMissing(path.Join(appPreferences.ZettlekastenHome, x))
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
		widget.NewLabel("S"),
		widget.NewLabel("M"),
		widget.NewLabel("T"),
		widget.NewLabel("W"),
		widget.NewLabel("T"),
		widget.NewLabel("F"),
		widget.NewLabel("S"),
	}
	thisDay := startOfMonthDisplay
	for i := 0; i < totalDays; i++ {
		mike := thisDay
		days = append(days, widget.NewButton(fmt.Sprintf("%d", thisDay.Day()), func() {
			x, _ := appStatus.CurrentZettleDKB.Get()
			saveZettle(markdownInput.Text, x)

			appStatus.CurrentZettleDBDate = mike
			appStatus.CurrentZettleDKB.Set(zettleFileName(appStatus.CurrentZettleDBDate))
			x, _ = appStatus.CurrentZettleDKB.Get()
			markdownInput.Text = getFileContentsAndCreateIfMissing(path.Join(appPreferences.ZettlekastenHome, x))
			markdownInput.Refresh()
			(*owningDialog).Hide()
		}))
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
		fmt.Printf("Date to show %s\n", dateToShow)
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

func markdownWindowSetup() {
	mainWindow.Resize(fyne.NewSize(800, 800))
	mainWindow.SetMaster()
	mainWindow.Hide()
	markdownInput = widget.NewEntry()
	markdownInput.MultiLine = true
	markdownInput.Wrapping = fyne.TextWrapWord
	previewWindow := thisApp.NewWindow("Preview")
	previewWindow.Resize(fyne.NewSize(800, 800))
	previewWindow.Hide()
	prevWindowVisible := false
	previewWindow.SetCloseIntercept(func() {
		prevWindowVisible = false
		previewWindow.Hide()
	})
	dateToShow := appStatus.CurrentZettleDBDate
	var deepdeep dialog.Dialog
	deepdeep = dialog.NewCustom(
		"Change date",
		"Nevermind",
		createDatePicker(dateToShow, &deepdeep),
		mainWindow,
	)
	menu := container.NewBorder(
		nil,
		nil,
		container.NewHBox(
			widget.NewButtonWithIcon("", theme.NavigateBackIcon(), func() {
				x, _ := appStatus.CurrentZettleDKB.Get()
				saveZettle(markdownInput.Text, x)
				markdownInput.Text = moveZettleDate(-24)
				markdownInput.Refresh()
			}),
			widget.NewButtonWithIcon("", theme.HistoryIcon(), func() {
				deepdeep.Show()
			}),
		),
		container.NewHBox(
			widget.NewButtonWithIcon("", theme.DocumentPrintIcon(), func() {
				if prevWindowVisible {
					prevWindowVisible = false
					previewWindow.Hide()
				} else {
					var previewMarkdown *widget.RichText
					if markdownInput.Text[0:3] == "---" {
						previewMarkdown = widget.NewRichTextFromMarkdown(strings.Split(markdownInput.Text, "...")[1])
					} else {
						previewMarkdown = widget.NewRichTextFromMarkdown(markdownInput.Text)
					}
					previewMarkdown.Wrapping = fyne.TextWrapWord
					previewWindow.SetContent(
						container.NewMax(
							previewMarkdown,
						),
					)
					prevWindowVisible = true
					previewWindow.Show()
				}
			}),
			widget.NewButtonWithIcon("", theme.NavigateNextIcon(), func() {
				x, _ := appStatus.CurrentZettleDKB.Get()
				saveZettle(markdownInput.Text, x)
				markdownInput.Text = moveZettleDate(24)
				markdownInput.Refresh()
			}),
		),
		widget.NewButton("Save", func() {
			x, _ := appStatus.CurrentZettleDKB.Get()
			writeFileContents(path.Join(appPreferences.ZettlekastenHome, x), markdownInput.Text)
		}),
	)
	content := container.NewBorder(menu, widget.NewLabelWithData(appStatus.CurrentZettleDKB), nil, nil, markdownInput)
	mainWindow.SetContent(content)
	mainWindow.SetCloseIntercept(func() {
		mainWindow.Hide()
		// Save contents
		x, _ := appStatus.CurrentZettleDKB.Get()
		saveZettle(markdownInput.Text, x)
	})
}

func preferencesWindowSetup() {
	appPreferences = AppPreferences{}
	appPreferences.ZettlekastenHome = thisApp.Preferences().StringWithFallback("ZettlekastenHome", "F:\\")
	appPreferences.RouterUsername = thisApp.Preferences().StringWithFallback("RouterUsername", "")
	appPreferences.RouterPassword = thisApp.Preferences().StringWithFallback("RouterPassword", "")
	zettlePath := widget.NewEntry()
	zettlePath.SetText(appPreferences.ZettlekastenHome)
	routerUser := widget.NewEntry()
	routerUser.SetText(appPreferences.RouterUsername)
	routerPass := widget.NewPasswordEntry()
	routerPass.SetText(appPreferences.RouterPassword)
	preferencesWindow.Resize(fyne.NewSize(400, 400))
	preferencesWindow.Hide()
	preferencesWindow.SetCloseIntercept(func() {
		preferencesWindow.Hide()
		// SavePreferences
		appPreferences.ZettlekastenHome = zettlePath.Text
		thisApp.Preferences().SetString("ZettlekastenHome", appPreferences.ZettlekastenHome)
		appPreferences.RouterUsername = routerUser.Text
		thisApp.Preferences().SetString("RouterUsername", appPreferences.RouterUsername)
		appPreferences.RouterPassword = routerPass.Text
		thisApp.Preferences().SetString("RouterPassword", appPreferences.RouterPassword)
	})
	preferencesWindow.SetContent(
		container.NewVBox(
			container.NewBorder(
				nil,
				nil,
				widget.NewLabel("Zettlekasten Path"),
				nil,
				zettlePath,
			),
			container.NewBorder(
				nil,
				nil,
				widget.NewLabel("Username"),
				nil,
				routerUser,
			),
			container.NewBorder(
				nil,
				nil,
				widget.NewLabel("Password"),
				nil,
				routerPass,
			),
		),
	)
}

func internetWindowSetup() {
	internetWindow.Resize(fyne.NewSize(430, 250))
	internetWindow.Hide()
	internetWindow.SetCloseIntercept(func() {
		internetWindow.Hide()
	})
	internetWindow.SetContent(
		container.NewGridWrap(
			fyne.NewSize(200, 50),
			widget.NewButton("Allow 15 mins", func() { tellModemToAllowForPeriod(time.Minute * 15) }),
			widget.NewButton("Allow 30 mins", func() { tellModemToAllowForPeriod(time.Minute * 35) }),
			widget.NewButton("Allow 1 hr", func() { tellModemToAllowForPeriod(time.Minute * 60) }),
			widget.NewButton("Allow +5 mins", func() { tellModemToAllowForPeriod(time.Minute * -5) }),
			widget.NewButton("Allow +15 mins", func() { tellModemToAllowForPeriod(time.Minute * -15) }),
			widget.NewButton("Allow +1 hr", func() { tellModemToAllowForPeriod(time.Minute * -60) }),
		),
	)
}

func getFileContentsAndCreateIfMissing(filename string) string {
	content, err := ioutil.ReadFile(filename)
	if errors.Is(err, os.ErrNotExist) {
		content = []byte(fmt.Sprintf("---\nDate: %s\n...\n", appStatus.CurrentZettleDBDate.Local().Format("2006-01-02")))
		os.WriteFile(filename, content, 0666)
	}
	return string(content)
}

func writeFileContents(filename string, content string) {
	err := os.WriteFile(filename, []byte(content), 0666)
	if err != nil {
		fmt.Printf("Failed to save\n%s\n", err)
	}
}

func zettleFileName(date time.Time) string {
	return fmt.Sprintf("%s-retro.markdown", date.Local().Format("20060102"))
}
