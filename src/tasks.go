package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
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
}

var GSMAuthWebServer *http.Server

func GetGSM() {
	if AuthenticationTokens.GSM.access_token == "" || AuthenticationTokens.GSM.expiration.Before(time.Now()) {
		// Login if expired
		go func() {
			GSMAuthWebServer := &http.Server{Addr: ":84", Handler: nil}
			http.HandleFunc("/cherwell", authenticateToCherwell)
			if err := GSMAuthWebServer.ListenAndServe(); err != nil {
				log.Fatal(err)
			}
		}()
		browser.OpenURL(`https://serviceportal.griffith.edu.au/cherwellapi/saml/login.cshtml?finalUri=http://localhost:84/cherwell?code=xx`)
	} else {
		var tasksResponse CherwellSearchResponse
		// Get my Tasks
		AppStatus.MyTasksFromGSM = [][]string{}
		// Download Tasks
		for page := 1; page < 200; page++ {
			response, _ := GetMyTasksFromGSMForPage(page)
			_ = json.Unmarshal(response, &tasksResponse)
			if len(tasksResponse.BusinessObjects) > 0 {
				for _, x := range tasksResponse.BusinessObjects {
					row := []string{}
					for _, y := range x.Fields {
						row = append(row, y.Value)
					}
					AppStatus.MyTasksFromGSM = append(AppStatus.MyTasksFromGSM, row)
				}
			}
			if len(tasksResponse.BusinessObjects) != 200 {
				break
			}
		}
		// Download Incidents
		AppStatus.MyIncidentsFromGSM = [][]string{}
		for page := 1; page < 200; page++ {
			response, _ := GetMyIncidentsFromGSMForPage(page)
			_ = json.Unmarshal(response, &tasksResponse)
			if len(tasksResponse.BusinessObjects) > 0 {
				for _, x := range tasksResponse.BusinessObjects {
					row := []string{}
					for _, y := range x.Fields {
						row = append(row, y.Value)
					}
					AppStatus.MyIncidentsFromGSM = append(AppStatus.MyIncidentsFromGSM, row)
				}
			}
			if len(tasksResponse.BusinessObjects) != 200 {
				break
			}
		}
		// Download my requests
		AppStatus.MyRequestsInGSM = [][]string{}
		for page := 1; page < 200; page++ {
			response, _ := GetMyRequestsInGSMForPage(page)
			_ = json.Unmarshal(response, &tasksResponse)
			if len(tasksResponse.BusinessObjects) > 0 {
				for _, x := range tasksResponse.BusinessObjects {
					row := []string{}
					for _, y := range x.Fields {
						row = append(row, y.Value)
					}
					AppStatus.MyRequestsInGSM = append(AppStatus.MyRequestsInGSM, row)
				}
			}
			if len(tasksResponse.BusinessObjects) != 200 {
				break
			}
		}
		// Download Team
		AppStatus.MyTeamIncidentsFromGSM = [][]string{}
		for page := 1; page < 200; page++ {
			response, _ := GetMyTeamIncidentsInGSMForPage(page)
			_ = json.Unmarshal(response, &tasksResponse)
			if len(tasksResponse.BusinessObjects) > 0 {
				for _, x := range tasksResponse.BusinessObjects {
					fmt.Printf("%s vs %s\n", x.Fields[6].Value, AuthenticationTokens.GSM.cherwelluser)
					if x.Fields[6].Value == AuthenticationTokens.GSM.cherwelluser {
						continue
					}
					row := []string{}
					for _, y := range x.Fields {
						row = append(row, y.Value)
					}
					AppStatus.MyTeamIncidentsFromGSM = append(AppStatus.MyTeamIncidentsFromGSM, row)
				}
			}
			fmt.Printf("Found %d\n", len(AppStatus.MyTeamIncidentsFromGSM))
			if len(tasksResponse.BusinessObjects) != 200 {
				break
			}
		}
		taskWindowRefresh()
		// Add personal priorities
		// Return
	}
}

func GetMyTasksFromGSMForPage(page int) ([]byte, error) {
	return SearchCherwellFor(GSMSearchQuery{
		Filters: []GSMFilter{
			{FieldId: "93cfd5a4e1d0ba5d3423e247b08dfd1286cae772cf", Operator: "eq", Value: AuthenticationTokens.GSM.cherwelluser},
			{FieldId: "9368f0fb7b744108a666984c21afc932562eb7dc16", Operator: "eq", Value: "Acknowledged"},
			{FieldId: "9368f0fb7b744108a666984c21afc932562eb7dc16", Operator: "eq", Value: "New"},
			{FieldId: "9368f0fb7b744108a666984c21afc932562eb7dc16", Operator: "eq", Value: "In Progress"},
			{FieldId: "9368f0fb7b744108a666984c21afc932562eb7dc16", Operator: "eq", Value: "On Hold"},
		},
		BusObjId:   "9355d5ed41e384ff345b014b6cb1c6e748594aea5b",
		PageNumber: page,
		PageSize:   200,
		Fields: []string{
			"BO:9355d5ed41e384ff345b014b6cb1c6e748594aea5b,FI:9355d5ed416bbc9408615c4145978ff8538a3f6eb4",                                     // Created Date/Time
			"BO:6dd53665c0c24cab86870a21cf6434ae,FI:6ae282c55e8e4266ae66ffc070c17fa3,RE:93694ed12e2e9bb908131846b7a9c67ec72b811676",           // Incident ID
			"BO:6dd53665c0c24cab86870a21cf6434ae,FI:93e8ea93ff67fd95118255419690a50ef2d56f910c,RE:93694ed12e2e9bb908131846b7a9c67ec72b811676", // Incident Short Desc
			"93ad98a2d68a61778eda3d4d9cbb30acbfd458aea4", // Task Title
			"9368f0fb7b744108a666984c21afc932562eb7dc16", // Status
			"9355d6d84625cc7c1a7a48435ea878328f1646c7af", // Parent Type ID
			"9355d6d6f3d7531087eab4456482100476d46ac59b", // Parent RecID
			"9355d5fabd7763ad02894d43eca25b5432e555e1c6", // Completion Details
			"93d5409c4bcbf7a38ed75a47dd92671f374236fa32", // TaskID
			"BO:6dd53665c0c24cab86870a21cf6434ae,FI:83c36313e97b4e6b9028aff3b401b71c,RE:93694ed12e2e9bb908131846b7a9c67ec72b811676", // Incident Priority
		},
		Sorting: []GSMSort{
			{FieldID: "BO:6dd53665c0c24cab86870a21cf6434ae,FI:83c36313e97b4e6b9028aff3b401b71c,RE:93694ed12e2e9bb908131846b7a9c67ec72b811676", SortDirection: 1},
			{FieldID: "9355d5ed416bbc9408615c4145978ff8538a3f6eb4", SortDirection: 1},
		},
	})
}

func GetMyIncidentsFromGSMForPage(page int) ([]byte, error) {
	return SearchCherwellFor(GSMSearchQuery{
		Filters: []GSMFilter{
			{FieldId: "BO:6dd53665c0c24cab86870a21cf6434ae,FI:9339fc404e39ae705648ab43969f29262e6d167606", Operator: "eq", Value: AuthenticationTokens.GSM.cherwelluser},
			{FieldId: "5eb3234ae1344c64a19819eda437f18d", Operator: "eq", Value: "Acknowledged"},
			{FieldId: "5eb3234ae1344c64a19819eda437f18d", Operator: "eq", Value: "New"},
			{FieldId: "5eb3234ae1344c64a19819eda437f18d", Operator: "eq", Value: "In Progress"},
			{FieldId: "5eb3234ae1344c64a19819eda437f18d", Operator: "eq", Value: "On Hold"},
		},
		BusObjId:   "6dd53665c0c24cab86870a21cf6434ae",
		PageNumber: page,
		PageSize:   200,
		Fields: []string{
			"BO:6dd53665c0c24cab86870a21cf6434ae,FI:c1e86f31eb2c4c5f8e8615a5189e9b19",           // Created Date/Time
			"BO:6dd53665c0c24cab86870a21cf6434ae,FI:6ae282c55e8e4266ae66ffc070c17fa3",           // Incident ID
			"BO:6dd53665c0c24cab86870a21cf6434ae,FI:93e8ea93ff67fd95118255419690a50ef2d56f910c", // Incident Short Desc
			"BO:6dd53665c0c24cab86870a21cf6434ae,FI:5eb3234ae1344c64a19819eda437f18d",           // Status
			"BO:6dd53665c0c24cab86870a21cf6434ae,FI:83c36313e97b4e6b9028aff3b401b71c",           // Incident Priority
		},
		Sorting: []GSMSort{
			{FieldID: "BO:6dd53665c0c24cab86870a21cf6434ae,FI:83c36313e97b4e6b9028aff3b401b71c", SortDirection: 1},
			{FieldID: "BO:6dd53665c0c24cab86870a21cf6434ae,FI:c1e86f31eb2c4c5f8e8615a5189e9b19", SortDirection: 1},
		},
	})
}

func GetMyRequestsInGSMForPage(page int) ([]byte, error) {
	return SearchCherwellFor(GSMSearchQuery{
		Filters: []GSMFilter{
			{FieldId: "BO:6dd53665c0c24cab86870a21cf6434ae,FI:941aa0889094428a6f4c054dbea345b09b4d87c77e", Operator: "eq", Value: AuthenticationTokens.GSM.userid},
			{FieldId: "5eb3234ae1344c64a19819eda437f18d", Operator: "eq", Value: "Acknowledged"},
			{FieldId: "5eb3234ae1344c64a19819eda437f18d", Operator: "eq", Value: "New"},
			{FieldId: "5eb3234ae1344c64a19819eda437f18d", Operator: "eq", Value: "In Progress"},
			{FieldId: "5eb3234ae1344c64a19819eda437f18d", Operator: "eq", Value: "On Hold"},
		},
		BusObjId:   "6dd53665c0c24cab86870a21cf6434ae",
		PageNumber: page,
		PageSize:   200,
		Fields: []string{
			"BO:6dd53665c0c24cab86870a21cf6434ae,FI:c1e86f31eb2c4c5f8e8615a5189e9b19",           // Created Date/Time
			"BO:6dd53665c0c24cab86870a21cf6434ae,FI:6ae282c55e8e4266ae66ffc070c17fa3",           // Incident ID
			"BO:6dd53665c0c24cab86870a21cf6434ae,FI:93e8ea93ff67fd95118255419690a50ef2d56f910c", // Incident Short Desc
			"BO:6dd53665c0c24cab86870a21cf6434ae,FI:5eb3234ae1344c64a19819eda437f18d",           // Status
			"BO:6dd53665c0c24cab86870a21cf6434ae,FI:83c36313e97b4e6b9028aff3b401b71c",           // Incident Priority
		},
		Sorting: []GSMSort{
			{FieldID: "BO:6dd53665c0c24cab86870a21cf6434ae,FI:83c36313e97b4e6b9028aff3b401b71c", SortDirection: 1},
			{FieldID: "BO:6dd53665c0c24cab86870a21cf6434ae,FI:c1e86f31eb2c4c5f8e8615a5189e9b19", SortDirection: 1},
		},
	})
}
func GetMyTeamIncidentsInGSMForPage(page int) ([]byte, error) {
	baseFilter := []GSMFilter{
		{FieldId: "5eb3234ae1344c64a19819eda437f18d", Operator: "eq", Value: "Acknowledged"},
		{FieldId: "5eb3234ae1344c64a19819eda437f18d", Operator: "eq", Value: "New"},
		{FieldId: "5eb3234ae1344c64a19819eda437f18d", Operator: "eq", Value: "In Progress"},
		{FieldId: "5eb3234ae1344c64a19819eda437f18d", Operator: "eq", Value: "On Hold"},
	}
	for _, x := range AuthenticationTokens.GSM.teams {
		baseFilter = append(baseFilter, GSMFilter{FieldId: "BO:6dd53665c0c24cab86870a21cf6434ae,FI:9339fc404e312b6d43436041fc8af1c07c6197f559", Operator: "eq", Value: x})
	}
	return SearchCherwellFor(GSMSearchQuery{
		Filters:    baseFilter,
		BusObjId:   "6dd53665c0c24cab86870a21cf6434ae",
		PageNumber: page,
		PageSize:   200,
		Fields: []string{
			"BO:6dd53665c0c24cab86870a21cf6434ae,FI:c1e86f31eb2c4c5f8e8615a5189e9b19",           // Created Date/Time
			"BO:6dd53665c0c24cab86870a21cf6434ae,FI:6ae282c55e8e4266ae66ffc070c17fa3",           // Incident ID
			"BO:6dd53665c0c24cab86870a21cf6434ae,FI:93e8ea93ff67fd95118255419690a50ef2d56f910c", // Incident Short Desc
			"BO:6dd53665c0c24cab86870a21cf6434ae,FI:5eb3234ae1344c64a19819eda437f18d",           // Status
			"BO:6dd53665c0c24cab86870a21cf6434ae,FI:83c36313e97b4e6b9028aff3b401b71c",           // Incident Priority
			"BO:6dd53665c0c24cab86870a21cf6434ae,FI:9339fc404e4c93350bf5be446fb13d693b0bb7f219", // Owner Name
			"BO:6dd53665c0c24cab86870a21cf6434ae,FI:9339fc404e39ae705648ab43969f29262e6d167606", // Owner to ID
		},
		Sorting: []GSMSort{
			{FieldID: "BO:6dd53665c0c24cab86870a21cf6434ae,FI:83c36313e97b4e6b9028aff3b401b71c", SortDirection: 1},
			{FieldID: "BO:6dd53665c0c24cab86870a21cf6434ae,FI:c1e86f31eb2c4c5f8e8615a5189e9b19", SortDirection: 1},
		},
	})
}

func SearchCherwellFor(toSend GSMSearchQuery) ([]byte, error) {
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
					AuthenticationTokens.GSM.expiration, _ = time.Parse("Mon, 02 Jan 2016 15:04:05 GMT", CherwellToken.Expires)
				}
				var response []byte
				var decodedResponse CherwellSearchResponse
				// Get my UserID in Cherwell
				response, _ = SearchCherwellFor(GSMSearchQuery{
					Filters: []GSMFilter{
						{FieldId: "941798910e24d4b1ae3c6a408cb3f8c5eba26e2b2b", Operator: "eq", Value: AuthenticationTokens.GSM.userid},
					},
					BusObjId: "9338216b3c549b75607cf54667a4e67d1f644d9fed",
					Fields:   []string{"933a15d17f1f10297df7604b58a76734d6106ac428"},
				})
				_ = json.Unmarshal(response, &decodedResponse)
				AuthenticationTokens.GSM.cherwelluser = decodedResponse.BusinessObjects[0].BusObRecId
				// Get my TeamIDs in Cherwell
				var teamResponse struct {
					Teams []struct {
						TeamID   string `json:"teamId"`
						TeamName string `json:"teamName"`
					} `json:"teams"`
				}
				response, _ = getStuffFromCherwell(
					"GET",
					"api/V2/getusersteams/userrecordid/"+AuthenticationTokens.GSM.cherwelluser,
					[]byte{})
				_ = json.Unmarshal(response, &teamResponse)
				AuthenticationTokens.GSM.teams = []string{}
				for _, x := range teamResponse.Teams {
					AuthenticationTokens.GSM.teams = append(AuthenticationTokens.GSM.teams, x.TeamID)
				}
				GetGSM()
			}
		}
	} else {
		// Redirect to MS Auth
		browser.OpenURL(`https://serviceportal.griffith.edu.au/cherwellapi/saml/login.cshtml?finalUri=http://localhost:84/cherwell?code=xx`)
	}
}

func getStuffFromCherwell(method string, path string, payload []byte) ([]byte, error) {
	var result []byte
	client := &http.Client{
		Timeout: time.Second * 10,
	}
	newpath, _ := url.JoinPath("https://griffith.cherwellondemand.com/CherwellAPI/", path)
	req, _ := http.NewRequest(method, newpath, bytes.NewReader(payload))
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", AuthenticationTokens.GSM.access_token))
	req.Header.Set("Content-type", "application/json")
	resp, err := client.Do(req)
	if err == nil {
		defer resp.Body.Close()
		result, err = ioutil.ReadAll(resp.Body)
	}
	return result, err
}

func GetJIRA() {

}

func GetPlanner() {

}

func LoginToMS() {

}
