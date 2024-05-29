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

type MSTokens struct {
	Access_token  string
	Refresh_token string
	Expiration    time.Time
}
type Tokens struct {
	MS MSTokens
}

var AuthenticationTokens = Tokens{
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
	Links            []string
	History          string
}

/** If we move this into /tasks, keep these here and refer in main as tasks.* ? */
var Planner = &PlannerStruct{}
var Jira = &JiraStruct{}
var Snow = &SNOWStruct{}
var Zettle = &ZettleStruct{}

func InitTasks(appPreferences *TaskPreferencesStruct, connectionStatusBoxRef func(bool, string), taskWindowRefreshRef func(string), activeTaskStatusUpdateRef func(int)) {
	TaskWindowRefresh = taskWindowRefreshRef
	ActiveTaskStatusUpdate = activeTaskStatusUpdateRef
	ConnectionStatusBox = connectionStatusBoxRef
	AppPreferences = *appPreferences
	if AppPreferences.JiraActive {
		Jira.Init()
	}
	if AppPreferences.MSPlannerActive {
		Planner.Init("http://localhost:84/", "", "", time.Now())
	}
	if appPreferences.SnowActive {
		Snow.Init(
			"http://localhost:84/",
			"",
			"",
			time.Now())
	}
	Zettle.Init()
}

func GetAllTasks(
	jiraActive,
	msplannerActive,
	snowActive bool,
	taskWindowRefresh func(string),
	updateFunc func(int),
	zettleHome string) {
	if msplannerActive {
		go func() {
			Planner.Download()
			taskWindowRefresh("Planner")
		}()
	}
	if jiraActive {
		go func() {
			Jira.Download()
			taskWindowRefresh("Jira")
		}()
	}
	if snowActive {
		go func() {
			Snow.Download(
				func() { taskWindowRefresh("SNIncidents") },
				func() { taskWindowRefresh("SNRequests") },
				func() { taskWindowRefresh("SNTeamIncidents") },
			)
			taskWindowRefresh("Snow")
		}()
	}
	Zettle.Download(zettleHome)
	taskWindowRefresh("Zettle")
}

type TaskPreferencesStruct struct {
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
	SnowGroup          string
}

var AuthWebServer *http.Server
var TaskWindowRefresh func(string)
var ActiveTaskStatusUpdate func(int)
var ConnectionStatusBox func(bool, string)
var AppPreferences TaskPreferencesStruct

func StartLocalServers() {
	if AppPreferences.MSPlannerActive {
		http.HandleFunc(Planner.RedirectPath, func(w http.ResponseWriter, r *http.Request) {
			Planner.Authenticate(w, r)
		})
	}
	if AppPreferences.SnowActive {
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
	MSPlanner map[string]string `json:"msplanner"`
	Jira      map[string]string `json:"jira"`
	SNow      map[string]string `json:"snow"`
}

var PriorityOverrides PriorityOverridesStruct

func LoadPriorityOverride(preferences string) {
	r, e := os.Open(preferences)
	if errors.Is(e, os.ErrNotExist) {
		PriorityOverrides = PriorityOverridesStruct{
			MSPlanner: map[string]string{
				"x": "y",
			},
			Jira: map[string]string{
				"x": "y",
			},
			SNow: map[string]string{"x": "y"},
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
