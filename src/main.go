package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	icon "vonexplaino.com/m/v2/hq/icon"
	"vonexplaino.com/m/v2/hq/iserver"
	"vonexplaino.com/m/v2/hq/kube"
	"vonexplaino.com/m/v2/hq/tasks"

	fyne "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
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

var iServer iserver.IServerStruct

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
	tasks.StartLocalServers()
	if appPreferences.TaskPreferences.MSPlannerActive {
		go waitAndRun(func() {
			tasks.Planner.Download()
			taskWindowRefresh("Planner")
		})
	}
	if appPreferences.TaskPreferences.JiraActive {
		// Regular task refresh
		go waitAndRun(func() {
			tasks.Jira.Download()
			taskWindowRefresh("Jira")
		})
	}
	if appPreferences.TaskPreferences.SnowActive {
		// Regular task refresh
		go waitAndRun(func() {
			tasks.Snow.Download(
				func() { taskWindowRefresh("SNIncidents") },
				func() { taskWindowRefresh("SNRequests") },
				func() { taskWindowRefresh("SNTeamIncidents") },
			)
		})
	}
	go waitAndRun(func() {
		tasks.Zettle.Download(filepath.Dir(appPreferences.ZettlekastenHome))
		taskWindowRefresh("Zettle")
	})
}

func waitAndRun(runThis func()) {
	for {
		time.Sleep(5 * time.Minute)
		hour := time.Now().Hour()
		weekday := time.Now().Weekday()
		if hour > 8 && hour < 17 && weekday > 0 && weekday < 6 {
			runThis()
		}
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
			appPreferences.TaskPreferences.MSPlannerActive,
			appPreferences.TaskPreferences.SnowActive,
			taskWindowRefresh,
			activeTaskStatusUpdate,
			filepath.Dir(appPreferences.ZettlekastenHome),
		)
		iServer.Init("http://localhost:86/")
	}()

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

func taskWindowSetup() {
	taskWindow.Resize(fyne.NewSize(430, 550))
	taskWindow.Hide()
	taskStatusWidget := widget.NewLabelWithData(AppStatus.TaskTaskStatus)
	connectionStatusContainer = container.NewGridWithColumns(4)
	connectionStatusBox(false, "D")
	connectionStatusBox(false, "K")
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
	TaskTabsIndexes["Dashboard"] = 0
	TaskTabs.Append(
		container.NewTabItem(
			"Dashboard",
			container.NewBorder(
				nil,
				nil,
				nil,
				nil,
				container.NewAdaptiveGrid(2, widget.NewLabel("# of open tasks\nStatus of projects\nLookup of iServer for RSDF\n* Other things")),
			),
		),
	)
	if appPreferences.TaskPreferences.MSPlannerActive {
		TaskTabsIndexes["Planner"] = len(TaskTabsIndexes)
		TaskTabs.Append(
			container.NewTabItem(
				"Planner",
				container.NewBorder(
					nil,
					nil,
					nil,
					nil,
					container.NewWithoutLayout(),
				),
			),
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
	// Kubernetes tasks
	TaskTabsIndexes["Kube"] = len(TaskTabsIndexes)
	TaskTabs.Append(
		container.NewTabItem("Kubernetes", container.NewBorder(
			nil, nil, nil, nil,
			setupKubenetesWindow(),
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
					thisID := x.BusObRecId
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
								browser.OpenURL(tasks.Snow.BaseURL + "/now/sow/record/task/" + thisID)
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

			TaskTabs.Items[TaskTabsIndexes["SNIncidents"]].Text = fmt.Sprintf("Incidents (%d)", len(tasks.Snow.MyIncidents))
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
			TaskTabs.Items[TaskTabsIndexes["SNTeamIncidents"]].Text = fmt.Sprintf("Team Incidents (%d)", len(tasks.Snow.TeamIncidents))
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
			TaskTabs.Items[TaskTabsIndexes["SNRequests"]].Text = fmt.Sprintf("Requests (%d)", len(tasks.Snow.LoggedIncidents))
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
					container.NewStack(
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
				col4.Objects = append(col4.Objects, container.NewStack(
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
							tasks.Planner.Download()
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
		TaskTabs.Items[TaskTabsIndexes["Planner"]].Text = fmt.Sprintf("Plans (%d)", len(tasks.Planner.MyTasks))
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
					container.NewStack(
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
						container.NewStack(
							getJiraTypeIconFor(x.Type, taskIcons),
							widget.NewLabelWithStyle(x.Type[0:1], fyne.TextAlignCenter, fyne.TextStyle{Bold: true})),
						blocked,
						widget.NewLabelWithStyle(fmt.Sprintf("[%s] %s", thisID, x.Title), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
					))
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
				col4.Objects = append(col4.Objects, container.NewStack(
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
		TaskTabs.Items[TaskTabsIndexes["Jira"]].Text = fmt.Sprintf("Jira (%d)", len(tasks.Jira.MyTasks))
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
				col4.Objects = append(col4.Objects, container.NewStack(
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
		TaskTabs.Items[TaskTabsIndexes["Zettle"]].Text = fmt.Sprintf("Zettlekasten (%d)", len(tasks.Zettle.MyTasks))
	}
	/*
		if _, ok := TaskTabsIndexes["Kube"]; ok && (specific == "" || specific == "Kube") {

		}
		if _, ok := TaskTabsIndexes["Dashboard"]; ok && (specific == "" || specific == "Dashboard") {

		}
	*/
	taskWindow.Content().Refresh()
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
