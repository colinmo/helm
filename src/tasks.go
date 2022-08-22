package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"sort"
	"time"

	"github.com/pkg/browser"
)

// Connecting to the various task sources
// - GSM
// - JIRA
// - Planner

type Tokens struct {
	GSM struct {
		access_token  string
		refresh_token string
		userid        string
		expiration    time.Time
		cherwelluser  string
		teams         []string
	}
	MS struct {
		access_token  string
		refresh_token string
		expiration    time.Time
	}
}

var AuthenticationTokens Tokens

type GSMFilter struct {
	FieldId  string `json:"fieldId"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
}
type GSMSort struct {
	FieldID       string `json:"fieldId"`
	SortDirection int    `json:"sortDirection"`
}
type GSMSearchQuery struct {
	Filters    []GSMFilter `json:"filters"`
	BusObjId   string      `json:"busObId"`
	PageNumber int         `json:"pageNumber"`
	PageSize   int         `json:"pageSize"`
	Fields     []string    `json:"fields"`
	Sorting    []GSMSort   `json:"sorting"`
}

func GetAllTasks() {
	GetGSM()
	if appPreferences.MSPlannerActive {
		GetPlanner()
	}
	if appPreferences.JiraActive {
		GetJira()
	}
}

var AuthWebServer *http.Server

func startLocalServers() {
	http.HandleFunc("/cherwell", authenticateToCherwell)
	http.HandleFunc("/ms", authenticateToMS)
	go func() {
		AuthWebServer = &http.Server{Addr: ":84", Handler: nil}
		if err := AuthWebServer.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()
}

func isConnectedToGSM() bool {
	if AuthenticationTokens.GSM.refresh_token != "" && AuthenticationTokens.GSM.expiration.Before(time.Now()) {
		refreshGSM()
		activeTaskStatusUpdate(1)
		return false
	}
	if AuthenticationTokens.GSM.access_token == "" || AuthenticationTokens.GSM.expiration.Before(time.Now()) {
		browser.OpenURL(`https://serviceportal.griffith.edu.au/cherwellapi/saml/login.cshtml?finalUri=http://localhost:84/cherwell?code=xx`)
		activeTaskStatusUpdate(1)
		return false
	}
	return true

}

func GetGSM() {
	if isConnectedToGSM() {
		DownloadTasks()
		taskWindowRefresh("CWTasks")
		DownloadIncidents()
		taskWindowRefresh("CWIncidents")
		DownloadMyRequests()
		taskWindowRefresh("CWRequests")
		DownloadTeam()
		taskWindowRefresh("CWTeamIncidents")
	}
}

func DownloadTasks() {
	if isConnectedToGSM() {
		activeTaskStatusUpdate(1)
		defer activeTaskStatusUpdate(-1)
		AppStatus.MyTasksFromGSM = [][]string{}
		for page := 1; page < 200; page++ {
			tasksResponse, _ := GetMyTasksFromGSMForPage(page)
			if len(tasksResponse.BusinessObjects) > 0 {
				for _, x := range tasksResponse.BusinessObjects {
					row := []string{}
					for _, y := range x.Fields {
						row = append(row, y.Value)
					}
					if val, ok := priorityOverrides.CWIncidents[row[1]]; ok {
						orig := len(row) - 1
						row = append(row, row[orig])
						row[orig] = val
					}
					AppStatus.MyTasksFromGSM = append(AppStatus.MyTasksFromGSM, row)
				}
			} else {
				fmt.Printf("Nothing found")
			}
			if len(tasksResponse.BusinessObjects) != 200 {
				break
			}
		}
		sort.SliceStable(
			AppStatus.MyTasksFromGSM,
			func(i, j int) bool {
				var toReturn bool
				if AppStatus.MyTasksFromGSM[i][6] == AppStatus.MyTasksFromGSM[j][6] {
					toReturn = AppStatus.MyTasksFromGSM[i][0] < AppStatus.MyTasksFromGSM[j][0]
				} else {
					toReturn = AppStatus.MyTasksFromGSM[i][6] < AppStatus.MyTasksFromGSM[j][6]
				}
				return toReturn
			},
		)
	}
}

func DownloadIncidents() {
	if isConnectedToGSM() {
		activeTaskStatusUpdate(1)
		defer activeTaskStatusUpdate(-1)
		AppStatus.MyIncidentsFromGSM = [][]string{}
		for page := 1; page < 200; page++ {
			tasksResponse, _ := GetMyIncidentsFromGSMForPage(page)
			if len(tasksResponse.BusinessObjects) > 0 {
				for _, x := range tasksResponse.BusinessObjects {
					row := []string{}
					for _, y := range x.Fields {
						row = append(row, y.Value)
					}
					lastIndex := len(row) - 1
					row = append(row, row[lastIndex])
					if val, ok := priorityOverrides.CWIncidents[row[1]]; ok {
						row[lastIndex] = val
					}
					AppStatus.MyIncidentsFromGSM = append(AppStatus.MyIncidentsFromGSM, row)
				}
			}
			if len(tasksResponse.BusinessObjects) != 200 {
				break
			}
		}
	}
}

func DownloadMyRequests() {
	if isConnectedToGSM() {
		activeTaskStatusUpdate(1)
		defer activeTaskStatusUpdate(-1)
		AppStatus.MyRequestsInGSM = [][]string{}
		for page := 1; page < 200; page++ {
			tasksResponse, _ := GetMyRequestsInGSMForPage(page)
			if len(tasksResponse.BusinessObjects) > 0 {
				for _, x := range tasksResponse.BusinessObjects {
					row := []string{}
					for _, y := range x.Fields {
						row = append(row, y.Value)
					}
					lastIndex := len(row) - 1
					row = append(row, row[lastIndex])
					if val, ok := priorityOverrides.CWIncidents[row[1]]; ok {
						row[lastIndex] = val
					}
					AppStatus.MyRequestsInGSM = append(AppStatus.MyRequestsInGSM, row)
				}
			}
			if len(tasksResponse.BusinessObjects) != 200 {
				break
			}
		}
	}
}

func DownloadTeam() {
	if isConnectedToGSM() {
		activeTaskStatusUpdate(1)
		defer activeTaskStatusUpdate(-1)
		AppStatus.MyTeamIncidentsFromGSM = [][]string{}
		for page := 1; page < 200; page++ {
			tasksResponse, _ := GetMyTeamIncidentsInGSMForPage(page)
			if len(tasksResponse.BusinessObjects) > 0 {
				for _, x := range tasksResponse.BusinessObjects {
					if x.Fields[6].Value == AuthenticationTokens.GSM.cherwelluser {
						continue
					}
					row := []string{}
					for _, y := range x.Fields {
						row = append(row, y.Value)
					}
					lastIndex := len(row) - 1
					row = append(row, row[lastIndex])
					if val, ok := priorityOverrides.CWIncidents[row[1]]; ok {
						row[lastIndex] = val
					}
					AppStatus.MyTeamIncidentsFromGSM = append(AppStatus.MyTeamIncidentsFromGSM, row)
				}
			}
			if len(tasksResponse.BusinessObjects) != 200 {
				break
			}
		}
	}
}

var CWFields struct {
	Task struct {
		OwnerID           string
		CreatedDateTime   string
		TaskTitle         string
		TaskStatus        string
		TaskID            string
		IncidentID        string
		IncidentShortDesc string
		IncidentPriority  string
	}
	Incident struct {
		OwnerID          string
		Status           string
		CreatedDateTime  string
		IncidentID       string
		ShortDesc        string
		Priority         string
		RequestorSNumber string
		TeamID           string
		OwnerName        string
	}
}

func GetMyTasksFromGSMForPage(page int) (CherwellSearchResponse, error) {
	var tasksResponse CherwellSearchResponse
	r, err := SearchCherwellFor(GSMSearchQuery{
		Filters: []GSMFilter{
			{FieldId: CWFields.Task.OwnerID, Operator: "eq", Value: AuthenticationTokens.GSM.cherwelluser},
			{FieldId: CWFields.Task.TaskStatus, Operator: "eq", Value: "Acknowledged"},
			{FieldId: CWFields.Task.TaskStatus, Operator: "eq", Value: "New"},
			{FieldId: CWFields.Task.TaskStatus, Operator: "eq", Value: "In Progress"},
			{FieldId: CWFields.Task.TaskStatus, Operator: "eq", Value: "On Hold"},
			{FieldId: CWFields.Task.TaskStatus, Operator: "eq", Value: "Pending"},
		},
		BusObjId:   "9355d5ed41e384ff345b014b6cb1c6e748594aea5b",
		PageNumber: page,
		PageSize:   200,
		Fields: []string{
			CWFields.Task.CreatedDateTime,
			CWFields.Task.IncidentID,
			CWFields.Task.IncidentShortDesc,
			CWFields.Task.TaskTitle,
			CWFields.Task.TaskStatus,
			CWFields.Task.TaskID,
			CWFields.Task.IncidentPriority,
		},
		Sorting: []GSMSort{
			{FieldID: CWFields.Task.IncidentPriority, SortDirection: 1},
			{FieldID: CWFields.Task.CreatedDateTime, SortDirection: 1},
		},
	})
	if err == nil {
		defer r.Close()
		_ = json.NewDecoder(r).Decode(&tasksResponse)
	}
	return tasksResponse, err
}

func GetMyIncidentsFromGSMForPage(page int) (CherwellSearchResponse, error) {
	var tasksResponse CherwellSearchResponse
	r, err := SearchCherwellFor(GSMSearchQuery{
		Filters: []GSMFilter{
			{FieldId: CWFields.Incident.OwnerID, Operator: "eq", Value: AuthenticationTokens.GSM.cherwelluser},
			{FieldId: CWFields.Incident.Status, Operator: "eq", Value: "Acknowledged"},
			{FieldId: CWFields.Incident.Status, Operator: "eq", Value: "New"},
			{FieldId: CWFields.Incident.Status, Operator: "eq", Value: "In Progress"},
			{FieldId: CWFields.Incident.Status, Operator: "eq", Value: "On Hold"},
			{FieldId: CWFields.Incident.Status, Operator: "eq", Value: "Pending"},
		},
		BusObjId:   "6dd53665c0c24cab86870a21cf6434ae",
		PageNumber: page,
		PageSize:   200,
		Fields: []string{
			CWFields.Incident.CreatedDateTime,
			CWFields.Incident.IncidentID,
			CWFields.Incident.ShortDesc,
			CWFields.Incident.Status,
			CWFields.Incident.Priority,
		},
		Sorting: []GSMSort{
			{FieldID: CWFields.Incident.Priority, SortDirection: 1},
			{FieldID: CWFields.Incident.CreatedDateTime, SortDirection: 1},
		},
	})
	if err == nil {
		defer r.Close()
		_ = json.NewDecoder(r).Decode(&tasksResponse)
	}
	return tasksResponse, err
}

func GetMyRequestsInGSMForPage(page int) (CherwellSearchResponse, error) {
	var tasksResponse CherwellSearchResponse
	r, err := SearchCherwellFor(GSMSearchQuery{
		Filters: []GSMFilter{
			{FieldId: CWFields.Incident.RequestorSNumber, Operator: "eq", Value: AuthenticationTokens.GSM.userid},
			{FieldId: CWFields.Incident.Status, Operator: "eq", Value: "Acknowledged"},
			{FieldId: CWFields.Incident.Status, Operator: "eq", Value: "New"},
			{FieldId: CWFields.Incident.Status, Operator: "eq", Value: "In Progress"},
			{FieldId: CWFields.Incident.Status, Operator: "eq", Value: "On Hold"},
			{FieldId: CWFields.Incident.Status, Operator: "eq", Value: "Pending"},
		},
		BusObjId:   "6dd53665c0c24cab86870a21cf6434ae",
		PageNumber: page,
		PageSize:   200,
		Fields: []string{
			CWFields.Incident.CreatedDateTime,
			CWFields.Incident.IncidentID,
			CWFields.Incident.ShortDesc,
			CWFields.Incident.Status,
			CWFields.Incident.Priority,
		},
		Sorting: []GSMSort{
			{FieldID: CWFields.Incident.Priority, SortDirection: 1},
			{FieldID: CWFields.Incident.CreatedDateTime, SortDirection: 1},
		},
	})
	if err == nil {
		defer r.Close()
		_ = json.NewDecoder(r).Decode(&tasksResponse)
	}
	return tasksResponse, err
}
func GetMyTeamIncidentsInGSMForPage(page int) (CherwellSearchResponse, error) {
	var tasksResponse CherwellSearchResponse
	baseFilter := []GSMFilter{
		{FieldId: CWFields.Incident.Status, Operator: "eq", Value: "Acknowledged"},
		{FieldId: CWFields.Incident.Status, Operator: "eq", Value: "New"},
		{FieldId: CWFields.Incident.Status, Operator: "eq", Value: "In Progress"},
		{FieldId: CWFields.Incident.Status, Operator: "eq", Value: "On Hold"},
		{FieldId: CWFields.Incident.Status, Operator: "eq", Value: "Pending"},
	}
	for _, x := range AuthenticationTokens.GSM.teams {
		baseFilter = append(baseFilter, GSMFilter{FieldId: CWFields.Incident.TeamID, Operator: "eq", Value: x})
	}
	r, err := SearchCherwellFor(GSMSearchQuery{
		Filters:    baseFilter,
		BusObjId:   "6dd53665c0c24cab86870a21cf6434ae",
		PageNumber: page,
		PageSize:   200,
		Fields: []string{
			CWFields.Incident.CreatedDateTime,
			CWFields.Incident.IncidentID,
			CWFields.Incident.ShortDesc,
			CWFields.Incident.Status,
			CWFields.Incident.Priority,
			CWFields.Incident.OwnerName,
			CWFields.Incident.OwnerID,
		},
		Sorting: []GSMSort{
			{FieldID: CWFields.Incident.Priority, SortDirection: 1},
			{FieldID: CWFields.Incident.CreatedDateTime, SortDirection: 1},
		},
	})
	if err == nil {
		defer r.Close()
		_ = json.NewDecoder(r).Decode(&tasksResponse)
	}
	return tasksResponse, err
}

func SearchCherwellFor(toSend GSMSearchQuery) (io.ReadCloser, error) {
	var err error
	sendThis, _ := json.Marshal(toSend)
	result, err := getStuffFromCherwell(
		"POST",
		"api/V1/getsearchresults",
		sendThis)
	return result, err
}

type CherwellAuthResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
	Expires          string `json:"expires"`
	ExpiresDate      time.Time
	Issued           string `json:"issued"`
	AccessToken      string `json:"access_token"`
	RefreshToken     string `json:"refresh_token"`
	TokenType        string `json:"token_type"`
	Username         string `json:"username"`
}
type CherwellSearchResponse struct {
	BusinessObjects []struct {
		BusObId       string `json:"busObId"`
		BusObPublicId string `json:"busObPublicId"`
		BusObRecId    string `json:"busObRecId"`
		Fields        []struct {
			Dirty       bool   `json:"dirty"`
			DisplayName string `json:"displayName"`
			FieldId     string `json:"fieldId"`
			FullFieldId string `json:"fullFieldId"`
			HTML        string `json:"html"`
			Name        string `json:"name"`
			Value       string `json:"value"`
		} `json:"fields"`
		Links []struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"links"`
	} `json:"businessObjects"`
	HasPrompts          bool     `json:"hasPrompts"`
	Links               []string `json:"links"`
	Prompts             []string `json:"prompts"`
	SearchResultsFields []string `json:"searchResultsFields"`
}

func authenticateToCherwell(w http.ResponseWriter, r *http.Request) {
	var CherwellToken CherwellAuthResponse
	query := r.URL.Query()
	if query.Get("code") != "" {
		// Get body
		ticket := r.PostFormValue("ticket")
		if len(ticket) > 0 {
			payload := url.Values{
				"grant_type":    {"password"},
				"client_id":     {"814f9a74-c86a-451e-b6bb-deea65acf72a"},
				"username":      {r.PostFormValue("userId")},
				"password":      {ticket},
				"refresh_token": {""},
				"site_name":     {""},
			}
			resp, err := http.PostForm(
				"https://griffith.cherwellondemand.com/CherwellAPI/token?auth_mode=SAML",
				payload,
			)
			if err != nil {
				log.Fatalf("Login failed %s\n", err)
			} else {
				json.NewDecoder(resp.Body).Decode(&CherwellToken)
				if len(CherwellToken.Error) > 0 {
					log.Fatalf("Failed %s\n", CherwellToken.ErrorDescription)
				}
				AuthenticationTokens.GSM.access_token = CherwellToken.AccessToken
				AuthenticationTokens.GSM.refresh_token = CherwellToken.RefreshToken
				AuthenticationTokens.GSM.userid = CherwellToken.Username
				if CherwellToken.Expires == "" {
					AuthenticationTokens.GSM.expiration = time.Now().Add(2000 * time.Hour)
				} else {
					AuthenticationTokens.GSM.expiration, _ = time.Parse(time.RFC1123, CherwellToken.Expires)
				}
				var decodedResponse CherwellSearchResponse
				// Get my UserID in Cherwell
				r, _ := SearchCherwellFor(GSMSearchQuery{
					Filters: []GSMFilter{
						{FieldId: "941798910e24d4b1ae3c6a408cb3f8c5eba26e2b2b", Operator: "eq", Value: AuthenticationTokens.GSM.userid},
					},
					BusObjId: "9338216b3c549b75607cf54667a4e67d1f644d9fed",
					Fields:   []string{"933a15d131ff727ca7ed3f4e1c8f528d719b99b82d", "933a15d17f1f10297df7604b58a76734d6106ac428"},
				})
				defer r.Close()
				_ = json.NewDecoder(r).Decode(&decodedResponse)
				AuthenticationTokens.GSM.cherwelluser = decodedResponse.BusinessObjects[0].BusObRecId
				// Get my TeamIDs in Cherwell
				var teamResponse struct {
					Teams []struct {
						TeamID   string `json:"teamId"`
						TeamName string `json:"teamName"`
					} `json:"teams"`
				}
				r, _ = getStuffFromCherwell(
					"GET",
					"api/V2/getusersteams/userrecordid/"+AuthenticationTokens.GSM.cherwelluser,
					[]byte{})
				defer r.Close()
				_ = json.NewDecoder(r).Decode(&teamResponse)
				AuthenticationTokens.GSM.teams = []string{
					decodedResponse.BusinessObjects[0].Fields[1].Value,
				}
				for _, x := range teamResponse.Teams {
					AuthenticationTokens.GSM.teams = append(AuthenticationTokens.GSM.teams, x.TeamID)
				}
				activeTaskStatusUpdate(-1)
				GetGSM()
			}
		}
	} else {
		// Redirect to Cherwell AUTH
		browser.OpenURL(`https://serviceportal.griffith.edu.au/cherwellapi/saml/login.cshtml?finalUri=http://localhost:84/cherwell?code=xx`)
		activeTaskStatusUpdate(1)
	}
}

func refreshGSM() {
	var CherwellToken CherwellAuthResponse
	payload := url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {"814f9a74-c86a-451e-b6bb-deea65acf72a"},
		"username":      {AuthenticationTokens.GSM.userid},
		"password":      {""},
		"refresh_token": {AuthenticationTokens.GSM.refresh_token},
		"site_name":     {""},
	}
	resp, err := http.PostForm(
		"https://griffith.cherwellondemand.com/CherwellAPI/token?auth_mode=SAML",
		payload,
	)
	if err != nil {
		log.Fatalf("Login failed %s\n", err)
	}
	json.NewDecoder(resp.Body).Decode(&CherwellToken)
	if len(CherwellToken.Error) > 0 {
		log.Fatalf("Failed %s\n", CherwellToken.ErrorDescription)
	}
	AuthenticationTokens.GSM.access_token = CherwellToken.AccessToken
	AuthenticationTokens.GSM.refresh_token = CherwellToken.RefreshToken
	if CherwellToken.Expires == "" {
		AuthenticationTokens.GSM.expiration = time.Now().Add(2000 * time.Hour)
	} else {
		AuthenticationTokens.GSM.expiration, _ = time.Parse(time.RFC1123, CherwellToken.Expires)
	}
	activeTaskStatusUpdate(-1)
	GetGSM()
}

func getStuffFromCherwell(method string, path string, payload []byte) (io.ReadCloser, error) {
	client := &http.Client{
		Timeout: time.Second * 10,
	}
	newpath, _ := url.JoinPath("https://griffith.cherwellondemand.com/CherwellAPI/", path)
	req, _ := http.NewRequest(method, newpath, bytes.NewReader(payload))
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", AuthenticationTokens.GSM.access_token))
	req.Header.Set("Content-type", "application/json")
	resp, err := client.Do(req)
	return resp.Body, err
}

func GetJIRA() {

}

var MSAuthWebServer *http.Server

type MSAuthResponse struct {
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
	ExpiresIn    int    `json:"expires_in"`
	ExpiresDate  time.Time
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func GetPlanner() {
	if AuthenticationTokens.MS.access_token == "" || AuthenticationTokens.MS.expiration.Before(time.Now()) {
		// Login if expired
		browser.OpenURL(
			fmt.Sprintf(`https://login.microsoftonline.com/%s/oauth2/v2.0/authorize?finalUri=?code=xy&client_id=%s&response_type=code&redirect_uri=http://localhost:84/ms&response_mode=query&scope=%s`,
				msApplicationTenant,
				msApplicationClientId,
				msScopes),
		)
	} else {
		DownloadPlanners()
		// Add personal priorities
		// Return
	}
}

func authenticateToMS(w http.ResponseWriter, r *http.Request) {
	var MSToken MSAuthResponse
	query := r.URL.Query()
	if query.Get("code") != "" {
		payload := url.Values{
			"client_id":     {msApplicationClientId},
			"scope":         {msScopes},
			"code":          {query.Get("code")},
			"redirect_uri":  {"http://localhost:84/ms"},
			"grant_type":    {"authorization_code"},
			"client_secret": {msApplicationSecret},
		}
		resp, err := http.PostForm(
			fmt.Sprintf(`https://login.microsoftonline.com/%s/oauth2/v2.0/token`,
				msApplicationTenant,
			),
			payload,
		)
		if err != nil {
			log.Fatalf("Login failed %s\n", err)
		} else {
			err := json.NewDecoder(resp.Body).Decode(&MSToken)
			if err != nil {
				log.Fatalf("Failed MS %s\n", err)
			}
			AuthenticationTokens.MS.access_token = MSToken.AccessToken
			AuthenticationTokens.MS.refresh_token = MSToken.RefreshToken
			seconds, _ := time.ParseDuration(fmt.Sprintf("%ds", MSToken.ExpiresIn))
			AuthenticationTokens.MS.expiration = time.Now().Add(seconds)
			GetPlanner()
		}
	} else {
		// Redirect to Cherwell AUTH
		browser.OpenURL(
			fmt.Sprintf(`https://login.microsoftonline.com/%s/oauth2/v2.0/authorize?finalUri=?code=xy&client_id=%s&response_type=code&redirect_uri=http://localhost:84/ms&response_mode=query&scope=%s`,
				msApplicationTenant,
				msApplicationClientId,
				msScopes),
		)
	}
}

type myTasksGraphResponse struct {
	NextPage string `json:"@odata.nextLink"`
	Value    []struct {
		TaskID          string `json:"id"`
		PlanID          string `json:"planId"`
		BucketID        string `json:"bucketId"`
		Title           string `json:"title"`
		OrderHint       string `json:"orderHint"`
		PercentComplete int    `json:"percentComplete"`
		CreatedDateTime string `json:"createdDateTime"`
		Priority        int    `json:"priority"`
		Details         struct {
			Description string `json:"description"`
		} `json:"details"`
	} `json:"value"`
}

func DownloadPlanners() {
	activeTaskStatusUpdate(1)
	defer activeTaskStatusUpdate(-1)

	// * @todo Add Sort
	AppStatus.MyTasksFromPlanner = [][]string{}
	var teamResponse myTasksGraphResponse
	urlToCall := "/me/planner/tasks"
	for page := 1; page < 200; page++ {
		r, err := callGraphURI("GET", urlToCall, []byte{}, "$expand=details")
		if err == nil {
			defer r.Close()
			_ = json.NewDecoder(r).Decode(&teamResponse)

			for _, y := range teamResponse.Value {
				if y.PercentComplete < 100 {
					AppStatus.MyTasksFromPlanner = append(
						AppStatus.MyTasksFromPlanner,
						[]string{
							y.TaskID,
							y.PlanID,
							y.BucketID,
							y.Title,
							y.OrderHint,
							y.CreatedDateTime,
							teamPriorityToGSMPriority(y.Priority),
							y.Details.Description,
							fmt.Sprintf("%d", y.PercentComplete),
						},
					)
				}
			}
			if len(teamResponse.NextPage) == 0 {
				break
			} else {
				x, e := url.Parse(teamResponse.NextPage)
				if e == nil {
					urlToCall = x.Path
				} else {
					break
				}
			}
		} else {
			fmt.Printf("Failed to get Graph Tasks %s\n", err)
		}
	}
	// sort
	sort.SliceStable(AppStatus.MyTasksFromPlanner, func(i, j int) bool {
		if AppStatus.MyTasksFromPlanner[i][7] == AppStatus.MyTasksFromPlanner[j][7] {
			return AppStatus.MyTasksFromPlanner[i][5] < AppStatus.MyTasksFromPlanner[j][5]
		}
		return AppStatus.MyTasksFromPlanner[i][7] < AppStatus.MyTasksFromPlanner[j][7]
	})
	taskWindowRefresh("MSPlanner")
}

func teamPriorityToGSMPriority(priority int) string {
	switch priority {
	case 0, 1:
		return "1"
	case 2, 3, 4:
		return "2"
	case 5, 6, 7:
		return "3"
	case 8:
		return "4"
	case 9:
		return "5"
	default:
		return "5"
	}
}

func callGraphURI(method string, path string, payload []byte, query string) (io.ReadCloser, error) {
	client := &http.Client{
		Timeout: time.Second * 10,
	}
	newpath, _ := url.JoinPath("https://graph.microsoft.com/v1.0/", path)
	if len(query) > 0 {
		newpath = newpath + "?" + query
	}
	req, _ := http.NewRequest(method, newpath, bytes.NewReader(payload))
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", AuthenticationTokens.MS.access_token))
	req.Header.Set("Content-type", "application/json")

	resp, err := client.Do(req)
	return resp.Body, err
}

type JiraResponseType struct {
	StartAt    int `json:"startAt"`
	MaxResults int `json:"maxResults"`
	Total      int `json:"total"`
	Issues     []struct {
		ID     string `json:"id"`
		Key    string `json:"key"`
		Fields struct {
			Summary  string `json:"summary"`
			Priority struct {
				Name    string `json:"name"`
				IconURL string `json:"iconUr"`
			} `json:"priority"`
			CreatedDateTime string `json:"created"`
			Status          struct {
				Name    string `json:"name"`
				IconURL string `json:"iconUrl"`
			} `json:"status"`
		} `json:"fields"`
	} `json:"issues"`
}

func GetJira() {
	if appPreferences.JiraActive && appPreferences.JiraKey > "" {
		activeTaskStatusUpdate(1)
		defer activeTaskStatusUpdate(-1)
		AppStatus.MyTasksFromJira = [][]string{}
		var jiraResponse JiraResponseType
		baseQuery := "jql=assignee=currentuser()+and+status+not+in+(Done)+order+by+priority,+created+asc&fields=status,summary,created,priority&orderBy=priority,created"
		queryToCall := fmt.Sprintf("%s&startAt=0", baseQuery)
		for page := 1; page < 200; page++ {
			r, err := callJiraURI("GET", "search", []byte{}, queryToCall)
			if err == nil {
				defer r.Close()
				_ = json.NewDecoder(r).Decode(&jiraResponse)

				for _, y := range jiraResponse.Issues {
					AppStatus.MyTasksFromJira = append(
						AppStatus.MyTasksFromJira,
						[]string{
							y.Key,
							y.Fields.Summary,
							y.Fields.CreatedDateTime,
							jiraPriorityToGSMPriority(y.Fields.Priority.Name),
							y.Fields.Status.Name,
						},
					)
				}
				if len(jiraResponse.Issues) == 0 {
					break
				} else {
					queryToCall = fmt.Sprintf("%s&startAt=%d", baseQuery, jiraResponse.MaxResults*page)
				}
			} else {
				fmt.Printf("Failed to get Jira Tasks %s\n", err)
			}
		}
		// sort
		sort.SliceStable(AppStatus.MyTasksFromPlanner, func(i, j int) bool {
			if AppStatus.MyTasksFromPlanner[i][3] == AppStatus.MyTasksFromPlanner[j][3] {
				return AppStatus.MyTasksFromPlanner[i][2] < AppStatus.MyTasksFromPlanner[j][2]
			}
			return AppStatus.MyTasksFromPlanner[i][3] < AppStatus.MyTasksFromPlanner[j][3]
		})
		taskWindowRefresh("Jira")
	}
}
func callJiraURI(method string, path string, payload []byte, query string) (io.ReadCloser, error) {
	client := &http.Client{
		Timeout: time.Second * 10,
	}
	newpath, _ := url.JoinPath("https://griffith.atlassian.net/rest/api/2/", path)
	if len(query) > 0 {
		newpath = newpath + "?" + query
	}
	req, _ := http.NewRequest(method, newpath, bytes.NewReader(payload))
	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", appPreferences.JiraUsername, appPreferences.JiraKey)))))
	req.Header.Set("Content-type", "application/json")

	resp, err := client.Do(req)
	return resp.Body, err
}
func jiraPriorityToGSMPriority(priority string) string {
	switch priority {
	case "Highest":
		return "1"
	case "High":
		return "2"
	case "Medium":
		return "3"
	case "Low":
		return "4"
	case "Lowest":
		return "5"
	default:
		return "5"
	}
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
