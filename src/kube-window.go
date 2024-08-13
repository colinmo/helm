package main

import (
	"fmt"
	"log"
	"os/exec"
	"strings"

	fyne "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"vonexplaino.com/m/v2/hq/kube"
	vonwidget "vonexplaino.com/m/v2/hq/widget"
)

func kubernetesWindowSetup() {
	kubernetesWindow.Resize(fyne.NewSize(800, 500))
	kubernetesWindow.Hide()
	kubernetesWindow.SetCloseIntercept(func() {
		kubernetesWindow.Hide()
	})
	kubernetesWindow.SetContent(setupKubenetesWindow())
}

func setupKubenetesWindow() *fyne.Container {
	// Variable holders
	deployments := binding.BindStringList(&[]string{})
	pods := binding.BindStringList(&[]string{})

	// Get kubes available
	contexts, e := kube.GetContexts()
	if e != nil {
		fmt.Printf("Failed to get contexts %s\n", e.Error())
	}
	namespaces, e := kube.GetNamespaces()
	if e != nil {
		fmt.Printf("Failed to get contexts %s\n", e.Error())
	}
	// Selectors
	namespaceSelector := widget.NewSelect(
		namespaces,
		func(selected string) {
		},
	)
	namespaceSelector.SetSelected(kube.GetNamespace())

	contextSelector := widget.NewSelect(
		contexts,
		func(selected string) {
		},
	)
	contextSelector.SetSelected(kube.GetContext())
	dataContainer := container.NewStack()

	var podSelectList *widget.List
	var deploySelectList *widget.List
	deploySelectList = widget.NewListWithData(
		deployments,
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewLabel(""),
			)
		},
		func(i binding.DataItem, o fyne.CanvasObject) {
			lab, _ := i.(binding.String).Get()
			myIcon := theme.SearchIcon()
			if kube.FilteredByDeployment == lab {
				myIcon = theme.ZoomInIcon()
			}

			filterButton := widget.NewButtonWithIcon("", myIcon, func() {})
			filterButton.OnTapped = func() {
				if kube.FilteredByDeployment == "" || kube.FilteredByDeployment != lab {
					kube.FilteredByDeployment = lab
					filterButton.Icon = theme.ZoomInIcon()
				} else {
					kube.FilteredByDeployment = ""
					filterButton.Icon = theme.SearchIcon()
				}
				filterButton.Refresh()
				deploySelectList.Refresh()
				x, err := kube.GetPods()
				if err == nil {
					pods.Set(x)
				}
				podSelectList.Refresh()
			}
			o.(*fyne.Container).Objects = []fyne.CanvasObject{
				filterButton,
				widget.NewButtonWithIcon("", theme.ComputerIcon(), func() {
					dataContainer.Objects = []fyne.CanvasObject{monitorDeployment(lab, contextSelector.Selected, namespaceSelector.Selected)}
					dataContainer.Refresh()
				}),
				widget.NewLabel(lab),
				layout.NewSpacer(),
			}
		})

	podSelectList = widget.NewListWithData(
		pods,
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewLabel(""),
			)
		},
		func(i binding.DataItem, o fyne.CanvasObject) {
			lab, _ := i.(binding.String).Get()
			deets := strings.Split(lab, "|")
			switch deets[1] {
			case "Pending":
				deets[1] = "…"
			case "Running":
				deets[1] = "☺"
			case "Succeeded":
				deets[1] = "✔"
			case "Failed":
				deets[1] = "☠"
			case "Unknown":
				deets[1] = "‽"
			}
			o.(*fyne.Container).Objects = []fyne.CanvasObject{
				widget.NewButtonWithIcon("", theme.ComputerIcon(), func() {
					dataContainer.Objects = []fyne.CanvasObject{monitorPod(lab, contextSelector.Selected, namespaceSelector.Selected)}
					dataContainer.Refresh()
				}),
				widget.NewLabel(deets[1]),
				widget.NewLabel(deets[0]),
				layout.NewSpacer(),
			}
		},
	)

	contextSelector.OnChanged = func(new string) {
		kube.SwitchContext(new)
		namespaces, e = kube.GetNamespaces()
		namespaceSelector.Options = namespaces
		if e != nil {
			fmt.Printf("Failed to get contexts %s\n", e.Error())
		}
		deployments.Set([]string{})
		pods.Set([]string{})
	}
	namespaceSelector.OnChanged = func(new string) {
		kube.SwitchNamespace(new)
		deployments.Set([]string{})
		pods.Set([]string{})
	}

	return container.NewBorder(
		container.NewHBox(
			widget.NewLabel("Context"),
			contextSelector,
			widget.NewLabel("Namespace"),
			namespaceSelector,
			widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
				x, _ := kube.GetDeployments()
				deployments.Set(x)
				y, _ := kube.GetPods()
				pods.Set(y)
			}),
		),
		nil,
		nil,
		nil,
		container.NewGridWithColumns(
			2,
			// Deployments
			container.NewGridWithColumns(
				2,
				container.NewBorder(
					widget.NewLabelWithStyle("Deployments", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
					widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
						x, _ := kube.GetDeployments()
						deployments.Set(x)
					}),
					nil,
					nil,
					container.NewStack(
						deploySelectList,
					),
				),
				// Pods
				container.NewBorder(
					widget.NewLabelWithStyle("Pods", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
					widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
						x, _ := kube.GetPods()
						pods.Set(x)
					}),
					nil,
					nil,
					podSelectList,
				), // Monitor
			),
			dataContainer,
		),
	)
}

func monitorDeployment(deploymentId, context, namespace string) (returnme *fyne.Container) {
	returnme = container.NewBorder(
		widget.NewLabel(fmt.Sprintf("Deployment %s", deploymentId)),
		nil,
		nil,
		nil,
		container.NewBorder(
			container.NewGridWrap(
				fyne.NewSize(80, 25),
				widget.NewButton(
					"Describe",
					func() {
						commandText := fmt.Sprintf(
							`kubectl describe deployment %s --context=%s --namespace=%s`,
							deploymentId,
							context,
							namespace,
						)
						kubernetesWindow.Clipboard().SetContent(commandText)

						cmdArray := []string{`/usr/bin/osascript`,
							"-e",
							fmt.Sprintf("tell app \"Terminal\"\ndo script \"%s\"\nend tell", commandText),
						}
						cmd := exec.Command(cmdArray[0], cmdArray[1:]...)
						err := cmd.Start()
						if err != nil {
							log.Fatal(err)
						}
					}),
				widget.NewButton("Delete", func() {
					dialog.ShowConfirm(
						fmt.Sprintf("Delete %s", deploymentId),
						"Deleting this pod will allow the deployment to recreate a new pod.",
						func(ok bool) {
							if ok {
								kube.DeleteDeployment(deploymentId)
							}
						},
						kubernetesWindow,
					)
				}),
			),
			nil,
			nil,
			nil,
			nil,
		),
	)
	return
}

func monitorPod(podId, context, namespace string) (returnme *fyne.Container) {
	// Memory monitor
	maxMemory := 0
	memoryMonitor := vonwidget.NewLinegraphWidget(
		0,
		400,
		[]int{},
		"",
		"MiB",
	)
	timingResults := binding.BindIntList(&[]int{})
	timingResults.AddListener(binding.NewDataListener(func() {
		x, _ := timingResults.Get()
		memoryMonitor.UpdateItemsAndMax(x, maxMemory)
		memoryMonitor.Refresh()
	}))
	podIdOnly := strings.Split(podId, "|")[0]
	returnme = container.NewBorder(
		widget.NewLabel(fmt.Sprintf("Pod %s", podId)),
		nil,
		nil,
		nil,
		container.NewBorder(
			container.NewGridWrap(
				fyne.NewSize(80, 25),
				widget.NewButton(
					"Logs",
					func() {
						commandText := fmt.Sprintf(
							`kubectl logs -f %s --context=%s --namespace=%s`,
							podIdOnly,
							context,
							namespace,
						)

						kubernetesWindow.Clipboard().SetContent(commandText)
						cmdArray := []string{`/usr/bin/osascript`,
							"-e",
							fmt.Sprintf("tell app \"Terminal\"\ndo script \"%s\"\nend tell", commandText),
						}
						cmd := exec.Command(cmdArray[0], cmdArray[1:]...)
						err := cmd.Start()
						if err != nil {
							log.Fatal(err)
						}
					},
				),
				widget.NewButton("Delete", func() {
					dialog.ShowConfirm(
						fmt.Sprintf("Delete %s", podIdOnly),
						"Deleting this pod will allow the deployment to recreate a new pod.",
						func(ok bool) {
							if ok {
								kube.DeletePod(podIdOnly)
							}
						},
						kubernetesWindow,
					)
				}),
				widget.NewButton("Describe", func() {
					commandText := fmt.Sprintf(
						`kubectl describe pod %s --context=%s --namespace=%s`,
						podIdOnly,
						context,
						namespace,
					)

					kubernetesWindow.Clipboard().SetContent(commandText)
					cmdArray := []string{`/usr/bin/osascript`,
						"-e",
						fmt.Sprintf("tell app \"Terminal\"\ndo script \"%s\"\nend tell", commandText),
					}
					cmd := exec.Command(cmdArray[0], cmdArray[1:]...)
					err := cmd.Start()
					if err != nil {
						log.Fatal(err)
					}

				}),
				widget.NewButton("Monitor", func() {
					kube.GetMemoryForPod(podIdOnly, timingResults, &maxMemory)
				}),
			),
			nil,
			nil,
			nil,
			memoryMonitor,
		),
	)
	return

}
