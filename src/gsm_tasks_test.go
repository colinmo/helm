package main

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

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
