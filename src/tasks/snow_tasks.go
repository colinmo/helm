package tasks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/pkg/browser"
	"golang.org/x/oauth2"
	"golang.org/x/sync/singleflight"
)

type TokenStatusType int64

const (
	Pending TokenStatusType = iota
	Inactive
	Active
	Spring
)

type SNOWStruct struct {
	Task
	RedirectPath    string
	RedirectURI     string
	UserID          string
	UserSnumber     string
	UserTeams       []string
	UserEmail       string
	DefaultTeam     string
	MyTasks         []TaskResponseStruct
	MyIncidents     []TaskResponseStruct
	LoggedIncidents []TaskResponseStruct
	TeamIncidents   []TaskResponseStruct
	TeamTasks       []TaskResponseStruct
	BaseURL         string
	AuthURL         string
	StatusCallback  func(bool, string)
	TokenStatus     TokenStatusType
	G               *singleflight.Group
	Token           *oauth2.Token
}

var SnowStatuses = map[string]string{
	"1": "New", "2": "In Progress", "3": "On Hold", "6": "Resolved", "8": "Cancelled",
}
var SNContactTypes = map[string]string{
	"-- None --":           "-- None --",
	"chat":                 "Chat",
	"messaging":            "Messaging",
	"phone":                "Phone",
	"video":                "Video",
	"microsoft_teams_chat": "Microsoft Teams Chat",
	"email":                "Email",
	"event":                "Event",
	"walk-in":              "Walk-in",
	"incident_creation":    "Incident creation by employee",
	"self-service":         "Self-service",
}

var SNContactTypeLabels = []string{
	"-- None --",
	"Chat",
	"Messaging",
	"Phone",
	"Video",
	"Microsoft Teams Chat",
	"Email",
	"Event",
	"Walk-in",
	"Incident creation by employee",
	"Self-service",
}

var SNImpact = map[string]string{
	"1": "1 - Organisation",
	"2": "2 - Group",
	"3": "3 - Individual",
}

var SNImpactLabels = []string{
	"1 - Organisation",
	"2 - Group",
	"3 - Individual",
}

var SNUrgency = map[string]string{
	"1": "1 - High",
	"2": "2 - Medium",
	"3": "3 - Low",
}
var SNUrgencyLabels = []string{
	"1 - High",
	"2 - Medium",
	"3 - Low",
}

var snowConf *oauth2.Config

var snowTokenLock sync.Mutex

func (snow *SNOWStruct) Init(
	baseRedirect string,
	accessToken string,
	refreshToken string,
	expiration time.Time) {
	snowTokenLock.Lock()
	snow.TokenStatus = Inactive
	snow.RedirectPath = "/snow"
	snow.BaseURL = snBaseUrl
	snowConf = &oauth2.Config{
		ClientID:     snApplicationClientId,
		ClientSecret: snApplicationSecret,
		RedirectURL: func() string {
			thisUrl, _ := url.JoinPath(baseRedirect, snow.RedirectPath)
			return thisUrl
		}(),
		Scopes: []string{},
		Endpoint: oauth2.Endpoint{
			AuthURL: snAuthUrl,
			TokenURL: func() string {
				thisUrl, _ := url.JoinPath(snBaseUrl, "oauth_token.do")
				return thisUrl
			}(),
		},
	}
	// Start the background runner
	snow.G = &singleflight.Group{}
	snow.Login()
}

func (snow *SNOWStruct) Login() {
	go func() {
		snow.G.Do(
			"SNOWLogin",
			func() (interface{}, error) {
				snow.TokenStatus = Pending
				browser.OpenURL(snowConf.AuthCodeURL("some-user-state", oauth2.AccessTypeOffline))
				for {
					if snow.TokenStatus == Active {
						return "", nil
					}
				}
			},
		)
	}()
}

func (snow *SNOWStruct) Download(incidentWindow func(), requestsWindow func(), teamIncidentWindow func()) {
	snow.DownloadIncidents(incidentWindow)
	snow.DownloadMyRequests(requestsWindow)
	snow.DownloadTeamIncidents(teamIncidentWindow)
}

func (snow *SNOWStruct) Authenticate(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	if query.Get("code") != "" {
		t, err := snowConf.Exchange(context.Background(), query.Get("code"))
		if err != nil {
			ConnectionStatusBox(false, "S")
		} else {
			snow.Token = t
			ConnectionStatusBox(true, "S")
			w.Header().Add("Content-type", "text/html")
			fmt.Fprintf(w, "<html><head></head><body><H1>Authenticated<p>You are authenticated to Service Now, you may close this window.<script>window.close();</script></body></html>")
			AppPreferences.SnowAccessToken = snow.Token.AccessToken
			AppPreferences.SnowSRefreshToken = snow.Token.RefreshToken
			AppPreferences.SnowExpiresAt = snow.Token.Expiry
			snowTokenLock.Unlock()
		}
	}
}

func (snow *SNOWStruct) DownloadIncidents(afterFunc func()) {
	go func() {
		snow.G.Do(
			"DownloadIncidents",
			func() (interface{}, error) {
				ActiveTaskStatusUpdate(1)
				defer ActiveTaskStatusUpdate(-1)
				snow.MyIncidents = []TaskResponseStruct{}
				for offset := 0; offset < 200; offset++ {
					res, _ := snow.GetMyIncidentsForPage(offset)
					for _, e := range res {
						myOverride := e.Priority
						if val, ok := PriorityOverrides.SNow[e.ID]; ok {
							myOverride = val
						}
						snow.MyIncidents = append(
							snow.MyIncidents,
							TaskResponseStruct{
								BusObRecId:       e.ID,
								ID:               e.Number,
								Title:            e.Description,
								CreatedDateTime:  e.Created,
								Priority:         e.Priority,
								PriorityOverride: myOverride,
								Status:           e.Status,
							},
						)
					}
					if len(res) != 10 {
						break
					}
				}
				sort.SliceStable(
					snow.MyIncidents,
					func(i, j int) bool {
						var toReturn bool
						if snow.MyIncidents[i].PriorityOverride == snow.MyIncidents[j].PriorityOverride {
							toReturn = snow.MyIncidents[i].CreatedDateTime.Before(snow.MyIncidents[j].CreatedDateTime)
						} else {
							toReturn = snow.MyIncidents[i].PriorityOverride < snow.MyIncidents[j].PriorityOverride
						}
						return toReturn
					},
				)
				afterFunc()
				return "", nil
			},
		)
	}()
}

func (snow *SNOWStruct) DownloadMyRequests(afterFunc func()) {
	go func() {
		snow.G.Do(
			"DownloadMyRequests",
			func() (interface{}, error) {
				ActiveTaskStatusUpdate(1)
				defer ActiveTaskStatusUpdate(-1)
				snow.LoggedIncidents = []TaskResponseStruct{}
				for offset := 0; offset < 200; offset++ {
					res, _ := snow.GetMyRequestsForPage(offset)
					for _, e := range res {
						myOverride := e.Priority
						if val, ok := PriorityOverrides.SNow[e.ID]; ok {
							myOverride = val
						}
						snow.LoggedIncidents = append(
							snow.LoggedIncidents,
							TaskResponseStruct{
								BusObRecId:       e.ID,
								ID:               e.Number,
								Title:            e.Description,
								CreatedDateTime:  e.Created,
								Priority:         e.Priority,
								PriorityOverride: myOverride,
								Status:           e.Status,
							},
						)
					}
					if len(res) != 10 {
						break
					}
				}
				sort.SliceStable(
					snow.LoggedIncidents,
					func(i, j int) bool {
						var toReturn bool
						if snow.LoggedIncidents[i].PriorityOverride == snow.LoggedIncidents[j].PriorityOverride {
							toReturn = snow.LoggedIncidents[i].CreatedDateTime.Before(snow.LoggedIncidents[j].CreatedDateTime)
						} else {
							toReturn = snow.LoggedIncidents[i].PriorityOverride < snow.LoggedIncidents[j].PriorityOverride
						}
						return toReturn
					},
				)
				afterFunc()
				return "", nil
			},
		)
	}()
}

func (snow *SNOWStruct) DownloadTeamIncidents(afterFunc func()) {
	go func() {
		snow.G.Do(
			"DownloadTeamIncidents",
			func() (interface{}, error) {
				ActiveTaskStatusUpdate(1)
				defer ActiveTaskStatusUpdate(-1)
				snow.TeamIncidents = []TaskResponseStruct{}
				for offset := 0; offset < 200; offset++ {
					res, _ := snow.GetMyTeamIncidentsForPage(offset)
					for _, e := range res {
						myOverride := e.Priority
						if val, ok := PriorityOverrides.SNow[e.ID]; ok {
							myOverride = val
						}
						snow.TeamIncidents = append(
							snow.TeamIncidents,
							TaskResponseStruct{
								BusObRecId:       e.ID,
								ID:               e.Number,
								Title:            e.Description,
								CreatedDateTime:  e.Created,
								Priority:         e.Priority,
								PriorityOverride: myOverride,
								Status:           e.Status,
							},
						)
					}
					if len(res) != 10 {
						break
					}
				}
				sort.SliceStable(
					snow.TeamIncidents,
					func(i, j int) bool {
						var toReturn bool
						if snow.TeamIncidents[i].PriorityOverride == snow.TeamIncidents[j].PriorityOverride {
							toReturn = snow.TeamIncidents[i].CreatedDateTime.Before(snow.TeamIncidents[j].CreatedDateTime)
						} else {
							toReturn = snow.TeamIncidents[i].PriorityOverride < snow.TeamIncidents[j].PriorityOverride
						}
						return toReturn
					},
				)
				afterFunc()
				return "", nil
			},
		)
	}()
}

func (snow *SNOWStruct) GetMyIncidentsForPage(page int) ([]SnowIncident, error) {
	r, err := snow.SearchSnowFor(
		"incident", // table
		[]string{"number", "short_description", "sys_id", "priority", "sys_created_on", "state"}, // fields to return
		map[string]string{"assigned_to": AppPreferences.SnowUser, "state": "IN1,2,3,4,5,7"},      // filters
		page,
	)
	return r, err
}

func (snow *SNOWStruct) GetMyRequestsForPage(page int) ([]SnowIncident, error) {
	r, err := snow.SearchSnowFor(
		"incident", // table
		[]string{"number", "short_description", "sys_id", "priority", "sys_created_on", "state"}, // fields to return
		map[string]string{"opened_by": AppPreferences.SnowUser, "state": "IN1,2,3,4,5,7"},        // filters
		page,
	)
	return r, err
}

func (snow *SNOWStruct) GetMyTeamIncidentsForPage(page int) ([]SnowIncident, error) {
	r, err := snow.SearchSnowFor(
		"incident", // table
		[]string{"number", "short_description", "sys_id", "priority", "sys_created_on", "state"},                      // fields to return
		map[string]string{"assigned_to": "=", "state": "IN1,2,3,4,5,7", "assignment_group": AppPreferences.SnowGroup}, // filters
		page,
	)
	return r, err
}

type SnowIncidentCreate struct {
	AffectedUser     string `json:"caller_id"`
	Service          string `json:"business_service"`
	ServiceOffering  string `json:"service_offering"`
	OpenedBy         string `json:"opened_by"`
	ShortDescription string `json:"short_description"`
	ContactType      string `json:"contact_type"`
	Impact           string `json:"impact"`
	Urgency          string `json:"urgency"`
	AssignmentGroup  string `json:"assignment_group"`
	AssignedTo       string `json:"assigned_to"`
	Description      string `json:"description"`
}
type CreateIncidentResponse struct {
	Result struct {
		Error  string `json:"error"`
		Number string `json:"number"`
	} `json:"result"`
}

func (snow *SNOWStruct) CreateNewIncident(newIncident SnowIncidentCreate) (string, string, error) {
	newIncidentNumber := ""
	urlForIncident := ""
	bob, err := json.Marshal(newIncident)
	if err == nil {
		body, respCode, headers, err := snow.getStuffAndHeadersFromURL(
			"POST",
			"/api/now/table/incident",
			"",
			bob,
		)
		if err == nil {
			if respCode == 201 {
				urlForIncident = strings.Replace(headers["Location"][0], "api/now/table/incident", "now/sow/record/incident", 1)
				// Process r
				var incidentsResponse CreateIncidentResponse
				err = json.NewDecoder(body).Decode(&incidentsResponse)
				if err == nil {
					newIncidentNumber = incidentsResponse.Result.Number
					if incidentsResponse.Result.Error != "" {
						return newIncidentNumber, urlForIncident, fmt.Errorf("failed to save incident %s", incidentsResponse.Result.Error)
					}
				}
			} else {
				return newIncidentNumber, urlForIncident, fmt.Errorf("failed to create, got %d", respCode)
			}
		}
	}
	return newIncidentNumber, urlForIncident, err
}

type SnowIncident struct {
	ID          string
	Number      string
	Created     time.Time
	Priority    string
	Status      string
	Description string
}
type SnowResponseIncidents struct {
	Number           string `json:"number"`
	ShortDescription string `json:"short_description"`
	ID               string `json:"sys_id"`
	Created          string `json:"sys_created_on"`
	Priority         string `json:"priority"`
	State            string `json:"state"`
}

type SnowResponse struct {
	Results []SnowResponseIncidents `json:"result"`
}

func (snow *SNOWStruct) GetAnyTable(table string, fields []string, filter map[string]string, sort string, page int) ([]byte, error) {
	if sort != "" {
		sort = "^" + sort
	}
	filter["active"] = "=true"
	result, err := snow.getStuffFromURL(
		"GET",
		"/api/now/table/"+table,
		fmt.Sprintf(
			"sysparm_limit=20&sysparm_fields=%s&sysparm_query=%s&sysparm_offset=%d",
			strings.Join(fields, ","),
			createKeyValuePairsForQuery(filter)+sort,
			page),
		[]byte{},
	)
	var toReturn []byte
	if err == nil {
		defer result.Close()
		toReturn, err = io.ReadAll(result)
	}
	return toReturn, err
}

func (snow *SNOWStruct) SearchSnowFor(table string, fields []string, filter map[string]string, page int) ([]SnowIncident, error) {
	var incidentsResponse SnowResponse
	result, err := snow.getStuffFromURL(
		"GET",
		"/api/now/table/"+table,
		fmt.Sprintf(
			"sysparm_limit=10&sysparm_fields=%s&sysparm_query=%s&sysparm_offset=%d",
			strings.Join(fields, ","),
			createKeyValuePairsForQuery(filter),
			page),
		[]byte{},
	)
	toReturn := []SnowIncident{}
	if err == nil {
		defer result.Close()
		err = json.NewDecoder(result).Decode(&incidentsResponse)
		if err == nil {
			for _, x := range incidentsResponse.Results {
				created, _ := time.Parse("2006-01-02 15:04:05", x.Created)
				me := SnowIncident{
					ID:          x.ID,
					Number:      x.Number,
					Created:     created,
					Priority:    x.Priority,
					Status:      x.State,
					Description: x.ShortDescription,
				}
				toReturn = append(toReturn, me)
			}
		} else {
			log.Fatal(err)
		}
	} else {
		log.Fatal(err)
	}
	return toReturn, err
}

func createKeyValuePairsForQuery(m map[string]string) string {
	b := new(bytes.Buffer)
	for key, value := range m {
		fmt.Fprintf(b, "%s%s^", key, url.QueryEscape(value))
	}
	toReturn := b.String()
	return toReturn[0 : len(toReturn)-1]
}

func (snow *SNOWStruct) getStuffFromURL(method string, path string, query string, payload []byte) (io.ReadCloser, error) {
	snowTokenLock.Lock()
	client := snowConf.Client(context.Background(), snow.Token)
	newpath, _ := url.JoinPath(snow.BaseURL, path)
	req, e := http.NewRequest(method, newpath, bytes.NewReader(payload))
	req.URL.RawQuery = query

	if e != nil {
		log.Fatal(e)
	}
	req.Header.Set("Content-type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	snowTokenLock.Unlock()
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	return resp.Body, err
}

func (snow *SNOWStruct) getStuffAndHeadersFromURL(method string, path string, query string, payload []byte) (io.ReadCloser, int, map[string][]string, error) {
	snowTokenLock.Lock()
	client := snowConf.Client(context.Background(), snow.Token)
	newpath, _ := url.JoinPath(snow.BaseURL, path)
	req, e := http.NewRequest(method, newpath, bytes.NewReader(payload))
	req.URL.RawQuery = query

	if e != nil {
		log.Fatal(e)
	}

	req.Header.Set("Content-type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	snowTokenLock.Unlock()
	if err != nil {
		log.Fatal(err)
		return nil, 0, map[string][]string{}, err
	}
	return resp.Body, resp.StatusCode, resp.Header, err
}
