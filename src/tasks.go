package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

// Connecting to the various task sources
// - GSM
// - JIRA
// - Planner

// @todo: Use Golang parallelism to refresh a token in the background, leaving a message/ action in its wake to resolve before processing a request
//        Or, record pending requests and then access them again once a refresh is complete.
//        OR have a message based process for handling requests for information from the remote client that handles all the authentication etc. by itself
//        So requests for refresh are sent to this parallel process via a message queue, and it processes them individually and in order.

type GSMTokens struct {
	access_token  string
	refresh_token string
	userid        string
	expiration    time.Time
	cherwelluser  string
	allteams      []string
	myteam        string
}
type MSTokens struct {
	access_token  string
	refresh_token string
	expiration    time.Time
}
type Tokens struct {
	GSM GSMTokens
	MS  MSTokens
}

var AuthenticationTokens = Tokens{
	GSM: GSMTokens{
		access_token:  "",
		refresh_token: "",
		userid:        "",
		expiration:    time.Now(),
		cherwelluser:  "",
		allteams:      []string{},
		myteam:        "",
	},
	MS: MSTokens{
		access_token:  "",
		refresh_token: "",
		expiration:    time.Now(),
	},
}

type TaskResponseStruct struct {
	ID               string
	BusObRecId       string
	Title            string
	ParentTitle      string
	ParentID         string
	ParentIDInternal string
	CreatedDateTime  time.Time
	Priority         string
	Status           string
	PriorityOverride string
	Owner            string
	OwnerID          string
}

var planner = Planner{}
var jira = Jira{}
var gsm = Cherwell{}

func InitTasks() {
	if appPreferences.JiraActive {
		planner.Init("http://localhost:84/", connectionStatusBox, "", "", time.Now())
		planner.Login()
	}
	if appPreferences.GSMActive {
		gsm.Init("http://localhost:84/", connectionStatusBox, "", "", time.Now())
		gsm.Login()
	}
}

func GetAllTasks() {
	if appPreferences.GSMActive {
		gsm.Download(
			func() { taskWindowRefresh("CWTasks") },
			func() { taskWindowRefresh("CWIncidents") },
			func() { taskWindowRefresh("CWRequests") },
			func() { taskWindowRefresh("CWTeamIncidents") },
			func() { taskWindowRefresh("CWTeamTasks") },
		)
	}
	if appPreferences.MSPlannerActive {
		planner.Refresh()
		planner.Download("")
		taskWindowRefresh("MSPlanner")
	}
	if appPreferences.JiraActive {
		jira.Download()
	}
}

var AuthWebServer *http.Server

func startLocalServers() {
	if appPreferences.GSMActive {
		http.HandleFunc(gsm.RedirectPath, func(w http.ResponseWriter, r *http.Request) {
			gsm.AuthenticateToCherwell(w, r)
		})
	}
	if appPreferences.JiraActive {
		http.HandleFunc(planner.RedirectPath, func(w http.ResponseWriter, r *http.Request) {
			planner.Authenticate(w, r)
		})
	}
	go func() {
		AuthWebServer = &http.Server{Addr: ":84", Handler: nil}
		if err := AuthWebServer.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()
}

type PriorityOverrides struct {
	CWTasks     map[string]string `json:"cwtasks"`
	CWIncidents map[string]string `json:"cwincidents"`
	MSPlanner   map[string]string `json:"msplanner"`
}

var priorityOverrides PriorityOverrides

func loadPriorityOverride() {
	r, e := os.Open(appPreferences.PriorityOverride)
	if errors.Is(e, os.ErrNotExist) {
		priorityOverrides = PriorityOverrides{
			CWTasks: map[string]string{
				"x": "y",
			},
			CWIncidents: map[string]string{
				"x": "y",
			},
			MSPlanner: map[string]string{
				"x": "y",
			},
		}
		savePriorityOverride()
		r, e = os.Open(appPreferences.PriorityOverride)
	}
	if e == nil {
		defer r.Close()
		_ = json.NewDecoder(r).Decode(&priorityOverrides)
	}
}

func savePriorityOverride() {
	f, err := os.OpenFile(appPreferences.PriorityOverride, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.ModeExclusive)
	if err == nil {
		defer f.Close()
		x, err := json.Marshal(priorityOverrides)
		if err == nil {
			fmt.Fprintln(f, string(x))
		}
	}
}

func TruncateShort(s string, i int) string {
	var runes = []rune(s)
	if len(runes) > i {
		return string(runes[:i]) + "..."
	}
	return s
}

// Generic task object needs
type Task struct {
}

func (t *Task) Init() {
	// Initialising
	fmt.Printf("Generic init")
}

func (t *Task) Authenticate(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("Authenticating")
}
