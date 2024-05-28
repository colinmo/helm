package tasks

import (
	"bytes"
	"context"
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

	"github.com/pkg/browser"
	"golang.org/x/oauth2"
)

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
	Token                          *oauth2.Token
	GettingToken                   bool
	PlanTitles                     map[string]string
	RedirectPath                   string
	RedirectURI                    string
	StatusCallback                 func(bool, string)
	MyTasks                        []TaskResponseStruct
}

var msTokenLock sync.Mutex

var planConf *oauth2.Config

func (p *PlannerStruct) Init(
	baseRedirect string,
	accessToken string,
	refreshToken string,
	expiration time.Time) {
	msTokenLock.Lock()
	p.PlanTitles = map[string]string{}
	p.RedirectPath = "/ms"
	p.MyTasks = []TaskResponseStruct{}
	planConf = &oauth2.Config{
		ClientID:     msApplicationClientId,
		ClientSecret: msApplicationSecret,
		RedirectURL: func() string {
			thisUrl, _ := url.JoinPath(baseRedirect, p.RedirectPath)
			return thisUrl
		}(),
		Scopes: strings.Split(msScopes, " "),
		Endpoint: oauth2.Endpoint{
			AuthURL: fmt.Sprintf(
				`https://login.microsoftonline.com/%s/oauth2/v2.0/authorize`,
				MsApplicationTenant),
			TokenURL: fmt.Sprintf(
				`https://login.microsoftonline.com/%s/oauth2/v2.0/token`,
				MsApplicationTenant,
			),
		},
	}
	p.Login()
}

func (p *PlannerStruct) Authenticate(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	if query.Get("code") != "" {
		t, err := planConf.Exchange(context.Background(), query.Get("code"))
		if err != nil {
			ConnectionStatusBox(false, "M")
			w.Header().Add("Content-type", "text/html")
			fmt.Fprintf(w, "<html><head></head><body><H1>Failed to Authenticate<p>%s</body></html>", err.Error())
		} else {
			p.Token = t
			ConnectionStatusBox(true, "M")
			w.Header().Add("Content-type", "text/html")
			fmt.Fprintf(w, "<html><head></head><body><H1>Authenticated<p>You are authenticated, you may close this window.<script>window.close();</script></body></html>")
			msTokenLock.Unlock()
		}
	}
}

func (p *PlannerStruct) Login() {
	browser.OpenURL(planConf.AuthCodeURL("some-user-state", oauth2.AccessTypeOffline))
}

func (p *PlannerStruct) Download() {
	ActiveTaskStatusUpdate(1)
	defer ActiveTaskStatusUpdate(-1)

	p.MyTasks = []TaskResponseStruct{}
	uniquePlans := map[string][]int{}
	var teamResponse myTasksGraphResponse
	urlToCall := "/me/planner/tasks"
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
	msTokenLock.Lock()
	client := planConf.Client(context.Background(), p.Token)
	newpath, _ := url.JoinPath("https://graph.microsoft.com/v1.0/", path)
	req, _ := http.NewRequest(method, newpath, bytes.NewReader(payload))
	req.URL.RawQuery = query
	req.Header.Set("Content-type", "application/json")

	resp, err := client.Do(req)
	msTokenLock.Unlock()
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
