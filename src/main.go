package main

import (
	"bytes"
	"errors"
	"fmt"
	"image/color"
	"io/ioutil"
	"log"
	"os"
	"path"
	"runtime"
	"strings"
	"time"

	icon "vonexplaino.com/m/v2/helm/icon"

	fyne "fyne.io/fyne/v2"
	app "fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/pkg/browser"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

/**
* This is a systray item for:
*   - Markdown Daily status
*   - Internet Parent control
**/

type AppStatusStruct struct {
	CurrentZettleDBDate time.Time
	CurrentZettleDKB    binding.String
	TaskList            binding.StringList
	TasksFromGSM        [][]string
}

type AppPreferences struct {
	ZettlekastenHome string
	RouterUsername   string
	RouterPassword   string
	MSAccessToken    string
	MSRefreshToken   string
	MSExpiresAt      time.Time
}

var activeInternetTimeChan (chan time.Duration)
var thisApp fyne.App
var mainWindow fyne.Window
var preferencesWindow fyne.Window
var internetWindow fyne.Window
var taskWindow fyne.Window
var markdownInput *widget.Entry
var AppStatus AppStatusStruct
var appPreferences AppPreferences
var priorityColours map[string]color.Color

func setup() {
	os.Setenv("TZ", "Australia/Brisbane")
	priorityColours = map[string]color.Color{
		"1": color.NRGBA{R: 255, B: 0, G: 0, A: 255},
		"2": color.NRGBA{R: 255, B: 125, G: 125, A: 255},
		"3": color.NRGBA{R: 0, B: 255, G: 255, A: 255},
		"4": color.NRGBA{R: 0, B: 125, G: 255, A: 255},
		"5": color.NRGBA{R: 0, B: 0, G: 255, A: 255},
	}
	AppStatus = AppStatusStruct{
		CurrentZettleDBDate: time.Now().Local(),
		CurrentZettleDKB:    binding.NewString(),
		TaskList:            binding.NewStringList(),
	}
}
func main() {
	setup()
	// Get initial statuses for things
	activeInternetTimeChan = make(chan time.Duration, 10)
	AppStatus.CurrentZettleDKB.Set(zettleFileName(time.Now().Local()))
	go waitingForInternetCommand()

	thisApp = app.NewWithID("com.vonexplaino.helm.preferences")
	thisApp.SetIcon(fyne.NewStaticResource("Systray", icon.Data))
	preferencesWindow = thisApp.NewWindow("Preferences")
	preferencesWindowSetup()
	internetWindow = thisApp.NewWindow("Internet Control")
	internetWindowSetup()
	mainWindow = thisApp.NewWindow("Markdown Daily Knowledgebase")
	markdownWindowSetup()
	taskWindow = thisApp.NewWindow("Tasks")
	taskWindowSetup()
	// TASKS
	GetAllTasks()
	if desk, ok := thisApp.(desktop.App); ok {
		m := fyne.NewMenu("MyApp",
			fyne.NewMenuItem("Todays Notes", func() {
				mainWindow.Show()
				// Reload from file
				x, _ := AppStatus.CurrentZettleDKB.Get()
				markdownInput.Text = getFileContentsAndCreateIfMissing(path.Join(appPreferences.ZettlekastenHome, x))
				markdownInput.Refresh()
			}),
			fyne.NewMenuItem("Internet control", func() {
				if runtime.GOOS == "windows" {
					internetWindow.Show()
				}
			}),
			fyne.NewMenuItem("Tasks", func() {
				taskWindow.Show()
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
	AppStatus.CurrentZettleDBDate = AppStatus.CurrentZettleDBDate.Add(time.Hour * hours)
	AppStatus.CurrentZettleDKB.Set(zettleFileName(AppStatus.CurrentZettleDBDate))
	x, _ := AppStatus.CurrentZettleDKB.Get()
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
	fmt.Printf("Today is %s\n", todayString)
	for i := 0; i < totalDays; i++ {
		mike := thisDay
		bg := canvas.NewRectangle(color.NRGBA{R: 220, G: 220, B: 220, A: 0})
		if thisDay.Format("01/02/2006") == todayString {
			bg = canvas.NewRectangle(color.NRGBA{R: 100, G: 200, B: 150, A: 255})
		}
		days = append(days, container.NewMax(bg, widget.NewButton(fmt.Sprintf("%d", thisDay.Day()), func() {
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
	dateToShow := AppStatus.CurrentZettleDBDate
	var deepdeep dialog.Dialog
	deepdeep = dialog.NewCustom(
		"Change date",
		"Nevermind",
		createDatePicker(dateToShow, &deepdeep),
		mainWindow,
	)
	searchEntry := widget.NewEntry()
	menu := container.NewBorder(
		nil,
		container.NewBorder(
			nil,
			nil,
			nil,
			widget.NewButton(
				"Search",
				func() {
					finds, err := searchFiles(
						appPreferences.ZettlekastenHome,
						searchEntry.Text,
					)
					if err == nil {
						var selectFile dialog.Dialog
						fileList := widget.NewList(
							func() int {
								return len(finds)
							},
							func() fyne.CanvasObject {
								return widget.NewButton("FoundMe", func() {})
							},
							func(i widget.ListItemID, o fyne.CanvasObject) {
								o.(*widget.Button).SetText(finds[i])
								o.(*widget.Button).OnTapped = func() {
									// Save and Load
									x, _ := AppStatus.CurrentZettleDKB.Get()
									saveZettle(markdownInput.Text, x)
									AppStatus.CurrentZettleDBDate, _ = time.Parse("20060102", o.(*widget.Button).Text[0:8])
									AppStatus.CurrentZettleDKB.Set(zettleFileName(AppStatus.CurrentZettleDBDate))
									x, _ = AppStatus.CurrentZettleDKB.Get()
									markdownInput.Text = getFileContentsAndCreateIfMissing(path.Join(appPreferences.ZettlekastenHome, x))
									markdownInput.Refresh()
									selectFile.Hide()
								}
							},
						)
						selectFile = dialog.NewCustom(
							fmt.Sprintf("Found %d", len(finds)),
							"Nevermind",
							container.NewMax(
								fileList,
							),
							mainWindow,
						)
						selectFile.Show()
						fileList.Resize(fileList.MinSize().AddWidthHeight(100, 400))
						selectFile.Resize(selectFile.MinSize().AddWidthHeight(100, 400))
					} else {
						dialog.ShowInformation(
							"Search failed",
							err.Error(),
							mainWindow,
						)
					}
					fmt.Printf("Finds: %s\nError: %s\n", finds, err)
				},
			),
			searchEntry,
		),
		container.NewHBox(
			widget.NewButtonWithIcon("", theme.NavigateBackIcon(), func() {
				x, _ := AppStatus.CurrentZettleDKB.Get()
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
				} else {
					var mep string
					if markdownInput.Text[0:3] == "---" {
						mep = strings.Split(markdownInput.Text, "...")[1]
					} else {
						mep = markdownInput.Text
					}
					md := goldmark.New(
						goldmark.WithExtensions(extension.GFM),
						goldmark.WithParserOptions(
							parser.WithAutoHeadingID(),
						),
						goldmark.WithRendererOptions(
							html.WithHardWraps(),
							html.WithXHTML(),
						),
					)
					var buf bytes.Buffer
					if err := md.Convert([]byte(mep), &buf); err != nil {
						panic(err)
					}
					tmpFile, _ := ioutil.TempFile(os.TempDir(), "markdownpreview-*.html")
					defer os.Remove(tmpFile.Name())
					tmpFile.Write(buf.Bytes())
					tmpFile.Close()
					browser.OpenFile(tmpFile.Name())
					time.Sleep(time.Second * 2)
				}
			}),
			widget.NewButtonWithIcon("", theme.NavigateNextIcon(), func() {
				x, _ := AppStatus.CurrentZettleDKB.Get()
				saveZettle(markdownInput.Text, x)
				markdownInput.Text = moveZettleDate(24)
				markdownInput.Refresh()
			}),
		),
		widget.NewButton("Save", func() {
			x, _ := AppStatus.CurrentZettleDKB.Get()
			writeFileContents(path.Join(appPreferences.ZettlekastenHome, x), markdownInput.Text)
		}),
	)
	content := container.NewBorder(menu, widget.NewLabelWithData(AppStatus.CurrentZettleDKB), nil, nil, markdownInput)
	mainWindow.SetContent(content)
	mainWindow.SetCloseIntercept(func() {
		mainWindow.Hide()
		// Save contents
		x, _ := AppStatus.CurrentZettleDKB.Get()
		saveZettle(markdownInput.Text, x)
	})
}

func preferencesWindowSetup() {
	stringDateFormat := "20060102T15:04:05"
	appPreferences = AppPreferences{}
	appPreferences.ZettlekastenHome = thisApp.Preferences().StringWithFallback("ZettlekastenHome", "F:\\")
	appPreferences.RouterUsername = thisApp.Preferences().StringWithFallback("RouterUsername", "")
	appPreferences.RouterPassword = thisApp.Preferences().StringWithFallback("RouterPassword", "")
	appPreferences.MSAccessToken = thisApp.Preferences().StringWithFallback("MSAccessToken", "")
	appPreferences.MSRefreshToken = thisApp.Preferences().StringWithFallback("MSRefreshToken", "")
	fmt.Printf("%s\n", thisApp.Preferences().String("MSExpiresAt"))
	var e error
	appPreferences.MSExpiresAt, e = time.Parse(stringDateFormat, thisApp.Preferences().StringWithFallback("MSExpiresAt", "20060102T15:04:05"))
	if e != nil {
		log.Fatalf("Nope %s\n", e)
	}
	fmt.Printf("%s\n", appPreferences.MSExpiresAt.Local())

	zettlePath := widget.NewEntry()
	zettlePath.SetText(appPreferences.ZettlekastenHome)
	routerUser := widget.NewEntry()
	routerUser.SetText(appPreferences.RouterUsername)
	routerPass := widget.NewPasswordEntry()
	routerPass.SetText(appPreferences.RouterPassword)
	accessToken := widget.NewEntry()
	accessToken.SetText(appPreferences.MSAccessToken)
	refreshToken := widget.NewEntry()
	refreshToken.SetText(appPreferences.MSRefreshToken)
	expiresAt := widget.NewEntry()
	expiresAt.SetText(appPreferences.MSExpiresAt.Local().Format(stringDateFormat))

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
		appPreferences.MSAccessToken = accessToken.Text
		thisApp.Preferences().SetString("MSAccessToken", appPreferences.MSAccessToken)
		appPreferences.MSRefreshToken = refreshToken.Text
		thisApp.Preferences().SetString("MSRefreshToken", appPreferences.MSRefreshToken)
		appPreferences.MSExpiresAt, _ = time.Parse("20060102T15:04:05", expiresAt.Text)
		fmt.Printf("New: %s\n", appPreferences.MSExpiresAt.Local())
		thisApp.Preferences().SetString("MSExpiresAt", appPreferences.MSExpiresAt.Format(stringDateFormat))
		fmt.Printf("New: %s\n", thisApp.Preferences().String("MSExpiresAt"))
	})
	preferencesWindow.SetContent(
		container.New(
			layout.NewFormLayout(),
			widget.NewLabel("Zettlekasten Path"),
			zettlePath,
			widget.NewLabel("Username"),
			routerUser,
			widget.NewLabel("Password"),
			routerPass,
			widget.NewLabel("MS Access Token"),
			accessToken,
			widget.NewLabel("MS Refresh Token"),
			refreshToken,
			widget.NewLabel("MS Expires At"),
			expiresAt,
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
		content = []byte(fmt.Sprintf("---\nDate: %s\n...\n", AppStatus.CurrentZettleDBDate.Local().Format("2006-01-02")))
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

func taskWindowSetup() {
	taskWindow.Resize(fyne.NewSize(430, 250))
	taskWindow.Hide()
	taskWindow.SetCloseIntercept(func() {
		taskWindow.Hide()
	})
}

func taskWindowRefresh() {
	var list fyne.CanvasObject
	if len(AppStatus.TasksFromGSM) == 0 {
		list = widget.NewLabel("No tasks")
	} else {
		/*
			col1 := container.NewVBox(widget.NewRichTextFromMarkdown(`## Incident`))
			col2 := container.NewVBox(widget.NewRichTextFromMarkdown(`## Task`))
			col3 := container.NewVBox(widget.NewRichTextFromMarkdown(`## Age`))
			col4 := container.NewVBox(widget.NewRichTextFromMarkdown(`## Priority`))
			col5 := container.NewVBox(widget.NewRichTextFromMarkdown(`## Status`))
			for _, x := range AppStatus.TasksFromGSM[1:] {
				col1.Objects = append(col1.Objects, widget.NewLabelWithStyle(fmt.Sprintf("[%s] %s", x[1], x[2]), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
				col2.Objects = append(col2.Objects, widget.NewLabel(fmt.Sprintf("[%s] %s", x[8], x[3])))
				dt, _ := time.Parse("1/2/2006 3:04:05 PM", x[0])
				col3.Objects = append(col3.Objects, widget.NewLabel(dateSinceNowInString(dt)))
				col4.Objects = append(col4.Objects, container.NewMax(
					canvas.NewRectangle(priorityColours[x[9]]),
					container.NewPadded(widget.NewLabel("Priority: "+x[9]))))
				col5.Objects = append(col5.Objects, widget.NewLabel(x[4]))
			}
			list = container.NewVScroll(
				container.NewHBox(
					col1,
					col2,
					col3,
					col4,
					col5,
				),
			)
		*/
		stuff := []fyne.CanvasObject{}
		for _, row := range AppStatus.TasksFromGSM {
			dt, _ := time.Parse("1/2/2006 3:04:05 PM", row[0])
			crd := widget.NewCard(
				fmt.Sprintf("[%s] %s", row[1], row[2]),
				fmt.Sprintf("[%s] %s", row[8], row[3]),
				container.NewVBox(
					container.NewGridWithColumns(2, widget.NewLabel(dateSinceNowInString(dt)), widget.NewLabel(row[4])),
					container.NewHBox(
						widget.NewButton("Visit", func() {}),
						widget.NewButton("Reassign", func() {}),
						widget.NewButton("Status", func() {}),
						widget.NewButton("Journals", func() {}))),
			)
			stuff = append(
				stuff,
				crd,
			)
		}
		list = container.NewGridWrap(fyne.NewSize(300, 200), stuff...)
	}

	taskWindow.SetContent(list)
	taskWindow.Content().Refresh()
}

func gsmTaskToRow(row []string) fyne.CanvasObject {
	x, _ := time.Parse("1/2/2006 3:04:05 PM", row[0])
	header := widget.NewLabelWithStyle(fmt.Sprintf("[%s] %s", row[1], row[2]), fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	header.Wrapping = fyne.TextWrapWord
	block := container.NewMax(
		canvas.NewRectangle(color.Black),
		container.NewPadded(
			canvas.NewRectangle(color.White),
			container.NewVBox(
				container.NewBorder(
					nil,
					nil,
					nil,
					container.NewMax(
						canvas.NewRectangle(priorityColours[row[9]]),
						container.NewPadded(widget.NewLabel("Priority: "+row[9]))),
					header,
				),
				widget.NewLabel(fmt.Sprintf("[%s] %s", row[8], row[3])),
				container.NewGridWithColumns(2, widget.NewLabel(dateSinceNowInString(x)), widget.NewLabel(row[4])),
				container.NewHBox(
					widget.NewButton("Visit", func() {}),
					widget.NewButton("Reassign", func() {}),
					widget.NewButton("Status", func() {}),
					widget.NewButton("Journals", func() {})),
			),
		),
	)
	return block
}

func dateSinceNowInString(oldDate time.Time) string {
	bob := time.Since(oldDate)
	mep := bob.Seconds()
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
	if mep > 1 {
		metric = metric + "s"
	}
	return fmt.Sprintf("%d %s", int(mep), metric)
}
