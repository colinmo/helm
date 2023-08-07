package tasks

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

	"golang.org/x/sync/singleflight"

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

type GSMTokenStatus int64

const (
	Inactive GSMTokenStatus = iota
	Pending
	Active
	Expired
)

type CherwellStruct struct {
	Task
	RedirectPath    string
	RedirectURI     string
	AccessToken     string
	UserID          string
	UserSnumber     string
	UserTeams       []string
	UserEmail       string
	DefaultTeam     string
	RefreshToken    string
	Expiration      time.Time
	MyTasks         []TaskResponseStruct
	MyIncidents     []TaskResponseStruct
	LoggedIncidents []TaskResponseStruct
	TeamIncidents   []TaskResponseStruct
	TeamTasks       []TaskResponseStruct
	BaseURL         string
	AuthURL         string
	StatusCallback  func(bool, string)
	TokenStatus     GSMTokenStatus
	G               *singleflight.Group
}

func (gsm *CherwellStruct) Init(baseRedirect string, accessToken string, refreshToken string, expiration time.Time) {
	gsm.TokenStatus = Inactive
	if accessToken != "" && time.Now().After(expiration) {
		gsm.TokenStatus = Expired
	} else if accessToken != "" {
		gsm.AccessToken = accessToken
		gsm.RefreshToken = refreshToken
		gsm.Expiration = expiration
		gsm.TokenStatus = Active
	}
	gsm.BaseURL = `https://griffith.cherwellondemand.com/CherwellAPI/`
	gsm.AuthURL = `https://serviceportal.griffith.edu.au/cherwellapi/saml/login.cshtml?finalUri=http://localhost:84/cherwell?code=xx`
	gsm.RedirectPath = "/cherwell"
	gsm.RedirectURI, _ = url.JoinPath(baseRedirect, gsm.RedirectPath)

	// Start the background runner
	gsm.G = &singleflight.Group{}
}

func (gsm *CherwellStruct) getGSMAccessToken() (bool, error) {
	_, e, _ := gsm.G.Do(
		"GetGSMToken",
		func() (interface{}, error) {
			if gsm.TokenStatus == Active && time.Now().After(gsm.Expiration) {
				// Expired token
				gsm.AccessToken = ""
				ConnectionStatusBox(false, "G")
				gsm.TokenStatus = Expired
				gsm.Refresh()
			}
			if gsm.TokenStatus != Active {
				// No token
				gsm.AccessToken = ""
				ConnectionStatusBox(false, "G")
				gsm.Refresh()
				return "", fmt.Errorf("pending token")
			}
			// Valid token
			return gsm.AccessToken, nil
		},
	)
	return e == nil, e
}

func (gsm *CherwellStruct) Login() {
	go func() {
		gsm.G.Do(
			"GSMLogin",
			func() (interface{}, error) {
				gsm.TokenStatus = Pending
				browser.OpenURL(gsm.AuthURL)
				for {
					if gsm.TokenStatus == Active {
						return "", nil
					}
				}
			},
		)
	}()
}

func (gsm *CherwellStruct) Download(taskWindow func(), incidentWindow func(), requestsWindow func(), teamIncidentWindow func(), teamTaskWindow func()) {
	b, _ := gsm.getGSMAccessToken()
	if b {
		gsm.DownloadTasks(taskWindow)
		gsm.DownloadIncidents(incidentWindow)
		gsm.DownloadMyRequests(requestsWindow)
		gsm.DownloadTeam(teamIncidentWindow)
		gsm.DownloadTeamTasks(teamTaskWindow)
	} else {
		gsm.Login()
	}
}

func (gsm *CherwellStruct) DownloadTasks(afterFunc func()) {
	ok, _ := gsm.getGSMAccessToken()
	if ok {
		go func() {
			gsm.G.Do(
				"DownloadGSMTasks",
				func() (interface{}, error) {
					ActiveTaskStatusUpdate(1)
					defer ActiveTaskStatusUpdate(-1)
					gsm.MyTasks = []TaskResponseStruct{}
					for page := 1; page < 200; page++ {
						tasksResponse, _ := gsm.GetMyTasksFromGSMForPage(page)
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
								if val, ok := PriorityOverrides.CWIncidents[row.ID]; ok {
									row.PriorityOverride = val
								}
								gsm.MyTasks = append(gsm.MyTasks, row)
							}
						}
						if len(tasksResponse.BusinessObjects) != 200 {
							break
						}
					}
					sort.SliceStable(
						gsm.MyTasks,
						func(i, j int) bool {
							var toReturn bool
							if gsm.MyTasks[i].PriorityOverride == gsm.MyTasks[j].PriorityOverride {
								toReturn = gsm.MyTasks[i].CreatedDateTime.Before(gsm.MyTasks[j].CreatedDateTime)
							} else {
								toReturn = gsm.MyTasks[i].PriorityOverride < gsm.MyTasks[j].PriorityOverride
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
}

func (gsm *CherwellStruct) DownloadIncidents(afterFunc func()) {
	ok, _ := gsm.getGSMAccessToken()
	if ok {
		go func() {
			gsm.G.Do(
				"DownloadGSMIncidents",
				func() (interface{}, error) {
					ActiveTaskStatusUpdate(1)
					defer ActiveTaskStatusUpdate(-1)
					gsm.MyIncidents = []TaskResponseStruct{}
					for page := 1; page < 200; page++ {
						tasksResponse, _ := gsm.GetMyIncidentsFromGSMForPage(page)
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
								if val, ok := PriorityOverrides.CWIncidents[row.ID]; ok {
									row.PriorityOverride = val
								}
								gsm.MyIncidents = append(gsm.MyIncidents, row)
							}
						}
						if len(tasksResponse.BusinessObjects) != 200 {
							break
						}
					}
					sort.SliceStable(
						gsm.MyIncidents,
						func(i, j int) bool {
							var toReturn bool
							if gsm.MyIncidents[i].PriorityOverride == gsm.MyIncidents[j].PriorityOverride {
								toReturn = gsm.MyIncidents[i].CreatedDateTime.Before(gsm.MyIncidents[j].CreatedDateTime)
							} else {
								toReturn = gsm.MyIncidents[i].PriorityOverride < gsm.MyIncidents[j].PriorityOverride
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
}

func (gsm *CherwellStruct) DownloadMyRequests(afterFunc func()) {
	ok, _ := gsm.getGSMAccessToken()
	if ok {
		go func() {
			gsm.G.Do(
				"DownloadGSMMyRequests",
				func() (interface{}, error) {
					ActiveTaskStatusUpdate(1)
					defer ActiveTaskStatusUpdate(-1)
					gsm.LoggedIncidents = []TaskResponseStruct{}
					for page := 1; page < 200; page++ {
						tasksResponse, _ := gsm.GetMyRequestsInGSMForPage(page)
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
								if val, ok := PriorityOverrides.CWIncidents[row.ID]; ok {
									row.PriorityOverride = val
								}
								gsm.LoggedIncidents = append(gsm.LoggedIncidents, row)
							}
						}
						if len(tasksResponse.BusinessObjects) != 200 {
							break
						}
					}
					sort.SliceStable(
						gsm.LoggedIncidents,
						func(i, j int) bool {
							var toReturn bool
							if gsm.LoggedIncidents[i].PriorityOverride == gsm.LoggedIncidents[j].PriorityOverride {
								toReturn = gsm.LoggedIncidents[i].CreatedDateTime.Before(gsm.LoggedIncidents[j].CreatedDateTime)
							} else {
								toReturn = gsm.LoggedIncidents[i].PriorityOverride < gsm.LoggedIncidents[j].PriorityOverride
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
}

func (gsm *CherwellStruct) DownloadTeam(afterFunc func()) {
	ok, _ := gsm.getGSMAccessToken()
	if ok {
		go func() {
			gsm.G.Do(
				"DownloadGSMTeam",
				func() (interface{}, error) {
					ActiveTaskStatusUpdate(1)
					defer ActiveTaskStatusUpdate(-1)
					gsm.TeamIncidents = []TaskResponseStruct{}
					for page := 1; page < 200; page++ {
						tasksResponse, _ := gsm.GetMyTeamIncidentsInGSMForPage(page)
						if len(tasksResponse.BusinessObjects) > 0 {
							for _, x := range tasksResponse.BusinessObjects {
								if x.Fields[6].Value == gsm.UserID {
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
									case CWFields.Incident.OwnerID:
										row.OwnerID = y.Value
									}
								}
								if val, ok := PriorityOverrides.CWIncidents[row.ID]; ok {
									row.PriorityOverride = val
								}
								if row.OwnerID == "" {
									gsm.TeamIncidents = append(gsm.TeamIncidents, row)
								}
							}
						}
						if len(tasksResponse.BusinessObjects) != 200 {
							break
						}
					}
					sort.SliceStable(
						gsm.TeamIncidents,
						func(i, j int) bool {
							var toReturn bool
							if gsm.TeamIncidents[i].PriorityOverride == gsm.TeamIncidents[j].PriorityOverride {
								toReturn = gsm.TeamIncidents[i].CreatedDateTime.Before(gsm.TeamIncidents[j].CreatedDateTime)
							} else {
								toReturn = gsm.TeamIncidents[i].PriorityOverride < gsm.TeamIncidents[j].PriorityOverride
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
}

func (gsm *CherwellStruct) DownloadTeamTasks(afterFunc func()) {
	ok, _ := gsm.getGSMAccessToken()
	if ok {
		go func() {
			gsm.G.Do(
				"DownloadGSMTeamTasks",
				func() (interface{}, error) {
					ActiveTaskStatusUpdate(1)
					defer ActiveTaskStatusUpdate(-1)
					gsm.TeamTasks = []TaskResponseStruct{}
					for page := 1; page < 200; page++ {
						tasksResponse, _ := gsm.GetMyTeamTasksFromGSMForPage(page)
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
									case CWFields.Task.OwnerName:
										row.Owner = y.Value
									case CWFields.Task.OwnerID:
										row.OwnerID = y.Value
									}
								}
								if val, ok := PriorityOverrides.CWIncidents[row.ID]; ok {
									row.PriorityOverride = val
								}
								if row.OwnerID == "" {
									gsm.TeamTasks = append(gsm.TeamTasks, row)
								}
							}
						}
						if len(tasksResponse.BusinessObjects) != 200 {
							break
						}
					}
					sort.SliceStable(
						gsm.TeamTasks,
						func(i, j int) bool {
							var toReturn bool
							if gsm.TeamTasks[i].PriorityOverride == gsm.TeamTasks[j].PriorityOverride {
								toReturn = gsm.TeamTasks[i].CreatedDateTime.Before(gsm.TeamTasks[j].CreatedDateTime)
							} else {
								toReturn = gsm.TeamTasks[i].PriorityOverride < gsm.TeamTasks[j].PriorityOverride
							}
							return toReturn
						},
					)
					afterFunc()
					return "", nil
				})
		}()
	}
}

type CWFieldIDSTask struct {
	BusObId            string
	BusObPubRecId      string
	OwnerID            string
	OwnedBy            string
	OwnerName          string
	OwnedByEmail       string
	CreatedDateTime    string
	TaskTitle          string
	TaskStatus         string
	TaskID             string
	IncidentID         string
	IncidentInternalID string
	IncidentShortDesc  string
	IncidentPriority   string
	TeamID             string
	TeamName           string
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
		OwnerID:            "BO:9355d5ed41e384ff345b014b6cb1c6e748594aea5b,FI:93cfd5a4e1d0ba5d3423e247b08dfd1286cae772cf",
		OwnedBy:            "BO:9355d5ed41e384ff345b014b6cb1c6e748594aea5b,FI:93cfd5a4e13f7d4a4de1914f638abebee3a982bb50",
		OwnerName:          "BO:9355d5ed41e384ff345b014b6cb1c6e748594aea5b,FI:93cfd5a4e13f7d4a4de1914f638abebee3a982bb50",
		OwnedByEmail:       "BO:9355d5ed41e384ff345b014b6cb1c6e748594aea5b,FI:93b3d14c5a6204e3ef30e94d69a4b8a6542c22a5e4",
		CreatedDateTime:    "BO:9355d5ed41e384ff345b014b6cb1c6e748594aea5b,FI:9355d5ed416bbc9408615c4145978ff8538a3f6eb4",
		TaskTitle:          "BO:9355d5ed41e384ff345b014b6cb1c6e748594aea5b,FI:93ad98a2d68a61778eda3d4d9cbb30acbfd458aea4",
		TaskStatus:         "BO:9355d5ed41e384ff345b014b6cb1c6e748594aea5b,FI:9368f0fb7b744108a666984c21afc932562eb7dc16",
		TaskID:             "BO:9355d5ed41e384ff345b014b6cb1c6e748594aea5b,FI:93d5409c4bcbf7a38ed75a47dd92671f374236fa32",
		IncidentID:         "BO:6dd53665c0c24cab86870a21cf6434ae,FI:6ae282c55e8e4266ae66ffc070c17fa3,RE:93694ed12e2e9bb908131846b7a9c67ec72b811676",
		IncidentInternalID: "BO:6dd53665c0c24cab86870a21cf6434ae,FI:fa03d51b709e4a6eb2d52885b2ef7e04,RE:93694ed12e2e9bb908131846b7a9c67ec72b811676",
		IncidentShortDesc:  "BO:6dd53665c0c24cab86870a21cf6434ae,FI:93e8ea93ff67fd95118255419690a50ef2d56f910c,RE:93694ed12e2e9bb908131846b7a9c67ec72b811676",
		IncidentPriority:   "BO:6dd53665c0c24cab86870a21cf6434ae,FI:83c36313e97b4e6b9028aff3b401b71c,RE:93694ed12e2e9bb908131846b7a9c67ec72b811676",
		TeamID:             "BO:9355d5ed41e384ff345b014b6cb1c6e748594aea5b,FI:93cfd5a4e15baa9882a7994995b332c30e677bdcee",
		TeamName:           "BO:9355d5ed41e384ff345b014b6cb1c6e748594aea5b,FI:93cfd5a4e10af4933a573444d08cbc412da491b42e",
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

func (gsm *CherwellStruct) GetMyTasksFromGSMForPage(page int) (CherwellSearchResponse, error) {
	var tasksResponse CherwellSearchResponse
	r, err := gsm.SearchCherwellFor(GSMSearchQuery{
		Filters: []GSMFilter{
			{FieldId: CWFields.Task.OwnerID, Operator: "eq", Value: gsm.UserID},
			{FieldId: CWFields.Task.TaskStatus, Operator: "eq", Value: "Acknowledged"},
			{FieldId: CWFields.Task.TaskStatus, Operator: "eq", Value: "New"},
			{FieldId: CWFields.Task.TaskStatus, Operator: "eq", Value: "Assigned"},
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
			CWFields.Task.OwnerID,
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

func (gsm *CherwellStruct) GetMyIncidentsFromGSMForPage(page int) (CherwellSearchResponse, error) {
	var tasksResponse CherwellSearchResponse
	r, err := gsm.SearchCherwellFor(GSMSearchQuery{
		Filters: []GSMFilter{
			{FieldId: CWFields.Incident.OwnerID, Operator: "eq", Value: gsm.UserID},
			{FieldId: CWFields.Incident.Status, Operator: "eq", Value: "Acknowledged"},
			{FieldId: CWFields.Incident.Status, Operator: "eq", Value: "New"},
			{FieldId: CWFields.Incident.Status, Operator: "eq", Value: "Assigned"},
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
	}, true)
	if err == nil {
		defer r.Close()
		_ = json.NewDecoder(r).Decode(&tasksResponse)
	}
	return tasksResponse, err
}

func (gsm *CherwellStruct) GetMyRequestsInGSMForPage(page int) (CherwellSearchResponse, error) {
	var tasksResponse CherwellSearchResponse
	r, err := gsm.SearchCherwellFor(GSMSearchQuery{
		Filters: []GSMFilter{
			{FieldId: CWFields.Incident.RequestorSNumber, Operator: "eq", Value: gsm.UserSnumber},
			{FieldId: CWFields.Incident.Status, Operator: "eq", Value: "Acknowledged"},
			{FieldId: CWFields.Incident.Status, Operator: "eq", Value: "New"},
			{FieldId: CWFields.Incident.Status, Operator: "eq", Value: "Assigned"},
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
	}, true)
	if err == nil {
		defer r.Close()
		_ = json.NewDecoder(r).Decode(&tasksResponse)
	}
	return tasksResponse, err
}

func (gsm *CherwellStruct) GetMyTeamIncidentsInGSMForPage(page int) (CherwellSearchResponse, error) {
	var tasksResponse CherwellSearchResponse
	baseFilter := []GSMFilter{
		{FieldId: CWFields.Incident.Status, Operator: "eq", Value: "Acknowledged"},
		{FieldId: CWFields.Incident.Status, Operator: "eq", Value: "New"},
		{FieldId: CWFields.Incident.Status, Operator: "eq", Value: "Assigned"},
		{FieldId: CWFields.Incident.Status, Operator: "eq", Value: "In Progress"},
		{FieldId: CWFields.Incident.Status, Operator: "eq", Value: "On Hold"},
		{FieldId: CWFields.Incident.Status, Operator: "eq", Value: "Pending"},
	}
	//
	for _, x := range gsm.UserTeams {
		baseFilter = append(baseFilter, GSMFilter{FieldId: CWFields.Incident.TeamID, Operator: "eq", Value: x})
	}
	if len(gsm.DefaultTeam) > 0 {
		baseFilter = append(baseFilter, GSMFilter{FieldId: CWFields.Incident.TeamID, Operator: "eq", Value: gsm.DefaultTeam})
	}
	r, err := gsm.SearchCherwellFor(GSMSearchQuery{
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

func (gsm *CherwellStruct) GetMyTeamTasksFromGSMForPage(page int) (CherwellSearchResponse, error) {
	var tasksResponse CherwellSearchResponse
	baseFilter := []GSMFilter{
		{FieldId: CWFields.Task.TaskStatus, Operator: "eq", Value: "Acknowledged"},
		{FieldId: CWFields.Task.TaskStatus, Operator: "eq", Value: "New"},
		{FieldId: CWFields.Task.TaskStatus, Operator: "eq", Value: "Assigned"},
		{FieldId: CWFields.Task.TaskStatus, Operator: "eq", Value: "In Progress"},
		{FieldId: CWFields.Task.TaskStatus, Operator: "eq", Value: "On Hold"},
		{FieldId: CWFields.Task.TaskStatus, Operator: "eq", Value: "Pending"},
		{FieldId: CWFields.Task.OwnerName, Operator: "eq", Value: ""},
	}
	//
	for _, x := range gsm.UserTeams {
		baseFilter = append(baseFilter, GSMFilter{FieldId: CWFields.Task.TeamID, Operator: "eq", Value: x})
	}
	if len(gsm.DefaultTeam) > 0 {
		baseFilter = append(baseFilter, GSMFilter{FieldId: CWFields.Task.TeamID, Operator: "eq", Value: gsm.DefaultTeam})
	}
	builtQuery := GSMSearchQuery{
		Filters:    baseFilter,
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
			CWFields.Task.OwnerName,
			CWFields.Task.OwnerID,
		},
	}
	r, err := gsm.SearchCherwellFor(builtQuery, true)
	if err == nil {
		defer r.Close()
		_ = json.NewDecoder(r).Decode(&tasksResponse)
	}
	return tasksResponse, err
}

func (gsm *CherwellStruct) FindPeopleToReasignTo(lookfor string) ([]FoundGSMPeople, error) {
	var searchResponse CherwellSearchResponse
	var foundPeople []FoundGSMPeople
	alreadyLooked := map[string]bool{}
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
	r, err := gsm.SearchCherwellFor(searchingFor, true)
	if err == nil {
		defer r.Close()
		_ = json.NewDecoder(r).Decode(&searchResponse)
		for _, x1 := range searchResponse.BusinessObjects {
			person := FoundGSMPeople{
				UserID: x1.BusObRecId,
			}
			person.Teams = gsm.GetTeamsForUser(person.UserID)
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
	r, err = gsm.SearchCherwellFor(searchingFor, true)
	if err == nil {
		defer r.Close()
		_ = json.NewDecoder(r).Decode(&searchResponse)
		for _, x1 := range searchResponse.BusinessObjects {
			if _, ok := alreadyLooked[x1.BusObRecId]; !ok {
				person := FoundGSMPeople{
					UserID: x1.BusObRecId,
				}
				person.Teams = gsm.GetTeamsForUser(person.UserID)
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

func (gsm *CherwellStruct) GetTeamsForUser(userId string) map[string]string {
	var response UserTeamsResponse
	toReturn := map[string]string{}
	r, err := gsm.getStuffFromCherwell(
		"GET",
		"api/V2/getusersteams/userrecordid/"+userId,
		[]byte{},
		true)
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

func (gsm *CherwellStruct) ReassignTaskToPersonInTeam(taskId string, userId string, teamName string) error {
	payload, _ := json.Marshal(ReassignStruct{
		BusObId:    CWFields.Task.BusObId,
		BusObRecId: taskId,
		Fields: []struct {
			Dirty       bool   `json:"dirty"`
			DisplayName string `json:"displayName"`
			FieldId     string `json:"fieldId"`
			FullFieldId string `json:"fullFieldId"`
			HTML        string `json:"html"`
			Name        string `json:"name"`
			Value       string `json:"value"`
		}{
			{FieldId: CWFields.Task.OwnerID, Name: "OwnedByID", DisplayName: "Owned By ID", Value: userId, Dirty: true},
			{FieldId: CWFields.Task.TeamName, Name: "OwnedByTeam", DisplayName: "Owned By Team", Value: teamName, Dirty: true},
		},
	})
	fmt.Printf("payload: %s", string(payload))
	_, err := gsm.getStuffFromCherwell(
		"POST",
		"api/V1/savebusinessobject",
		payload,
		true)
	return err
}

func (gsm *CherwellStruct) SearchCherwellFor(toSend GSMSearchQuery, refreshToken bool) (io.ReadCloser, error) {
	var err error
	sendThis, _ := json.Marshal(toSend)
	result, err := gsm.getStuffFromCherwell(
		"POST",
		"api/V1/getsearchresults",
		sendThis,
		refreshToken)
	return result, err
}

type CherwellAuthResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
	Expires          string `json:".expires"`
	ExpiresDate      time.Time
	Issued           string `json:".issued"`
	AccessToken      string `json:"access_token"`
	RefreshToken     string `json:"refresh_token"`
	TokenType        string `json:"token_type"`
	Username         string `json:"username"`
	ClientID         string `json:"as:client_id"`
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

func (gsm *CherwellStruct) GetJournalNotesForIncident(incident string) ([]CherwellJournal, error) {
	var tasksResponse CherwellRelatedResponse
	var toReturn []CherwellJournal
	gsm.getGSMAccessToken()

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
	result, err := gsm.getStuffFromCherwell(
		"POST",
		"api/V1/getrelatedbusinessobject",
		sendThis,
		true)
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

func (gsm *CherwellStruct) AuthenticateToCherwell(w http.ResponseWriter, r *http.Request) {
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
			targetURL, _ := url.JoinPath(gsm.BaseURL, "token")
			targetURL += "?auth_mode=SAML"
			resp, err := http.PostForm(
				targetURL,
				payload,
			)
			if err == nil {
				json.NewDecoder(resp.Body).Decode(&CherwellToken)
				if len(CherwellToken.Error) > 0 {
					log.Fatalf("Failed 1 %s\n", CherwellToken.ErrorDescription)
				}
				gsm.AccessToken = CherwellToken.AccessToken
				gsm.RefreshToken = CherwellToken.RefreshToken
				gsm.UserSnumber = CherwellToken.Username
				gsm.UserID = CherwellToken.ClientID
				if CherwellToken.Expires == "" {
					gsm.Expiration = time.Now().Add(1 * time.Hour)
				} else {
					gsm.Expiration, _ = time.Parse(time.RFC1123, CherwellToken.Expires)
				}
				var decodedResponse CherwellSearchResponse
				// Get my UserID in Cherwell
				r, _ := gsm.SearchCherwellFor(GSMSearchQuery{
					Filters: []GSMFilter{
						{FieldId: "941798910e24d4b1ae3c6a408cb3f8c5eba26e2b2b", Operator: "eq", Value: gsm.UserSnumber},
					},
					BusObjId: "9338216b3c549b75607cf54667a4e67d1f644d9fed",
					Fields:   []string{"933a15d131ff727ca7ed3f4e1c8f528d719b99b82d", "933a15d17f1f10297df7604b58a76734d6106ac428", "933821793f43a638cf23e34723b907956d324ad303"},
				}, true)
				defer r.Close()
				_ = json.NewDecoder(r).Decode(&decodedResponse)
				gsm.UserID = decodedResponse.BusinessObjects[0].BusObRecId
				gsm.DefaultTeam = decodedResponse.BusinessObjects[0].Fields[1].Value
				gsm.UserEmail = decodedResponse.BusinessObjects[0].Fields[2].Value
				teams := gsm.GetTeamsForUser(gsm.UserID)
				teams2 := []string{}
				for id := range teams {
					teams2 = append(teams2, id)
				}
				gsm.UserTeams = teams2
				gsm.TokenStatus = Active
				ConnectionStatusBox(true, "G")
				gsm.Refresh()
				w.Header().Add("Content-type", "text/html")
				fmt.Fprintf(w, "<html><head></head><body><H1>Authenticated<p>You are authenticated, you may close this window.</body></html>")
				gsm.Download(
					func() { TaskWindowRefresh("CWTasks") },
					func() { TaskWindowRefresh("CWIncidents") },
					func() { TaskWindowRefresh("CWRequests") },
					func() { TaskWindowRefresh("CWTeamIncidents") },
					func() { TaskWindowRefresh("CWTeamTasks") },
				)
			} else {
				w.Header().Add("Content-type", "text/html")
				fmt.Fprintf(w, "<html><head></head><body><H1>AUTHENTICATION FAILED</h1><p>Something is sad.</p></body></html>")

			}
			ActiveTaskStatusUpdate(-1)
		}
	} else {
		// Redirect to Cherwell AUTH
		gsm.Login()
		ActiveTaskStatusUpdate(1)
	}
}

// Doesn't work for SAML?
func (gsm *CherwellStruct) Refresh() {
	payload := url.Values{
		"grant_type": {"password"},
		"client_id":  {"814f9a74-c86a-451e-b6bb-deea65acf72a"},
		"username":   {gsmUsername},
		"password":   {gsmPassword},
		"auth_mode":  {"internal"},
	}
	targetURL, _ := url.JoinPath(gsm.BaseURL, "token")
	resp, err := http.PostForm(
		targetURL,
		payload,
	)
	if err == nil {
		var CherwellToken CherwellAuthResponse
		json.NewDecoder(resp.Body).Decode(&CherwellToken)
		if len(CherwellToken.Error) > 0 {
			log.Fatalf("Failed 1 %s\n", CherwellToken.ErrorDescription)
		}
		gsm.AccessToken = CherwellToken.AccessToken
		gsm.RefreshToken = CherwellToken.RefreshToken
		if CherwellToken.Expires == "" {
			gsm.Expiration = time.Now().Add(2000 * time.Hour)
		} else {
			gsm.Expiration = time.Now().Add(1 * time.Hour)
		}
		ConnectionStatusBox(true, "G")
		gsm.TokenStatus = Active
	} else {
		gsm.RefreshToken = ""
		gsm.AccessToken = ""
		gsm.Expiration = gsm.Expiration.Add(-200 * time.Hour)
		ConnectionStatusBox(false, "G")
		gsm.TokenStatus = Expired
	}
}

func (gsm *CherwellStruct) getStuffFromCherwell(method string, path string, payload []byte, refreshToken bool) (io.ReadCloser, error) {
	client := &http.Client{
		Timeout: time.Second * 10,
	}
	newpath, _ := url.JoinPath(gsm.BaseURL, path)
	req, _ := http.NewRequest(method, newpath, bytes.NewReader(payload))

	if gsm.AccessToken == "" || time.Now().After(gsm.Expiration) {
		gsm.Refresh()
		return gsm.getStuffFromCherwell(method, path, payload, false)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", gsm.AccessToken))
	req.Header.Set("Content-type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == 401 && refreshToken {
		gsm.Refresh()
		return gsm.getStuffFromCherwell(method, path, payload, false)
	}
	return resp.Body, err
}
