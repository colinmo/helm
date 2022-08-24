package main

import (
	"bytes"
	"errors"
	"fmt"
	"image/color"
	"math"
	"os"
	"os/user"
	"path"
	"path/filepath"
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
	CurrentZettleDBDate    time.Time
	CurrentZettleDKB       binding.String
	GSMGettingToken        bool
	MSGettingToken         bool
	MyTasksFromGSM         [][]string
	MyIncidentsFromGSM     [][]string
	MyRequestsInGSM        [][]string
	MyTeamIncidentsFromGSM [][]string
	TaskTaskCount          int
	TaskTaskStatus         binding.String
	MyTasksFromPlanner     [][]string
	MyTasksFromJira        [][]string
}

type AppPreferences struct {
	ZettlekastenHome string
	RouterUsername   string
	RouterPassword   string
	//
	MSPlannerActive bool
	MSAccessToken   string
	MSRefreshToken  string
	MSExpiresAt     time.Time
	MSGroups        string
	//
	CWActive       bool
	CWAccessToken  string
	CWRefreshToken string
	CWExpiresAt    time.Time
	//
	PriorityOverride string
	//
	JiraActive   bool
	JiraUsername string
	JiraKey      string
}

var thisApp fyne.App
var mainWindow fyne.Window
var preferencesWindow fyne.Window

// var internetWindow fyne.Window
var taskWindow fyne.Window
var markdownInput *widget.Entry
var AppStatus AppStatusStruct
var TaskTabsIndexes map[string]int
var TaskTabs *container.AppTabs
var appPreferences AppPreferences

func setup() {
	os.Setenv("TZ", "Australia/Brisbane")
	AppStatus = AppStatusStruct{
		CurrentZettleDBDate: time.Now().Local(),
		CurrentZettleDKB:    binding.NewString(),
		TaskTaskStatus:      binding.NewString(),
		TaskTaskCount:       0,
		GSMGettingToken:     false,
		MSGettingToken:      false,
	}
	AppStatus.CurrentZettleDKB.Set(zettleFileName(time.Now().Local()))
	CWFields.Task.OwnerID = "93cfd5a4e1d0ba5d3423e247b08dfd1286cae772cf"
	CWFields.Task.CreatedDateTime = "9355d5ed416bbc9408615c4145978ff8538a3f6eb4"
	CWFields.Task.TaskTitle = "93ad98a2d68a61778eda3d4d9cbb30acbfd458aea4"
	CWFields.Task.TaskStatus = "9368f0fb7b744108a666984c21afc932562eb7dc16"
	CWFields.Task.TaskID = "93d5409c4bcbf7a38ed75a47dd92671f374236fa32"
	CWFields.Task.IncidentID = "BO:6dd53665c0c24cab86870a21cf6434ae,FI:6ae282c55e8e4266ae66ffc070c17fa3,RE:93694ed12e2e9bb908131846b7a9c67ec72b811676"
	CWFields.Task.IncidentShortDesc = "BO:6dd53665c0c24cab86870a21cf6434ae,FI:93e8ea93ff67fd95118255419690a50ef2d56f910c,RE:93694ed12e2e9bb908131846b7a9c67ec72b811676"
	CWFields.Task.IncidentPriority = "BO:6dd53665c0c24cab86870a21cf6434ae,FI:83c36313e97b4e6b9028aff3b401b71c,RE:93694ed12e2e9bb908131846b7a9c67ec72b811676"
	CWFields.Incident.OwnerID = "9339fc404e39ae705648ab43969f29262e6d167606"
	CWFields.Incident.Status = "5eb3234ae1344c64a19819eda437f18d"
	CWFields.Incident.CreatedDateTime = "c1e86f31eb2c4c5f8e8615a5189e9b19"
	CWFields.Incident.IncidentID = "6ae282c55e8e4266ae66ffc070c17fa3"
	CWFields.Incident.ShortDesc = "93e8ea93ff67fd95118255419690a50ef2d56f910c"
	CWFields.Incident.Priority = "83c36313e97b4e6b9028aff3b401b71c"
	CWFields.Incident.RequestorSNumber = "941aa0889094428a6f4c054dbea345b09b4d87c77e"
	CWFields.Incident.TeamID = "9339fc404e312b6d43436041fc8af1c07c6197f559"
	CWFields.Incident.OwnerName = "9339fc404e4c93350bf5be446fb13d693b0bb7f219"
}
func overrides() {
	// Priority Overrides
	myself, error := user.Current()
	pribase := ""
	if error == nil {
		pribase = filepath.Join(myself.HomeDir, "/.helm")
	} else {
		pribase = filepath.Join(os.TempDir(), "/.helm")
	}
	appPreferences.PriorityOverride = thisApp.Preferences().StringWithFallback("PriorityOverride", pribase)
	thisApp.Preferences().SetString("PriorityOverride", appPreferences.PriorityOverride)
	loadPriorityOverride()
	startLocalServers()
	browser.OpenURL(`https://serviceportal.griffith.edu.au/cherwellapi/saml/login.cshtml?finalUri=http://localhost:84/cherwell?code=xx`)
	go singleThreadReturnOrGetGSMAccessToken()
	//	activeInternetTimeChan = make(chan time.Duration, 10)
	//	go waitingForInternetCommand()
}
func main() {
	setup()
	thisApp = app.NewWithID("com.vonexplaino.helm.preferences")
	thisApp.SetIcon(fyne.NewStaticResource("Systray", icon.Data))
	overrides()
	preferencesWindow = thisApp.NewWindow("Preferences")
	preferencesWindowSetup()
	//	internetWindow = thisApp.NewWindow("Internet Control")
	//	internetWindowSetup()
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
			//			fyne.NewMenuItem("Internet control", func() {
			//				if runtime.GOOS == "windows" {
			//					internetWindow.Show()
			//				}
			//			}),
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
					tmpFile, _ := os.CreateTemp(os.TempDir(), "markdownpreview-*.html")
					defer os.Remove(tmpFile.Name())
					tmpFile.Write([]byte(markdownHTMLHeader))
					tmpFile.Write(buf.Bytes())
					tmpFile.Write([]byte(markdownHTMLFooter))
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
	appPreferences.ZettlekastenHome = thisApp.Preferences().StringWithFallback("ZettlekastenHome", os.TempDir())
	//	appPreferences.RouterUsername = thisApp.Preferences().StringWithFallback("RouterUsername", "")
	//	appPreferences.RouterPassword = thisApp.Preferences().StringWithFallback("RouterPassword", "")
	appPreferences.MSPlannerActive = thisApp.Preferences().BoolWithFallback("MSPlannerActive", false)
	appPreferences.MSAccessToken = thisApp.Preferences().StringWithFallback("MSAccessToken", "")
	appPreferences.MSRefreshToken = thisApp.Preferences().StringWithFallback("MSRefreshToken", "")
	appPreferences.MSExpiresAt, _ = time.Parse(stringDateFormat, thisApp.Preferences().StringWithFallback("MSExpiresAt", "20060102T15:04:05"))
	appPreferences.MSGroups = thisApp.Preferences().StringWithFallback("MSGroups", "")
	appPreferences.PriorityOverride = thisApp.Preferences().String("PriorityOverride")
	appPreferences.JiraActive = thisApp.Preferences().BoolWithFallback("JiraActive", false)
	appPreferences.JiraKey = thisApp.Preferences().StringWithFallback("JiraKey", "")
	appPreferences.JiraUsername = thisApp.Preferences().StringWithFallback("JiraUsername", "")

	zettlePath := widget.NewEntry()
	zettlePath.SetText(appPreferences.ZettlekastenHome)
	//	routerUser := widget.NewEntry()
	//	routerUser.SetText(appPreferences.RouterUsername)
	//	routerPass := widget.NewPasswordEntry()
	//	routerPass.SetText(appPreferences.RouterPassword)
	plannerActive := widget.NewCheck("Active", func(res bool) {})
	plannerActive.SetChecked(appPreferences.MSPlannerActive)
	accessToken := widget.NewEntry()
	accessToken.SetText(appPreferences.MSAccessToken)
	refreshToken := widget.NewEntry()
	refreshToken.SetText(appPreferences.MSRefreshToken)
	expiresAt := widget.NewEntry()
	expiresAt.SetText(appPreferences.MSExpiresAt.Local().Format(stringDateFormat))
	groupsList := widget.NewEntry()
	groupsList.SetText(appPreferences.MSGroups)
	priorityOverride := widget.NewEntry()
	priorityOverride.SetText(appPreferences.PriorityOverride)
	jiraActive := widget.NewCheck("Active", func(res bool) {})
	jiraActive.SetChecked(appPreferences.JiraActive)
	jiraKey := widget.NewPasswordEntry()
	jiraKey.SetText(appPreferences.JiraKey)
	jiraUsername := widget.NewEntry()
	jiraUsername.SetText(appPreferences.JiraUsername)

	preferencesWindow.Resize(fyne.NewSize(400, 400))
	preferencesWindow.Hide()
	preferencesWindow.SetCloseIntercept(func() {
		preferencesWindow.Hide()
		// SavePreferences
		appPreferences.ZettlekastenHome = zettlePath.Text
		thisApp.Preferences().SetString("ZettlekastenHome", appPreferences.ZettlekastenHome)
		//		appPreferences.RouterUsername = routerUser.Text
		//		thisApp.Preferences().SetString("RouterUsername", appPreferences.RouterUsername)
		//		appPreferences.RouterPassword = routerPass.Text
		//		thisApp.Preferences().SetString("RouterPassword", appPreferences.RouterPassword)
		appPreferences.MSPlannerActive = plannerActive.Checked
		thisApp.Preferences().SetBool("MSPlannerActive", appPreferences.MSPlannerActive)
		appPreferences.MSAccessToken = accessToken.Text
		thisApp.Preferences().SetString("MSAccessToken", appPreferences.MSAccessToken)
		appPreferences.MSRefreshToken = refreshToken.Text
		thisApp.Preferences().SetString("MSRefreshToken", appPreferences.MSRefreshToken)
		appPreferences.MSExpiresAt, _ = time.Parse("20060102T15:04:05", expiresAt.Text)
		thisApp.Preferences().SetString("MSExpiresAt", appPreferences.MSExpiresAt.Format(stringDateFormat))
		appPreferences.MSGroups = groupsList.Text
		thisApp.Preferences().SetString("MSGroups", appPreferences.MSGroups)
		appPreferences.PriorityOverride = priorityOverride.Text
		thisApp.Preferences().SetString("PriorityOverride", appPreferences.PriorityOverride)
		appPreferences.JiraActive = jiraActive.Checked
		thisApp.Preferences().SetBool("JiraActive", appPreferences.JiraActive)
		appPreferences.JiraKey = jiraKey.Text
		thisApp.Preferences().SetString("JiraKey", appPreferences.JiraKey)
		appPreferences.JiraUsername = jiraUsername.Text
		thisApp.Preferences().SetString("JiraUsername", appPreferences.JiraUsername)
	})
	preferencesWindow.SetContent(
		container.New(
			layout.NewFormLayout(),
			widget.NewLabel("Zettlekasten Path"),
			zettlePath,
			//			widget.NewLabel("Username"),
			//			routerUser,
			//			widget.NewLabel("Password"),
			//			routerPass,
			widget.NewLabel("Planner active"),
			plannerActive,
			widget.NewLabel("MS Access Token"),
			accessToken,
			widget.NewLabel("MS Refresh Token"),
			refreshToken,
			widget.NewLabel("MS Expires At"),
			expiresAt,
			widget.NewLabel("Store priorities"),
			priorityOverride,
			widget.NewLabel("Jira active"),
			jiraActive,
			widget.NewLabel("Jira Key"),
			jiraKey,
			widget.NewLabel("Jira Username"),
			jiraUsername,
		),
	)
}

func getFileContentsAndCreateIfMissing(filename string) string {
	content, err := os.ReadFile(filename)
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
	taskWindow.Resize(fyne.NewSize(430, 550))
	taskWindow.Hide()
	taskStatusWidget := widget.NewLabelWithData(AppStatus.TaskTaskStatus)
	TaskTabs = container.NewAppTabs(
		container.NewTabItem("My Tasks", container.NewBorder(
			widget.NewToolbar(
				widget.NewToolbarAction(
					theme.ViewRefreshIcon(),
					func() {
						DownloadTasks()
						taskWindowRefresh("CWTasks")
					},
				),
				widget.NewToolbarSeparator(),
				widget.NewToolbarAction(
					theme.HistoryIcon(),
					func() {},
				),
				widget.NewToolbarAction(
					theme.ErrorIcon(),
					func() {},
				),
			),
			nil,
			nil,
			nil,
			container.NewWithoutLayout(),
		)),
		container.NewTabItem("My Incidents", container.NewBorder(
			widget.NewToolbar(
				widget.NewToolbarAction(
					theme.ViewRefreshIcon(),
					func() {
						DownloadIncidents()
						taskWindowRefresh("CWIncidents")
					},
				),
				widget.NewToolbarSeparator(),
				widget.NewToolbarAction(
					theme.HistoryIcon(),
					func() {},
				),
				widget.NewToolbarAction(
					theme.ErrorIcon(),
					func() {},
				),
			),
			nil,
			nil,
			nil,
			container.NewWithoutLayout(),
		)),
		container.NewTabItem("My Requests", container.NewBorder(
			widget.NewToolbar(
				widget.NewToolbarAction(
					theme.ViewRefreshIcon(),
					func() {
						DownloadTeam()
						taskWindowRefresh("CWTeamIncidents")
					},
				),
				widget.NewToolbarSeparator(),
				widget.NewToolbarAction(
					theme.HistoryIcon(),
					func() {},
				),
				widget.NewToolbarAction(
					theme.ErrorIcon(),
					func() {},
				),
			),
			nil,
			nil,
			nil,
			container.NewWithoutLayout(),
		)),
		container.NewTabItem("My Team Incidents", container.NewBorder(
			widget.NewToolbar(
				widget.NewToolbarAction(
					theme.ViewRefreshIcon(),
					func() {
						DownloadMyRequests()
						taskWindowRefresh("CWRequests")
					},
				),
				widget.NewToolbarSeparator(),
				widget.NewToolbarAction(
					theme.HistoryIcon(),
					func() {},
				),
				widget.NewToolbarAction(
					theme.ErrorIcon(),
					func() {},
				),
			),
			nil,
			nil,
			nil,
			container.NewWithoutLayout(),
		)),
	)
	TaskTabsIndexes = map[string]int{
		"CWTasks":         0,
		"CWIncidents":     1,
		"CWRequests":      2,
		"CWTeamIncidents": 3,
	}
	if appPreferences.MSPlannerActive {
		TaskTabsIndexes["MSPlanner"] = len(TaskTabsIndexes)
		TaskTabs.Append(
			container.NewTabItem("My Planner", container.NewBorder(
				widget.NewToolbar(
					widget.NewToolbarAction(
						theme.ViewRefreshIcon(),
						func() {
							DownloadPlanners()
							taskWindowRefresh("MSPlanner")
						},
					),
					widget.NewToolbarSeparator(),
					widget.NewToolbarAction(
						theme.HistoryIcon(),
						func() {},
					),
					widget.NewToolbarAction(
						theme.ErrorIcon(),
						func() {},
					),
				),
				nil,
				nil,
				nil,
				container.NewWithoutLayout(),
			)),
		)
	}
	if appPreferences.JiraActive {
		TaskTabsIndexes["Jira"] = len(TaskTabsIndexes)
		TaskTabs.Append(
			container.NewTabItem("My Jira", container.NewBorder(
				widget.NewToolbar(
					widget.NewToolbarAction(
						theme.ViewRefreshIcon(),
						func() {
							GetJira()
						},
					),
					widget.NewToolbarSeparator(),
					widget.NewToolbarAction(
						theme.HistoryIcon(),
						func() {},
					),
					widget.NewToolbarAction(
						theme.ErrorIcon(),
						func() {},
					),
				),
				nil,
				nil,
				nil,
				container.NewWithoutLayout(),
			)),
		)
	}
	taskWindow.SetContent(
		container.NewBorder(
			nil,
			container.NewMax(taskStatusWidget),
			nil,
			nil,
			TaskTabs,
		),
	)
	taskWindow.SetCloseIntercept(func() {
		taskWindow.Hide()
	})
	taskWindow.Content().Refresh()
}

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

func newTappableLabel(textLabel string) *tappableLabel {
	label := &tappableLabel{}
	label.ExtendBaseWidget(label)
	label.SetText(textLabel)
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

func activeTaskStatusUpdate(by int) {
	AppStatus.TaskTaskCount = int(math.Max(float64(AppStatus.TaskTaskCount+by), 0))
	if AppStatus.TaskTaskCount == 0 {
		AppStatus.TaskTaskStatus.Set("Idle")
	} else {
		AppStatus.TaskTaskStatus.Set(fmt.Sprintf("%d tasks underway", AppStatus.TaskTaskCount))
	}
}

func taskWindowRefresh(specific string) {
	var list fyne.CanvasObject

	fmt.Printf("Refreshing\n")
	priorityIcons := setupPriorityIcons()
	if specific == "" || specific == "CWTasks" {
		if len(AppStatus.MyTasksFromGSM) == 0 {
			list = widget.NewLabel("No tasks")
		} else {
			col0 := container.NewVBox(widget.NewRichTextFromMarkdown(`### `))
			col1 := container.NewVBox(widget.NewRichTextFromMarkdown(`### Incident`))
			col2 := container.NewVBox(widget.NewRichTextFromMarkdown(`### Task`))
			col3 := container.NewVBox(widget.NewRichTextFromMarkdown(`### Age`))
			col4 := container.NewVBox(widget.NewRichTextFromMarkdown(`### Priority`))
			col5 := container.NewVBox(widget.NewRichTextFromMarkdown(`### Status`))

			for _, x := range AppStatus.MyTasksFromGSM {
				thisID := x[1]
				myPriority := x[6]
				if len(x) >= 8 && x[6] != x[7] {
					myPriority = x[6] + "(" + x[7] + ")"
				}
				tempVar := ""
				col0.Objects = append(
					col0.Objects,
					container.NewMax(
						widget.NewLabel(""),
						newTappableIcon(theme.InfoIcon(), func(_ *fyne.PointEvent) {
							browser.OpenURL("https://griffith.cherwellondemand.com/CherwellClient/Access/incident/" + thisID)
						}),
					))
				col1.Objects = append(col1.Objects,
					widget.NewLabelWithStyle(fmt.Sprintf("[%s] %s", x[1], x[2]), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
				col2.Objects = append(col2.Objects, newTappableLabel(fmt.Sprintf("[%s] %s", x[5], x[3])))
				dt, _ := time.Parse("1/2/2006 3:04:05 PM", x[0])
				col3.Objects = append(col3.Objects, widget.NewLabel(dateSinceNowInString(dt)))
				tempFunc := func(_ *fyne.PointEvent) {
					dialog.ShowForm(
						"Priority override "+thisID,
						"Override",
						"Cancel",
						[]*widget.FormItem{
							widget.NewFormItem(
								"Priority",
								widget.NewSelect(
									[]string{"1", "2", "3", "4", "5"},
									func(changed string) {
										tempVar = changed
									},
								)),
						},
						func(isit bool) {
							if tempVar == x[6] || tempVar == "" {
								delete(priorityOverrides.CWIncidents, thisID)
							} else {
								priorityOverrides.CWIncidents[thisID] = tempVar
							}
							savePriorityOverride()
						},
						taskWindow,
					)
				}
				col4.Objects = append(col4.Objects, container.NewMax(
					priorityIcons[x[6]],
					newTappableLabelWithStyle(
						myPriority,
						fyne.TextAlignCenter,
						fyne.TextStyle{},
						tempFunc)))
				col5.Objects = append(col5.Objects, widget.NewLabel(x[4]))
			}
			list = container.NewVScroll(
				container.NewHBox(
					col0,
					col1,
					col2,
					col3,
					col4,
					col5,
				),
			)
		}
		TaskTabs.Items[TaskTabsIndexes["CWTasks"]].Content = container.NewBorder(
			widget.NewToolbar(
				widget.NewToolbarAction(
					theme.ViewRefreshIcon(),
					func() {
						DownloadTasks()
						taskWindowRefresh("CWTasks")
					},
				),
				widget.NewToolbarSeparator(),
				widget.NewToolbarAction(
					theme.HistoryIcon(),
					func() {},
				),
				widget.NewToolbarAction(
					theme.ErrorIcon(),
					func() {},
				),
			),
			nil,
			nil,
			nil,
			list,
		)
	}
	if specific == "" || specific == "CWIncidents" {
		var list2 fyne.CanvasObject
		if len(AppStatus.MyIncidentsFromGSM) == 0 {
			list2 = widget.NewLabel("No incidents")
		} else {
			col0 := container.NewVBox(widget.NewRichTextFromMarkdown(`### `))
			col1 := container.NewVBox(widget.NewRichTextFromMarkdown(`### Incident`))
			col3 := container.NewVBox(widget.NewRichTextFromMarkdown(`### Age`))
			col4 := container.NewVBox(widget.NewRichTextFromMarkdown(`### Priority`))
			col5 := container.NewVBox(widget.NewRichTextFromMarkdown(`### Status`))
			for _, x := range AppStatus.MyIncidentsFromGSM {
				thisID := x[1]
				col0.Objects = append(
					col0.Objects,
					container.NewMax(
						widget.NewLabel(""),
						newTappableIcon(theme.InfoIcon(), func(_ *fyne.PointEvent) {
							browser.OpenURL("https://griffith.cherwellondemand.com/CherwellClient/Access/incident/" + thisID)
						}),
					))
				col1.Objects = append(col1.Objects,
					widget.NewLabelWithStyle(fmt.Sprintf("[%s] %s", x[1], x[2]), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
				dt, _ := time.Parse("1/2/2006 3:04:05 PM", x[0])
				col3.Objects = append(col3.Objects, widget.NewLabel(dateSinceNowInString(dt)))
				col4.Objects = append(col4.Objects, container.NewMax(
					priorityIcons[x[4]], // canvas.NewCircle(priorityColours[x[4]]),
					widget.NewLabelWithStyle(x[4], fyne.TextAlignCenter, fyne.TextStyle{})))
				col5.Objects = append(col5.Objects, widget.NewLabel(x[3]))
			}
			list2 = container.NewVScroll(
				container.NewHBox(
					col0,
					col1,
					col3,
					col4,
					col5,
				),
			)
		}
		TaskTabs.Items[TaskTabsIndexes["CWIncidents"]].Content = container.NewBorder(
			widget.NewToolbar(
				widget.NewToolbarAction(
					theme.ViewRefreshIcon(),
					func() {
						DownloadIncidents()
						taskWindowRefresh("CWIncidents")
					},
				),
				widget.NewToolbarSeparator(),
				widget.NewToolbarAction(
					theme.HistoryIcon(),
					func() {},
				),
				widget.NewToolbarAction(
					theme.ErrorIcon(),
					func() {},
				),
			),
			nil,
			nil,
			nil,
			list2,
		)
	}
	if specific == "" || specific == "CWTeamIncidents" {
		var list3 fyne.CanvasObject
		if len(AppStatus.MyTeamIncidentsFromGSM) == 0 {
			list3 = widget.NewLabel("No incidents")
		} else {
			col0 := container.NewVBox(widget.NewRichTextFromMarkdown(`### `))
			col1 := container.NewVBox(widget.NewRichTextFromMarkdown(`### Incident`))
			col2 := container.NewVBox(widget.NewRichTextFromMarkdown(`### Owner`))
			col3 := container.NewVBox(widget.NewRichTextFromMarkdown(`### Age`))
			col4 := container.NewVBox(widget.NewRichTextFromMarkdown(`### Priority`))
			col5 := container.NewVBox(widget.NewRichTextFromMarkdown(`### Status`))
			for _, x := range AppStatus.MyTeamIncidentsFromGSM {
				thisID := x[1]
				col0.Objects = append(
					col0.Objects,
					container.NewMax(
						widget.NewLabel(""),
						newTappableIcon(theme.InfoIcon(), func(_ *fyne.PointEvent) {
							browser.OpenURL("https://griffith.cherwellondemand.com/CherwellClient/Access/incident/" + thisID)
						}),
					))
				if len(x) > 5 {
					col2.Objects = append(col2.Objects, widget.NewLabel(x[5]))
				} else {
					col2.Objects = append(col2.Objects, widget.NewLabel("none"))
				}
				col1.Objects = append(col1.Objects,
					widget.NewLabelWithStyle(fmt.Sprintf("[%s] %s", x[1], x[2]), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
				dt, _ := time.Parse("1/2/2006 3:04:05 PM", x[0])
				col3.Objects = append(col3.Objects, widget.NewLabel(dateSinceNowInString(dt)))
				col4.Objects = append(col4.Objects, container.NewMax(
					priorityIcons[x[4]],
					widget.NewLabelWithStyle(x[4], fyne.TextAlignCenter, fyne.TextStyle{})))
				col5.Objects = append(col5.Objects, widget.NewLabel(x[3]))
			}
			list3 = container.NewVScroll(
				container.NewHBox(
					col0,
					col1,
					col2,
					col3,
					col4,
					col5,
				),
			)
		}

		TaskTabs.Items[TaskTabsIndexes["CWTeamIncidents"]].Content = container.NewBorder(
			widget.NewToolbar(
				widget.NewToolbarAction(
					theme.ViewRefreshIcon(),
					func() {
						DownloadTeam()
						taskWindowRefresh("CWTeamIncidents")
					},
				),
				widget.NewToolbarSeparator(),
				widget.NewToolbarAction(
					theme.HistoryIcon(),
					func() {},
				),
				widget.NewToolbarAction(
					theme.ErrorIcon(),
					func() {},
				),
			),
			nil,
			nil,
			nil,
			list3,
		)
	}
	if specific == "" || specific == "CWRequests" {
		var list4 fyne.CanvasObject
		if len(AppStatus.MyRequestsInGSM) == 0 {
			list4 = widget.NewLabel("No requests")
		} else {
			col0 := container.NewVBox(widget.NewRichTextFromMarkdown(`### `))
			col1 := container.NewVBox(widget.NewRichTextFromMarkdown(`### Incident`))
			col3 := container.NewVBox(widget.NewRichTextFromMarkdown(`### Age`))
			col4 := container.NewVBox(widget.NewRichTextFromMarkdown(`### Priority`))
			col5 := container.NewVBox(widget.NewRichTextFromMarkdown(`### Status`))
			for _, x := range AppStatus.MyRequestsInGSM {
				thisID := x[1]
				col0.Objects = append(
					col0.Objects,
					container.NewMax(
						widget.NewLabel(""),
						newTappableIcon(theme.InfoIcon(), func(_ *fyne.PointEvent) {
							browser.OpenURL("https://griffith.cherwellondemand.com/CherwellClient/Access/incident/" + thisID)
						}),
					))
				col1.Objects = append(col1.Objects,
					widget.NewLabelWithStyle(fmt.Sprintf("[%s] %s", x[1], x[2]), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
				dt, _ := time.Parse("1/2/2006 3:04:05 PM", x[0])
				col3.Objects = append(col3.Objects, widget.NewLabel(dateSinceNowInString(dt)))
				col4.Objects = append(col4.Objects, container.NewMax(
					priorityIcons[x[4]],
					widget.NewLabelWithStyle(x[4], fyne.TextAlignCenter, fyne.TextStyle{})))
				col5.Objects = append(col5.Objects, widget.NewLabel(x[3]))
			}
			list4 = container.NewVScroll(
				container.NewHBox(
					col0,
					col1,
					col3,
					col4,
					col5,
				),
			)
		}

		TaskTabs.Items[TaskTabsIndexes["CWRequests"]].Content = container.NewBorder(
			widget.NewToolbar(
				widget.NewToolbarAction(
					theme.ViewRefreshIcon(),
					func() {
						DownloadMyRequests()
						taskWindowRefresh("CWRequests")
					},
				),
				widget.NewToolbarSeparator(),
				widget.NewToolbarAction(
					theme.HistoryIcon(),
					func() {},
				),
				widget.NewToolbarAction(
					theme.ErrorIcon(),
					func() {},
				),
			),
			nil,
			nil,
			nil,
			list4,
		)
	}
	if appPreferences.MSPlannerActive && (specific == "" || specific == "MSPlanner") {
		// MY PLANNER
		var list5 fyne.CanvasObject
		if len(AppStatus.MyTasksFromPlanner) == 0 {
			list5 = widget.NewLabel("No requests")
		} else {
			col0 := container.NewVBox(widget.NewRichTextFromMarkdown(`### `))
			col1 := container.NewVBox(widget.NewRichTextFromMarkdown(`### Title`))
			col3 := container.NewVBox(widget.NewRichTextFromMarkdown(`### Age`))
			col4 := container.NewVBox(widget.NewRichTextFromMarkdown(`### Priority`))
			col5 := container.NewVBox(widget.NewRichTextFromMarkdown(`### %`))
			for _, x := range AppStatus.MyTasksFromPlanner {
				thisID := x[0]
				col0.Objects = append(
					col0.Objects,
					container.NewMax(
						widget.NewLabel(""),
						newTappableIcon(theme.InfoIcon(), func(_ *fyne.PointEvent) {
							browser.OpenURL(
								fmt.Sprintf(
									"https://tasks.office.com/%s/Home/Task/%s",
									msApplicationTenant,
									thisID,
								),
							)
						}),
					))
				col1.Objects = append(col1.Objects,
					widget.NewLabelWithStyle(x[3], fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
				dt, _ := time.Parse("2006-01-02T15:04:05.999999999Z", x[5])
				col3.Objects = append(col3.Objects, widget.NewLabel(dateSinceNowInString(dt)))
				col4.Objects = append(col4.Objects, container.NewMax(
					priorityIcons[x[6]],
					widget.NewLabelWithStyle(x[6], fyne.TextAlignCenter, fyne.TextStyle{})))
				col5.Objects = append(col5.Objects, widget.NewLabel(x[8]))
			}
			list5 = container.NewVScroll(
				container.NewHBox(
					col0,
					col1,
					col3,
					col4,
					col5,
				),
			)
		}

		TaskTabs.Items[TaskTabsIndexes["MSPlanner"]].Content = container.NewBorder(
			widget.NewToolbar(
				widget.NewToolbarAction(
					theme.ViewRefreshIcon(),
					func() {
						DownloadPlanners()
						taskWindowRefresh("MSPlanner")
					},
				),
				widget.NewToolbarSeparator(),
				widget.NewToolbarAction(
					theme.HistoryIcon(),
					func() {},
				),
				widget.NewToolbarAction(
					theme.ErrorIcon(),
					func() {},
				),
			),
			nil,
			nil,
			nil,
			list5,
		)
	}
	if appPreferences.JiraActive && (specific == "" || specific == "Jira") {
		var list fyne.CanvasObject
		if len(AppStatus.MyTasksFromJira) == 0 {
			list = widget.NewLabel("No requests")
		} else {
			col0 := container.NewVBox(widget.NewRichTextFromMarkdown(`### `))
			col1 := container.NewVBox(widget.NewRichTextFromMarkdown(`### Title`))
			col3 := container.NewVBox(widget.NewRichTextFromMarkdown(`### Age`))
			col4 := container.NewVBox(widget.NewRichTextFromMarkdown(`### Priority`))
			col5 := container.NewVBox(widget.NewRichTextFromMarkdown(`### Status`))
			for _, x := range AppStatus.MyTasksFromJira {
				thisID := x[0]
				col0.Objects = append(
					col0.Objects,
					container.NewMax(
						widget.NewLabel(""),
						newTappableIcon(theme.InfoIcon(), func(_ *fyne.PointEvent) {
							browser.OpenURL(
								fmt.Sprintf(
									"https://griffith.atlassian.net/browse/%s",
									thisID,
								),
							)
						}),
					))
				col1.Objects = append(col1.Objects,
					widget.NewLabelWithStyle(x[1], fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
				dt, _ := time.Parse("2006-01-02T15:04:05.999-0700", x[2])
				col3.Objects = append(col3.Objects, widget.NewLabel(dateSinceNowInString(dt)))
				col4.Objects = append(col4.Objects, container.NewMax(
					priorityIcons[x[3]],
					widget.NewLabelWithStyle(x[3], fyne.TextAlignCenter, fyne.TextStyle{})))
				col5.Objects = append(col5.Objects, widget.NewLabel(x[4]))
			}
			list = container.NewVScroll(
				container.NewHBox(
					col0,
					col1,
					col3,
					col4,
					col5,
				),
			)
		}
		TaskTabs.Items[TaskTabsIndexes["Jira"]].Content = container.NewBorder(
			widget.NewToolbar(
				widget.NewToolbarAction(
					theme.ViewRefreshIcon(),
					func() {
						GetJira()
					},
				),
				widget.NewToolbarSeparator(),
				widget.NewToolbarAction(
					theme.HistoryIcon(),
					func() {},
				),
				widget.NewToolbarAction(
					theme.ErrorIcon(),
					func() {},
				),
			),
			nil,
			nil,
			nil,
			list,
		)
	}
	taskWindow.Content().Refresh()
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
