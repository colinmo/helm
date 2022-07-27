package main

import (
	"fmt"
	"os"
	"time"

	"github.com/tebeka/selenium"
	"github.com/tebeka/selenium/firefox"
)

func GetToParentalControlsAndClick(lookfor string) {
	const (
		// These paths will be different on your system.
		geckoDriverPath = `F:\Dropbox\swap\golang\helm\src\geckodriver.exe`
		port            = 8080
	)
	opts := []selenium.ServiceOption{
		selenium.GeckoDriver(geckoDriverPath), // Specify the path to GeckoDriver in order to use Firefox.
		selenium.Output(os.Stderr),            // Output debug information to STDERR.
	}
	selenium.SetDebug(false)
	service, err := selenium.NewGeckoDriverService(
		geckoDriverPath,
		port,
		opts...)
	if err != nil {
		panic(err) // panic is used only as an example and is not otherwise recommended.
	}
	defer service.Stop()

	// Connect to the WebDriver instance running locally.
	caps := selenium.Capabilities{"browserName": "firefox"}
	firefoxCaps := firefox.Capabilities{
		Args: []string{
			"--headless",
			"--no-sandbox",
		},
	}
	caps.AddFirefox(firefoxCaps)
	webDriver, err := selenium.NewRemote(caps, fmt.Sprintf("http://localhost:%d", port))
	if err != nil {
		panic(err)
	}
	defer webDriver.Quit()

	err = webDriver.Get("http://192.168.1.1")
	if err != nil {
		fmt.Printf("Failed to load page: %s\n", err)
		return
	}

	if title, err := webDriver.Title(); err == nil {
		fmt.Printf("Page title: %s\n", title)
	} else {
		fmt.Printf("Failed to get page title: %s", err)
		return
	}

	var elem selenium.WebElement
	elem, err = webDriver.FindElement(selenium.ByCSSSelector, "#pc-login-user")
	if err != nil {
		fmt.Printf("Failed to find element: %s\n", err)
		return
	}
	elem.SendKeys(thisApp.Preferences().StringWithFallback("RouterUsername", ""))
	elem, err = webDriver.FindElement(selenium.ByCSSSelector, "#pc-login-password")
	if err != nil {
		fmt.Printf("Failed to find element: %s\n", err)
		return
	}
	elem.SendKeys(thisApp.Preferences().StringWithFallback("RouterPassword", ""))
	elem, err = webDriver.FindElement(selenium.ByCSSSelector, "#pc-login-btn")
	if err != nil {
		fmt.Printf("Failed to find element: %s\n", err)
		return
	}
	elem.Click()

	elem, err = webDriver.FindElement(selenium.ByCSSSelector, "#confirm-yes")
	if err == nil {
		elem.Click()
	}

	time.Sleep(5 * time.Second)
	/*
		elementFound := func(wd selenium.WebDriver) (bool, error) {
			elem, err = webDriver.FindElement(selenium.ByCSSSelector, "div.help-container")
			if err == nil {
				return elem.IsDisplayed()
			}
			return false, err
		}
		err = webDriver.WaitWithTimeout(elementFound, 3*time.Second)
	*/
	elem, err = webDriver.FindElement(selenium.ByCSSSelector, "a[url='parentCtrl.htm'] span")
	if err != nil {
		fmt.Printf("Failed to find element: %s\n", err)
		return
	}
	elem.Click()
	elem.Click()

	time.Sleep(3 * time.Second)

	_, err = webDriver.FindElement(selenium.ByCSSSelector, lookfor)
	if err == nil {
		webDriver.ExecuteScript(
			fmt.Sprintf("document.querySelector('%s').click()", lookfor),
			[]interface{}{})
	}
	time.Sleep(4 * time.Second)
}

func DisableParentalControls() {
	GetToParentalControlsAndClick("#enableParentalCtrlOff")
}
func EnableParentalControls() {
	GetToParentalControlsAndClick("#enableParentalCtrlOn")
}