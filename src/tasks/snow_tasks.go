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
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/pkg/browser"
)

type SNOWStruct struct {
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

var SnowStatuses = map[string]string{
	"1": "New", "2": "In Progress", "3": "On Hold", "6": "Resolved", "8": "Cancelled",
}

// var snowTokenLock sync.Mutex

func (snow *SNOWStruct) Init(
	baseRedirect string,
	accessToken string,
	refreshToken string,
	expiration time.Time) {
	snow.TokenStatus = Inactive
	if accessToken != "" && time.Now().After(expiration) {
		snow.TokenStatus = Expired
	} else if accessToken != "" {
		snow.AccessToken = accessToken
		snow.RefreshToken = refreshToken
		snow.Expiration = expiration
		snow.TokenStatus = Active
	}
	snow.BaseURL = `https://griffithdev.service-now.com/`
	snow.AuthURL = `https://griffithdev.service-now.com/oauth_auth.do?response_type=code&client_id=d6f39a303ab48610e2fd1ea7144278e0&state=1111`
	snow.RedirectPath = "/snow"
	snow.RedirectURI, _ = url.JoinPath(baseRedirect, snow.RedirectPath)

	// Start the background runner
	snow.G = &singleflight.Group{}
}

func (snow *SNOWStruct) getAccessToken() (bool, error) {
	_, e, _ := snow.G.Do(
		"GetSNOWToken",
		func() (interface{}, error) {
			if snow.TokenStatus == Active && time.Now().After(snow.Expiration) {
				// Expired token
				snow.AccessToken = ""
				ConnectionStatusBox(false, "S")
				snow.TokenStatus = Expired
				snow.Refresh()
			}
			if snow.TokenStatus != Active {
				// No token
				snow.AccessToken = ""
				ConnectionStatusBox(false, "S")
				if snow.RefreshToken == "" {
					snow.Login()
				} else {
					snow.Refresh()
				}
				return "", fmt.Errorf("pending token")
			}
			// Valid token
			return snow.AccessToken, nil
		},
	)
	return e == nil, e
}

func (snow *SNOWStruct) Login() {
	go func() {
		snow.G.Do(
			"SNOWLogin",
			func() (interface{}, error) {
				snow.TokenStatus = Pending
				browser.OpenURL(snow.AuthURL)
				for {
					if snow.TokenStatus == Active {
						return "", nil
					}
				}
			},
		)
	}()
}

var snowTokenLock sync.Mutex

// Reconnect to GSM via username/password
func (snow *SNOWStruct) Refresh() {
	snowTokenLock.Lock()
	if snow.RefreshToken == "" {
		snow.Login()
		return
	}
	payload := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {snow.RefreshToken},
		"redirect_uri":  {"http://localhost:84/snow"},
		"client_id":     {snApplicationClientId},
		"client_secret": {snApplicationSecret},
	}
	targetURL, _ := url.JoinPath(snow.BaseURL, "oauth_token.do")
	resp, err := http.PostForm(
		targetURL,
		payload,
	)
	if err == nil {
		var SNToken MSAuthResponse
		json.NewDecoder(resp.Body).Decode(&SNToken)
		snow.AccessToken = SNToken.AccessToken
		snow.RefreshToken = SNToken.RefreshToken
		seconds, _ := time.ParseDuration(fmt.Sprintf("%ds", SNToken.ExpiresIn-10))
		snow.Expiration = time.Now().Add(seconds)
		ConnectionStatusBox(true, "S")
		snow.TokenStatus = Active
		AppPreferences.SnowAccessToken = snow.AccessToken
		AppPreferences.SnowSRefreshToken = snow.RefreshToken
		AppPreferences.SnowExpiresAt = snow.Expiration
	} else {
		snow.TokenStatus = Expired
		snow.RefreshToken = ""
		snow.AccessToken = ""
		snow.Expiration = snow.Expiration.Add(-200 * time.Hour)
		AppPreferences.SnowAccessToken = snow.AccessToken
		AppPreferences.SnowSRefreshToken = snow.RefreshToken
		AppPreferences.SnowExpiresAt = snow.Expiration
		ConnectionStatusBox(false, "S")
	}
	snowTokenLock.Unlock()
}

func (snow *SNOWStruct) Download(incidentWindow func(), requestsWindow func(), teamIncidentWindow func()) {
	snow.getAccessToken()
	for {
		if snow.TokenStatus == Active {
			break
		}
	}
	snow.DownloadIncidents(incidentWindow)
	snow.DownloadMyRequests(requestsWindow)
	snow.DownloadTeamIncidents(teamIncidentWindow)
}

func (snow *SNOWStruct) Authenticate(w http.ResponseWriter, r *http.Request) {
	var SNToken MSAuthResponse
	query := r.URL.Query()
	if query.Get("code") != "" {
		payload := url.Values{
			"client_id":           {snApplicationClientId},
			"code":                {query.Get("code")},
			"redirect_uri":        {snow.RedirectURI},
			"grant_type":          {"authorization_code"},
			"client_secret":       {snApplicationSecret},
			"requested_token_use": {"on_behalf_of"},
		}
		targetURL, _ := url.JoinPath(snow.BaseURL, "oauth_token.do")
		resp, err := http.PostForm(
			targetURL,
			payload,
		)
		if err != nil {
			log.Fatalf("SN Login failed %s\n", err)
			ConnectionStatusBox(false, "S")
		} else {
			err := json.NewDecoder(resp.Body).Decode(&SNToken)
			if err != nil {
				log.Fatalf("Failed SN %s\n", err)
			}
			if SNToken.AccessToken == "" {
				x, y := json.Marshal(SNToken)
				log.Fatalf("Failed to authenticate to SN %s\n%v\n", x, y)
			}
			snow.RefreshToken = SNToken.RefreshToken
			seconds, _ := time.ParseDuration(fmt.Sprintf("%ds", SNToken.ExpiresIn-10))
			snow.Expiration = time.Now().Add(seconds)
			ConnectionStatusBox(true, "S")
			w.Header().Add("Content-type", "text/html")
			fmt.Fprintf(w, "<html><head></head><body><H1>Authenticated<p>You are authenticated to Service Now, you may close this window.<script>window.close();</script></body></html>")
			snow.AccessToken = SNToken.AccessToken
			snow.TokenStatus = Active
			AppPreferences.SnowAccessToken = snow.AccessToken
			AppPreferences.SnowSRefreshToken = snow.RefreshToken
			AppPreferences.SnowExpiresAt = snow.Expiration
		}
	}
}

func (snow *SNOWStruct) DownloadIncidents(afterFunc func()) {
	ok, _ := snow.getAccessToken()
	if ok {
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
}

func (snow *SNOWStruct) DownloadMyRequests(afterFunc func()) {
	ok, _ := snow.getAccessToken()
	if ok {
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
}

func (snow *SNOWStruct) DownloadTeamIncidents(afterFunc func()) {
	ok, _ := snow.getAccessToken()
	if ok {
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

func (snow *SNOWStruct) SearchSnowFor(table string, fields []string, filter map[string]string, page int) ([]SnowIncident, error) {
	var incidentsResponse SnowResponse
	result, err := snow.getStuffFromURL(
		"GET",
		"/api/now/table/"+table,
		fmt.Sprintf(
			"sysparm_limit=10&sysparm_fields=%s&sysparm_query=%s&sysparm_offset=%d",
			strings.Join(fields, ","),
			createKeyValuePairs(filter),
			page),
		[]byte{},
		true,
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

func createKeyValuePairs(m map[string]string) string {
	b := new(bytes.Buffer)
	for key, value := range m {
		fmt.Fprintf(b, "%s%s^", key, value)
	}
	toReturn := b.String()
	return toReturn[0 : len(toReturn)-1]
}

func (snow *SNOWStruct) getStuffFromURL(method string, path string, query string, payload []byte, refreshToken bool) (io.ReadCloser, error) {
	client := &http.Client{
		Timeout: time.Second * 10,
	}
	newpath, _ := url.JoinPath(snow.BaseURL, path)
	req, e := http.NewRequest(method, newpath, bytes.NewReader(payload))
	req.URL.RawQuery = query

	if e != nil {
		log.Fatal(e)
	}

	if snow.AccessToken == "" || time.Now().After(snow.Expiration) {
		snow.Refresh()
		return snow.getStuffFromURL(method, path, query, payload, false)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", snow.AccessToken))
	req.Header.Set("Content-type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	if resp.StatusCode == 401 && refreshToken {
		snow.Refresh()
		return snow.getStuffFromURL(method, path, query, payload, false)
	}
	return resp.Body, err
}
