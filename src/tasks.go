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
	Title            string
	ParentTitle      string
	ParentID         string
	CreatedDateTime  time.Time
	Priority         string
	Status           string
	PriorityOverride string
	Owner            string
}

func GetAllTasks() {
	GetGSM()
	if appPreferences.MSPlannerActive {
		DownloadPlanners()
		taskWindowRefresh("MSPlanner")
	}
	if appPreferences.JiraActive {
		GetJira()
	}
}

var AuthWebServer *http.Server

func startLocalServers() {
	fmt.Printf("Server Active\n")
	http.HandleFunc("/cherwell", authenticateToCherwell)
	http.HandleFunc("/ms", authenticateToMS)
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

// @todo - add a cleanup somewhere so that if the priorty matches the actual value, don't save it
// OR add an element to the dropdown saying "Default" that removes an override
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
