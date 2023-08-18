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

type PlannerStruct struct {
	Task
	PlannerAccessTokenChan         chan string
	PlannerAccessTokenRequestsChan chan string
	AccessToken                    string
	RefreshToken                   string
	Expiration                     time.Time
	GettingToken                   bool
	PlanTitles                     map[string]string
	RedirectPath                   string
	RedirectURI                    string
	StatusCallback                 func(bool, string)
	MyTasks                        []TaskResponseStruct
}

func (p *PlannerStruct) Init(
	baseRedirect string,
	accessToken string,
	refreshToken string,
	expiration time.Time) {
	p.PlannerAccessTokenChan = make(chan string)
	p.PlannerAccessTokenRequestsChan = make(chan string)
	p.PlanTitles = map[string]string{}
	p.RedirectPath = "/ms"
	p.RedirectURI, _ = url.JoinPath(baseRedirect, p.RedirectPath)
	p.MyTasks = []TaskResponseStruct{}
	if accessToken != "" {
		p.AccessToken = accessToken
		p.RefreshToken = refreshToken
		p.Expiration = expiration
	}
	// Start the background runner
	p.SingleThreadReturnOrGetPlannerAccessToken()
}

func (p *PlannerStruct) SingleThreadReturnOrGetPlannerAccessToken() {
	if p.AccessToken == "" {
		p.Login()
	}
	go func() {
		for {
			_, ok := <-p.PlannerAccessTokenRequestsChan
			if !ok {
				break
			}
			for {
				if p.AccessToken != "" {
					break
				}
				time.Sleep(1 * time.Second)
			}
			if p.Expiration.Before(time.Now()) {
				p.Refresh()
			}
			p.PlannerAccessTokenChan <- p.AccessToken
		}
	}()
}

func (p *PlannerStruct) Authenticate(w http.ResponseWriter, r *http.Request) {
	var MSToken MSAuthResponse
	query := r.URL.Query()
	if query.Get("code") != "" {
		payload := url.Values{
			"client_id":           {msApplicationClientId},
			"scope":               {msScopes},
			"code":                {query.Get("code")},
			"redirect_uri":        {p.RedirectURI},
			"grant_type":          {"authorization_code"},
			"client_secret":       {msApplicationSecret},
			"requested_token_use": {"on_behalf_of"},
		}
		resp, err := http.PostForm(
			fmt.Sprintf(
				`https://login.microsoftonline.com/%s/oauth2/v2.0/token`,
				MsApplicationTenant,
			),
			payload,
		)
		if err != nil {
			log.Fatalf("Login failed %s\n", err)
			ConnectionStatusBox(false, "M")
		} else {
			err := json.NewDecoder(resp.Body).Decode(&MSToken)
			if err != nil {
				log.Fatalf("Failed MS %s\n", err)
			}
			p.RefreshToken = MSToken.RefreshToken
			seconds, _ := time.ParseDuration(fmt.Sprintf("%ds", MSToken.ExpiresIn-10))
			p.Expiration = time.Now().Add(seconds)
			ConnectionStatusBox(true, "M")
			w.Header().Add("Content-type", "text/html")
			fmt.Fprintf(w, "<html><head></head><body><H1>Authenticated<p>You are authenticated, you may close this window.</body></html>")
			p.AccessToken = MSToken.AccessToken
		}
	}
}

func (p *PlannerStruct) Login() {
	browser.OpenURL(
		fmt.Sprintf(`https://login.microsoftonline.com/%s/oauth2/v2.0/authorize?finalUri=?code=xy&client_id=%s&response_type=code&redirect_uri=http://localhost:84/ms&response_mode=query&scope=%s`,
			MsApplicationTenant,
			msApplicationClientId,
			msScopes),
	)
}

func (p *PlannerStruct) Download(specific string) {
	p.GetAccessToken()
	ActiveTaskStatusUpdate(1)
	defer ActiveTaskStatusUpdate(-1)

	p.MyTasks = []TaskResponseStruct{}
	uniquePlans := map[string][]int{}
	var teamResponse myTasksGraphResponse
	urlToCall := "/me/planner/tasks"
	fmt.Printf("Downloading\n")
	for page := 1; page < 200; page++ {
		r, err := p.CallGraphURI("GET", urlToCall, []byte{}, "$select=id,details,planid,title,priority,percentcomplete,createdDateTime")
		if err == nil {
			err = json.NewDecoder(r).Decode(&teamResponse)
			if err != nil {
				log.Fatal("Bad decode")
			}
			r.Close()

			for _, y := range teamResponse.Value {
				if y.PercentComplete < 100 {
					row := TaskResponseStruct{
						ID:               y.TaskID,
						ParentID:         y.PlanID,
						Title:            y.Title,
						Priority:         p.TeamPriorityToGSMPriority(y.Priority),
						PriorityOverride: p.TeamPriorityToGSMPriority(y.Priority),
					}
					row.CreatedDateTime, _ = time.Parse("2006-01-02T15:04:05.999999999Z", y.CreatedDateTime)
					switch y.PercentComplete {
					case 0:
						row.Status = "Not started (0)"
					case 50:
						row.Status = "In progress (50)"
					default:
						row.Status = fmt.Sprintf("Unknown (%d)", y.PercentComplete)
					}
					if val, ok := PriorityOverrides.MSPlanner[row.ID]; ok {
						row.PriorityOverride = val
					}
					p.MyTasks = append(
						p.MyTasks,
						row,
					)
					uniquePlans[y.PlanID] = append(uniquePlans[y.PlanID], len(p.MyTasks)-1)
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
		if _, ok := p.PlanTitles[id]; !ok {
			r, err := p.CallGraphURI("GET", "planner/plans/"+id, []byte{}, "")
			if err == nil {
				defer r.Close()
				var planResponse PlanGraphResponse
				_ = json.NewDecoder(r).Decode(&planResponse)
				p.PlanTitles[id] = planResponse.Title
			}
		}
		for _, index := range members {
			p.MyTasks[index].Title = fmt.Sprintf("[%s]: %s", p.PlanTitles[id], p.MyTasks[index].Title)
		}
	}

	sort.SliceStable(p.MyTasks, func(i, j int) bool {
		if p.MyTasks[i].PriorityOverride == p.MyTasks[j].PriorityOverride {
			return p.MyTasks[i].CreatedDateTime.Before(p.MyTasks[j].CreatedDateTime)
		}
		return p.MyTasks[i].PriorityOverride < p.MyTasks[j].PriorityOverride
	})

}

func (p *PlannerStruct) CallGraphURI(method string, path string, payload []byte, query string) (io.ReadCloser, error) {
	client := &http.Client{
		Timeout: time.Second * 10,
	}
	newpath, _ := url.JoinPath("https://graph.microsoft.com/v1.0/", path)
	if len(query) > 0 {
		newpath = newpath + "?" + query
	}
	req, _ := http.NewRequest(method, newpath, bytes.NewReader(payload))
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.AccessToken))
	req.Header.Set("Content-type", "application/json")

	resp, err := client.Do(req)
	if err == nil && resp.StatusCode == 200 {
		return resp.Body, err
	}
	return nil, err
}

func (p *PlannerStruct) TeamPriorityToGSMPriority(priority int) string {
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

func (p *PlannerStruct) Refresh() {
	var MSToken MSAuthResponse
	if len(p.RefreshToken) == 0 {
		return
	}
	payload := url.Values{
		"client_id":     {msApplicationClientId},
		"scope":         {msScopes},
		"refresh_token": {p.RefreshToken},
		"redirect_uri":  {p.RedirectURI},
		"grant_type":    {"refresh_token"},
		"client_secret": {msApplicationSecret},
	}
	resp, err := http.PostForm(
		fmt.Sprintf(`https://login.microsoftonline.com/%s/oauth2/v2.0/token`,
			MsApplicationTenant,
		),
		payload,
	)
	if err != nil {
		ConnectionStatusBox(false, "M")
		log.Fatalf("Login failed %s\n", err)
	} else {
		err := json.NewDecoder(resp.Body).Decode(&MSToken)
		if err != nil {
			log.Fatalf("Failed MS %s\n", err)
		}
		p.RefreshToken = MSToken.RefreshToken
		seconds, _ := time.ParseDuration(fmt.Sprintf("%ds", MSToken.ExpiresIn-10))
		p.Expiration = time.Now().Add(seconds)
		p.AccessToken = MSToken.AccessToken
		ConnectionStatusBox(true, "M")
	}
}

func (p *PlannerStruct) GetAccessToken() string {
	p.PlannerAccessTokenRequestsChan <- time.Now().String()
	return <-p.PlannerAccessTokenChan
}
