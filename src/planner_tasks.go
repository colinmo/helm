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

type MSAuthResponse struct {
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
	ExpiresIn    int    `json:"expires_in"`
	ExpiresDate  time.Time
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

var PlannerAccessTokenRequestsChan = make(chan string)
var PlannerAccessTokenChan = make(chan string)

func singleThreadReturnOrGetPlannerAccessToken() {
	for {
		_, ok := <-PlannerAccessTokenRequestsChan
		if !ok {
			break
		}
		for {
			if AuthenticationTokens.MS.access_token != "" &&
				!AppStatus.MSGettingToken {
				break
			}
			time.Sleep(1 * time.Second)
		}
		if AuthenticationTokens.MS.expiration.Before(time.Now()) {
			refreshMS()
		}
		PlannerAccessTokenChan <- AuthenticationTokens.MS.access_token
	}
}

func returnOrGetPlannerAccessToken() string {
	PlannerAccessTokenRequestsChan <- time.Now().String()
	return <-PlannerAccessTokenChan
}

func authenticateToMS(w http.ResponseWriter, r *http.Request) {
	var MSToken MSAuthResponse
	query := r.URL.Query()
	if query.Get("code") != "" {
		payload := url.Values{
			"client_id":           {msApplicationClientId},
			"scope":               {msScopes},
			"code":                {query.Get("code")},
			"redirect_uri":        {"http://localhost:84/ms"},
			"grant_type":          {"authorization_code"},
			"client_secret":       {msApplicationSecret},
			"requested_token_use": {"on_behalf_of"},
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
			connectionStatusBox(true, "M")
			w.Header().Add("Content-type", "text/html")
			fmt.Fprintf(w, "<html><head></head><body><H1>Authenticated<p>You are authenticated, you may close this window.</body></html>")
			AppStatus.MSGettingToken = false
		}
	} else {
		LoginToMS()
	}
}

func LoginToMS() {
	browser.OpenURL(
		fmt.Sprintf(`https://login.microsoftonline.com/%s/oauth2/v2.0/authorize?finalUri=?code=xy&client_id=%s&response_type=code&redirect_uri=http://localhost:84/ms&response_mode=query&scope=%s`,
			msApplicationTenant,
			msApplicationClientId,
			msScopes),
	)
}
func refreshMS() {
	var MSToken MSAuthResponse
	AppStatus.MSGettingToken = true
	if len(AuthenticationTokens.MS.refresh_token) == 0 || time.Now().After(AuthenticationTokens.MS.expiration) {
		LoginToMS()
		return
	}
	payload := url.Values{
		"client_id":     {msApplicationClientId},
		"scope":         {msScopes},
		"refresh_token": {AuthenticationTokens.MS.refresh_token},
		"redirect_uri":  {"http://localhost:84/ms"},
		"grant_type":    {"refresh_token"},
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
		AppStatus.MSGettingToken = false
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

type PlanGraphResponse struct {
	Title string `json:"title"`
}

var MSPlannerPlanTitles = map[string]string{}

func DownloadPlanners() {
	go func() {
		fmt.Printf("Downloading Planner\n")
		returnOrGetPlannerAccessToken()
		activeTaskStatusUpdate(1)
		defer activeTaskStatusUpdate(-1)

		AppStatus.MyTasksFromPlanner = []TaskResponseStruct{}
		uniquePlans := map[string][]int{}
		var teamResponse myTasksGraphResponse
		urlToCall := "/me/planner/tasks"
		for page := 1; page < 200; page++ {
			r, err := callGraphURI("GET", urlToCall, []byte{}, "$expand=details")
			if err == nil {
				defer r.Close()
				_ = json.NewDecoder(r).Decode(&teamResponse)

				for _, y := range teamResponse.Value {
					if y.PercentComplete < 100 {
						row := TaskResponseStruct{
							ID:               y.TaskID,
							ParentID:         y.PlanID,
							Title:            y.Title,
							Priority:         teamPriorityToGSMPriority(y.Priority),
							PriorityOverride: teamPriorityToGSMPriority(y.Priority),
							// TruncateShort(y.Details.Description, 60),
						}
						row.CreatedDateTime, _ = time.Parse("2006-01-02T15:04:05.999999999Z", y.CreatedDateTime)
						switch y.PercentComplete {
						case 0:
							row.Status = "Not started (0)"
						case 50:
							row.Status = "In progress (50)"
						}
						if val, ok := priorityOverrides.MSPlanner[row.ID]; ok {
							row.PriorityOverride = val
						}
						AppStatus.MyTasksFromPlanner = append(
							AppStatus.MyTasksFromPlanner,
							row,
						)
						uniquePlans[y.PlanID] = append(uniquePlans[y.PlanID], len(AppStatus.MyTasksFromPlanner)-1)
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
		// Get the Plan names
		for id, members := range uniquePlans {
			if _, ok := MSPlannerPlanTitles[id]; !ok {
				r, err := callGraphURI("GET", "planner/plans/"+id, []byte{}, "")
				if err == nil {
					defer r.Close()
					var planResponse PlanGraphResponse
					_ = json.NewDecoder(r).Decode(&planResponse)
					MSPlannerPlanTitles[id] = planResponse.Title
				}
			}
			for _, index := range members {
				AppStatus.MyTasksFromPlanner[index].Title = fmt.Sprintf("[%s]: %s", MSPlannerPlanTitles[id], AppStatus.MyTasksFromPlanner[index].Title)
			}
		}

		sort.SliceStable(AppStatus.MyTasksFromPlanner, func(i, j int) bool {
			if AppStatus.MyTasksFromPlanner[i].PriorityOverride == AppStatus.MyTasksFromPlanner[j].PriorityOverride {
				return AppStatus.MyTasksFromPlanner[i].CreatedDateTime.Before(AppStatus.MyTasksFromPlanner[j].CreatedDateTime)
			}
			return AppStatus.MyTasksFromPlanner[i].PriorityOverride < AppStatus.MyTasksFromPlanner[j].PriorityOverride
		})

		taskWindowRefresh("MSPlanner")
	}()
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
