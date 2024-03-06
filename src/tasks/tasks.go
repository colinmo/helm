package tasks

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
	Access_token  string
	Refresh_token string
	Userid        string
	Expiration    time.Time
	Cherwelluser  string
	Allteams      []string
	Myteam        string
}
type MSTokens struct {
	Access_token  string
	Refresh_token string
	Expiration    time.Time
}
type Tokens struct {
	GSM GSMTokens
	MS  MSTokens
}

var AuthenticationTokens = Tokens{
	GSM: GSMTokens{
		Access_token:  "",
		Refresh_token: "",
		Userid:        "",
		Expiration:    time.Now(),
		Cherwelluser:  "",
		Allteams:      []string{},
		Myteam:        "",
	},
	MS: MSTokens{
		Access_token:  "",
		Refresh_token: "",
		Expiration:    time.Now(),
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
	Type             string
	Blocked          bool
}

/** If we move this into /tasks, keep these here and refer in main as tasks.* ? */
var Planner = &PlannerStruct{}
var Jira = &JiraStruct{}
var Gsm = &CherwellStruct{}
var Snow = &SNOWStruct{}

func InitTasks(appPreferences *TaskPreferencesStruct, connectionStatusBoxRef func(bool, string), taskWindowRefreshRef func(string), activeTaskStatusUpdateRef func(int)) {
	TaskWindowRefresh = taskWindowRefreshRef
	ActiveTaskStatusUpdate = activeTaskStatusUpdateRef
	ConnectionStatusBox = connectionStatusBoxRef
	AppPreferences = *appPreferences
	if AppPreferences.JiraActive {
		Jira.Init()
	}
	if AppPreferences.GSMActive {
		Gsm.Init("http://localhost:84/", "", "", time.Now())
	}
	if AppPreferences.MSPlannerActive {
		Planner.Init("http://localhost:84/", "", "", time.Now())
	}
	if appPreferences.SnowActive {
		Snow.Init("http://localhost:84/", "", "", time.Now())
	}
}

func GetAllTasks(jiraActive, gsmActive, msplannerActive, snowActive bool, taskWindowRefresh func(string), updateFunc func(int)) {
	if gsmActive {
		Gsm.Download(
			func() { taskWindowRefresh("CWTasks") },
			func() { taskWindowRefresh("CWIncidents") },
			func() { taskWindowRefresh("CWRequests") },
			func() { taskWindowRefresh("CWTeamIncidents") },
			func() { taskWindowRefresh("CWTeamTasks") },
		)
	}
	if msplannerActive {
		Planner.Refresh()
		Planner.Download("")
		taskWindowRefresh("Planner")
	}
	if jiraActive {
		Jira.Download()
		taskWindowRefresh("Jira")
	}
	if snowActive {
		Snow.Download(
			func() { taskWindowRefresh("SNIncidents") },
			func() { taskWindowRefresh("SNRequests") },
			func() { taskWindowRefresh("SNTeamIncidents") },
		)
		taskWindowRefresh("Snow")
	}
}

type TaskPreferencesStruct struct {
	GSMActive          bool
	MSPlannerActive    bool
	MSAccessToken      string
	MSRefreshToken     string
	MSExpiresAt        time.Time
	MSGroups           string
	CWActive           bool
	PriorityOverride   string
	JiraProjectHome    string
	JiraActive         bool
	JiraUsername       string
	JiraKey            string
	JiraDefaultProject string
	DynamicsActive     bool
	DynamicsKey        string
	SnowActive         bool
	SnowAccessToken    string
	SnowSRefreshToken  string
	SnowExpiresAt      time.Time
	SnowUser           string // 7fcaa702933002009c8579b4f47ffbde
}

var AuthWebServer *http.Server
var TaskWindowRefresh func(string)
var ActiveTaskStatusUpdate func(int)
var ConnectionStatusBox func(bool, string)
var AppPreferences TaskPreferencesStruct

func StartLocalServers() {
	if AppPreferences.GSMActive {
		http.HandleFunc(Gsm.RedirectPath, func(w http.ResponseWriter, r *http.Request) {
			Gsm.AuthenticateToCherwell(w, r)
		})
	}
	if AppPreferences.MSPlannerActive {
		http.HandleFunc(Planner.RedirectPath, func(w http.ResponseWriter, r *http.Request) {
			Planner.Authenticate(w, r)
		})
	}
	if AppPreferences.SnowActive {
		fmt.Printf("Snow redirect %s\n", Snow.RedirectPath)
		http.HandleFunc(Snow.RedirectPath, func(w http.ResponseWriter, r *http.Request) {
			Snow.Authenticate(w, r)
		})
	}
	go func() {
		AuthWebServer = &http.Server{Addr: ":84", Handler: nil}
		if err := AuthWebServer.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()
}

type PriorityOverridesStruct struct {
	CWTasks     map[string]string `json:"cwtasks"`
	CWIncidents map[string]string `json:"cwincidents"`
	MSPlanner   map[string]string `json:"msplanner"`
	Jira        map[string]string `json:"jira"`
}

var PriorityOverrides PriorityOverridesStruct

func LoadPriorityOverride(preferences string) {
	r, e := os.Open(preferences)
	if errors.Is(e, os.ErrNotExist) {
		PriorityOverrides = PriorityOverridesStruct{
			CWTasks: map[string]string{
				"x": "y",
			},
			CWIncidents: map[string]string{
				"x": "y",
			},
			MSPlanner: map[string]string{
				"x": "y",
			},
			Jira: map[string]string{
				"x": "y",
			},
		}
		SavePriorityOverride()
		r, e = os.Open(AppPreferences.PriorityOverride)
	}
	if e == nil {
		defer r.Close()
		_ = json.NewDecoder(r).Decode(&PriorityOverrides)
	}
}

func SavePriorityOverride() {
	f, err := os.OpenFile(AppPreferences.PriorityOverride, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.ModeExclusive)
	if err == nil {
		defer f.Close()
		x, err := json.Marshal(PriorityOverrides)
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
