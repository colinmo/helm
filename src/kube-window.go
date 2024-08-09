package main

import (
	"fmt"

	fyne "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"vonexplaino.com/m/v2/hq/kube"
	vonwidget "vonexplaino.com/m/v2/hq/widget"
)

func kubernetesWindowSetup() {
	kubernetesWindow.Resize(fyne.NewSize(500, 500))
	kubernetesWindow.Hide()
	kubernetesWindow.SetCloseIntercept(func() {
		kubernetesWindow.Hide()
	})
	kubernetesWindow.SetContent(setupKubenetesWindow())
}

func setupKubenetesWindow() *fyne.Container {
	deployments := binding.BindStringList(&[]string{})
	pods := binding.BindStringList(&[]string{})
	timingResults := binding.BindIntList(&[]int{})
	contexts, e := kube.GetContexts()
	if e != nil {
		fmt.Printf("Failed to get contexts %s\n", e.Error())
	}
	namespaces, e := kube.GetNamespaces()
	if e != nil {
		fmt.Printf("Failed to get contexts %s\n", e.Error())
	}

	maxMemory := 0
	memoryMonitor := vonwidget.NewLinegraphWidget(
		0,
		400,
		[]int{},
		"",
		"MiB",
	)
	timingResults.AddListener(binding.NewDataListener(func() {
		x, _ := timingResults.Get()
		memoryMonitor.UpdateItemsAndMax(x, maxMemory)
		memoryMonitor.Refresh()
	}))
	namespaceSelector := widget.NewSelect(
		namespaces,
		func(selected string) {
			kube.SwitchNamespace(selected)
		},
	)
	namespaceSelector.SetSelected(kube.GetNamespace())

	contextSelector := widget.NewSelect(
		contexts,
		func(selected string) {
			kube.SwitchContext(selected)
			namespaces, e = kube.GetNamespaces()
			namespaceSelector.Options = namespaces
			if e != nil {
				fmt.Printf("Failed to get contexts %s\n", e.Error())
			}
		},
	)
	contextSelector.SetSelected(kube.GetContext())

	return container.NewBorder(
		container.NewHBox(
			widget.NewLabel("Context"),
			contextSelector,
			widget.NewLabel("Namespace"),
			namespaceSelector,
		),

		nil,
		nil,
		nil, container.NewGridWrap(
			fyne.NewSize(300, 300),
			// Deployments
			container.NewBorder(
				widget.NewLabelWithStyle("Deployments", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
				widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
					x, err := kube.GetDeployments()
					if err == nil {
						deployments.Set(x)
					}
				}),
				nil,
				nil,
				widget.NewListWithData(
					deployments,
					func() fyne.CanvasObject {
						return container.NewHBox(
							widget.NewLabel(""),
						)
					},
					func(i binding.DataItem, o fyne.CanvasObject) {
						lab, _ := i.(binding.String).Get()
						o.(*fyne.Container).Objects = []fyne.CanvasObject{
							widget.NewLabel(lab),
							layout.NewSpacer(),
							widget.NewButtonWithIcon("", theme.SearchIcon(), func() {
								if kube.FilteredByDeployment == "" || kube.FilteredByDeployment != lab {
									kube.FilteredByDeployment = lab
								} else {
									kube.FilteredByDeployment = ""
								}
								x, err := kube.GetPods()
								if err == nil {
									pods.Set(x)
								}
							}),
						}
					}),
			),
			// Pods
			container.NewBorder(
				widget.NewLabelWithStyle("Pods", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
				widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
					x, err := kube.GetPods()
					if err == nil {
						pods.Set(x)
					}
				}), nil, nil,
				widget.NewListWithData(
					pods,
					func() fyne.CanvasObject {
						return container.NewHBox(
							widget.NewLabel(""),
						)
					},
					func(i binding.DataItem, o fyne.CanvasObject) {
						lab, _ := i.(binding.String).Get()
						o.(*fyne.Container).Objects = []fyne.CanvasObject{
							widget.NewLabel(lab),
							layout.NewSpacer(),
							widget.NewButtonWithIcon("", theme.ZoomInIcon(), func() {
								timingResults.Set([]int{})
								kube.GetMemoryForPod(lab, timingResults, &maxMemory)
							}),
						}
					},
				),
			),
			// Monitor
			container.NewBorder(
				memoryMonitor,
				nil, nil, nil, nil,
			),
		),
	)
}
