package main

import (
	"fmt"
	"log"
	"net/http"
	"testing"
	"time"

	"fyne.io/fyne/v2/data/binding"
	"github.com/pkg/browser"
)

func TestSingleThreadReturnOrGetGSMAccessToken(t *testing.T) {
	go singleThreadReturnOrGetGSMAccessToken()
	AuthenticationTokens.GSM.access_token = "x"
	AuthenticationTokens.GSM.expiration = time.Now().Add(200 * time.Hour)
	val := returnOrGetGSMAccessToken()
	if val != AuthenticationTokens.GSM.access_token {
		t.Fatalf("Didn't get the access token expected")
	}
}

func TestGoodAccessToken(t *testing.T) {
	AppStatus = AppStatusStruct{
		TaskTaskStatus:  binding.NewString(),
		TaskTaskCount:   0,
		GSMGettingToken: false,
		MSGettingToken:  false,
	}
	startFakeMS("http://localhost:84/cherwell?code=ok", 301, []string{})
	go singleThreadReturnOrGetGSMAccessToken()
	browser.OpenURL(GSMAuthURL)
	val := returnOrGetGSMAccessToken()
	if val != "OKToken" {
		t.Fatalf("Didn't get the access token expected [%s]", val)
	}
}

func startFakeMS(authReturnLocation string, authReturnCode int, apiResponses []string) {
	GSMBaseUrl = `http://localhost:85/CherwellAPI/`
	GSMAuthURL = `http://localhost:86/cherwellapi/saml/login.cshtml?finalUri=http://localhost:84/cherwell?code=xx`

	http.HandleFunc("/cherwell", func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("Local Auth Endpoint called\n")
		query := r.URL.Query()
		if query.Get("code") == "ok" {
			AuthenticationTokens.GSM.access_token = "OKToken"
			AuthenticationTokens.GSM.refresh_token = "OKToken"
			AuthenticationTokens.GSM.userid = "OKUser"
			AuthenticationTokens.GSM.expiration = time.Now().Add(2000 * time.Hour)
			AuthenticationTokens.GSM.cherwelluser = "Mike"
			AuthenticationTokens.GSM.allteams = []string{}
			AuthenticationTokens.GSM.myteam = ""
			AppStatus.GSMGettingToken = false
			w.Header().Add("Content-type", "text/html")
			fmt.Fprintf(w, "<html><head></head><body><script>window.close()</script></body></html>")
		}
	})
	go func() {
		AuthWebServer = &http.Server{Addr: ":84", Handler: nil}
		if err := AuthWebServer.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()

	http.HandleFunc("/cherwellapi", func(w http.ResponseWriter, r *http.Request) {

	})
	go func() {
		AuthWebServer = &http.Server{Addr: ":85", Handler: nil}
		if err := AuthWebServer.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()

	http.HandleFunc(
		"/cherwellapi/saml/login.cshtml",
		func(w http.ResponseWriter, r *http.Request) {
			fmt.Printf("AuthURL called\n")
			if authReturnCode == 301 {
				w.Header().Add("Location", authReturnLocation)
			}
			w.WriteHeader(authReturnCode)
		},
	)
	go func() {
		AuthWebServer = &http.Server{Addr: ":86", Handler: nil}
		if err := AuthWebServer.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()

}
