package main

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	icon "vonexplaino.com/m/v2/hq/icon"
	"vonexplaino.com/m/v2/hq/kube"
	"vonexplaino.com/m/v2/hq/tasks"

	fyne "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/pkg/browser"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

type AppStatusStruct struct {
	CurrentZettleDBDate time.Time
	CurrentZettleDKB    binding.String
	TaskTaskCount       int
	TaskTaskStatus      binding.String
}

type AppPreferencesStruct struct {
	ZettlekastenHome string
	TaskPreferences  tasks.TaskPreferencesStruct
	KubePreferences  kube.PreferencesStruct
	LinkPreferences  []struct {
		Label string
		URL   string
	}
}

var thisApp fyne.App
var mainWindow fyne.Window
var preferencesWindow fyne.Window
var taskWindow fyne.Window
var kubernetesWindow fyne.Window

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
	loc, _ := time.LoadLocation("Australia/Brisbane")
	AppStatus = AppStatusStruct{
		CurrentZettleDBDate: time.Now().Local(),
		CurrentZettleDKB:    binding.NewString(),
		TaskTaskStatus:      binding.NewString(),
		TaskTaskCount:       0,
	}
	AppStatus.CurrentZettleDKB.Set(zettleFileName(time.Now().In(loc).Local()))
}

var scheduledTasksList map[string]func()

func pullUpdates() {
	ticker := time.NewTicker(5 * time.Minute)
	quit := make(chan struct{})
	// Run on interval
	go func() {
		for {
			select {
			case <-ticker.C:
				hour := time.Now().Hour()
				weekday := time.Now().Weekday()
				if hour > 8 && hour < 17 && weekday > 0 && weekday < 6 {
					for _, y := range scheduledTasksList {
						go func() { y() }()
					}
				}
				// do stuff
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()
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
				tasks.Planner.Download()
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
}

func startTasks() {
	scheduledTasksList = map[string]func(){}
	if appPreferences.TaskPreferences.JiraActive {
		tasks.Jira.Init()
		scheduledTasksList["Jira"] = func() {
			tasks.Jira.Download()
			taskWindowRefresh("Jira")
		}
		tasks.Jira.Download()
		taskWindowRefresh("Jira")
	}
	if appPreferences.TaskPreferences.MSPlannerActive {
		tasks.Planner.Init(
			"http://localhost:84/",
			time.Now(),
		)
		scheduledTasksList["Planner"] = func() {
			tasks.Planner.Download()
			taskWindowRefresh("Planner")
		}
	}
	if appPreferences.TaskPreferences.SnowActive {
		tasks.Snow.Init(
			"http://localhost:84/",
			time.Now(),
			func() { taskWindowRefresh("SNIncidents") },
			func() { taskWindowRefresh("SNRequests") },
			func() { taskWindowRefresh("SNTeamIncidents") },
		)
		scheduledTasksList["Jira"] = func() {
			tasks.Snow.Download()
		}
	}
	tasks.StartLocalServers()

	tasks.Zettle.Init()
	scheduledTasksList["Zettle"] = func() {
		tasks.Zettle.Download(filepath.Dir(appPreferences.ZettlekastenHome))
		taskWindowRefresh("Zettle")
	}
	scheduledTasksList["Zettle"]()
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
	startTasks()
	pullUpdates()

	//kubernetesWindow = thisApp.NewWindow("Kubernetes")
	//kubernetesWindowSetup()

	if desk, ok := thisApp.(desktop.App); ok {
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
			//fyne.NewMenuItem("Kubernetes", func() {
			//	kubernetesWindow.Show()
			//}),
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
							container.NewStack(
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
			widget.NewButtonWithIcon("", theme.CheckButtonCheckedIcon(), func() {
				lines := strings.Split(markdownInput.Text, "\n")
				cursorRow := markdownInput.CursorRow
				if markdownInput.CursorColumn == 0 {
					lines[cursorRow] = " * [ ] \n" + lines[cursorRow][markdownInput.CursorColumn:]
				} else {
					lines[cursorRow] = lines[cursorRow][0:markdownInput.CursorColumn] + "\n * [ ] " + lines[cursorRow][markdownInput.CursorColumn:]
				}
				markdownInput.SetText(strings.Join(lines, "\n"))
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
	content := container.NewBorder(
		menu,
		widget.NewLabelWithData(AppStatus.CurrentZettleDKB),
		nil,
		nil,
		markdownInput,
	)
	mainWindow.SetContent(content)
	mainWindow.SetCloseIntercept(func() {
		mainWindow.Hide()
		// Save contents
		x, _ := AppStatus.CurrentZettleDKB.Get()
		saveZettle(markdownInput.Text, x)
	})
}
