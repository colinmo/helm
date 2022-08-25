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

var GSMAccessTokenRequestsChan = make(chan string)
var GSMAccessTokenChan = make(chan string)
var GSMBaseUrl = `https://griffith.cherwellondemand.com/CherwellAPI/`
var GSMAuthURL = `https://serviceportal.griffith.edu.au/cherwellapi/saml/login.cshtml?finalUri=http://localhost:84/cherwell?code=xx`

func singleThreadReturnOrGetGSMAccessToken() {
	for {
		_, ok := <-GSMAccessTokenRequestsChan
		if !ok {
			break
		}
		for {
			if AuthenticationTokens.GSM.access_token != "" && !AppStatus.GSMGettingToken {
				break
			}
			time.Sleep(1 * time.Second)
		}
		if AuthenticationTokens.GSM.expiration.Before(time.Now()) {
			AuthenticationTokens.GSM.access_token = ""
			browser.OpenURL(GSMAuthURL)
			for {
				if AuthenticationTokens.GSM.access_token != "" && !AppStatus.GSMGettingToken {
					break
				}
				time.Sleep(1 * time.Second)
			}
		}
		GSMAccessTokenChan <- AuthenticationTokens.GSM.access_token
	}
}

func returnOrGetGSMAccessToken() string {
	GSMAccessTokenRequestsChan <- time.Now().String()
	return <-GSMAccessTokenChan
}

func GetGSM() {
	fmt.Printf("Getting all GSM\n")
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

type CWFieldIDSTask struct {
	OwnerID           string
	CreatedDateTime   string
	TaskTitle         string
	TaskStatus        string
	TaskID            string
	IncidentID        string
	IncidentShortDesc string
	IncidentPriority  string
}
type CWFieldIDSIncident struct {
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
type CWFieldIDs struct {
	Task     CWFieldIDSTask
	Incident CWFieldIDSIncident
}

var CWFields = CWFieldIDs{
	Task: CWFieldIDSTask{
		OwnerID:           "93cfd5a4e1d0ba5d3423e247b08dfd1286cae772cf",
		CreatedDateTime:   "9355d5ed416bbc9408615c4145978ff8538a3f6eb4",
		TaskTitle:         "93ad98a2d68a61778eda3d4d9cbb30acbfd458aea4",
		TaskStatus:        "9368f0fb7b744108a666984c21afc932562eb7dc16",
		TaskID:            "93d5409c4bcbf7a38ed75a47dd92671f374236fa32",
		IncidentID:        "BO:6dd53665c0c24cab86870a21cf6434ae,FI:6ae282c55e8e4266ae66ffc070c17fa3,RE:93694ed12e2e9bb908131846b7a9c67ec72b811676",
		IncidentShortDesc: "BO:6dd53665c0c24cab86870a21cf6434ae,FI:93e8ea93ff67fd95118255419690a50ef2d56f910c,RE:93694ed12e2e9bb908131846b7a9c67ec72b811676",
		IncidentPriority:  "BO:6dd53665c0c24cab86870a21cf6434ae,FI:83c36313e97b4e6b9028aff3b401b71c,RE:93694ed12e2e9bb908131846b7a9c67ec72b811676",
	},
	Incident: CWFieldIDSIncident{
		OwnerID:          "9339fc404e39ae705648ab43969f29262e6d167606",
		Status:           "5eb3234ae1344c64a19819eda437f18d",
		CreatedDateTime:  "c1e86f31eb2c4c5f8e8615a5189e9b19",
		IncidentID:       "6ae282c55e8e4266ae66ffc070c17fa3",
		ShortDesc:        "93e8ea93ff67fd95118255419690a50ef2d56f910c",
		Priority:         "83c36313e97b4e6b9028aff3b401b71c",
		RequestorSNumber: "941aa0889094428a6f4c054dbea345b09b4d87c77e",
		TeamID:           "9339fc404e312b6d43436041fc8af1c07c6197f559",
		OwnerName:        "9339fc404e4c93350bf5be446fb13d693b0bb7f219",
	}}

func GetMyTasksFromGSMForPage(page int) (CherwellSearchResponse, error) {
	var tasksResponse CherwellSearchResponse
	returnOrGetGSMAccessToken() // Wait for CherwellUser to be a value
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
	}, true)
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
	}, true)
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
	}, true)
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
	}, true)
	if err == nil {
		defer r.Close()
		_ = json.NewDecoder(r).Decode(&tasksResponse)
	}
	return tasksResponse, err
}

func SearchCherwellFor(toSend GSMSearchQuery, refreshToken bool) (io.ReadCloser, error) {
	var err error
	sendThis, _ := json.Marshal(toSend)
	result, err := getStuffFromCherwell(
		"POST",
		"api/V1/getsearchresults",
		sendThis,
		refreshToken)
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
			targetURL, _ := url.JoinPath(GSMBaseUrl, "token")
			targetURL += "?auth_mode=SAML"
			resp, err := http.PostForm(
				targetURL,
				payload,
			)
			if err != nil {
				log.Fatalf("Login failed %s\n", err)
			} else {
				json.NewDecoder(resp.Body).Decode(&CherwellToken)
				if len(CherwellToken.Error) > 0 {
					log.Fatalf("Failed 1 %s\n", CherwellToken.ErrorDescription)
				}
				fmt.Printf("Getting token\n")
				AppStatus.GSMGettingToken = true
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
				}, false)
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
				r, err = getStuffFromCherwell(
					"GET",
					"api/V2/getusersteams/userrecordid/"+AuthenticationTokens.GSM.cherwelluser,
					[]byte{},
					false)
				if err == nil {
					defer r.Close()
					_ = json.NewDecoder(r).Decode(&teamResponse)
					AuthenticationTokens.GSM.teams = []string{
						decodedResponse.BusinessObjects[0].Fields[1].Value,
					}
					for _, x := range teamResponse.Teams {
						AuthenticationTokens.GSM.teams = append(AuthenticationTokens.GSM.teams, x.TeamID)
					}
					fmt.Printf("Token got\n")
					w.Header().Add("Content-type", "text/html")
					fmt.Fprintf(w, "<html><head></head><body><script>window.close()</script></body></html>")
					fmt.Printf("Authenticated, time to refresh\n")
				}
				activeTaskStatusUpdate(-1)
				AppStatus.GSMGettingToken = false

			}
		}
	} else {
		// Redirect to Cherwell AUTH
		browser.OpenURL(GSMAuthURL)
		activeTaskStatusUpdate(1)
	}
}

// @Todo - GSM doesn't support refresh on SAML tokens
// so just do a standard browser access token thing
func refreshGSM() {
	browser.OpenURL(GSMAuthURL)
	/*
		var CherwellToken CherwellAuthResponse
		payload := url.Values{
			"grant_type":    {"refresh_token"},
			"client_id":     {"814f9a74-c86a-451e-b6bb-deea65acf72a"},
			"username":      {AuthenticationTokens.GSM.userid},
			"refresh_token": {AuthenticationTokens.GSM.refresh_token},
			"site_name":     {""},
		}
		targetURL, _ := url.JoinPath(GSMBaseUrl, "token")
		targetURL += "?auth_mode=SAML"
		resp, err := http.PostForm(
			targetURL,
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
	*/
}

func getStuffFromCherwell(method string, path string, payload []byte, refreshToken bool) (io.ReadCloser, error) {
	client := &http.Client{
		Timeout: time.Second * 10,
	}
	newpath, _ := url.JoinPath(GSMBaseUrl, path)
	req, _ := http.NewRequest(method, newpath, bytes.NewReader(payload))
	if refreshToken {
		returnOrGetGSMAccessToken()
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", AuthenticationTokens.GSM.access_token))
	req.Header.Set("Content-type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("What happened? %s\n", err)
		return nil, err
	}
	return resp.Body, err
}