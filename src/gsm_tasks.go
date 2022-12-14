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
	"strings"
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

func singleThreadReturnGSMAccessToken() {
	for {
		_, ok := <-GSMAccessTokenRequestsChan
		if !ok {
			// Close channel
			break
		}
		if time.Now().After(AuthenticationTokens.GSM.expiration) {
			AuthenticationTokens.GSM.access_token = ""
			connectionStatusBox(false, "G")
		}
		for {
			if AuthenticationTokens.GSM.access_token != "" {
				break
			}
			connectionStatusBox(false, "G")
			time.Sleep(1 * time.Second)
		}
		GSMAccessTokenChan <- AuthenticationTokens.GSM.access_token
	}
}

func returnOrGetGSMAccessToken() string {
	GSMAccessTokenRequestsChan <- time.Now().String()
	return <-GSMAccessTokenChan
}

func AuthenticateToGSM() {
	AuthenticationTokens.GSM.access_token = ""
	browser.OpenURL(GSMAuthURL)
}

func GetGSM() {
	go func() {
		DownloadTasks()
		taskWindowRefresh("CWTasks")
	}()
	go func() {
		DownloadIncidents()
		taskWindowRefresh("CWIncidents")
	}()
	go func() {
		DownloadMyRequests()
		taskWindowRefresh("CWRequests")
	}()
	go func() {
		DownloadTeam()
		taskWindowRefresh("CWTeamIncidents")
	}()
}

func DownloadTasks() {
	returnOrGetGSMAccessToken()
	activeTaskStatusUpdate(1)
	defer activeTaskStatusUpdate(-1)
	AppStatus.MyTasksFromGSM = []TaskResponseStruct{}
	for page := 1; page < 200; page++ {
		tasksResponse, _ := GetMyTasksFromGSMForPage(page)
		if len(tasksResponse.BusinessObjects) > 0 {
			for _, x := range tasksResponse.BusinessObjects {
				row := TaskResponseStruct{
					ID:         x.BusObPublicId,
					BusObRecId: x.BusObRecId,
				}
				for _, y := range x.Fields {
					switch y.FieldId {
					case CWFields.Task.TaskStatus:
						row.Status = y.Value
					case CWFields.Task.TaskTitle:
						row.Title = y.Value
					case CWFields.Task.IncidentShortDesc:
						row.ParentTitle = y.Value
					case CWFields.Task.IncidentID:
						row.ParentID = y.Value
					case CWFields.Task.IncidentInternalID:
						row.ParentIDInternal = y.Value
					case CWFields.Task.CreatedDateTime:
						row.CreatedDateTime, _ = time.Parse("1/2/2006 3:04:05 PM", y.Value)
					case CWFields.Task.IncidentPriority:
						row.Priority = y.Value
						row.PriorityOverride = y.Value
					}
				}
				if val, ok := priorityOverrides.CWIncidents[row.ID]; ok {
					row.PriorityOverride = val
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
			if AppStatus.MyTasksFromGSM[i].PriorityOverride == AppStatus.MyTasksFromGSM[j].PriorityOverride {
				toReturn = AppStatus.MyTasksFromGSM[i].CreatedDateTime.Before(AppStatus.MyTasksFromGSM[j].CreatedDateTime)
			} else {
				toReturn = AppStatus.MyTasksFromGSM[i].PriorityOverride < AppStatus.MyTasksFromGSM[j].PriorityOverride
			}
			return toReturn
		},
	)
}

func DownloadIncidents() {
	returnOrGetGSMAccessToken()
	activeTaskStatusUpdate(1)
	defer activeTaskStatusUpdate(-1)
	AppStatus.MyIncidentsFromGSM = []TaskResponseStruct{}
	for page := 1; page < 200; page++ {
		tasksResponse, _ := GetMyIncidentsFromGSMForPage(page)
		if len(tasksResponse.BusinessObjects) > 0 {
			for _, x := range tasksResponse.BusinessObjects {
				row := TaskResponseStruct{
					ID: x.BusObPublicId,
				}
				for _, y := range x.Fields {
					switch y.FieldId {
					case CWFields.Incident.Status:
						row.Status = y.Value
					case CWFields.Incident.ShortDesc:
						row.Title = y.Value
					case CWFields.Incident.CreatedDateTime:
						row.CreatedDateTime, _ = time.Parse("1/2/2006 3:04:05 PM", y.Value)
					case CWFields.Incident.Priority:
						row.Priority = y.Value
						row.PriorityOverride = y.Value
					}
				}
				if val, ok := priorityOverrides.CWIncidents[row.ID]; ok {
					row.PriorityOverride = val
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
	returnOrGetGSMAccessToken()
	activeTaskStatusUpdate(1)
	defer activeTaskStatusUpdate(-1)
	AppStatus.MyRequestsInGSM = []TaskResponseStruct{}
	for page := 1; page < 200; page++ {
		tasksResponse, _ := GetMyRequestsInGSMForPage(page)
		if len(tasksResponse.BusinessObjects) > 0 {
			for _, x := range tasksResponse.BusinessObjects {
				row := TaskResponseStruct{
					ID: x.BusObPublicId,
				}
				for _, y := range x.Fields {
					switch y.FieldId {
					case CWFields.Incident.Status:
						row.Status = y.Value
					case CWFields.Incident.ShortDesc:
						row.Title = y.Value
					case CWFields.Incident.CreatedDateTime:
						row.CreatedDateTime, _ = time.Parse("1/2/2006 3:04:05 PM", y.Value)
					case CWFields.Incident.Priority:
						row.Priority = y.Value
						row.PriorityOverride = y.Value
					}
				}
				if val, ok := priorityOverrides.CWIncidents[row.ID]; ok {
					row.PriorityOverride = val
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
	returnOrGetGSMAccessToken()
	activeTaskStatusUpdate(1)
	defer activeTaskStatusUpdate(-1)
	AppStatus.MyTeamIncidentsFromGSM = []TaskResponseStruct{}
	for page := 1; page < 200; page++ {
		tasksResponse, _ := GetMyTeamIncidentsInGSMForPage(page)
		if len(tasksResponse.BusinessObjects) > 0 {
			for _, x := range tasksResponse.BusinessObjects {
				if x.Fields[6].Value == AuthenticationTokens.GSM.cherwelluser {
					continue
				}
				row := TaskResponseStruct{
					ID: x.BusObPublicId,
				}
				for _, y := range x.Fields {
					switch y.FieldId {
					case CWFields.Incident.Status:
						row.Status = y.Value
					case CWFields.Incident.ShortDesc:
						row.Title = y.Value
					case CWFields.Incident.CreatedDateTime:
						row.CreatedDateTime, _ = time.Parse("1/2/2006 3:04:05 PM", y.Value)
					case CWFields.Incident.Priority:
						row.Priority = y.Value
						row.PriorityOverride = y.Value
					case CWFields.Incident.OwnerName:
						row.Owner = y.Value
					}
				}
				if val, ok := priorityOverrides.CWIncidents[row.ID]; ok {
					row.PriorityOverride = val
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
	BusObId            string
	BusObPubRecId      string
	OwnerID            string
	OwnerName          string
	CreatedDateTime    string
	TaskTitle          string
	TaskStatus         string
	TaskID             string
	IncidentID         string
	IncidentInternalID string
	IncidentShortDesc  string
	IncidentPriority   string
	TeamID             string
}
type CWFieldIDSIncident struct {
	OwnerID             string
	Status              string
	CreatedDateTime     string
	IncidentID          string
	ShortDesc           string
	Priority            string
	RequestorSNumber    string
	TeamID              string
	OwnerName           string
	BusObId             string
	JournalRelationship string
}
type CWFieldIDSJournal struct {
	TypeName    string
	DateCreated string
	Details     string
}
type CWFieldIDSUser struct {
	BusObId       string
	UserID        string
	Name          string
	Email         string
	DefaultTeamID string
}
type CWFieldIDs struct {
	Task     CWFieldIDSTask
	Incident CWFieldIDSIncident
	Journal  CWFieldIDSJournal
	User     CWFieldIDSUser
}

var CWFields = CWFieldIDs{
	Task: CWFieldIDSTask{
		BusObId:            "9355d5ed41e384ff345b014b6cb1c6e748594aea5b",
		OwnerID:            "93cfd5a4e1d0ba5d3423e247b08dfd1286cae772cf",
		OwnerName:          "93cfd5a4e13f7d4a4de1914f638abebee3a982bb50",
		CreatedDateTime:    "9355d5ed416bbc9408615c4145978ff8538a3f6eb4",
		TaskTitle:          "93ad98a2d68a61778eda3d4d9cbb30acbfd458aea4",
		TaskStatus:         "9368f0fb7b744108a666984c21afc932562eb7dc16",
		TaskID:             "93d5409c4bcbf7a38ed75a47dd92671f374236fa32",
		IncidentID:         "BO:6dd53665c0c24cab86870a21cf6434ae,FI:6ae282c55e8e4266ae66ffc070c17fa3,RE:93694ed12e2e9bb908131846b7a9c67ec72b811676",
		IncidentInternalID: "BO:6dd53665c0c24cab86870a21cf6434ae,FI:fa03d51b709e4a6eb2d52885b2ef7e04,RE:93694ed12e2e9bb908131846b7a9c67ec72b811676",
		IncidentShortDesc:  "BO:6dd53665c0c24cab86870a21cf6434ae,FI:93e8ea93ff67fd95118255419690a50ef2d56f910c,RE:93694ed12e2e9bb908131846b7a9c67ec72b811676",
		IncidentPriority:   "BO:6dd53665c0c24cab86870a21cf6434ae,FI:83c36313e97b4e6b9028aff3b401b71c,RE:93694ed12e2e9bb908131846b7a9c67ec72b811676",
		TeamID:             "93cfd5a4e15baa9882a7994995b332c30e677bdcee",
	},
	Incident: CWFieldIDSIncident{
		OwnerID:             "9339fc404e39ae705648ab43969f29262e6d167606",
		Status:              "5eb3234ae1344c64a19819eda437f18d",
		CreatedDateTime:     "c1e86f31eb2c4c5f8e8615a5189e9b19",
		IncidentID:          "6ae282c55e8e4266ae66ffc070c17fa3",
		ShortDesc:           "93e8ea93ff67fd95118255419690a50ef2d56f910c",
		Priority:            "83c36313e97b4e6b9028aff3b401b71c",
		RequestorSNumber:    "941aa0889094428a6f4c054dbea345b09b4d87c77e",
		TeamID:              "9339fc404e312b6d43436041fc8af1c07c6197f559",
		OwnerName:           "9339fc404e4c93350bf5be446fb13d693b0bb7f219",
		BusObId:             "6dd53665c0c24cab86870a21cf6434ae",
		JournalRelationship: "934d819237a4ec95ae69394e539440a17591e9d490",
	},
	Journal: CWFieldIDSJournal{
		TypeName:    "93412229b86b8c228bf6ef4380932b030f30fdb408",
		DateCreated: "93412229b8324b8c66ae4648dc9531d9506e5dadd6",
		Details:     "9341223bbcef1e2b8dfa6048a2bb4be1e94bad60ac",
	},
	User: CWFieldIDSUser{
		BusObId:       "9338216b3c549b75607cf54667a4e67d1f644d9fed",
		UserID:        "941798910e24d4b1ae3c6a408cb3f8c5eba26e2b2b",
		Name:          "93382178280a07634f62d74fc4bc587e3b3f479776",
		Email:         "933821793f43a638cf23e34723b907956d324ad303",
		DefaultTeamID: "933a15d17f1f10297df7604b58a76734d6106ac428",
	},
}

type FoundGSMPeople struct {
	UserID        string
	Name          string
	Email         string
	Teams         map[string]string
	DefaultTeamID string
}

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
			CWFields.Task.IncidentInternalID,
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
	}, false)
	if err == nil {
		defer r.Close()
		_ = json.NewDecoder(r).Decode(&tasksResponse)
	}
	return tasksResponse, err
}

func GetMyIncidentsFromGSMForPage(page int) (CherwellSearchResponse, error) {
	var tasksResponse CherwellSearchResponse
	returnOrGetGSMAccessToken()
	r, err := SearchCherwellFor(GSMSearchQuery{
		Filters: []GSMFilter{
			{FieldId: CWFields.Incident.OwnerID, Operator: "eq", Value: AuthenticationTokens.GSM.cherwelluser},
			{FieldId: CWFields.Incident.Status, Operator: "eq", Value: "Acknowledged"},
			{FieldId: CWFields.Incident.Status, Operator: "eq", Value: "New"},
			{FieldId: CWFields.Incident.Status, Operator: "eq", Value: "In Progress"},
			{FieldId: CWFields.Incident.Status, Operator: "eq", Value: "On Hold"},
			{FieldId: CWFields.Incident.Status, Operator: "eq", Value: "Pending"},
		},
		BusObjId:   CWFields.Incident.BusObId,
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
	}, false)
	if err == nil {
		defer r.Close()
		_ = json.NewDecoder(r).Decode(&tasksResponse)
	}
	return tasksResponse, err
}

func GetMyRequestsInGSMForPage(page int) (CherwellSearchResponse, error) {
	var tasksResponse CherwellSearchResponse
	returnOrGetGSMAccessToken()
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
		PageNumber: 1,
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
	}, false)
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
	// Refresh my teams, just in case.
	// Get my TeamIDs in Cherwell
	returnOrGetGSMAccessToken()
	var teamResponse struct {
		Teams []struct {
			TeamID   string `json:"teamId"`
			TeamName string `json:"teamName"`
		} `json:"teams"`
	}
	r, err := getStuffFromCherwell(
		"GET",
		"api/V2/getusersteams/userrecordid/"+AuthenticationTokens.GSM.cherwelluser,
		[]byte{},
		false)
	if err == nil {
		defer r.Close()
		_ = json.NewDecoder(r).Decode(&teamResponse)
		AuthenticationTokens.GSM.allteams = []string{}
		for _, x := range teamResponse.Teams {
			AuthenticationTokens.GSM.allteams = append(AuthenticationTokens.GSM.allteams, x.TeamID)
		}
	}
	//
	for _, x := range AuthenticationTokens.GSM.allteams {
		baseFilter = append(baseFilter, GSMFilter{FieldId: CWFields.Incident.TeamID, Operator: "eq", Value: x})
	}
	if len(AuthenticationTokens.GSM.myteam) > 0 {
		baseFilter = append(baseFilter, GSMFilter{FieldId: CWFields.Incident.TeamID, Operator: "eq", Value: AuthenticationTokens.GSM.myteam})
	}
	r, err = SearchCherwellFor(GSMSearchQuery{
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
	}, false)
	if err == nil {
		defer r.Close()
		_ = json.NewDecoder(r).Decode(&tasksResponse)
	}
	return tasksResponse, err
}

func FindPeopleToReasignTo(lookfor string) ([]FoundGSMPeople, error) {
	var searchResponse CherwellSearchResponse
	var foundPeople []FoundGSMPeople
	alreadyLooked := map[string]bool{}
	fmt.Printf("Search built\n")
	searchingFor := GSMSearchQuery{
		Filters: []GSMFilter{
			{FieldId: CWFields.User.Name, Operator: "contains", Value: lookfor},
		},
		BusObjId:   CWFields.User.BusObId,
		PageNumber: 1,
		PageSize:   30,
		Fields: []string{
			CWFields.User.UserID,
			CWFields.User.DefaultTeamID,
			CWFields.User.Name,
			CWFields.User.Email,
		},
		Sorting: []GSMSort{
			{FieldID: CWFields.User.Name, SortDirection: 1},
		},
	}
	r, err := SearchCherwellFor(searchingFor, false)
	if err == nil {
		defer r.Close()
		_ = json.NewDecoder(r).Decode(&searchResponse)
		for _, x1 := range searchResponse.BusinessObjects {
			person := FoundGSMPeople{
				UserID: x1.BusObRecId,
			}
			person.Teams = GetTeamsForUser(person.UserID)
			for _, y := range x1.Fields {
				switch y.Name {
				case "DefaultTeamID":
					person.DefaultTeamID = y.Value
				case "Email":
					person.Email = y.Value
				case "FullName":
					person.Name = y.Value
				}
			}
			alreadyLooked[x1.BusObRecId] = true
			foundPeople = append(foundPeople, person)
		}
	}
	searchingFor = GSMSearchQuery{
		Filters: []GSMFilter{
			{FieldId: CWFields.User.Email, Operator: "contains", Value: lookfor},
		},
		BusObjId:   CWFields.User.BusObId,
		PageNumber: 1,
		PageSize:   30,
		Fields: []string{
			CWFields.User.UserID,
			CWFields.User.DefaultTeamID,
			CWFields.User.Name,
			CWFields.User.Email,
		},
		Sorting: []GSMSort{
			{FieldID: CWFields.User.Name, SortDirection: 1},
		},
	}
	r, err = SearchCherwellFor(searchingFor, false)
	if err == nil {
		defer r.Close()
		_ = json.NewDecoder(r).Decode(&searchResponse)
		for _, x1 := range searchResponse.BusinessObjects {
			if _, ok := alreadyLooked[x1.BusObRecId]; !ok {
				person := FoundGSMPeople{
					UserID: x1.BusObRecId,
				}
				person.Teams = GetTeamsForUser(person.UserID)
				for _, y := range x1.Fields {
					switch y.Name {
					case "DefaultTeamID":
						person.DefaultTeamID = y.Value
					case "Email":
						person.Email = y.Value
					case "FullName":
						person.Name = y.Value
					}
				}
				foundPeople = append(foundPeople, person)
			}
		}
	}
	return foundPeople, err
}

type UserTeamsResponse struct {
	Teams []struct {
		TeamID   string `json:"teamId"`
		TeamName string `json:"teamName"`
	} `json:"teams"`
}

func GetTeamsForUser(userId string) map[string]string {
	var response UserTeamsResponse
	toReturn := map[string]string{}
	r, err := getStuffFromCherwell(
		"GET",
		"api/V2/getusersteams/userrecordid/"+userId,
		[]byte{},
		false)
	if err == nil {
		defer r.Close()
		err = json.NewDecoder(r).Decode(&response)
		if err == nil {
			for _, x := range response.Teams {
				toReturn[x.TeamID] = x.TeamName
			}
		}
	}
	return toReturn
}

type ReassignStruct struct {
	BusObId    string `json:"busObId"`
	BusObRecId string `json:"busObRecId"`
	Fields     []struct {
		Dirty       bool   `json:"dirty"`
		DisplayName string `json:"displayName"`
		FieldId     string `json:"fieldId"`
		FullFieldId string `json:"fullFieldId"`
		HTML        string `json:"html"`
		Name        string `json:"name"`
		Value       string `json:"value"`
	} `json:"fields"`
}

func ReassignTaskToPersonInTeam(incidentId string, userId string, teamId string) error {
	payload, _ := json.Marshal(ReassignStruct{
		BusObId:    CWFields.Task.BusObId,
		BusObRecId: incidentId,
		Fields: []struct {
			Dirty       bool   `json:"dirty"`
			DisplayName string `json:"displayName"`
			FieldId     string `json:"fieldId"`
			FullFieldId string `json:"fullFieldId"`
			HTML        string `json:"html"`
			Name        string `json:"name"`
			Value       string `json:"value"`
		}{
			{FieldId: CWFields.Task.OwnerID, Name: "OwnedByID", DisplayName: "Owned By ID", Value: userId, Dirty: false},
			{FieldId: CWFields.Task.TeamID, Name: "OwnedByTeamID", DisplayName: "Owned By Team ID", Value: teamId, Dirty: false},
		},
	})
	fmt.Printf("%s\n", payload)
	_, err := getStuffFromCherwell(
		"POST",
		"api/V1/savebusinessobject",
		payload,
		false)
	return err
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

type GSMRelatedBusinessObject struct {
	Filters          []GSMFilter `json:"filter"`
	ParentBusObId    string      `json:"parentBusObId"`
	ParentBusObRecId string      `json:"parentBusObRecId"`
	RelationshipId   string      `json:"relationshipId"`
	Fields           []string    `json:"fields"`
	Sorting          []GSMSort   `json:"sorting"`
}

type CherwellRelatedbusinessObject struct {
	Fields []struct {
		Dirty       bool   `json:"dirty"`
		DisplayName string `json:"displayName"`
		FieldId     string `json:"fieldId"`
		FullFieldId string `json:"fullFieldId"`
		HTML        string `json:"html"`
		Name        string `json:"name"`
		Value       string `json:"value"`
	} `json:"fields"`
}
type CherwellRelatedResponse struct {
	RelatedBusinessObjects []CherwellRelatedbusinessObject `json:"relatedBusinessObjects"`
}
type CherwellJournal struct {
	Date    time.Time
	Class   string
	Details string
}

func GetJournalNotesForIncident(incident string) ([]CherwellJournal, error) {
	var tasksResponse CherwellRelatedResponse
	var toReturn []CherwellJournal
	returnOrGetGSMAccessToken()

	toSend := GSMRelatedBusinessObject{
		Filters: []GSMFilter{
			{FieldId: CWFields.Journal.TypeName, Operator: "eq", Value: "Journal - Note"},
			{FieldId: CWFields.Journal.TypeName, Operator: "eq", Value: "Journal - Mail History"},
			{FieldId: CWFields.Journal.TypeName, Operator: "eq", Value: "Journal - SLM History"},
			{FieldId: CWFields.Journal.TypeName, Operator: "eq", Value: "Journal - Comment"},
		},
		ParentBusObId:    CWFields.Incident.BusObId,
		ParentBusObRecId: incident,
		RelationshipId:   CWFields.Incident.JournalRelationship,
		Fields: []string{
			CWFields.Journal.DateCreated,
			CWFields.Journal.Details,
			CWFields.Journal.TypeName,
		},
		Sorting: []GSMSort{
			{FieldID: CWFields.Journal.DateCreated, SortDirection: 0},
		},
	}
	sendThis, _ := json.Marshal(toSend)
	result, err := getStuffFromCherwell(
		"POST",
		"api/V1/getrelatedbusinessobject",
		sendThis,
		false)
	if err == nil {
		defer result.Close()
		_ = json.NewDecoder(result).Decode(&tasksResponse)
		for _, x := range tasksResponse.RelatedBusinessObjects {
			me := CherwellJournal{}
			for _, z := range x.Fields {
				switch z.Name {
				case "JournalTypeName":
					me.Class = strings.Replace(z.Value, "Journal - ", "", -1)
				case "Details":
					me.Details = z.Value
				case "CreatedDateTime":
					y, _ := time.Parse("1/2/2006 3:04:05 PM", z.Value)
					me.Date = y
				}
			}
			toReturn = append(toReturn, me)
		}
	}
	return toReturn, err
}

func authenticateToCherwell(w http.ResponseWriter, r *http.Request) {
	var CherwellToken CherwellAuthResponse
	query := r.URL.Query()
	if query.Get("code") != "" {
		// Get body
		ticket := r.PostFormValue("ticket")
		if len(ticket) > 0 {
			payload := url.Values{
				"grant_type": {"password"},
				"client_id":  {"814f9a74-c86a-451e-b6bb-deea65acf72a"},
				"username":   {r.PostFormValue("userId")},
				"password":   {ticket},
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
				fmt.Printf("Getting token active\n")
				AppStatus.GSMGettingToken = true
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
				fmt.Printf("Getting my user id")
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
				AuthenticationTokens.GSM.myteam = decodedResponse.BusinessObjects[0].Fields[1].Value
				connectionStatusBox(true, "G")
				AppStatus.GSMGettingToken = false
				fmt.Printf("Token got\n")
				w.Header().Add("Content-type", "text/html")
				fmt.Fprintf(w, "<html><head></head><body><H1>Authenticated<p>You are authenticated, you may close this window.</body></html>")
				activeTaskStatusUpdate(-1)
			}
		}
	} else {
		// Redirect to Cherwell AUTH
		browser.OpenURL(GSMAuthURL)
		activeTaskStatusUpdate(1)
	}
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
	if AuthenticationTokens.GSM.access_token == "" || time.Now().After(AuthenticationTokens.GSM.expiration) {
		gsmConnectionActive.Objects[1] = CloudDisconnect
		gsmConnectionActive.Refresh()
		return nil, fmt.Errorf("expired token")
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", AuthenticationTokens.GSM.access_token))
	req.Header.Set("Content-type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("What happened? %s\n", err)
		return nil, err
	}
	return resp.Body, err
}
