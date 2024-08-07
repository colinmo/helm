package iserver

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/pkg/browser"
	"golang.org/x/oauth2"
)

var defaultModel = "Baseline Architecture"

type IServerStruct struct {
	Token        *oauth2.Token
	RedirectPath string
}

var isTokenLock sync.Mutex
var AuthWebServer *http.Server
var planConf *oauth2.Config

func (p *IServerStruct) Init(baseRedirect string) {
	isTokenLock.Lock()
	p.RedirectPath = "/iserv"
	planConf = &oauth2.Config{
		ClientID:     iSERVER_AZURE_CLIENT_ID,
		ClientSecret: iSERVER_AZURE_CLIENT_SECRET,
		RedirectURL: func() string {
			thisUrl, _ := url.JoinPath(baseRedirect, p.RedirectPath)
			return thisUrl
		}(),
		Scopes: strings.Split(iSERVER_AZURE_SCOPES, " "),
		Endpoint: oauth2.Endpoint{
			AuthURL: fmt.Sprintf(
				`https://login.microsoftonline.com/%s/oauth2/v2.0/authorize`,
				iSERVER_AZURE_TENANT_ID),
			TokenURL: fmt.Sprintf(
				`https://login.microsoftonline.com/%s/oauth2/v2.0/token`,
				iSERVER_AZURE_TENANT_ID,
			),
		},
	}
	p.StartLocalServers()
	p.Login()
}
func (p *IServerStruct) StartLocalServers() {
	http.HandleFunc(p.RedirectPath, func(w http.ResponseWriter, r *http.Request) {
		p.Authenticate(w, r)
	})
	go func() {
		AuthWebServer = &http.Server{Addr: ":86", Handler: nil}
		if err := AuthWebServer.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()
}

func (p *IServerStruct) Authenticate(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	if query.Get("code") != "" {
		t, err := planConf.Exchange(context.Background(), query.Get("code"))
		if err != nil {
			w.Header().Add("Content-type", "text/html")
			fmt.Fprintf(w, "<html><head></head><body><H1>Failed to Authenticate<p>%s</body></html>", err.Error())
		} else {
			p.Token = t
			w.Header().Add("Content-type", "text/html")
			fmt.Fprintf(w, "<html><head></head><body><H1>Authenticated<p>You are authenticated, you may close this window.<script>window.close();</script></body></html>")
			isTokenLock.Unlock()
		}
	}
}

func (p *IServerStruct) Login() {
	browser.OpenURL(planConf.AuthCodeURL("some-user-state", oauth2.AccessTypeOffline))
}

func (p *IServerStruct) CallRestEndpoint(method string, path string, payload []byte, query string) (io.ReadCloser, error) {
	isTokenLock.Lock()

	client := planConf.Client(context.Background(), p.Token)
	newpath, _ := url.JoinPath("https://griffith-api.iserver365.com/", path)
	req, _ := http.NewRequest(method, newpath, bytes.NewReader(payload))
	req.URL.RawQuery = query
	req.Header.Set("Content-type", "application/json")

	resp, err := client.Do(req)
	isTokenLock.Unlock()
	if err == nil && resp.StatusCode == 200 {
		return resp.Body, err
	}
	return nil, err
}
