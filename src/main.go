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
	"vonexplaino.com/m/v2/helm/tasks"

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
*   - tasks.Jira/ MS/ Cherwell job visibility
**/

// @todo - Sorting or remove buttons

type AppStatusStruct struct {
	CurrentZettleDBDate time.Time
	CurrentZettleDKB    binding.String
	TaskTaskCount       int
	TaskTaskStatus      binding.String
}

type AppPreferencesStruct struct {
	ZettlekastenHome string
	TaskPreferences  tasks.TaskPreferencesStruct
}

var thisApp fyne.App
var mainWindow fyne.Window
var preferencesWindow fyne.Window

var taskWindow fyne.Window
var markdownInput *widget.Entry
var AppStatus AppStatusStruct
var TaskTabsIndexes map[string]int
var TaskTabs *container.AppTabs
var appPreferences AppPreferencesStruct
var gsmConnectionActive *fyne.Container
var msConnectionActive *fyne.Container
var jiraConnectionActive *fyne.Container
var connectionStatusContainer *fyne.Container
var connectionStatusBox = func(bool, string) {
	// nothing
}

func setup() {
	os.Setenv("TZ", "Australia/Brisbane")
	AppStatus = AppStatusStruct{
		CurrentZettleDBDate: time.Now().Local(),
		CurrentZettleDKB:    binding.NewString(),
		TaskTaskStatus:      binding.NewString(),
		TaskTaskCount:       0,
	}
	AppStatus.CurrentZettleDKB.Set(zettleFileName(time.Now().Local()))
}

func overrides() {
	preferencesToLocalVar()
	tasks.LoadPriorityOverride(appPreferences.TaskPreferences.PriorityOverride)
	connectionStatusBox = func(onl bool, label string) {
		icon := CloudDisconnect
		if onl {
			icon = CloudConnect
		}
		button := widget.NewButton(label, func() {})
		button.Importance = widget.LowImportance
		switch label[0:1] {
		case "G":
			button.OnTapped = func() {
				if onl {
					tasks.Gsm.Download(
						func() { taskWindowRefresh("CWTasks") },
						func() { taskWindowRefresh("CWIncidents") },
						func() { taskWindowRefresh("CWRequests") },
						func() { taskWindowRefresh("CWTeamIncidents") },
						func() { taskWindowRefresh("CWTeamTasks") },
					)
				} else {
					taskWindowRefresh("CWTasks")
					taskWindowRefresh("CWIncidents")
					taskWindowRefresh("CWRequests")
					taskWindowRefresh("CWTeamIncidents")
					taskWindowRefresh("CWTeamTasks")
				}
			}
			if !onl {
				tasks.Gsm.MyTasks = []tasks.TaskResponseStruct{}
				tasks.Gsm.MyIncidents = []tasks.TaskResponseStruct{}
				tasks.Gsm.LoggedIncidents = []tasks.TaskResponseStruct{}
				tasks.Gsm.TeamIncidents = []tasks.TaskResponseStruct{}
				tasks.Gsm.TeamTasks = []tasks.TaskResponseStruct{}
				taskWindowRefresh("CWTasks")
				taskWindowRefresh("CWIncidents")
				taskWindowRefresh("CWRequests")
				taskWindowRefresh("CWTeamIncidents")
				taskWindowRefresh("CWTeamTasks")
			}
			gsmConnectionActive.Objects = container.NewMax(
				button,
				icon,
			).Objects
			gsmConnectionActive.Refresh()
		case "J":
			button.OnTapped = func() {
				if onl {
					tasks.Jira.Download()
					taskWindowRefresh("Jira")
				}
			}
			jiraConnectionActive.Objects = container.NewMax(
				button,
				icon,
			).Objects
			jiraConnectionActive.Refresh()
		case "M":
			button.OnTapped = func() {
				tasks.Planner.Download("")
				taskWindowRefresh("Planner")
			}
			msConnectionActive.Objects = container.NewMax(
				button,
				icon,
			).Objects
			msConnectionActive.Refresh()
		}
		taskWindow.Content().Refresh()
	}
	gsmConnectionActive = container.NewMax()
	jiraConnectionActive = container.NewMax()
	msConnectionActive = container.NewMax()
	tasks.InitTasks(
		&appPreferences.TaskPreferences,
		connectionStatusBox,
		taskWindowRefresh,
		activeTaskStatusUpdate,
	)
	tasks.StartLocalServers()
	if appPreferences.TaskPreferences.GSMActive {
		go func() {
			for {
				time.Sleep(5 * time.Minute)
				// Stop refresh if out of business hours
				hour := time.Now().Hour()
				weekday := time.Now().Weekday()
				if hour > 8 && hour < 17 && weekday > 0 && weekday < 6 {
					tasks.Gsm.Download(
						func() { taskWindowRefresh("CWTasks") },
						func() { taskWindowRefresh("CWIncidents") },
						func() { taskWindowRefresh("CWRequests") },
						func() { taskWindowRefresh("CWTeamIncidents") },
						func() { taskWindowRefresh("CWTeamTasks") },
					)
				}
			}
		}()
	}
	if appPreferences.TaskPreferences.MSPlannerActive {
		go func() {
			for {
				time.Sleep(5 * time.Minute)
				hour := time.Now().Hour()
				weekday := time.Now().Weekday()
				if hour > 8 && hour < 17 && weekday > 0 && weekday < 6 {
					tasks.Planner.Download("")
					taskWindowRefresh("Planner")
				}
			}
		}()
	}
	if appPreferences.TaskPreferences.JiraActive {
		go func() {
			for {
				time.Sleep(5 * time.Minute)
				hour := time.Now().Hour()
				weekday := time.Now().Weekday()
				if hour > 8 && hour < 17 && weekday > 0 && weekday < 6 {
					tasks.Jira.Download()
					taskWindowRefresh("Jira")
				}
			}
		}()
	}
}

func main() {
	setup()
	thisApp = app.NewWithID("com.vonexplaino.helm.preferences")
	thisApp.SetIcon(fyne.NewStaticResource("Systray", icon.Data))
	overrides()
	preferencesWindow = thisApp.NewWindow("Preferences")
	preferencesWindowSetup()
	mainWindow = thisApp.NewWindow("Markdown Daily Knowledgebase")
	markdownWindowSetup()
	taskWindow = thisApp.NewWindow("Tasks")
	taskWindowSetup()
	go func() {
		tasks.GetAllTasks(
			appPreferences.TaskPreferences.JiraActive,
			appPreferences.TaskPreferences.GSMActive,
			appPreferences.TaskPreferences.MSPlannerActive,
			taskWindowRefresh,
			activeTaskStatusUpdate,
		)
	}()
	if desk, ok := thisApp.(desktop.App); ok {
		desk.SetSystemTrayIcon(fyne.NewStaticResource("Systray", icon.IconWhite))
		m := fyne.NewMenu("MyApp",
			fyne.NewMenuItem("Todays Notes", func() {
				mainWindow.Show()
				x, _ := AppStatus.CurrentZettleDKB.Get()
				if len(markdownInput.Text) > 0 {
					saveZettle(markdownInput.Text, x)
				}
				// Reload from file
				markdownInput.Text = getFileContentsAndCreateIfMissing(path.Join(appPreferences.ZettlekastenHome, x))
				markdownInput.Refresh()
			}),
			fyne.NewMenuItem("Tasks", func() {
				taskWindow.Show()
			}),
			fyne.NewMenuItemSeparator(),
			fyne.NewMenuItem("Preferences", func() {
				preferencesWindowSetup()
				preferencesWindow.Show()
			}),
		)
		desk.SetSystemTrayMenu(m)
	}
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
						mep = strings.Split(markdownInput.Text[3:], "---")[1]
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

func preferencesToLocalVar() {
	appPreferences = AppPreferencesStruct{}
	appPreferences.ZettlekastenHome = thisApp.Preferences().StringWithFallback("ZettlekastenHome", path.Join(os.TempDir(), "zett"))
	appPreferences.TaskPreferences.JiraProjectHome = thisApp.Preferences().StringWithFallback("JiraProjectHome", path.Join(os.TempDir(), "project"))
	appPreferences.TaskPreferences.GSMActive = thisApp.Preferences().BoolWithFallback("GSMActive", true)
	appPreferences.TaskPreferences.MSPlannerActive = thisApp.Preferences().BoolWithFallback("MSPlannerActive", false)
	appPreferences.TaskPreferences.MSGroups = thisApp.Preferences().StringWithFallback("MSGroups", "")
	appPreferences.TaskPreferences.JiraActive = thisApp.Preferences().BoolWithFallback("JiraActive", false)
	appPreferences.TaskPreferences.JiraKey = thisApp.Preferences().StringWithFallback("JiraKey", "")
	appPreferences.TaskPreferences.JiraUsername = thisApp.Preferences().StringWithFallback("JiraUsername", "")
	appPreferences.TaskPreferences.PriorityOverride = thisApp.Preferences().StringWithFallback("PriorityOverride", "")
	if appPreferences.TaskPreferences.PriorityOverride == "" {
		myself, error := user.Current()
		pribase := ""
		if error == nil {
			pribase = filepath.Join(myself.HomeDir, "/.hq")
		} else {
			pribase = filepath.Join(os.TempDir(), "/.hq")
		}
		appPreferences.TaskPreferences.PriorityOverride = thisApp.Preferences().StringWithFallback("PriorityOverride", pribase)
	}
}
func preferencesWindowSetup() {
	stringDateFormat := "20060102T15:04:05"

	// Fields
	zettlePath := widget.NewEntry()
	zettlePath.SetText(appPreferences.ZettlekastenHome)
	jiraPath := widget.NewEntry()
	jiraPath.SetText(appPreferences.TaskPreferences.JiraProjectHome)
	// MSPlanner
	plannerActive := widget.NewCheck("Active", func(res bool) {})
	plannerActive.SetChecked(appPreferences.TaskPreferences.MSPlannerActive)
	accessToken := widget.NewEntry()
	accessToken.SetText(tasks.AuthenticationTokens.MS.Access_token)
	refreshToken := widget.NewEntry()
	refreshToken.SetText(tasks.AuthenticationTokens.MS.Refresh_token)
	expiresAt := widget.NewEntry()
	expiresAt.SetText(tasks.AuthenticationTokens.MS.Expiration.Local().Format(stringDateFormat))
	groupsList := widget.NewEntry()
	groupsList.SetText(appPreferences.TaskPreferences.MSGroups)
	priorityOverride := widget.NewEntry()
	priorityOverride.SetText(appPreferences.TaskPreferences.PriorityOverride)
	// tasks.Jira
	jiraActive := widget.NewCheck("Active", func(res bool) {})
	jiraActive.SetChecked(appPreferences.TaskPreferences.JiraActive)
	jiraKey := widget.NewPasswordEntry()
	jiraKey.SetText(appPreferences.TaskPreferences.JiraKey)
	jiraUsername := widget.NewEntry()
	jiraUsername.SetText(appPreferences.TaskPreferences.JiraUsername)
	// GSM/ Cherwell
	gsmActive := widget.NewCheck("Active", func(res bool) {})
	gsmActive.SetChecked(appPreferences.TaskPreferences.GSMActive)
	// Dynamics
	dynamicsActive := widget.NewCheck("Active", func(res bool) {})
	dynamicsActive.SetChecked(appPreferences.TaskPreferences.DynamicsActive)
	dynamicsKey := widget.NewPasswordEntry()
	dynamicsKey.SetText(appPreferences.TaskPreferences.DynamicsKey)

	preferencesWindow.Resize(fyne.NewSize(500, 500))
	preferencesWindow.Hide()
	preferencesWindow.SetCloseIntercept(func() {
		preferencesWindow.Hide()
		// SavePreferences
		appPreferences.ZettlekastenHome = zettlePath.Text
		thisApp.Preferences().SetString("ZettlekastenHome", appPreferences.ZettlekastenHome)
		appPreferences.TaskPreferences.JiraProjectHome = jiraPath.Text
		thisApp.Preferences().SetString("JiraProjectHome", appPreferences.TaskPreferences.JiraProjectHome)
		appPreferences.TaskPreferences.PriorityOverride = priorityOverride.Text
		thisApp.Preferences().SetString("PriorityOverride", appPreferences.TaskPreferences.PriorityOverride)

		appPreferences.TaskPreferences.MSPlannerActive = plannerActive.Checked
		thisApp.Preferences().SetBool("MSPlannerActive", appPreferences.TaskPreferences.MSPlannerActive)
		tasks.AuthenticationTokens.MS.Access_token = accessToken.Text
		tasks.AuthenticationTokens.MS.Refresh_token = refreshToken.Text
		tasks.AuthenticationTokens.MS.Expiration, _ = time.Parse("20060102T15:04:05", expiresAt.Text)
		appPreferences.TaskPreferences.MSGroups = groupsList.Text
		thisApp.Preferences().SetString("MSGroups", appPreferences.TaskPreferences.MSGroups)

		appPreferences.TaskPreferences.JiraActive = jiraActive.Checked
		thisApp.Preferences().SetBool("JiraActive", appPreferences.TaskPreferences.JiraActive)
		appPreferences.TaskPreferences.JiraKey = jiraKey.Text
		thisApp.Preferences().SetString("JiraKey", appPreferences.TaskPreferences.JiraKey)
		appPreferences.TaskPreferences.JiraUsername = jiraUsername.Text
		thisApp.Preferences().SetString("JiraUsername", appPreferences.TaskPreferences.JiraUsername)

		appPreferences.TaskPreferences.GSMActive = gsmActive.Checked
		thisApp.Preferences().SetBool("GSMActive", appPreferences.TaskPreferences.GSMActive)
	})
	preferencesWindow.SetContent(
		container.New(
			layout.NewFormLayout(),
			widget.NewLabel(""),
			widget.NewLabelWithStyle("Paths", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			widget.NewLabel("Zettlekasten Path"),
			zettlePath,
			widget.NewLabel("Priority-override file"),
			priorityOverride,
			widget.NewLabel("Jira Project Path"),
			jiraPath,
			widget.NewLabel(""),
			widget.NewLabelWithStyle("Planner", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			widget.NewLabel("Planner active"),
			plannerActive,
			widget.NewLabel("Access Token"),
			accessToken,
			widget.NewLabel("Refresh Token"),
			refreshToken,
			widget.NewLabel("Expires At"),
			expiresAt,
			widget.NewLabel(""),
			widget.NewLabelWithStyle("JIRA", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			widget.NewLabel("Jira active"),
			jiraActive,
			widget.NewLabel("Key"),
			jiraKey,
			widget.NewLabel("Username"),
			jiraUsername,
			widget.NewLabel(""),
			widget.NewLabelWithStyle("GSM", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			widget.NewLabel("GSM Active"),
			gsmActive,
		),
	)
}

func getFileContentsAndCreateIfMissing(filename string) string {
	content, err := os.ReadFile(filename)
	if errors.Is(err, os.ErrNotExist) {
		content = []byte(fmt.Sprintf("---\nDate: %s\ntags: [\"status\"]\n---\n", AppStatus.CurrentZettleDBDate.Local().Format("2006-01-02")))
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
	return fmt.Sprintf("%s-retro.md", date.Local().Format("20060102"))
}

func taskWindowSetup() {
	taskWindow.Resize(fyne.NewSize(430, 550))
	taskWindow.Hide()
	taskStatusWidget := widget.NewLabelWithData(AppStatus.TaskTaskStatus)
	connectionStatusContainer = container.NewGridWithColumns(3)
	connectionStatusBox(false, "G")
	connectionStatusBox(false, "M")
	connectionStatusBox(false, "J")
	connectionStatusContainer = container.NewGridWithColumns(3,
		gsmConnectionActive,
		msConnectionActive,
		jiraConnectionActive,
	)
	TaskTabsIndexes = map[string]int{}
	TaskTabs = container.NewAppTabs()
	if appPreferences.TaskPreferences.GSMActive {
		TaskTabs = container.NewAppTabs(
			container.NewTabItem("My Tasks", container.NewBorder(
				widget.NewToolbar(
					widget.NewToolbarAction(
						theme.ViewRefreshIcon(),
						func() {
							tasks.Gsm.DownloadTasks(func() { taskWindowRefresh("CWTasks") })
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
							tasks.Gsm.DownloadIncidents(func() { taskWindowRefresh("CWIncidents") })
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
							tasks.Gsm.DownloadTeam(func() { taskWindowRefresh("CWRequests") })
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
			container.NewTabItem("Team Unass. Inc.", container.NewBorder(
				widget.NewToolbar(
					widget.NewToolbarAction(
						theme.ViewRefreshIcon(),
						func() {
							tasks.Gsm.DownloadMyRequests(func() { taskWindowRefresh("CWTeamIncidents") })
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
			container.NewTabItem("Team Unass. Tasks", container.NewBorder(
				widget.NewToolbar(
					widget.NewToolbarAction(
						theme.ViewRefreshIcon(),
						func() {
							tasks.Gsm.DownloadMyRequests(func() { taskWindowRefresh("CWTeamTasks") })
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
			"CWTeamTasks":     4,
		}
	}
	if appPreferences.TaskPreferences.MSPlannerActive {
		TaskTabsIndexes["Planner"] = len(TaskTabsIndexes)
		TaskTabs.Append(
			container.NewTabItem("Planner", container.NewBorder(
				widget.NewToolbar(
					widget.NewToolbarAction(
						theme.ViewRefreshIcon(),
						func() {
							go func() {
								tasks.Planner.Download("")
								taskWindowRefresh("Planner")
							}()
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
	if appPreferences.TaskPreferences.JiraActive {
		TaskTabsIndexes["Jira"] = len(TaskTabsIndexes)
		TaskTabs.Append(
			container.NewTabItem("Jira", container.NewBorder(
				widget.NewToolbar(
					widget.NewToolbarAction(
						theme.ViewRefreshIcon(),
						func() {
							go func() {
								tasks.Jira.Download()
								taskWindowRefresh("Jira")
							}()
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
			container.NewHBox(
				taskStatusWidget,
				layout.NewSpacer(),
				connectionStatusContainer,
			),
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

	priorityIcons := setupPriorityIcons()
	taskIcons := setupJiraTaskTypeIcons()
	if appPreferences.TaskPreferences.GSMActive {
		if _, ok := TaskTabsIndexes["CWTasks"]; ok && (specific == "" || specific == "CWTasks") {
			if len(tasks.Gsm.MyTasks) == 0 {
				list = widget.NewLabel("No tasks")
			} else {
				col0 := container.NewVBox(widget.NewRichTextFromMarkdown(`### `))
				col1 := container.NewVBox(widget.NewRichTextFromMarkdown(`### Incident`))
				col2 := container.NewVBox(widget.NewRichTextFromMarkdown(`### Task`))
				col3 := container.NewVBox(widget.NewRichTextFromMarkdown(`### Age`))
				col4 := container.NewVBox(widget.NewRichTextFromMarkdown(`### Priority`))
				col5 := container.NewVBox(widget.NewRichTextFromMarkdown(`### Status`))

				for _, x := range tasks.Gsm.MyTasks {
					thisID := x.ID
					thisObjRecId := x.BusObRecId
					thisParent := x.ParentID
					thisParentInternal := x.ParentIDInternal
					myPriority := x.PriorityOverride
					if x.Priority != x.PriorityOverride {
						myPriority = fmt.Sprintf("%s(%s)", x.PriorityOverride, x.Priority)
					}
					tempVar := ""
					col0.Objects = append(
						col0.Objects,
						container.NewMax(
							widget.NewLabel(""),
							container.NewHBox(
								newTappableIcon(theme.InfoIcon(), func(_ *fyne.PointEvent) {
									browser.OpenURL("https://griffith.cherwellondemand.com/CherwellClient/Access/incident/" + thisParent)
								}),
								newTappableIcon(theme.DocumentIcon(), func(_ *fyne.PointEvent) {
									journals, err := tasks.Gsm.GetJournalNotesForIncident(thisParentInternal)

									if err == nil {
										list := container.NewVBox()
										for _, x := range journals {
											deets := widget.NewLabel(x.Details)
											deets.Wrapping = fyne.TextWrapBreak
											list.Add(container.NewVBox(
												container.NewHBox(
													widget.NewLabel(x.Date.Format("2006-01-02 15:04:05")),
													layout.NewSpacer(),
													widget.NewLabel(x.Class),
												),
												deets,
											))
										}

										deepdeep := thisApp.NewWindow("Journals for " + thisParent)
										deepdeep.SetContent(container.NewVScroll(list))
										deepdeep.Resize(fyne.NewSize(400, 500))
										deepdeep.Show()
									} else {
										fmt.Printf("FAILED %v\n", err)
									}
								}),
								newTappableIcon(theme.AccountIcon(), func(_ *fyne.PointEvent) {
									var deepdeep dialog.Dialog
									var foundPeople []struct {
										Label  string
										Target string
									}
									foundsList := widget.NewList(
										func() int { return len(foundPeople) },
										func() fyne.CanvasObject { return newTappableLabel("x", func(_ *fyne.PointEvent) {}) },
										func(lii widget.ListItemID, co fyne.CanvasObject) {
											me := foundPeople[lii]
											co.(*tappableLabel).SetText(me.Label)
											co.(*tappableLabel).OnTapGo = func(_ *fyne.PointEvent) {
												splits := strings.Split(me.Target, "!")
												fmt.Printf("Target: %s|%s\n", me.Target, splits[0])
												tasks.Gsm.ReassignTaskToPersonInTeam(thisObjRecId, splits[0], splits[1])
												fmt.Printf("Reassigning to %s|%s\n", me.Label, me.Target)
												deepdeep.Hide()
											}
										},
									)
									foundsList.Resize(fyne.NewSize(300, 500))
									foundsContainer := container.NewMax(foundsList)
									lookinFor := widget.NewEntry()
									deepdeep = dialog.NewCustom("Reassign task",
										"Actually, no",
										container.NewBorder(
											container.NewVBox(
												container.New(layout.NewFormLayout(),
													widget.NewLabel("Reassign to"),
													lookinFor,
												),
												widget.NewButtonWithIcon(
													"Search",
													theme.SearchIcon(),
													func() {
														founds, err := tasks.Gsm.FindPeopleToReasignTo(lookinFor.Text)
														foundPeople = []struct {
															Label  string
															Target string
														}{}
														if err == nil {
															for _, c := range founds {
																for _, tName := range c.Teams {
																	foundPeople = append(foundPeople, struct {
																		Label  string
																		Target string
																	}{
																		Label:  fmt.Sprintf("%s - %s", c.Name, tName),
																		Target: fmt.Sprintf("%s!%s", c.UserID, tName),
																	})
																}
															}
															foundsList.Refresh()
															foundsContainer.Refresh()
														} else {
															fmt.Printf("Failed %v\n", err)
														}
													},
												),
											), nil, nil, nil,
											foundsContainer,
										),
										taskWindow,
									)
									deepdeep.Resize(fyne.NewSize(300, 500))
									deepdeep.Show()
									//
								})),
						))
					col1.Objects = append(col1.Objects,
						widget.NewLabelWithStyle(fmt.Sprintf("[%s] %s", x.ParentID, x.ParentTitle), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
					col2.Objects = append(
						col2.Objects,
						newTappableLabel(
							fmt.Sprintf("[%s] %s", x.ID, x.Title),
							func(_ *fyne.PointEvent) {
								browser.OpenURL("https://griffith.cherwellondemand.com/CherwellClient/Access/task/" + thisID)
							},
						),
					)
					col3.Objects = append(col3.Objects, widget.NewLabel(dateSinceNowInString(x.CreatedDateTime)))
					tempFunc := func(_ *fyne.PointEvent) {
						dialog.ShowForm(
							"Priority override "+thisID,
							"Override",
							"Cancel",
							[]*widget.FormItem{
								widget.NewFormItem(
									"Priority",
									widget.NewSelect(
										[]string{"1", "2", "3", "4", "5", "6"},
										func(changed string) {
											tempVar = changed
										},
									)),
							},
							func(isit bool) {
								if tempVar == x.Priority || tempVar == "" {
									delete(tasks.PriorityOverrides.CWIncidents, thisID)
								} else {
									tasks.PriorityOverrides.CWIncidents[thisID] = tempVar
								}
								tasks.SavePriorityOverride()
							},
							taskWindow,
						)
					}
					col4.Objects = append(col4.Objects, container.NewMax(
						getPriorityIconFor(x.PriorityOverride, priorityIcons),
						newTappableLabelWithStyle(
							myPriority,
							fyne.TextAlignCenter,
							fyne.TextStyle{},
							tempFunc)))
					col5.Objects = append(col5.Objects, widget.NewLabel(x.Status))
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
							tasks.Gsm.DownloadTasks(func() { taskWindowRefresh("CWTasks") })
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
		if _, ok := TaskTabsIndexes["CWIncidents"]; ok && (specific == "" || specific == "CWIncidents") {
			var list2 fyne.CanvasObject
			if len(tasks.Gsm.MyIncidents) == 0 {
				list2 = widget.NewLabel("No incidents")
			} else {
				col0 := container.NewVBox(widget.NewRichTextFromMarkdown(`### `))
				col1 := container.NewVBox(widget.NewRichTextFromMarkdown(`### Incident`))
				col3 := container.NewVBox(widget.NewRichTextFromMarkdown(`### Age`))
				col4 := container.NewVBox(widget.NewRichTextFromMarkdown(`### Priority`))
				col5 := container.NewVBox(widget.NewRichTextFromMarkdown(`### Status`))
				for _, x := range tasks.Gsm.MyIncidents {
					thisID := x.ID
					myPriority := x.PriorityOverride
					if x.Priority != x.PriorityOverride {
						myPriority = fmt.Sprintf("%s(%s)", x.PriorityOverride, x.Priority)
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
						widget.NewLabelWithStyle(fmt.Sprintf("[%s] %s", x.ID, x.Title), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
					col3.Objects = append(col3.Objects, widget.NewLabel(dateSinceNowInString(x.CreatedDateTime)))
					tempFunc := func(_ *fyne.PointEvent) {
						dialog.ShowForm(
							"Priority override "+thisID,
							"Override",
							"Cancel",
							[]*widget.FormItem{
								widget.NewFormItem(
									"Priority",
									widget.NewSelect(
										[]string{"1", "2", "3", "4", "5", "6"},
										func(changed string) {
											tempVar = changed
										},
									)),
							},
							func(isit bool) {
								if tempVar == x.Priority || tempVar == "" {
									delete(tasks.PriorityOverrides.CWIncidents, thisID)
								} else {
									tasks.PriorityOverrides.CWIncidents[thisID] = tempVar
								}
								tasks.SavePriorityOverride()
							},
							taskWindow,
						)
					}
					col4.Objects = append(col4.Objects, container.NewMax(
						getPriorityIconFor(x.PriorityOverride, priorityIcons),
						newTappableLabelWithStyle(
							myPriority,
							fyne.TextAlignCenter,
							fyne.TextStyle{},
							tempFunc)))
					col5.Objects = append(col5.Objects, widget.NewLabel(x.Status))
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
							tasks.Gsm.DownloadIncidents(func() { taskWindowRefresh("CWIncidents") })
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
		if _, ok := TaskTabsIndexes["CWTeamIncidents"]; ok && (specific == "" || specific == "CWTeamIncidents") {
			var list3 fyne.CanvasObject
			if len(tasks.Gsm.TeamIncidents) == 0 {
				list3 = widget.NewLabel("No incidents")
			} else {
				col0 := container.NewVBox(widget.NewRichTextFromMarkdown(`### `))
				col1 := container.NewVBox(widget.NewRichTextFromMarkdown(`### Incident`))
				col2 := container.NewVBox(widget.NewRichTextFromMarkdown(`### Owner`))
				col3 := container.NewVBox(widget.NewRichTextFromMarkdown(`### Age`))
				col4 := container.NewVBox(widget.NewRichTextFromMarkdown(`### Priority`))
				col5 := container.NewVBox(widget.NewRichTextFromMarkdown(`### Status`))
				for _, x := range tasks.Gsm.TeamIncidents {
					thisID := x.ID
					col0.Objects = append(
						col0.Objects,
						container.NewMax(
							widget.NewLabel(""),
							newTappableIcon(theme.InfoIcon(), func(_ *fyne.PointEvent) {
								browser.OpenURL("https://griffith.cherwellondemand.com/CherwellClient/Access/incident/" + thisID)
							}),
						))
					col2.Objects = append(col2.Objects, widget.NewLabel(x.Owner))
					col1.Objects = append(col1.Objects,
						widget.NewLabelWithStyle(fmt.Sprintf("[%s] %s", x.ID, x.Title), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
					col3.Objects = append(col3.Objects, widget.NewLabel(dateSinceNowInString(x.CreatedDateTime)))
					col4.Objects = append(col4.Objects, container.NewMax(
						getPriorityIconFor(x.PriorityOverride, priorityIcons),
						widget.NewLabelWithStyle(x.PriorityOverride, fyne.TextAlignCenter, fyne.TextStyle{})))
					col5.Objects = append(col5.Objects, widget.NewLabel(x.Status))
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
							tasks.Gsm.DownloadTeam(func() { taskWindowRefresh("CWTeamIncidents") })
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
		if _, ok := TaskTabsIndexes["CWTeamTasks"]; ok && (specific == "" || specific == "CWTeamTasks") {
			var list fyne.CanvasObject
			if len(tasks.Gsm.TeamTasks) == 0 {
				list = widget.NewLabel("No tasks")
			} else {
				col0 := container.NewVBox(widget.NewRichTextFromMarkdown(`### `))
				col1 := container.NewVBox(widget.NewRichTextFromMarkdown(`### Incident`))
				col2 := container.NewVBox(widget.NewRichTextFromMarkdown(`### Task`))
				col3 := container.NewVBox(widget.NewRichTextFromMarkdown(`### Age`))
				col4 := container.NewVBox(widget.NewRichTextFromMarkdown(`### Priority`))
				col5 := container.NewVBox(widget.NewRichTextFromMarkdown(`### Status`))
				for _, x := range tasks.Gsm.TeamTasks {
					thisID := x.ID
					thisObjRecId := x.BusObRecId
					thisParent := x.ParentID
					thisParentInternal := x.ParentIDInternal
					myPriority := x.PriorityOverride
					if x.Priority != x.PriorityOverride {
						myPriority = fmt.Sprintf("%s(%s)", x.PriorityOverride, x.Priority)
					}
					tempVar := ""
					col0.Objects = append(
						col0.Objects,
						container.NewMax(
							widget.NewLabel(""),
							container.NewHBox(
								newTappableIcon(theme.InfoIcon(), func(_ *fyne.PointEvent) {
									browser.OpenURL("https://griffith.cherwellondemand.com/CherwellClient/Access/incident/" + thisParent)
								}),
								newTappableIcon(theme.DocumentIcon(), func(_ *fyne.PointEvent) {
									journals, err := tasks.Gsm.GetJournalNotesForIncident(thisParentInternal)

									if err == nil {
										list := container.NewVBox()
										for _, x := range journals {
											deets := widget.NewLabel(x.Details)
											deets.Wrapping = fyne.TextWrapBreak
											list.Add(container.NewVBox(
												container.NewHBox(
													widget.NewLabel(x.Date.Format("2006-01-02 15:04:05")),
													layout.NewSpacer(),
													widget.NewLabel(x.Class),
												),
												deets,
											))
										}

										deepdeep := thisApp.NewWindow("Journals for " + thisParent)
										deepdeep.SetContent(container.NewVScroll(list))
										deepdeep.Resize(fyne.NewSize(400, 500))
										deepdeep.Show()
									} else {
										fmt.Printf("FAILED %v\n", err)
									}
								}),
								newTappableIcon(theme.AccountIcon(), func(_ *fyne.PointEvent) {
									var deepdeep dialog.Dialog
									var foundPeople []struct {
										Label  string
										Target string
									}
									foundsList := widget.NewList(
										func() int {
											return len(foundPeople)
										},
										func() fyne.CanvasObject {
											return newTappableLabel("x", func(_ *fyne.PointEvent) {})
										},
										func(lii widget.ListItemID, co fyne.CanvasObject) {
											me := foundPeople[lii]
											co.(*tappableLabel).SetText(me.Label)
											co.(*tappableLabel).OnTapGo = func(_ *fyne.PointEvent) {
												splits := strings.Split(me.Target, "!")
												fmt.Printf("Target: %s|%s\n", me.Target, splits[0])
												tasks.Gsm.ReassignTaskToPersonInTeam(thisObjRecId, splits[0], splits[1])
												fmt.Printf("Reassigning %s to %s|%s\n", thisObjRecId, me.Label, me.Target)
												deepdeep.Hide()
											}
										},
									)
									foundsList.Resize(fyne.NewSize(300, 500))
									foundsContainer := container.NewMax(foundsList)
									lookinFor := widget.NewEntry()
									deepdeep = dialog.NewCustom("Reassign task",
										"Actually, no",
										container.NewBorder(
											container.NewVBox(
												container.New(layout.NewFormLayout(),
													widget.NewLabel("Reassign to"),
													lookinFor,
												),
												widget.NewButtonWithIcon(
													"Search",
													theme.SearchIcon(),
													func() {
														founds, err := tasks.Gsm.FindPeopleToReasignTo(lookinFor.Text)
														foundPeople = []struct {
															Label  string
															Target string
														}{}
														if err == nil {
															for _, c := range founds {
																for _, tName := range c.Teams {
																	foundPeople = append(foundPeople, struct {
																		Label  string
																		Target string
																	}{
																		Label:  fmt.Sprintf("%s - %s", c.Name, tName),
																		Target: fmt.Sprintf("%s!%s", c.UserID, tName),
																	})
																}
															}
															foundsList.Refresh()
															foundsContainer.Refresh()
														} else {
															fmt.Printf("Failed %v\n", err)
														}
													},
												),
											), nil, nil, nil,
											foundsContainer,
										),
										taskWindow,
									)
									deepdeep.Resize(fyne.NewSize(300, 500))
									deepdeep.Show()
									//
								})),
						))
					col1.Objects = append(col1.Objects,
						widget.NewLabelWithStyle(fmt.Sprintf("[%s] %s", x.ParentID, x.ParentTitle), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
					col2.Objects = append(
						col2.Objects,
						newTappableLabel(
							fmt.Sprintf("[%s] %s", x.ID, x.Title),
							func(_ *fyne.PointEvent) {
								browser.OpenURL("https://griffith.cherwellondemand.com/CherwellClient/Access/task/" + thisID)
							},
						),
					)
					col3.Objects = append(col3.Objects, widget.NewLabel(dateSinceNowInString(x.CreatedDateTime)))
					tempFunc := func(_ *fyne.PointEvent) {
						dialog.ShowForm(
							"Priority override "+thisID,
							"Override",
							"Cancel",
							[]*widget.FormItem{
								widget.NewFormItem(
									"Priority",
									widget.NewSelect(
										[]string{"1", "2", "3", "4", "5", "6"},
										func(changed string) {
											tempVar = changed
										},
									)),
							},
							func(isit bool) {
								if tempVar == x.Priority || tempVar == "" {
									delete(tasks.PriorityOverrides.CWIncidents, thisID)
								} else {
									tasks.PriorityOverrides.CWIncidents[thisID] = tempVar
								}
								tasks.SavePriorityOverride()
							},
							taskWindow,
						)
					}
					col4.Objects = append(col4.Objects, container.NewMax(
						getPriorityIconFor(x.PriorityOverride, priorityIcons),
						newTappableLabelWithStyle(
							myPriority,
							fyne.TextAlignCenter,
							fyne.TextStyle{},
							tempFunc)))
					col5.Objects = append(col5.Objects, widget.NewLabel(x.Status))
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
			TaskTabs.Items[TaskTabsIndexes["CWTeamTasks"]].Content = container.NewBorder(
				widget.NewToolbar(
					widget.NewToolbarAction(
						theme.ViewRefreshIcon(),
						func() {
							tasks.Gsm.DownloadTeamTasks(func() { taskWindowRefresh("CWTeamTasks") })
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
		if _, ok := TaskTabsIndexes["CWRequests"]; ok && (specific == "" || specific == "CWRequests") {
			var list4 fyne.CanvasObject
			if len(tasks.Gsm.LoggedIncidents) == 0 {
				list4 = widget.NewLabel("No requests")
			} else {
				col0 := container.NewVBox(widget.NewRichTextFromMarkdown(`### `))
				col1 := container.NewVBox(widget.NewRichTextFromMarkdown(`### Incident`))
				col3 := container.NewVBox(widget.NewRichTextFromMarkdown(`### Age`))
				col4 := container.NewVBox(widget.NewRichTextFromMarkdown(`### Priority`))
				col5 := container.NewVBox(widget.NewRichTextFromMarkdown(`### Status`))
				for _, x := range tasks.Gsm.LoggedIncidents {
					thisID := x.ID
					col0.Objects = append(
						col0.Objects,
						container.NewMax(
							widget.NewLabel(""),
							newTappableIcon(theme.InfoIcon(), func(_ *fyne.PointEvent) {
								browser.OpenURL("https://griffith.cherwellondemand.com/CherwellClient/Access/incident/" + thisID)
							}),
						))
					col1.Objects = append(col1.Objects,
						widget.NewLabelWithStyle(fmt.Sprintf("[%s] %s", x.ID, x.Title), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
					col3.Objects = append(col3.Objects, widget.NewLabel(dateSinceNowInString(x.CreatedDateTime)))
					col4.Objects = append(col4.Objects, container.NewMax(
						getPriorityIconFor(x.PriorityOverride, priorityIcons),
						widget.NewLabelWithStyle(x.PriorityOverride, fyne.TextAlignCenter, fyne.TextStyle{})))
					col5.Objects = append(col5.Objects, widget.NewLabel(x.Status))
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
							tasks.Gsm.DownloadMyRequests(func() { taskWindowRefresh("CWRequests") })
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
	}
	if _, ok := TaskTabsIndexes["Planner"]; ok &&
		appPreferences.TaskPreferences.MSPlannerActive &&
		(specific == "" || specific == "Planner") {
		// MY PLANNER
		var list5 fyne.CanvasObject
		if len(tasks.Planner.MyTasks) == 0 {
			list5 = widget.NewLabel("No requests")
		} else {
			col0 := container.NewVBox(widget.NewRichTextFromMarkdown(`### `))
			col1 := container.NewVBox(widget.NewRichTextFromMarkdown(`### Title`))
			col3 := container.NewVBox(widget.NewRichTextFromMarkdown(`### Age`))
			col4 := container.NewVBox(widget.NewRichTextFromMarkdown(`### Priority`))
			col5 := container.NewVBox(widget.NewRichTextFromMarkdown(`### %`))
			for _, x := range tasks.Planner.MyTasks {
				thisID := x.ID
				myPriority := x.PriorityOverride
				if x.Priority != x.PriorityOverride {
					myPriority = fmt.Sprintf("%s(%s)", x.PriorityOverride, x.Priority)
				}
				tempVar := ""
				col0.Objects = append(
					col0.Objects,
					container.NewMax(
						widget.NewLabel(""),
						newTappableIcon(theme.InfoIcon(), func(_ *fyne.PointEvent) {
							browser.OpenURL(
								fmt.Sprintf(
									"https://tasks.office.com/%s/Home/Task/%s",
									tasks.MsApplicationTenant,
									thisID,
								),
							)
						}),
					))
				col1.Objects = append(col1.Objects,
					widget.NewLabelWithStyle(x.Title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
				col3.Objects = append(col3.Objects, widget.NewLabel(dateSinceNowInString(x.CreatedDateTime)))
				iconContainer := container.NewMax(getPriorityIconFor(x.PriorityOverride, priorityIcons))
				textContainer := newTappableLabel(myPriority, func(_ *fyne.PointEvent) {})
				tempFunc := func(_ *fyne.PointEvent) {
					dialog.ShowForm(
						"Priority override",
						"Override",
						"Cancel",
						[]*widget.FormItem{
							widget.NewFormItem(
								"Priority",
								widget.NewSelect(
									[]string{"1", "2", "3", "4", "5", "6"},
									func(changed string) {
										tempVar = changed
									},
								)),
						},
						func(isit bool) {
							if isit {
								var thisPriority string
								if tempVar == x.Priority {
									delete(tasks.PriorityOverrides.MSPlanner, thisID)
									thisPriority = tempVar
								} else {
									tasks.PriorityOverrides.MSPlanner[thisID] = tempVar
									thisPriority = tempVar + "(" + x.Priority + ")"
								}
								tasks.SavePriorityOverride()
								iconContainer.Objects[0] = getPriorityIconFor(tempVar, priorityIcons)
								textContainer.Label.Text = thisPriority
								textContainer.Refresh()
								iconContainer.Refresh()
							}
						},
						taskWindow,
					)
				}
				textContainer = newTappableLabelWithStyle(
					myPriority,
					fyne.TextAlignCenter,
					fyne.TextStyle{},
					tempFunc)
				col4.Objects = append(col4.Objects, container.NewMax(
					iconContainer,
					textContainer,
				))
				col5.Objects = append(col5.Objects, widget.NewLabel(x.Status))
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

		TaskTabs.Items[TaskTabsIndexes["Planner"]].Content = container.NewBorder(
			widget.NewToolbar(
				widget.NewToolbarAction(
					theme.ViewRefreshIcon(),
					func() {
						go func() {
							tasks.Planner.Download("")
							taskWindowRefresh("Planner")
						}()
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
	if _, ok := TaskTabsIndexes["Jira"]; ok && appPreferences.TaskPreferences.JiraActive && (specific == "" || specific == "Jira") {
		var list fyne.CanvasObject
		if len(tasks.Jira.MyTasks) == 0 {
			list = widget.NewLabel("No requests")
		} else {
			col0 := container.NewVBox(widget.NewRichTextFromMarkdown(`### `))
			col1 := container.NewVBox(widget.NewRichTextFromMarkdown(`### Title`))
			col3 := container.NewVBox(widget.NewRichTextFromMarkdown(`### Age`))
			col4 := container.NewVBox(widget.NewRichTextFromMarkdown(`### Priority`))
			col5 := container.NewVBox(widget.NewRichTextFromMarkdown(`### Status`))
			for _, x := range tasks.Jira.MyTasks {
				thisID := x.ID
				myPriority := x.PriorityOverride
				if x.Priority != x.PriorityOverride {
					myPriority = fmt.Sprintf("%s(%s)", x.PriorityOverride, x.Priority)
				}
				tempVar := ""
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
				blocked := widget.NewIcon(theme.MediaPlayIcon())
				if x.Blocked {
					blocked = widget.NewIcon(theme.MediaPauseIcon())
				}
				col1.Objects = append(col1.Objects,
					container.NewHBox(
						container.NewMax(
							getJiraTypeIconFor(x.Type, taskIcons),
							widget.NewLabelWithStyle(x.Type[0:1], fyne.TextAlignCenter, fyne.TextStyle{Bold: true})),
						blocked,
						widget.NewLabelWithStyle(fmt.Sprintf("[%s] %s", thisID, x.Title), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
					))
				col3.Objects = append(col3.Objects, widget.NewLabel(dateSinceNowInString(x.CreatedDateTime)))
				iconContainer := container.NewMax(getPriorityIconFor(x.PriorityOverride, priorityIcons))
				textContainer := newTappableLabel(myPriority, func(_ *fyne.PointEvent) {})
				tempFunc := func(_ *fyne.PointEvent) {
					dialog.ShowForm(
						"Priority override",
						"Override",
						"Cancel",
						[]*widget.FormItem{
							widget.NewFormItem(
								"Priority",
								widget.NewSelect(
									[]string{"1", "2", "3", "4", "5", "6"},
									func(changed string) {
										tempVar = changed
									},
								)),
						},
						func(isit bool) {
							if isit {
								var thisPriority string
								if tempVar == x.Priority {
									delete(tasks.PriorityOverrides.Jira, thisID)
									thisPriority = tempVar
								} else {
									tasks.PriorityOverrides.Jira[thisID] = tempVar
									thisPriority = tempVar + "(" + x.Priority + ")"
								}
								tasks.SavePriorityOverride()
								iconContainer.Objects[0] = getPriorityIconFor(tempVar, priorityIcons)
								textContainer.Label.Text = thisPriority
								textContainer.Refresh()
								iconContainer.Refresh()
							}
						},
						taskWindow,
					)
				}
				textContainer = newTappableLabelWithStyle(
					myPriority,
					fyne.TextAlignCenter,
					fyne.TextStyle{},
					tempFunc)
				col4.Objects = append(col4.Objects, container.NewMax(
					iconContainer,
					textContainer,
				))
				col5.Objects = append(col5.Objects, widget.NewLabel(x.Status))
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
						go func() {
							tasks.Jira.Download()
							taskWindowRefresh("Jira")
						}()
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

var CloudConnect = widget.NewIcon(
	fyne.NewStaticResource("cloudconnect.png", []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d,
		0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x5a, 0x00, 0x00, 0x00, 0x5a,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x38, 0xa8, 0x41, 0x02, 0x00, 0x00, 0x00,
		0x06, 0x62, 0x4b, 0x47, 0x44, 0x00, 0xff, 0x00, 0xff, 0x00, 0xff, 0xa0,
		0xbd, 0xa7, 0x93, 0x00, 0x00, 0x02, 0xf1, 0x49, 0x44, 0x41, 0x54, 0x78,
		0x9c, 0xed, 0xda, 0xcb, 0x6b, 0x1d, 0x55, 0x1c, 0x07, 0xf0, 0xcf, 0x8d,
		0xd6, 0x1a, 0xdd, 0xd4, 0xd6, 0x4d, 0x7d, 0x55, 0x2d, 0x28, 0x54, 0x90,
		0x10, 0x74, 0x51, 0x8b, 0x88, 0x50, 0x45, 0xc5, 0x47, 0xdc, 0x17, 0xdd,
		0x14, 0xb4, 0x0b, 0x53, 0x50, 0x48, 0x17, 0xfe, 0x01, 0xdd, 0x89, 0x76,
		0xe3, 0x42, 0xa8, 0x20, 0xbe, 0x15, 0xa4, 0xa8, 0x2b, 0xa9, 0x1b, 0x05,
		0x1f, 0x28, 0x3e, 0xa0, 0x88, 0x1b, 0x2b, 0x21, 0x2a, 0x55, 0x2b, 0xd8,
		0x87, 0x4d, 0x52, 0x9b, 0x9f, 0x8b, 0x39, 0x39, 0xde, 0x8a, 0xa1, 0x20,
		0xb5, 0x87, 0x3b, 0xfc, 0x3e, 0x97, 0x0b, 0x97, 0x99, 0x73, 0xe1, 0xcb,
		0x97, 0xc3, 0x99, 0x99, 0xc3, 0x90, 0x52, 0x4a, 0x29, 0xa5, 0x94, 0x52,
		0x4a, 0x29, 0xa5, 0x94, 0x52, 0x4a, 0x29, 0xa5, 0x94, 0x52, 0x4a, 0x29,
		0xa5, 0x94, 0x52, 0x4a, 0x29, 0xa5, 0x94, 0x52, 0x4a, 0x29, 0xa5, 0x94,
		0x52, 0x4a, 0x29, 0xa5, 0x94, 0x52, 0x4a, 0x1b, 0xf0, 0x48, 0xeb, 0x10,
		0x7d, 0x37, 0x8e, 0xcf, 0x71, 0x0a, 0x53, 0x8d, 0xb3, 0xf4, 0xda, 0x0b,
		0x88, 0xf2, 0x3d, 0x8e, 0x9b, 0xda, 0xc6, 0xe9, 0xa7, 0x69, 0x84, 0x8b,
		0x85, 0xfb, 0x6a, 0xd9, 0x3f, 0xe2, 0xd2, 0xb6, 0xb1, 0xfa, 0xe5, 0x36,
		0x2c, 0x1a, 0x08, 0xaf, 0x08, 0xf7, 0xd6, 0xa2, 0xf7, 0x63, 0xac, 0x6d,
		0xb4, 0xfe, 0xb8, 0x12, 0x87, 0x10, 0x76, 0x09, 0x7b, 0x6a, 0xc9, 0xbf,
		0xe1, 0xaa, 0xa6, 0xc9, 0x7a, 0xe4, 0x42, 0x7c, 0x8a, 0xb0, 0x55, 0xf8,
		0x52, 0x18, 0xaf, 0x45, 0x3f, 0xd0, 0x36, 0x5a, 0xbf, 0x3c, 0x8f, 0x70,
		0xad, 0x30, 0x27, 0xdc, 0x50, 0x4b, 0x7e, 0xb6, 0x71, 0xae, 0x5e, 0x99,
		0xb1, 0x7c, 0xf1, 0xfb, 0x4a, 0xd8, 0x51, 0x4b, 0x3e, 0xa0, 0xbb, 0xcd,
		0x4b, 0x67, 0xc1, 0x3d, 0xf8, 0xd3, 0x40, 0x78, 0x4d, 0x78, 0x57, 0x18,
		0x08, 0xcc, 0x63, 0xa2, 0x71, 0xb6, 0xde, 0x98, 0xc0, 0x51, 0x84, 0xdd,
		0xc2, 0xac, 0xb0, 0xae, 0xce, 0xe6, 0x9d, 0x6d, 0xa3, 0xf5, 0xc7, 0x7a,
		0xcc, 0x22, 0x3c, 0x2c, 0x2c, 0x0a, 0xb7, 0xd4, 0x92, 0xdf, 0xc1, 0xa0,
		0x69, 0xba, 0x9e, 0x18, 0xc7, 0xc7, 0x08, 0x5b, 0x84, 0x79, 0xe1, 0x89,
		0x5a, 0xf2, 0xac, 0x7c, 0x30, 0x39, 0x2b, 0xc6, 0xf0, 0x26, 0xc2, 0x35,
		0xc2, 0xcf, 0xc2, 0xbe, 0xba, 0x2e, 0x2f, 0x62, 0x73, 0xdb, 0x78, 0xfd,
		0xb1, 0x07, 0x61, 0x8d, 0x70, 0x40, 0x38, 0x28, 0x5c, 0x52, 0x67, 0xf3,
		0xe3, 0x8d, 0xb3, 0xf5, 0xc6, 0x2e, 0x84, 0x0b, 0x84, 0xf7, 0xca, 0xba,
		0xbc, 0xb9, 0x96, 0xfc, 0xb6, 0x5c, 0x97, 0x57, 0x74, 0x27, 0xf6, 0xe9,
		0xd6, 0xdb, 0xbb, 0xcf, 0x30, 0x76, 0x1b, 0x96, 0x8c, 0x09, 0xaf, 0x0b,
		0x21, 0x3c, 0x56, 0x4b, 0xfe, 0x0e, 0x6b, 0xfe, 0xdf, 0xa8, 0xa3, 0x6b,
		0x5a, 0xb7, 0x57, 0xdc, 0x95, 0x35, 0x70, 0x02, 0x57, 0xac, 0x30, 0x76,
		0x2b, 0x16, 0x10, 0x9e, 0x29, 0x25, 0xbf, 0x54, 0x4b, 0x9e, 0x97, 0x5b,
		0xa0, 0x2b, 0x9a, 0xb2, 0x3c, 0x3b, 0x77, 0x0b, 0x0f, 0xd5, 0xd2, 0xf6,
		0xfe, 0xcb, 0xd8, 0x49, 0x1c, 0x41, 0x98, 0x29, 0x25, 0x7f, 0x21, 0x5c,
		0x54, 0xff, 0xf3, 0xe8, 0xb9, 0x8b, 0x3d, 0x7a, 0xf6, 0x1b, 0x9e, 0x9d,
		0xdf, 0x0b, 0xe7, 0xd5, 0xd9, 0xb9, 0x6e, 0x68, 0xdc, 0x26, 0xfc, 0x82,
		0xb0, 0x4d, 0x58, 0x12, 0x7e, 0x15, 0xae, 0xae, 0x25, 0x3f, 0x77, 0xee,
		0xa3, 0x8f, 0x96, 0x6e, 0x97, 0xed, 0xc3, 0x52, 0x74, 0x08, 0x77, 0xd5,
		0xf2, 0xa6, 0xcb, 0x98, 0x8d, 0xf8, 0x81, 0xb2, 0x79, 0xbf, 0x28, 0x9c,
		0x2c, 0x3b, 0x73, 0xdd, 0xb8, 0x8f, 0xb0, 0xba, 0x49, 0xfa, 0x11, 0xf2,
		0x14, 0xc2, 0xce, 0xa1, 0xa2, 0xdf, 0xa8, 0x05, 0x7e, 0xad, 0xdb, 0x57,
		0x3e, 0x88, 0x70, 0x87, 0x70, 0xa2, 0x8c, 0xf9, 0xfb, 0xa1, 0xe4, 0x27,
		0x5c, 0xde, 0x2c, 0xfd, 0x08, 0x99, 0x44, 0x58, 0x2b, 0x1c, 0x2b, 0x25,
		0x2e, 0x08, 0x97, 0xd5, 0x22, 0xbb, 0xcd, 0xfb, 0x5b, 0x85, 0xe3, 0xe5,
		0xfc, 0xcb, 0xf5, 0xdc, 0x02, 0xb6, 0x34, 0xcc, 0x3e, 0x72, 0x3e, 0x30,
		0xbc, 0x4e, 0x87, 0xf0, 0x74, 0x2d, 0x33, 0xdc, 0x2c, 0xfc, 0x5e, 0x8e,
		0x7f, 0x76, 0xda, 0xc5, 0x6f, 0x47, 0xd3, 0xd4, 0x23, 0xe8, 0x7e, 0xca,
		0x85, 0xed, 0x64, 0x29, 0xf4, 0x0f, 0x61, 0xbd, 0x70, 0xa3, 0x70, 0xb8,
		0x1c, 0x9b, 0x3b, 0x6d, 0xa6, 0xe7, 0xc5, 0xef, 0x3f, 0x18, 0xc3, 0x37,
		0x94, 0x65, 0x61, 0x78, 0xad, 0x3e, 0x54, 0x7e, 0x1f, 0x13, 0x26, 0x6b,
		0xc9, 0xef, 0x63, 0x55, 0xd3, 0xc4, 0x23, 0x6c, 0x3b, 0xc2, 0x44, 0xb9,
		0x75, 0x1b, 0xfe, 0x9c, 0x12, 0xa6, 0x6a, 0xc9, 0xdf, 0x62, 0x6d, 0xdb,
		0xa8, 0xa3, 0x6d, 0x35, 0xe6, 0x10, 0x5e, 0xfd, 0x47, 0xd1, 0x33, 0xb5,
		0xe4, 0xc3, 0xb8, 0xae, 0x69, 0xca, 0x9e, 0xe8, 0x66, 0xf5, 0x06, 0xe1,
		0x48, 0x29, 0x79, 0x6f, 0x2d, 0x79, 0x11, 0xb7, 0xb7, 0x8d, 0xd7, 0x1f,
		0xe7, 0xe3, 0x13, 0x84, 0x07, 0x75, 0xef, 0x62, 0xac, 0xaa, 0x45, 0x6f,
		0x6f, 0x1b, 0xad, 0x7f, 0xae, 0xb7, 0xfc, 0xa8, 0xdd, 0x7d, 0x97, 0xf0,
		0x64, 0xd3, 0x44, 0x3d, 0xb6, 0x09, 0x2f, 0xe2, 0x2d, 0xf9, 0xc2, 0x4b,
		0x4a, 0x29, 0xa5, 0x94, 0x52, 0x4a, 0x29, 0xa5, 0x94, 0x52, 0x4a, 0x29,
		0xa5, 0x9e, 0xfa, 0x0b, 0xcb, 0xa1, 0x55, 0x6d, 0xd9, 0xdd, 0x6b, 0xc1,
		0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
	}),
)
var CloudDisconnect = widget.NewIcon(
	fyne.NewStaticResource("clouddisconnect.png", []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d,
		0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x5a, 0x00, 0x00, 0x00, 0x5a,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x38, 0xa8, 0x41, 0x02, 0x00, 0x00, 0x00,
		0x06, 0x62, 0x4b, 0x47, 0x44, 0x00, 0xff, 0x00, 0xff, 0x00, 0xff, 0xa0,
		0xbd, 0xa7, 0x93, 0x00, 0x00, 0x03, 0xbe, 0x49, 0x44, 0x41, 0x54, 0x78,
		0x9c, 0xed, 0xda, 0x5f, 0x88, 0x54, 0x55, 0x18, 0x00, 0xf0, 0xdf, 0x3a,
		0xee, 0xba, 0xbb, 0x33, 0xeb, 0xee, 0xfa, 0x27, 0xa2, 0xcc, 0x24, 0x21,
		0x22, 0xcb, 0x84, 0x20, 0x7c, 0x28, 0x29, 0x7b, 0x28, 0x88, 0x28, 0xd0,
		0xea, 0xa1, 0x5e, 0xd2, 0x7a, 0xaa, 0x08, 0xa5, 0x12, 0x0a, 0x7a, 0x4b,
		0xa2, 0x8c, 0xa0, 0x30, 0xa8, 0x87, 0xd0, 0x20, 0x7a, 0xc8, 0x8a, 0x4a,
		0x0a, 0x83, 0x40, 0x28, 0x22, 0x7c, 0x28, 0x29, 0x22, 0x2b, 0xc1, 0xca,
		0x22, 0xa5, 0x36, 0xd7, 0x75, 0xfe, 0xac, 0xba, 0xb3, 0xa7, 0x87, 0xab,
		0x64, 0xdb, 0x9d, 0x5d, 0xc9, 0xdc, 0xd9, 0x59, 0xbf, 0x1f, 0xdc, 0x97,
		0x99, 0xef, 0xce, 0x7c, 0xe7, 0x9b, 0xcb, 0x77, 0xce, 0x3d, 0x77, 0x08,
		0x21, 0x84, 0x10, 0x42, 0x08, 0x21, 0x84, 0x10, 0x42, 0x08, 0x21, 0x84,
		0x10, 0x42, 0x08, 0x21, 0x84, 0x10, 0x42, 0x08, 0x21, 0x84, 0x10, 0x42,
		0x08, 0x21, 0x84, 0x10, 0x42, 0x08, 0x21, 0x84, 0x10, 0x42, 0x08, 0x21,
		0x84, 0x10, 0x42, 0x08, 0x21, 0x84, 0x10, 0x42, 0x08, 0x21, 0x84, 0x10,
		0xce, 0x9e, 0x42, 0xb3, 0x13, 0x68, 0x21, 0xf3, 0x67, 0x70, 0x7f, 0xa2,
		0x6d, 0x26, 0xf7, 0x8d, 0xf2, 0x2b, 0xfe, 0x6c, 0x76, 0x52, 0xad, 0x60,
		0x0e, 0x2e, 0xee, 0x66, 0x63, 0x17, 0xeb, 0x27, 0x88, 0x5d, 0xdc, 0xc3,
		0xb7, 0xbd, 0xd4, 0x91, 0xd6, 0x30, 0x52, 0xe4, 0x10, 0x16, 0x9d, 0xf5,
		0x2c, 0x5b, 0x58, 0x5b, 0x0f, 0x5b, 0x8a, 0xd4, 0x0a, 0xd4, 0x1f, 0x62,
		0x64, 0x0e, 0x15, 0x5c, 0xdd, 0x20, 0xfe, 0xb2, 0x22, 0x83, 0x2f, 0x52,
		0xff, 0x9c, 0xb4, 0x83, 0x94, 0x48, 0x9b, 0x18, 0xe9, 0xe3, 0xd3, 0xc9,
		0x4c, 0xbc, 0xa5, 0x14, 0x58, 0xbb, 0x8c, 0x72, 0x85, 0xf4, 0xdd, 0x89,
		0xa2, 0x6d, 0x21, 0xf5, 0xf3, 0x15, 0xda, 0xc6, 0x84, 0x2f, 0xe8, 0xe1,
		0x8f, 0xd7, 0x18, 0x4d, 0x27, 0x62, 0x4f, 0x1e, 0x7b, 0x49, 0xb3, 0x28,
		0xe7, 0x9c, 0x13, 0x60, 0x2e, 0xef, 0x6c, 0x1d, 0x53, 0xb4, 0x3a, 0xe9,
		0x4a, 0xca, 0x05, 0xee, 0x3e, 0x35, 0xb4, 0x97, 0x9f, 0x9f, 0xa3, 0x3e,
		0xb6, 0xc8, 0x55, 0xd2, 0xe5, 0x94, 0x67, 0x4d, 0xdc, 0x72, 0xce, 0x5d,
		0x05, 0xee, 0x5d, 0x4d, 0x75, 0x6c, 0xf1, 0x3e, 0x23, 0x95, 0x18, 0x40,
		0x37, 0x3a, 0x7b, 0xf9, 0xfa, 0x51, 0x8e, 0x8d, 0x8d, 0x4b, 0xa4, 0xd5,
		0x54, 0x4a, 0x6c, 0x6b, 0xf6, 0x58, 0xa6, 0xba, 0xfe, 0x2e, 0x6a, 0x43,
		0x39, 0x05, 0xbc, 0x95, 0xca, 0x2c, 0x1e, 0xe9, 0x63, 0xeb, 0x2a, 0xaa,
		0xa3, 0x39, 0x31, 0x4f, 0x73, 0xbc, 0xc4, 0x37, 0xe8, 0x6a, 0xf6, 0x40,
		0xa6, 0xbc, 0x7e, 0x3e, 0x7c, 0x29, 0xa7, 0xef, 0xee, 0x26, 0x75, 0x50,
		0xbb, 0x90, 0x72, 0xde, 0x0f, 0xb1, 0x83, 0xd4, 0x9d, 0x2d, 0xe9, 0x16,
		0x34, 0x7b, 0x0c, 0xad, 0x62, 0xe5, 0x62, 0x8e, 0xe4, 0xb5, 0x85, 0x15,
		0x0c, 0xbf, 0x95, 0xf3, 0xfa, 0x7e, 0x52, 0x5f, 0xb6, 0x3a, 0xb9, 0xae,
		0xd9, 0xc9, 0xb7, 0x92, 0xb6, 0x5e, 0x7e, 0xf9, 0x24, 0x67, 0x52, 0xfc,
		0x32, 0xa7, 0xc8, 0xc7, 0x48, 0xcb, 0x18, 0xea, 0xe0, 0xf1, 0x66, 0x27,
		0xde, 0x72, 0xda, 0x59, 0xb7, 0x2a, 0x67, 0x52, 0xcc, 0x3b, 0x1e, 0xa6,
		0x5a, 0xe2, 0x63, 0xb1, 0x94, 0xfb, 0x4f, 0xfa, 0x3a, 0xa9, 0x1e, 0x98,
		0xa0, 0xc8, 0xdb, 0x19, 0xed, 0xe6, 0x20, 0xe6, 0x36, 0x3b, 0xe1, 0x96,
		0xd5, 0xc3, 0xb6, 0xa7, 0x72, 0x26, 0xc5, 0x93, 0xc7, 0x3e, 0xd2, 0xec,
		0xec, 0xa6, 0xe4, 0x9a, 0x33, 0xfd, 0xae, 0x19, 0xff, 0x43, 0xbe, 0x2d,
		0x2b, 0xb1, 0x7f, 0x88, 0xe1, 0x46, 0xef, 0x3f, 0x48, 0xad, 0xca, 0xf3,
		0xd8, 0x75, 0xa6, 0xdf, 0x75, 0x2e, 0x17, 0xba, 0xb3, 0xc0, 0x0d, 0xed,
		0xcc, 0x6c, 0x14, 0x70, 0x3b, 0x9d, 0x25, 0xae, 0x9f, 0xc4, 0x9c, 0xa6,
		0x9d, 0xb6, 0x3e, 0xde, 0xbe, 0x8d, 0x5a, 0x7d, 0x9c, 0xfe, 0x7c, 0x8c,
		0x34, 0x2f, 0x6b, 0x1d, 0x8d, 0x36, 0x9c, 0xc2, 0x78, 0x7a, 0x78, 0xf6,
		0x2a, 0xaa, 0x95, 0xd3, 0x58, 0x71, 0x6c, 0xa2, 0xde, 0xc7, 0xf6, 0x66,
		0xe7, 0xdc, 0x72, 0xda, 0x59, 0x73, 0x01, 0xd5, 0x83, 0xa7, 0x51, 0xe4,
		0x44, 0x2a, 0x67, 0x13, 0x62, 0x05, 0x97, 0x36, 0x3b, 0xf7, 0x56, 0xb2,
		0xb2, 0x97, 0xca, 0x9e, 0x06, 0x45, 0x1d, 0x6e, 0xf0, 0xfa, 0x93, 0x1c,
		0x9f, 0xcd, 0xeb, 0xcd, 0x4e, 0xbe, 0x55, 0x5c, 0x52, 0x64, 0x68, 0x67,
		0x83, 0x62, 0x7e, 0x41, 0xba, 0x91, 0x91, 0xbc, 0x8d, 0xa4, 0x81, 0x6c,
		0x8f, 0xa3, 0x26, 0xf6, 0x38, 0x26, 0xd4, 0xd1, 0xcf, 0x9e, 0x17, 0x72,
		0xf6, 0x96, 0x13, 0xe9, 0x28, 0x69, 0x31, 0x47, 0x3a, 0x38, 0xf4, 0x6a,
		0xe3, 0xbb, 0xc3, 0xa3, 0x25, 0x36, 0x37, 0x7b, 0x20, 0x53, 0xda, 0x1c,
		0x5e, 0xbe, 0x99, 0xe1, 0xbc, 0xab, 0x35, 0x91, 0x36, 0x70, 0x74, 0x36,
		0x1f, 0x61, 0x59, 0x6f, 0x83, 0xfe, 0xfd, 0x1b, 0xa9, 0x2b, 0xeb, 0xd5,
		0xf3, 0x9a, 0x3d, 0x9e, 0xa9, 0xea, 0xa6, 0xf3, 0xa9, 0x1c, 0x6a, 0x50,
		0xe4, 0x5d, 0xa4, 0x22, 0x83, 0x38, 0x0f, 0x66, 0xb3, 0xf9, 0x4e, 0x6a,
		0x79, 0xb1, 0xb7, 0x50, 0x15, 0x4f, 0x55, 0x72, 0xb5, 0x97, 0xd8, 0xff,
		0x41, 0x83, 0x22, 0xd7, 0x48, 0x8b, 0xb2, 0x75, 0xf2, 0x1d, 0xa7, 0x9c,
		0x53, 0x2c, 0x71, 0xe0, 0xbd, 0x31, 0xb1, 0x3b, 0xb3, 0x1f, 0x64, 0x48,
		0xf4, 0xe9, 0x7f, 0xeb, 0x60, 0xc3, 0x4a, 0xca, 0x8d, 0x96, 0x6e, 0xeb,
		0xb3, 0x96, 0xf1, 0x7e, 0xce, 0xa9, 0xcb, 0xfb, 0xa8, 0xfe, 0x48, 0x1a,
		0x22, 0x6d, 0x24, 0x75, 0x64, 0x7f, 0x35, 0xb8, 0x67, 0x92, 0x87, 0xd0,
		0x12, 0x16, 0x76, 0x53, 0xde, 0x3b, 0xce, 0x2a, 0xa3, 0xc8, 0x61, 0xcc,
		0xcf, 0x3b, 0xb9, 0x87, 0xc7, 0x96, 0x50, 0x1e, 0x20, 0xbd, 0x9b, 0xad,
		0x3a, 0x0e, 0xa3, 0x34, 0xb9, 0x43, 0x68, 0x01, 0x33, 0x78, 0xe2, 0x01,
		0x8e, 0xe7, 0x15, 0xb9, 0x4e, 0x5a, 0xca, 0x91, 0x02, 0x6b, 0xc7, 0xf9,
		0x88, 0xb6, 0x22, 0xcf, 0xb4, 0x67, 0x57, 0xfd, 0x3e, 0x2c, 0x9f, 0xac,
		0xdc, 0x5b, 0xcd, 0x8a, 0x8b, 0x1a, 0x3c, 0xae, 0x7a, 0x85, 0xd1, 0x5e,
		0x76, 0x3b, 0xbd, 0x8d, 0xfc, 0x86, 0x9b, 0x4e, 0x21, 0xb3, 0xe6, 0xae,
		0x9c, 0xfe, 0xfc, 0xfb, 0xdf, 0xb7, 0xd4, 0x4b, 0x27, 0x33, 0x99, 0xe9,
		0xbc, 0x4d, 0x7a, 0xe0, 0xfb, 0x6c, 0x02, 0xfb, 0x87, 0x75, 0xd4, 0x46,
		0xd9, 0x22, 0xfb, 0x67, 0x52, 0xf8, 0x1f, 0x14, 0x4a, 0xfc, 0xf4, 0xc6,
		0x29, 0x57, 0xf3, 0x9b, 0xd9, 0x04, 0x38, 0x80, 0x9e, 0x66, 0x27, 0x37,
		0xdd, 0x2c, 0x29, 0x32, 0x78, 0x05, 0x83, 0x73, 0x29, 0xf7, 0xf2, 0x03,
		0xae, 0x6d, 0x76, 0x52, 0xd3, 0x55, 0xa7, 0x6c, 0xc5, 0xb0, 0x50, 0x3c,
		0xc5, 0x9e, 0xfe, 0xfe, 0x02, 0xac, 0xdd, 0x70, 0x07, 0xf8, 0xa7, 0xc1,
		0x3c, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae, 0x42, 0x60,
		0x82,
	}),
)
