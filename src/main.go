package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image/color"
	"log"
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
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/data/validation"
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
var snowConnectionActive *fyne.Container
var msConnectionActive *fyne.Container
var jiraConnectionActive *fyne.Container
var ztConnectionActive *fyne.Container
var connectionStatusContainer *fyne.Container
var connectionStatusBox = func(bool, string) {}
var stringDateFormat = "20060102T15:04:05"

var allAvailableTeams map[string]string
var alphaTeams []string

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
		case "S":
			button.OnTapped = func() {
				if onl {
					tasks.Snow.DownloadIncidents(func() { taskWindowRefresh("SNIncidents") })
					tasks.Snow.DownloadTeamIncidents(func() { taskWindowRefresh("SNTeamIncidents") })
					tasks.Snow.DownloadMyRequests(func() { taskWindowRefresh("SNRequests") })
				}
			}
			snowConnectionActive.Objects = container.NewStack(
				button,
				icon,
			).Objects
			snowConnectionActive.Refresh()
		case "J":
			button.OnTapped = func() {
				if onl {
					tasks.Jira.Download()
					taskWindowRefresh("Jira")
				}
			}
			jiraConnectionActive.Objects = container.NewStack(
				button,
				icon,
			).Objects
			jiraConnectionActive.Refresh()
		case "M":
			button.OnTapped = func() {
				tasks.Planner.Download("")
				taskWindowRefresh("Planner")
			}
			msConnectionActive.Objects = container.NewStack(
				button,
				icon,
			).Objects
			msConnectionActive.Refresh()
		case "Z":
			button.OnTapped = func() {
				tasks.Zettle.Download(filepath.Dir(appPreferences.ZettlekastenHome))
				taskWindowRefresh("Zettle")
			}
			ztConnectionActive.Objects = container.NewStack(
				button,
				icon,
			).Objects
			ztConnectionActive.Refresh()
		}
		taskWindow.Content().Refresh()
	}
	jiraConnectionActive = container.NewStack()
	msConnectionActive = container.NewStack()
	snowConnectionActive = container.NewStack()
	ztConnectionActive = container.NewStack()
	tasks.InitTasks(
		&appPreferences.TaskPreferences,
		connectionStatusBox,
		taskWindowRefresh,
		activeTaskStatusUpdate,
	)
	tasks.StartLocalServers()
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
		// Regular task refresh
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
	if appPreferences.TaskPreferences.SnowActive {
		// Regular task refresh
		go func() {
			for {
				time.Sleep(5 * time.Minute)
				hour := time.Now().Hour()
				weekday := time.Now().Weekday()
				if hour > 8 && hour < 17 && weekday > 0 && weekday < 6 {
					tasks.Snow.Download(
						func() { taskWindowRefresh("SNIncidents") },
						func() { taskWindowRefresh("SNRequests") },
						func() { taskWindowRefresh("SNTeamIncidents") },
					)
				}
			}
		}()
	}
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			hour := time.Now().Hour()
			weekday := time.Now().Weekday()
			if hour > 8 && hour < 17 && weekday > 0 && weekday < 6 {
				tasks.Zettle.Download(filepath.Dir(appPreferences.ZettlekastenHome))
				taskWindowRefresh("Zettle")
			}
		}
	}()
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
			appPreferences.TaskPreferences.MSPlannerActive,
			appPreferences.TaskPreferences.SnowActive,
			taskWindowRefresh,
			activeTaskStatusUpdate,
			filepath.Dir(appPreferences.ZettlekastenHome),
		)
	}()
	if desk, ok := thisApp.(desktop.App); ok {
		// desk.SetSystemTrayIcon(fyne.NewStaticResource("Systray", icon.IconWhite))
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
	appPreferences.TaskPreferences.MSPlannerActive = thisApp.Preferences().BoolWithFallback("MSPlannerActive", false)
	appPreferences.TaskPreferences.MSGroups = thisApp.Preferences().StringWithFallback("MSGroups", "")
	appPreferences.TaskPreferences.JiraActive = thisApp.Preferences().BoolWithFallback("JiraActive", false)
	appPreferences.TaskPreferences.JiraKey = thisApp.Preferences().StringWithFallback("JiraKey", "")
	appPreferences.TaskPreferences.JiraUsername = thisApp.Preferences().StringWithFallback("JiraUsername", "")
	appPreferences.TaskPreferences.JiraDefaultProject = thisApp.Preferences().StringWithFallback("JiraDefaultProject", "")
	appPreferences.TaskPreferences.PriorityOverride = thisApp.Preferences().StringWithFallback("PriorityOverride", "")
	appPreferences.TaskPreferences.SnowActive = thisApp.Preferences().BoolWithFallback("SnowActive", false)
	appPreferences.TaskPreferences.SnowUser = thisApp.Preferences().StringWithFallback("SnowUser", "")
	appPreferences.TaskPreferences.SnowGroup = thisApp.Preferences().StringWithFallback("SnowGroup", "")
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
	jiraDefaultProject := widget.NewEntry()
	jiraDefaultProject.SetText(appPreferences.TaskPreferences.JiraDefaultProject)
	// Dynamics
	dynamicsActive := widget.NewCheck("Active", func(res bool) {})
	dynamicsActive.SetChecked(appPreferences.TaskPreferences.DynamicsActive)
	dynamicsKey := widget.NewPasswordEntry()
	dynamicsKey.SetText(appPreferences.TaskPreferences.DynamicsKey)
	// Service Now
	snowActive := widget.NewCheck("Active", func(res bool) {})
	snowActive.SetChecked(appPreferences.TaskPreferences.SnowActive)
	snowUser := widget.NewEntry()
	snowUser.SetText(appPreferences.TaskPreferences.SnowUser)
	snowGroup := widget.NewEntry()
	snowGroup.SetText(appPreferences.TaskPreferences.SnowGroup)

	preferencesWindow.Resize(fyne.NewSize(500, 500))
	preferencesWindow.Hide()
	preferencesWindow.SetCloseIntercept(func() {
		preferencesWindow.Hide()
		// SavePreferences
		appPreferences.ZettlekastenHome = zettlePath.Text
		thisApp.Preferences().SetString("ZettlekastenHome", appPreferences.ZettlekastenHome)
		appPreferences.TaskPreferences.JiraProjectHome = jiraPath.Text
		thisApp.Preferences().SetString("JiraProjectHome", appPreferences.TaskPreferences.JiraProjectHome)
		appPreferences.TaskPreferences.JiraDefaultProject = jiraDefaultProject.Text
		thisApp.Preferences().SetString("JiraDefaultProject", appPreferences.TaskPreferences.JiraDefaultProject)
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

		appPreferences.TaskPreferences.SnowActive = snowActive.Checked
		thisApp.Preferences().SetBool("SnowActive", appPreferences.TaskPreferences.SnowActive)
		appPreferences.TaskPreferences.SnowUser = snowUser.Text
		thisApp.Preferences().SetString("SnowUser", appPreferences.TaskPreferences.SnowUser)
		appPreferences.TaskPreferences.SnowGroup = snowGroup.Text
		thisApp.Preferences().SetString("SnowGroup", appPreferences.TaskPreferences.SnowGroup)
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
			widget.NewLabel("Default project"),
			jiraDefaultProject,
			widget.NewLabel(""),
			widget.NewLabelWithStyle("Service Now", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			widget.NewLabel("Active"),
			snowActive,
			widget.NewLabel("UserID"),
			snowUser,
			widget.NewLabel("GroupID"),
			snowGroup,
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
	connectionStatusContainer = container.NewGridWithColumns(4)
	connectionStatusBox(false, "M")
	connectionStatusBox(false, "J")
	connectionStatusBox(false, "S")
	connectionStatusBox(true, "Z")
	connectionStatusContainer = container.NewGridWithColumns(4,
		msConnectionActive,
		jiraConnectionActive,
		snowConnectionActive,
		ztConnectionActive,
	)
	TaskTabsIndexes = map[string]int{}
	TaskTabs = container.NewAppTabs()
	if appPreferences.TaskPreferences.MSPlannerActive {
		TaskTabsIndexes["Planner"] = len(TaskTabsIndexes)
		TaskTabs.Append(
			container.NewTabItem("Planner", container.NewBorder(
				nil,
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
				nil,
				nil,
				nil,
				nil,
				container.NewWithoutLayout(),
			)),
		)
	}
	if appPreferences.TaskPreferences.SnowActive {
		TaskTabsIndexes["SNIncidents"] = len(TaskTabsIndexes)
		TaskTabs.Append(
			container.NewTabItem("Incidents", container.NewBorder(
				nil,
				nil,
				nil,
				nil,
				container.NewWithoutLayout(),
			)),
		)
		TaskTabsIndexes["SNTeamIncidents"] = len(TaskTabsIndexes)
		TaskTabs.Append(
			container.NewTabItem("Team Incidents", container.NewBorder(
				nil,
				nil,
				nil,
				nil,
				container.NewWithoutLayout(),
			)),
		)
		TaskTabsIndexes["SNRequests"] = len(TaskTabsIndexes)
		TaskTabs.Append(
			container.NewTabItem("My Requests", container.NewBorder(
				nil,
				nil,
				nil,
				nil,
				container.NewWithoutLayout(),
			)),
		)
	}
	// Zet tasks
	TaskTabsIndexes["Zettle"] = len(TaskTabsIndexes)
	TaskTabs.Append(
		container.NewTabItem("Zettlekasten", container.NewBorder(
			nil, nil, nil, nil,
			container.NewWithoutLayout(),
		)),
	)
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
	priorityIcons := setupPriorityIcons()
	taskIcons := setupJiraTaskTypeIcons()

	if appPreferences.TaskPreferences.SnowActive {
		if _, ok := TaskTabsIndexes["SNIncidents"]; ok && (specific == "" || specific == "SNIncidents") {
			var list2 fyne.CanvasObject
			if len(tasks.Snow.MyIncidents) == 0 {
				list2 = widget.NewLabel("No incidents")
			} else {
				col0 := container.NewVBox(widget.NewRichTextFromMarkdown(`### `))
				col1 := container.NewVBox(widget.NewRichTextFromMarkdown(`### Incident`))
				col3 := container.NewVBox(widget.NewRichTextFromMarkdown(`### Age`))
				col4 := container.NewVBox(widget.NewRichTextFromMarkdown(`### Priority`))
				col5 := container.NewVBox(widget.NewRichTextFromMarkdown(`### Status`))
				for _, x := range tasks.Snow.MyIncidents {
					thisID := x.ID
					myPriority := x.PriorityOverride
					if x.Priority != x.PriorityOverride {
						myPriority = fmt.Sprintf("%s(%s)", x.PriorityOverride, x.Priority)
					}
					tempVar := ""
					col0.Objects = append(
						col0.Objects,
						container.NewStack(
							widget.NewLabel(""),
							newTappableIcon(theme.InfoIcon(), func(_ *fyne.PointEvent) {
								browser.OpenURL(tasks.Snow.BaseURL + "/now/sow/record/incident/" + thisID)
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
									delete(tasks.PriorityOverrides.SNow, thisID)
								} else {
									tasks.PriorityOverrides.SNow[thisID] = tempVar
								}
								tasks.SavePriorityOverride()
							},
							taskWindow,
						)
					}
					col4.Objects = append(col4.Objects, container.NewStack(
						getPriorityIconFor(x.PriorityOverride, priorityIcons),
						newTappableLabelWithStyle(
							myPriority,
							fyne.TextAlignCenter,
							fyne.TextStyle{},
							tempFunc)))
					col5.Objects = append(col5.Objects, widget.NewLabel(tasks.SnowStatuses[x.Status]))
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
			TaskTabs.Items[TaskTabsIndexes["SNIncidents"]].Content = container.NewBorder(
				widget.NewToolbar(
					widget.NewToolbarAction(
						theme.ViewRefreshIcon(),
						func() {
							tasks.Snow.DownloadIncidents(func() { taskWindowRefresh("SNIncidents") })
						},
					),
					widget.NewToolbarAction(
						theme.DocumentCreateIcon(),
						func() {
							items := tasks.SnowIncidentCreate{
								AffectedUser:     "me",
								Service:          "",
								ServiceOffering:  "",
								ShortDescription: "",
								ContactType:      "Self-service",
								Impact:           "3 - Individual",
								Urgency:          "3 - Low",
								AssignmentGroup:  "",
								AssignedTo:       "",
								Description:      "",
							}
							saving := tasks.SnowIncidentCreate{
								AffectedUser:     appPreferences.TaskPreferences.SnowUser[1:],
								Service:          "",
								ServiceOffering:  "",
								OpenedBy:         appPreferences.TaskPreferences.SnowUser[1:],
								ShortDescription: "",
								ContactType:      "",
								Impact:           "",
								Urgency:          "",
								AssignmentGroup:  "",
								AssignedTo:       "",
								Description:      "",
							}
							descWidget := widget.NewEntryWithData(binding.BindString(&items.Description))
							descWidget.MultiLine = true
							foundAffects := map[string]string{}
							foundServices := map[string]string{}
							foundAffectsOfferings := map[string]string{}
							foundAssignmentGroups := map[string]string{}
							foundAssignedTo := map[string]string{}

							var widgets map[string]fyne.CanvasObject
							widgets = map[string]fyne.CanvasObject{
								"AffectedUser": container.NewBorder(
									nil, nil, nil,
									newTappableIcon(theme.SearchIcon(), func(_ *fyne.PointEvent) {
										var r struct {
											Result []struct {
												ID    string `json:"sys_id"`
												Name  string `json:"name"`
												Email string `json:"email"`
											} `json:"result"`
										}
										// sys_user
										result, err := tasks.Snow.GetAnyTable(
											"sys_user",
											[]string{"sys_id", "name", "email"},
											map[string]string{"name": "CONTAINS" + items.AffectedUser},
											"ORDERBYname",
											0,
										)
										if err == nil {
											results := []string{}
											foundAffects = map[string]string{}
											json.Unmarshal(result, &r)
											for _, e := range r.Result {
												results = append(results, e.Name+" ("+e.Email+")")
												foundAffects[e.Name] = e.ID
											}
											widgets["AffectedUser"].(*fyne.Container).Objects[0].(*widget.SelectEntry).SetOptions(results)
										} else {
											log.Fatal(err)
										}
									}),
									widget.NewSelectEntry([]string{})),
								"Service": container.NewBorder(
									nil, nil, nil,
									newTappableIcon(theme.SearchIcon(), func(_ *fyne.PointEvent) {
										// cmdb_ci_service
										var r struct {
											Result []struct {
												ID   string `json:"sys_id"`
												Name string `json:"name"`
											} `json:"result"`
										}
										// sys_user
										result, err := tasks.Snow.GetAnyTable(
											"cmdb_ci_service",
											[]string{"sys_id", "name"},
											map[string]string{
												"name": "CONTAINS" + items.Service},
											"ORDERBYname",
											0,
										)
										if err == nil {
											results := []string{}
											foundServices = map[string]string{}
											json.Unmarshal(result, &r)
											for _, e := range r.Result {
												results = append(results, e.Name)
												foundServices[e.Name] = e.ID
											}
											widgets["Service"].(*fyne.Container).Objects[0].(*widget.SelectEntry).SetOptions(results)
										} else {
											log.Fatal(err)
										}
									}),
									widget.NewSelectEntry([]string{})),
								"ServiceOffering": container.NewBorder(
									nil, nil, nil,
									newTappableIcon(theme.SearchIcon(), func(_ *fyne.PointEvent) {
										// service_offering
										var r struct {
											Result []struct {
												ID   string `json:"sys_id"`
												Name string `json:"name"`
											} `json:"result"`
										}
										criteria := map[string]string{}
										if len(items.ServiceOffering) > 0 {
											criteria["name"] = "CONTAINS" + items.ServiceOffering
										}
										if len(foundServices[items.Service]) > 0 {
											criteria["parent"] = "=" + foundServices[items.Service]
										}
										result, err := tasks.Snow.GetAnyTable(
											"service_offering",
											[]string{"sys_id", "name"},
											criteria,
											"ORDERBYparent^ORDERBYname",
											0,
										)
										fmt.Printf("Found: %v\n", foundServices)
										if err == nil {
											results := []string{}
											foundAffectsOfferings = map[string]string{}
											json.Unmarshal(result, &r)
											for _, e := range r.Result {
												results = append(results, e.Name)
												foundAffectsOfferings[e.Name] = e.ID
											}
											widgets["ServiceOffering"].(*fyne.Container).Objects[0].(*widget.SelectEntry).SetOptions(results)
										} else {
											log.Fatal(err)
										}
									}),
									widget.NewSelectEntry([]string{})),
								"ShortDescription": container.NewBorder(
									nil, nil, nil, nil,
									widget.NewEntryWithData(binding.BindString(&items.ShortDescription))),
								"ContactType": container.NewBorder(
									nil, nil, nil, nil,
									widget.NewSelect(tasks.SNContactTypeLabels, func(s string) { items.ContactType = tasks.SNContactTypes[s] })),
								"Impact": container.NewBorder(
									nil, nil, nil, nil,
									widget.NewSelect(tasks.SNImpactLabels, func(s string) { items.Impact = tasks.SNImpact[s] })),
								"Urgency": container.NewBorder(
									nil, nil, nil, nil,
									widget.NewSelect(tasks.SNUrgencyLabels, func(s string) { items.Urgency = tasks.SNUrgency[s] })),
								"AssignmentGroup": container.NewBorder(
									nil, nil, nil,
									newTappableIcon(theme.SearchIcon(), func(_ *fyne.PointEvent) {
										// sys_user_group
										var r struct {
											Result []struct {
												ID   string `json:"sys_id"`
												Name string `json:"name"`
											} `json:"result"`
										}
										criteria := map[string]string{}
										if len(items.AssignmentGroup) > 0 {
											criteria["name"] = "CONTAINS" + items.AssignmentGroup
										}
										result, err := tasks.Snow.GetAnyTable(
											"sys_user_group",
											[]string{"sys_id", "name"},
											criteria,
											"ORDERBYname",
											0,
										)
										if err == nil {
											results := []string{}
											foundAssignmentGroups = map[string]string{}
											json.Unmarshal(result, &r)
											for _, e := range r.Result {
												results = append(results, e.Name)
												foundAssignmentGroups[e.Name] = e.ID
											}
											widgets["AssignmentGroup"].(*fyne.Container).Objects[0].(*widget.SelectEntry).SetOptions(results)
										} else {
											log.Fatal(err)
										}
									}),
									widget.NewSelectEntry([]string{})),
								"AssignedTo": container.NewBorder(
									nil, nil, nil,
									newTappableIcon(theme.SearchIcon(), func(_ *fyne.PointEvent) {
										// sys_user_grmember
										var r struct {
											Result []struct {
												ID   string `json:"user.sys_id"`
												Name string `json:"user.name"`
											} `json:"result"`
										}
										criteria := map[string]string{}
										if len(items.AssignedTo) > 0 {
											criteria["user.name"] = "LIKE" + items.AssignedTo
										}
										if len(foundAssignmentGroups[items.AssignmentGroup]) > 0 {
											criteria["group.sys_id"] = "=" + foundAssignmentGroups[items.AssignmentGroup]
										}
										result, err := tasks.Snow.GetAnyTable(
											"sys_user_grmember",
											[]string{"user.sys_id", "user.name", "group.name"},
											criteria,
											"ORDERBYuser.name",
											0,
										)
										if err == nil {
											results := []string{}
											foundAssignedTo = map[string]string{}
											json.Unmarshal(result, &r)
											for _, e := range r.Result {
												results = append(results, e.Name)
												foundAssignedTo[e.Name] = e.ID
											}
											widgets["AssignedTo"].(*fyne.Container).Objects[0].(*widget.SelectEntry).SetOptions(results)
										} else {
											log.Fatal(err)
										}
									}),
									widget.NewSelectEntry([]string{})),
								"Description": container.NewBorder(
									nil, nil, nil, nil,
									descWidget),
							}
							widgets["AffectedUser"].(*fyne.Container).Objects[0].(*widget.SelectEntry).Entry.SetText(items.AffectedUser)
							widgets["AffectedUser"].(*fyne.Container).Objects[0].(*widget.SelectEntry).Entry.Validator = validation.NewRegexp(".+", "Field is required")
							widgets["AffectedUser"].(*fyne.Container).Objects[0].(*widget.SelectEntry).Entry.OnChanged = func(x string) {
								items.AffectedUser = x
							}
							widgets["Service"].(*fyne.Container).Objects[0].(*widget.SelectEntry).Entry.SetText(items.Service)
							widgets["Service"].(*fyne.Container).Objects[0].(*widget.SelectEntry).Entry.Validator = validation.NewRegexp(".+", "Field is required")
							widgets["Service"].(*fyne.Container).Objects[0].(*widget.SelectEntry).Entry.OnChanged = func(x string) {
								items.Service = x
							}
							widgets["ServiceOffering"].(*fyne.Container).Objects[0].(*widget.SelectEntry).Entry.SetText(items.ServiceOffering)
							widgets["ServiceOffering"].(*fyne.Container).Objects[0].(*widget.SelectEntry).Entry.Validator = validation.NewRegexp(".+", "Field is required")
							widgets["ServiceOffering"].(*fyne.Container).Objects[0].(*widget.SelectEntry).Entry.OnChanged = func(x string) {
								items.ServiceOffering = x
							}
							widgets["AssignmentGroup"].(*fyne.Container).Objects[0].(*widget.SelectEntry).Entry.SetText(items.AssignmentGroup)
							widgets["AssignmentGroup"].(*fyne.Container).Objects[0].(*widget.SelectEntry).Entry.Validator = validation.NewRegexp(".+", "Field is required")
							widgets["AssignmentGroup"].(*fyne.Container).Objects[0].(*widget.SelectEntry).Entry.OnChanged = func(x string) {
								items.AssignmentGroup = x
							}
							widgets["AssignedTo"].(*fyne.Container).Objects[0].(*widget.SelectEntry).Entry.SetText(items.AssignedTo)
							widgets["AssignedTo"].(*fyne.Container).Objects[0].(*widget.SelectEntry).Entry.OnChanged = func(x string) {
								items.AssignedTo = x
							}
							widgets["ContactType"].(*fyne.Container).Objects[0].(*widget.Select).SetSelected(items.ContactType)
							widgets["Impact"].(*fyne.Container).Objects[0].(*widget.Select).Selected = items.Impact
							widgets["Urgency"].(*fyne.Container).Objects[0].(*widget.Select).Selected = items.Urgency
							widgets["ShortDescription"].(*fyne.Container).Objects[0].(*widget.Entry).Validator = validation.NewRegexp(".+", "Field is required")
							widgets["Description"].(*fyne.Container).Objects[0].(*widget.Entry).Validator = validation.NewRegexp(".+", "Field is required")
							incidentWindow := thisApp.NewWindow("New Incident")
							incidentWindow.SetContent(
								container.NewBorder(
									widget.NewToolbar(
										widget.NewToolbarAction(
											theme.DocumentSaveIcon(),
											func() {
												// Validate form

												// Convert Items to Saving
												if items.AffectedUser == "me" {
													saving.AffectedUser = appPreferences.TaskPreferences.SnowUser[1:]
												}
												saving.Service = foundServices[items.Service]
												saving.ServiceOffering = foundServices[items.ServiceOffering]
												saving.ShortDescription = items.ShortDescription
												saving.ContactType = tasks.SNContactTypes[items.ContactType]
												saving.Impact = tasks.SNImpact[items.Impact]
												saving.Urgency = tasks.SNUrgency[items.Urgency]
												saving.AssignmentGroup = foundAssignmentGroups[items.AssignmentGroup]
												if x, ok := foundAssignedTo[items.AssignedTo]; ok {
													saving.AssignedTo = x
												}
												missing := []string{}
												for name, value := range map[string]string{
													"Affected User":     items.AffectedUser,
													"Service":           items.Service,
													"Service Offering":  items.ServiceOffering,
													"Short Description": items.ShortDescription,
													"Contact Type":      items.ContactType,
													"Impact":            items.Impact,
													"Urgency":           items.Urgency,
													"Assignment Group":  items.AssignmentGroup,
													"Description":       items.Description} {
													if value == "" {
														missing = append(missing, name)
													}
												}
												if len(missing) > 0 {
													dialog.ShowError(
														fmt.Errorf("you must fill values for %s", strings.Join(missing, ", ")),
														incidentWindow,
													)
												} else {
													// Save
													number, url, err := tasks.Snow.CreateNewIncident(saving)
													if err == nil && url > "" {
														me := dialog.NewCustom(
															"Saved incident as "+number,
															"Ok",
															container.NewVBox(
																widget.NewLabel("Created incident successfully"),
																container.NewHBox(
																	widget.NewButton("Visit", func() {
																		browser.OpenURL(url)
																	}),
																),
															),
															incidentWindow,
														)
														me.SetOnClosed(func() {
															incidentWindow.Close()
														})
														me.Show()
													} else {
														dialog.ShowError(
															err,
															incidentWindow,
														)
													}
												}
											},
										),
									),
									nil,
									nil,
									nil,
									widget.NewForm(
										widget.NewFormItem(
											"Affected User",
											widgets["AffectedUser"],
										),
										widget.NewFormItem(
											"Service",
											widgets["Service"],
										),
										widget.NewFormItem(
											"Service Offering",
											widgets["ServiceOffering"],
										),
										widget.NewFormItem(
											"Short Description",
											widgets["ShortDescription"],
										),
										widget.NewFormItem(
											"Contact Type",
											widgets["ContactType"],
										),
										widget.NewFormItem(
											"Impact",
											widgets["Impact"],
										),
										widget.NewFormItem(
											"Urgency",
											widgets["Urgency"],
										),
										widget.NewFormItem(
											"Assignment Group",
											widgets["AssignmentGroup"],
										),
										widget.NewFormItem(
											"Assigned To",
											widgets["AssignedTo"],
										),
										widget.NewFormItem(
											"Description",
											widgets["Description"],
										),
									),
								),
							)
							incidentWindow.SetCloseIntercept(func() {
								incidentWindow.Hide()
							})
							incidentWindow.Content().Refresh()
							incidentWindow.Resize(fyne.Size{Width: 500, Height: 600})
							incidentWindow.Show()
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
		if _, ok := TaskTabsIndexes["SNTeamIncidents"]; ok && (specific == "" || specific == "SNTeamIncidents") {
			var list3 fyne.CanvasObject
			if len(tasks.Snow.TeamIncidents) == 0 {
				list3 = widget.NewLabel("No incidents")
			} else {
				col0 := container.NewVBox(widget.NewRichTextFromMarkdown(`### `))
				col1 := container.NewVBox(widget.NewRichTextFromMarkdown(`### Incident`))
				col2 := container.NewVBox(widget.NewRichTextFromMarkdown(`### Owner`))
				col3 := container.NewVBox(widget.NewRichTextFromMarkdown(`### Age`))
				col4 := container.NewVBox(widget.NewRichTextFromMarkdown(`### Priority`))
				col5 := container.NewVBox(widget.NewRichTextFromMarkdown(`### Status`))
				for _, x := range tasks.Snow.TeamIncidents {
					thisID := x.BusObRecId
					col0.Objects = append(
						col0.Objects,
						container.NewStack(
							widget.NewLabel(""),
							newTappableIcon(theme.InfoIcon(), func(_ *fyne.PointEvent) {
								browser.OpenURL(tasks.Snow.BaseURL + "/now/sow/record/incident/" + thisID)
							}),
						))
					col2.Objects = append(col2.Objects, widget.NewLabel(x.Owner))
					col1.Objects = append(col1.Objects,
						widget.NewLabelWithStyle(fmt.Sprintf("[%s] %s", x.ID, x.Title), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
					col3.Objects = append(col3.Objects, widget.NewLabel(dateSinceNowInString(x.CreatedDateTime)))
					col4.Objects = append(col4.Objects, container.NewStack(
						getPriorityIconFor(x.PriorityOverride, priorityIcons),
						widget.NewLabelWithStyle(x.PriorityOverride, fyne.TextAlignCenter, fyne.TextStyle{})))
					col5.Objects = append(col5.Objects, widget.NewLabel(tasks.SnowStatuses[x.Status]))
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

			TaskTabs.Items[TaskTabsIndexes["SNTeamIncidents"]].Content = container.NewBorder(
				widget.NewToolbar(
					widget.NewToolbarAction(
						theme.ViewRefreshIcon(),
						func() {
							tasks.Snow.DownloadTeamIncidents(func() { taskWindowRefresh("SNTeamIncidents") })
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
		if _, ok := TaskTabsIndexes["SNRequests"]; ok && (specific == "" || specific == "SNRequests") {
			var list4 fyne.CanvasObject
			if len(tasks.Snow.LoggedIncidents) == 0 {
				list4 = widget.NewLabel("No requests")
			} else {
				col0 := container.NewVBox(widget.NewRichTextFromMarkdown(`### `))
				col1 := container.NewVBox(widget.NewRichTextFromMarkdown(`### Incident`))
				col3 := container.NewVBox(widget.NewRichTextFromMarkdown(`### Age`))
				col4 := container.NewVBox(widget.NewRichTextFromMarkdown(`### Priority`))
				col5 := container.NewVBox(widget.NewRichTextFromMarkdown(`### Status`))
				for _, x := range tasks.Snow.LoggedIncidents {
					thisID := x.BusObRecId
					col0.Objects = append(
						col0.Objects,
						container.NewStack(
							widget.NewLabel(""),
							newTappableIcon(theme.InfoIcon(), func(_ *fyne.PointEvent) {
								browser.OpenURL(tasks.Snow.BaseURL + "/now/sow/record/incident/" + thisID)
							}),
						))
					col1.Objects = append(col1.Objects,
						widget.NewLabelWithStyle(fmt.Sprintf("[%s] %s", x.ID, x.Title), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
					col3.Objects = append(col3.Objects, widget.NewLabel(dateSinceNowInString(x.CreatedDateTime)))
					col4.Objects = append(col4.Objects, container.NewStack(
						getPriorityIconFor(x.PriorityOverride, priorityIcons),
						widget.NewLabelWithStyle(x.PriorityOverride, fyne.TextAlignCenter, fyne.TextStyle{})))
					col5.Objects = append(col5.Objects, widget.NewLabel(tasks.SnowStatuses[x.Status]))
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

			TaskTabs.Items[TaskTabsIndexes["SNRequests"]].Content = container.NewBorder(
				widget.NewToolbar(
					widget.NewToolbarAction(
						theme.ViewRefreshIcon(),
						func() {
							tasks.Snow.DownloadMyRequests(func() { taskWindowRefresh("SNRequests") })
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
		// Get the teams
		go func() {
			allAvailableTeams, alphaTeams = tasks.Jira.TeamsLookup()
		}()
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
					theme.DocumentCreateIcon(),
					func() {
						createNewJiraTicket()
					},
				),
			),
			nil,
			nil,
			nil,
			list,
		)
	}
	if _, ok := TaskTabsIndexes["Zettle"]; ok && (specific == "" || specific == "Zettle") {
		// MY PLANNER
		var list5 fyne.CanvasObject
		if len(tasks.Zettle.MyTasks) == 0 {
			list5 = widget.NewLabel("No requests")
		} else {
			col0 := container.NewVBox(widget.NewRichTextFromMarkdown(`### `))
			col1 := container.NewVBox(widget.NewRichTextFromMarkdown(`### Title`))
			col3 := container.NewVBox(widget.NewRichTextFromMarkdown(`### Age`))
			col4 := container.NewVBox(widget.NewRichTextFromMarkdown(`### Priority`))
			col5 := container.NewVBox(widget.NewRichTextFromMarkdown(`### Status`))
			for _, x := range tasks.Zettle.MyTasks {
				thisID := x.ID
				myPriority := x.PriorityOverride
				if x.Priority != x.PriorityOverride {
					myPriority = fmt.Sprintf("%s(%s)", x.PriorityOverride, x.Priority)
				}
				tempVar := ""
				col0.Objects = append(
					col0.Objects,
					container.NewStack(
						widget.NewLabel(""),
						newTappableIcon(theme.InfoIcon(), func(_ *fyne.PointEvent) {
							fmt.Printf("%s\n", thisID)
							browser.OpenFile(thisID)
						}),
					))
				col1.Objects = append(col1.Objects,
					widget.NewLabelWithStyle(x.Title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
				col3.Objects = append(col3.Objects, widget.NewLabel(dateSinceNowInString(x.CreatedDateTime)))
				iconContainer := container.NewStack(getPriorityIconFor(x.PriorityOverride, priorityIcons))
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

		TaskTabs.Items[TaskTabsIndexes["Zettle"]].Content = container.NewBorder(
			widget.NewToolbar(
				widget.NewToolbarAction(
					theme.ViewRefreshIcon(),
					func() {
						go func() {
							tasks.Zettle.Download(filepath.Dir(appPreferences.ZettlekastenHome))
							taskWindowRefresh("Zettle")
						}()
					},
				),
			),
			nil,
			nil,
			nil,
			list5,
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

func createNewJiraTicket() {
	// Create new ticket window
	newJiraTicket := thisApp.NewWindow("New Jira Ticket")

	newJiraTicket.Resize(fyne.NewSize(500, 500))
	newJiraTicket.Hide()
	newJiraTicket.SetCloseIntercept(func() {
		// Prompt for save if changes
		newJiraTicket.Hide()
		// SavePreferences
	})
	// Fields
	type AtlassianContentType struct {
		Type    string                 `json:"type"`
		Text    string                 `json:"text,omitempty"`
		Content []AtlassianContentType `json:"content,omitempty"`
	}
	type AtlassianDocumentType struct {
		Type    string                 `json:"type"`
		Version int                    `json:"version"`
		Content []AtlassianContentType `json:"content"`
	}
	type IssueObject struct {
		Fields struct {
			Assignee struct {
				ID string `json:"id"`
			} `json:"assignee"`
			Team        string                `json:"customfield_10001"`
			Summary     string                `json:"summary"`
			Description AtlassianDocumentType `json:"description"`
			Issuetype   struct {
				Name string `json:"name"`
			} `json:"issuetype"`
			Priority struct {
				Name string `json:"name"`
			} `json:"priority"`
			Project struct {
				ID string `json:"id"`
			} `json:"project"`
			Reporter struct {
				ID string `json:"id"`
			} `json:"reporter"`
			Parent struct {
				Key string `json:"key,omitempty"`
			} `json:"parent,omitempty"`
			EpicName string `json:"customfield_10011,omitempty"`
		} `json:"fields"`
	}
	projectEntry := widget.NewSelectEntry(
		[]string{},
	)
	allProjects := map[string]string{}
	go func() {
		find, projectOrder := tasks.Jira.ProjectLookup(projectEntry.SelectedText())
		projectEntry.SetOptions(projectOrder)
		for k, v := range find {
			allProjects[k] = v
		}
		if appPreferences.TaskPreferences.JiraDefaultProject > "" {
			projectEntry.SetText(appPreferences.TaskPreferences.JiraDefaultProject)
		}
		if len(projectOrder) == 1 {
			projectEntry.SetText(projectOrder[0])
		}
	}()
	summaryEntry := widget.NewEntry()
	descriptionEntry := widget.NewMultiLineEntry()
	prioritySelect := widget.NewSelect([]string{"Highest", "High", "Medium", "Low", "Lowest"}, func(c string) {})
	prioritySelect.SetSelected("Medium")
	assigneeSelect := widget.NewSelectEntry(
		[]string{
			"Me", "Other",
		},
	)
	assigneeSelect.SetText("Me")
	reporterSelect := widget.NewSelectEntry(
		[]string{
			"Me", "Other",
		},
	)
	reporterSelect.SetText("Me")
	teamSelect := widget.NewSelect(
		alphaTeams,
		func(thisString string) {
			fmt.Printf("Selected: %s %s\n", thisString, allAvailableTeams[thisString])
		},
	)
	parentOptions := map[string]string{}
	selectableParentOptions := []string{}
	parentSelect := widget.NewSelectEntry(
		[]string{},
	)
	epicNameEntry := widget.NewEntry()
	issueTypeEntry := widget.NewSelect(
		[]string{
			"Story", "Epic", "Initiative",
		},
		func(chosen string) {
			switch chosen {
			case "Initiative":
				parentSelect.Disable()
				epicNameEntry.Disable()
			case "Epic":
				parentSelect.Enable()
				epicNameEntry.Enable()
			default:
				parentSelect.Enable()
				epicNameEntry.Disable()
			}
		},
	)
	foundPeople := map[string]string{}

	newJiraTicket.SetContent(
		container.NewBorder(
			widget.NewToolbar(
				widget.NewToolbarAction(
					theme.DocumentSaveIcon(),
					func() {
						go func() {
							// Validate
							// Save
							saveMe := IssueObject{}
							saveMe.Fields.Team = allAvailableTeams[teamSelect.Selected]
							saveMe.Fields.Summary = summaryEntry.Text
							saveMe.Fields.Description = AtlassianDocumentType{
								Type:    "doc",
								Version: 1,
								Content: []AtlassianContentType{{
									Type:    "paragraph",
									Content: []AtlassianContentType{},
								}},
							}
							paras := strings.Split(descriptionEntry.Text, "\n")
							for _, p := range paras {
								saveMe.Fields.Description.Content[0].Content = append(saveMe.Fields.Description.Content[0].Content, AtlassianContentType{
									Text: p + "\n",
									Type: "text",
								})
							}
							saveMe.Fields.Issuetype.Name = issueTypeEntry.Selected
							saveMe.Fields.Priority.Name = prioritySelect.Selected
							saveMe.Fields.Project.ID = allProjects[projectEntry.Entry.Text]
							if issueTypeEntry.Selected == "Initiative" {
								saveMe.Fields.Parent = struct {
									Key string "json:\"key,omitempty\""
								}{}
							} else {
								saveMe.Fields.Parent.Key = parentOptions[parentSelect.Entry.Text]
							}
							if len(epicNameEntry.Text) > 0 && issueTypeEntry.Selected == "Epic" {
								saveMe.Fields.EpicName = epicNameEntry.Text
							}
							if assigneeSelect.Entry.Text == "Me" {
								saveMe.Fields.Assignee.ID = tasks.Jira.GetMyId()
							} else {
								saveMe.Fields.Assignee.ID = foundPeople[assigneeSelect.Entry.Text]
							}
							if reporterSelect.Entry.Text == "Me" {
								saveMe.Fields.Reporter.ID = tasks.Jira.GetMyId()
							} else {
								saveMe.Fields.Reporter.ID = foundPeople[reporterSelect.Entry.Text]
							}

							objectToSend, _ := json.MarshalIndent(saveMe, "", " ")
							id, self, err := tasks.Jira.CreateTask(objectToSend)
							var deepdeep dialog.Dialog
							if err != nil {
								deepdeep = dialog.NewCustom(
									"Could not save",
									"Well biscuits",
									widget.NewLabel(fmt.Sprintf("%s", err)),
									newJiraTicket,
								)
							} else {
								deepdeep = dialog.NewCustom(
									"Saved",
									"Woohoo!",
									newTappableLabel(
										fmt.Sprintf("Issue %s has been created. Click here to see it.", id),
										func(_ *fyne.PointEvent) {
											browser.OpenURL(self)
										},
									),
									newJiraTicket,
								)

							}
							deepdeep.Show()
						}()
					},
				),
			), nil, nil, nil,
			container.New(
				layout.NewFormLayout(),
				widget.NewLabel("Project"), container.NewBorder(nil, nil, nil, nil, projectEntry), // widget.NewButtonWithIcon("", theme.SearchIcon(), func() {})
				widget.NewLabel("Issue type"), issueTypeEntry,
				widget.NewLabel("Priority"), prioritySelect,
				widget.NewLabel("Summary"), summaryEntry,
				widget.NewLabel("Description"), descriptionEntry,
				widget.NewLabel("Assignee"), container.NewBorder(nil, nil, nil, widget.NewButtonWithIcon("", theme.SearchIcon(), func() {
					go func() {
						find, order := tasks.Jira.PersonLookup(assigneeSelect.Text)
						assigneeSelect.SetOptions(order)
						for k, v := range find {
							foundPeople[k] = v
						}
						if len(order) == 1 {
							assigneeSelect.SetText(order[0])
						}
					}()
				}), assigneeSelect),
				widget.NewLabel("Reporter"), container.NewBorder(nil, nil, nil, widget.NewButtonWithIcon("", theme.SearchIcon(), func() {
					go func() {
						find, order := tasks.Jira.PersonLookup(reporterSelect.Text)
						reporterSelect.SetOptions(order)
						for k, v := range find {
							foundPeople[k] = v
						}
						if len(order) == 1 {
							reporterSelect.SetText(order[0])
						}
					}()
				}), reporterSelect),
				widget.NewLabel("Team"), teamSelect,
				widget.NewLabel("Parent"), container.NewBorder(nil, nil, nil, widget.NewButtonWithIcon("", theme.SearchIcon(), func() {
					issueType := issueTypeEntry.Selected
					parentOptions = map[string]string{}
					selectableParentOptions = []string{}
					founds := []tasks.TaskResponseStruct{}
					switch issueType {
					case "Story":
						founds = tasks.Jira.RelatedIssuesLookupByType("Epic", parentSelect.Text)
					case "Epic":
						founds = tasks.Jira.RelatedIssuesLookupByType("Initiative", parentSelect.Text)
					}
					for _, x := range founds {
						parentOptions[x.Title] = x.ID
						selectableParentOptions = append(selectableParentOptions, x.Title)
					}
					parentSelect.SetOptions(selectableParentOptions)
				}), parentSelect),
				widget.NewLabel("Epic name"), epicNameEntry,
			),
		),
	)

	newJiraTicket.Show()
}
