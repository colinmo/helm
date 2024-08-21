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
	"1": "New",
	"2": "In Progress",
	"3": "On Hold",
	"6": "Resolved",
	"7": "Closed",
	"8": "Cancelled",
}
var SNContactTypes = map[string]string{
	"-- None --":                    "-- None --",
	"Chat":                          "chat",
	"Messaging":                     "messaging",
	"Phone":                         "phone",
	"Video":                         "video",
	"Microsoft Teams Chat":          "microsoft_teams_chat",
	"Email":                         "email",
	"Event":                         "event",
	"Walk-in":                       "walk-in",
	"Incident creation by employee": "incident_creation",
	"Self-service":                  "self-service",
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
func (snow *SNOWStruct) MyIcon() []byte {
	return []byte(`<svg version="1.2" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1570 1403" width="1570" height="1403">
	<title>servicenow-header-logo-svg</title>
	<style>
		.s0 { fill: #62d84e } 
	</style>
	<path id="Layer" fill-rule="evenodd" class="s0" d="m1228.4 138.9c129.2 88.9 228.9 214.3 286.3 360.2 57.5 145.8 70 305.5 36 458.5-34 153-112.9 292.4-226.7 400.3-13.3 12.9-28.8 23.4-45.8 30.8-17 7.5-35.2 11.9-53.7 12.9-18.5 1.1-37.1-1.1-54.8-6.6-17.7-5.4-34.3-13.9-49.1-25.2-48.2-35.9-101.8-63.8-158.8-82.6-57.1-18.9-116.7-28.5-176.8-28.5-60.1 0-119.8 9.6-176.8 28.5-57 18.8-110.7 46.7-158.9 82.6-14.6 11.2-31 19.8-48.6 25.3-17.6 5.5-36 7.8-54.4 6.8-18.4-0.9-36.5-5.1-53.4-12.4-16.9-7.3-32.4-17.5-45.8-30.2-114.6-108.3-194.1-248.5-228.1-402.5-34-154-20.9-314.6 37.6-461 58.5-146.5 159.6-272 290.3-360.3 130.7-88.3 284.9-135.4 442.7-135 156.8 1.3 309.6 49.6 438.8 138.4zm-291.8 1014c48.2-19.2 92-48 128.7-84.6 36.7-36.7 65.5-80.4 84.7-128.6 19.2-48.1 28.4-99.7 27-151.5 0-103.9-41.3-203.5-114.8-277-73.5-73.5-173.2-114.8-277.2-114.8-104 0-203.7 41.3-277.2 114.8-73.5 73.5-114.8 173.1-114.8 277-1.4 51.8 7.8 103.4 27 151.5 19.2 48.2 48 91.9 84.7 128.6 36.7 36.6 80.5 65.4 128.6 84.6 48.2 19.2 99.8 28.4 151.7 27 51.8 1.4 103.4-7.8 151.6-27z"/>
</svg>`)
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
								Type:             e.Type,
								History:          e.History,
							},
						)
					}
					if len(res) == 0 {
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
								Type:             e.Type,
							},
						)
					}
					if len(res) == 0 {
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
								Type:             e.Type,
							},
						)
					}
					if len(res) == 0 {
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
		"task", // table
		[]string{"number", "short_description", "sys_id", "priority", "sys_created_on", "state", "sys_class_name", "approval_history"}, // fields to return
		map[string]string{"assigned_to": AppPreferences.SnowUser, "active": "=true"},                                                   // filters
		page,
	)
	return r, err
}

func (snow *SNOWStruct) GetMyRequestsForPage(page int) ([]SnowIncident, error) {
	r, err := snow.SearchSnowFor(
		"incident", // table
		[]string{"number", "short_description", "sys_id", "priority", "sys_created_on", "state", "sys_class_name"}, // fields to return
		map[string]string{"opened_by": AppPreferences.SnowUser, "active": "=true", "state": "!=6"},                 // filters
		page,
	)
	s, err2 := snow.SearchSnowFor(
		"sc_request", // table
		[]string{"number", "short_description", "sys_id", "priority", "sys_created_on", "state", "sys_class_name"}, // fields to return
		map[string]string{"active": "=true", "state": "!=6", "requested_for": AppPreferences.SnowUser},             // filters
		page,
	)
	t, err3 := snow.SearchSnowFor(
		"sc_req_item", // table
		[]string{"number", "short_description", "sys_id", "priority", "sys_created_on", "state", "sys_class_name"}, // fields to return
		map[string]string{"active": "=true", "state": "!=6", "requested_for": AppPreferences.SnowUser},             // filters
		page,
	)
	if err == nil && err == err2 && err2 == err3 {
		return append(append(r, s...), t...), nil
	}
	if err == nil {
		if err2 == nil {
			return append(r, s...), err3
		}
		if err3 == nil {
			return append(r, t...), err2
		}
		return r, err2
	}
	if err2 == nil {
		if err3 == nil {
			return append(s, t...), err
		}
		return s, err
	}
	if err3 == nil {
		return t, err
	}
	return nil, err
}

func (snow *SNOWStruct) GetMyTeamIncidentsForPage(page int) ([]SnowIncident, error) {
	r, err := snow.SearchSnowFor(
		"task", // table
		[]string{"number", "short_description", "variables.contract_title", "sys_id", "priority", "sys_created_on", "state"}, // fields to return
		map[string]string{"assigned_to": "=", "active": "=true", "assignment_group": AppPreferences.SnowGroup},               // filters
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
	Type        string
	History     string
}
type SnowResponseIncidents struct {
	Number           string `json:"number"`
	ShortDescription string `json:"short_description"`
	Contract         string `json:"variables.contract_title"`
	ID               string `json:"sys_id"`
	Created          string `json:"sys_created_on"`
	Priority         string `json:"priority"`
	State            string `json:"state"`
	ApprovalHistory  string `json:"approval_history"`
	Type             string `json:"sys_class_name"`
}

type SnowResponse struct {
	Results []SnowResponseIncidents `json:"result"`
}

func (snow *SNOWStruct) GetAnyTable(table string, fields []string, filter map[string]string, sort string, page int) ([]byte, error) {
	if sort != "" {
		sort = "^" + sort
	}
	pageLength := 20
	filter["active"] = "=true"
	result, err := snow.getStuffFromURL(
		"GET",
		"/api/now/table/"+table,
		fmt.Sprintf(
			"sysparm_limit=%d&sysparm_fields=%s&sysparm_query=%s&sysparm_offset=%d",
			pageLength,
			strings.Join(fields, ","),
			createKeyValuePairsForQuery(filter)+sort,
			page*pageLength),
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
	pageLength := 20
	result, err := snow.getStuffFromURL(
		"GET",
		"/api/now/table/"+table,
		fmt.Sprintf(
			"sysparm_limit=%d&sysparm_fields=%s&sysparm_query=%s&sysparm_offset=%d&sysparm_display_value=true",
			pageLength,
			strings.Join(fields, ","),
			createKeyValuePairsForQuery(filter),
			page*pageLength),
		[]byte{},
	)
	fmt.Printf(
		"/api/now/table/"+table+"?sysparm_limit=%d&sysparm_fields=%s&sysparm_query=%s&sysparm_offset=%d&sysparm_display_value=true\n",
		pageLength,
		strings.Join(fields, ","),
		createKeyValuePairsForQuery(filter),
		page*pageLength)
	toReturn := []SnowIncident{}
	if err == nil {
		defer result.Close()
		err = json.NewDecoder(result).Decode(&incidentsResponse)
		if err == nil {
			for _, x := range incidentsResponse.Results {
				created, _ := time.Parse("02/01/2006 15:04:04", x.Created)
				me := SnowIncident{
					ID:          x.ID,
					Number:      x.Number,
					Created:     created,
					Priority:    x.Priority[0:1],
					Status:      x.State,
					Description: x.ShortDescription,
					Type:        x.Type,
					History:     x.ApprovalHistory,
				}
				if len(x.Contract) > 0 {
					me.Description = fmt.Sprintf("%s (Contract)", x.Contract)
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
