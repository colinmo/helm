package main

import (
	"os"
	"os/user"
	"path"
	"path/filepath"
	"time"

	fyne "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"vonexplaino.com/m/v2/hq/kube"
	"vonexplaino.com/m/v2/hq/tasks"
)

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
	appPreferences.KubePreferences.Active = thisApp.Preferences().BoolWithFallback("KubeActive", false)
	appPreferences.KubePreferences.Context = thisApp.Preferences().StringWithFallback("KubeContext", "")
	appPreferences.KubePreferences.Namespace = thisApp.Preferences().StringWithFallback("KubeNamespace", "")
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
	kube.Setup(appPreferences.KubePreferences.Context, appPreferences.KubePreferences.Namespace)
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
	// Kubernetes
	kubeActive := widget.NewCheck("Active", func(res bool) {})
	kubeActive.SetChecked(appPreferences.KubePreferences.Active)
	kubeContext := widget.NewEntry()
	kubeContext.SetText(appPreferences.KubePreferences.Context)
	kubeNamespace := widget.NewEntry()
	kubeNamespace.SetText(appPreferences.KubePreferences.Namespace)

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

		appPreferences.KubePreferences.Active = kubeActive.Checked
		thisApp.Preferences().SetBool("KubeActive", appPreferences.KubePreferences.Active)
		appPreferences.KubePreferences.Context = kubeContext.Text
		thisApp.Preferences().SetString("KubeContext", appPreferences.KubePreferences.Context)
		appPreferences.KubePreferences.Namespace = kubeNamespace.Text
		thisApp.Preferences().SetString("KubeNamespace", appPreferences.KubePreferences.Namespace)
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
			widget.NewLabel(""),
			widget.NewLabelWithStyle("Kubernetes", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			widget.NewLabel("Active"),
			kubeActive,
			widget.NewLabel("Context"),
			kubeContext,
			widget.NewLabel("Namespace"),
			kubeNamespace,
		),
	)
}
