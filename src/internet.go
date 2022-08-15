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

// var activeInternetTimeChan (chan time.Duration)

/*
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
*/

/*
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
*/
