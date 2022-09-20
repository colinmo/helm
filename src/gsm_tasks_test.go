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

func TestSingleThreadReturnGSMAccessTokenActive(t *testing.T) {
	gsm := Cherwell{}
	gsm.SingleThreadReturnAccessToken()
	AuthenticationTokens.GSM.access_token = "x"
	AuthenticationTokens.GSM.expiration = time.Now().Add(200 * time.Hour)
	val := ""
	go func() { val = gsm.returnOrGetGSMAccessToken() }()
	time.Sleep(1 * time.Second)
	if val != AuthenticationTokens.GSM.access_token {
		t.Fatalf("Didn't get the access token expected")
	}
}

func TestSingleThreadReturnGSMAccessTokenExpired(t *testing.T) {
	gsm := Cherwell{}
	gsm.SingleThreadReturnAccessToken()
	AuthenticationTokens.GSM.access_token = "y"
	AuthenticationTokens.GSM.expiration = time.Now().Add(-200 * time.Hour)
	connectionStatusBox = func(bool, string) {}
	val := ""
	go func() { val = gsm.returnOrGetGSMAccessToken() }()
	time.Sleep(5 * time.Second)
	if val != "" {
		t.Fatalf("Didn't handle an expired token")
	}
}

func TestGoodAccessToken(t *testing.T) {
	gsm := Cherwell{}
	AppStatus = AppStatusStruct{
		TaskTaskStatus: binding.NewString(),
		TaskTaskCount:  0,
	}
	go func() { startFakeMS("http://localhost:84/cherwell?code=ok", 301, []string{}) }()
	connectionStatusBox = func(bool, string) {}
	time.Sleep(5 * time.Second)
	gsm.SingleThreadReturnAccessToken()
	browser.OpenURL(gsm.AuthURL)
	val := gsm.returnOrGetGSMAccessToken()
	if val != "OKToken" {
		t.Fatalf("Didn't get the access token expected [%s]", val)
	}
}

func startFakeMS(authReturnLocation string, authReturnCode int, apiResponses []string) {
	gsm := Cherwell{}
	gsm.BaseURL = `http://localhost:84/CherwellAPI/`
	gsm.AuthURL = `http://localhost:84/cherwellapi/saml/login.cshtml?finalUri=http://localhost:84/cherwell?code=xx`

	http.HandleFunc("/cherwell", func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("Local Auth Endpoint called\n")
		query := r.URL.Query()
		if query.Get("code") == "ok" {
			gsm.AccessToken = "OKToken"
			gsm.RefreshToken = "OKToken"
			gsm.UserSnumber = "OKUser"
			gsm.Expiration = time.Now().Add(2000 * time.Hour)
			gsm.UserID = "Mike"
			gsm.UserTeams = []string{}
			gsm.DefaultTeam = ""
			w.Header().Add("Content-type", "text/html")
			fmt.Fprintf(w, "<html><head></head><body><script>window.close()</script></body></html>")
		}
	})
	http.HandleFunc("/cherwellapi", func(w http.ResponseWriter, r *http.Request) {})
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
	AuthWebServer = &http.Server{Addr: ":84", Handler: nil}
	if err := AuthWebServer.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
