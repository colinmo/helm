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

var MSAuthWebServer *http.Server

type MSAuthResponse struct {
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
	ExpiresIn    int    `json:"expires_in"`
	ExpiresDate  time.Time
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func GetPlanner() bool {
	if AppStatus.MSGettingToken {
		return false
	}
	if AuthenticationTokens.GSM.refresh_token != "" && AuthenticationTokens.GSM.expiration.Before(time.Now()) {
		fmt.Printf("Refresh token with %s\n", AuthenticationTokens.GSM.refresh_token)
		refreshMS()
		activeTaskStatusUpdate(1)
		return true
	}
	if AuthenticationTokens.MS.access_token == "" || AuthenticationTokens.MS.expiration.Before(time.Now()) {
		// Login if expired
		browser.OpenURL(
			fmt.Sprintf(`https://login.microsoftonline.com/%s/oauth2/v2.0/authorize?finalUri=?code=xy&client_id=%s&response_type=code&redirect_uri=http://localhost:84/ms&response_mode=query&scope=%s`,
				msApplicationTenant,
				msApplicationClientId,
				msScopes),
		)
		return false
	}
	return true
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
			AppStatus.MSGettingToken = false
			DownloadPlanners()
			taskWindowRefresh("MSPlanner")
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

func refreshMS() {
	var MSToken MSAuthResponse
	AppStatus.MSGettingToken = true
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
		GetPlanner()
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
	if GetPlanner() {
		activeTaskStatusUpdate(1)
		defer activeTaskStatusUpdate(-1)

		// * @todo Add Sort
		AppStatus.MyTasksFromPlanner = [][]string{}
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
						row := []string{
							y.TaskID,
							y.PlanID,
							y.BucketID,
							y.Title,
							y.OrderHint,
							y.CreatedDateTime,
							teamPriorityToGSMPriority(y.Priority),
							TruncateShort(y.Details.Description, 60),
							fmt.Sprintf("%d", y.PercentComplete),
							teamPriorityToGSMPriority(y.Priority),
						}
						if val, ok := priorityOverrides.MSPlanner[row[0]]; ok {
							row[6] = val
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
				AppStatus.MyTasksFromPlanner[index][3] = fmt.Sprintf("[%s]: %s", MSPlannerPlanTitles[id], AppStatus.MyTasksFromPlanner[index][3])
			}
		}

		// sort
		sort.SliceStable(AppStatus.MyTasksFromPlanner, func(i, j int) bool {
			if AppStatus.MyTasksFromPlanner[i][6] == AppStatus.MyTasksFromPlanner[j][6] {
				return AppStatus.MyTasksFromPlanner[i][5] > AppStatus.MyTasksFromPlanner[j][5]
			}
			return AppStatus.MyTasksFromPlanner[i][6] < AppStatus.MyTasksFromPlanner[j][6]
		})
	}
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
