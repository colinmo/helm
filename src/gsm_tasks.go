package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sort"
	"time"

	"github.com/pkg/browser"
)

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

var GSMAccessTokenRequestsChan chan string
var GSMAccessTokenChan chan string

func singleThreadReturnOrGetGSMAccessToken() {
	for {
		_, ok := <-GSMAccessTokenRequestsChan
		if ok == false {
			break
		}
		fmt.Printf("Return Or Get\n")
		for {
			fmt.Printf("wait")
			if AuthenticationTokens.GSM.access_token != "" {
				break
			}
			time.Sleep(1 * time.Second)
		}
		fmt.Printf("\n")
		fmt.Printf("AT: %s\nRT: %s\nXP: %s\n", AuthenticationTokens.GSM.access_token, AuthenticationTokens.GSM.refresh_token, AuthenticationTokens.GSM.expiration)
		if !AuthenticationTokens.GSM.expiration.After(time.Now()) {
			fmt.Printf("Refresh\n")
			refreshGSM()
		}
		fmt.Printf("OK to go\n")
		GSMAccessTokenChan <- AuthenticationTokens.GSM.access_token
	}
}

func returnOrGetGSMAccessToken() string {
	GSMAccessTokenRequestsChan <- "hey"
	return <-GSMAccessTokenChan
}

func GetGSM() {
	DownloadTasks()
	taskWindowRefresh("CWTasks")
	DownloadIncidents()
	taskWindowRefresh("CWIncidents")
	DownloadMyRequests()
	taskWindowRefresh("CWRequests")
	DownloadTeam()
	taskWindowRefresh("CWTeamIncidents")
}

func DownloadTasks() {
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

func DownloadIncidents() {
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

func DownloadMyRequests() {
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

func DownloadTeam() {
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
					log.Fatalf("Failed 1 %s\n", CherwellToken.ErrorDescription)
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
				AppStatus.GSMGettingToken = false
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
	AppStatus.GSMGettingToken = true
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
		log.Fatalf("Failed 2 %s\n", CherwellToken.ErrorDescription)
	}
	AuthenticationTokens.GSM.access_token = CherwellToken.AccessToken
	AuthenticationTokens.GSM.refresh_token = CherwellToken.RefreshToken
	if CherwellToken.Expires == "" {
		AuthenticationTokens.GSM.expiration = time.Now().Add(2000 * time.Hour)
	} else {
		AuthenticationTokens.GSM.expiration, _ = time.Parse(time.RFC1123, CherwellToken.Expires)
	}
}

func getStuffFromCherwell(method string, path string, payload []byte) (io.ReadCloser, error) {
	client := &http.Client{
		Timeout: time.Second * 10,
	}
	newpath, _ := url.JoinPath("https://griffith.cherwellondemand.com/CherwellAPI/", path)
	req, _ := http.NewRequest(method, newpath, bytes.NewReader(payload))
	returnOrGetGSMAccessToken()
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", AuthenticationTokens.GSM.access_token))
	req.Header.Set("Content-type", "application/json")
	resp, err := client.Do(req)
	return resp.Body, err
}
