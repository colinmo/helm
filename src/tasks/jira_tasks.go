package tasks

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"time"
)

type JiraResponseType struct {
	StartAt    int `json:"startAt"`
	MaxResults int `json:"maxResults"`
	Total      int `json:"total"`
	Issues     []struct {
		ID     string `json:"id"`
		Key    string `json:"key"`
		Fields struct {
			Summary  string `json:"summary"`
			Priority struct {
				Name    string `json:"name"`
				IconURL string `json:"iconUr"`
			} `json:"priority"`
			CreatedDateTime string `json:"created"`
			Status          struct {
				Name    string `json:"name"`
				IconURL string `json:"iconUrl"`
			} `json:"status"`
			IssueType struct {
				Name    string `json:"name"`
				IconURL string `json:"iconUrl"`
			} `json:"issuetype"`
		} `json:"fields"`
	} `json:"issues"`
}

type JiraStruct struct {
	Task
	MyTasks []TaskResponseStruct
}

func (j *JiraStruct) Init() {
	//
}

func (j *JiraStruct) Download() {
	if AppPreferences.JiraActive && AppPreferences.JiraKey > "" {
		ActiveTaskStatusUpdate(1)
		defer ActiveTaskStatusUpdate(-1)
		j.MyTasks = []TaskResponseStruct{}
		indexedTasks := map[string]TaskResponseStruct{}
		ConnectionStatusBox(true, "J")
		var jiraResponse JiraResponseType
		baseQuery := `jql=assignee%3Dcurrentuser()%20AND%20status%20!%3D%20%22Done%22&fields=summary,created,priority,status,issuetype`
		queryToCall := fmt.Sprintf("%s&startAt=0", baseQuery)
		for page := 1; page < 200; page++ {
			r, err := j.callJiraURI("GET", "search", []byte{}, queryToCall)
			if err == nil {
				defer r.Close()
				_ = json.NewDecoder(r).Decode(&jiraResponse)

				for _, y := range jiraResponse.Issues {
					dt, _ := time.Parse("2006-01-02T15:04:05.999-0700", y.Fields.CreatedDateTime)
					myOverride := j.jiraPriorityToGSMPriority(y.Fields.Priority.Name)
					if val, ok := PriorityOverrides.Jira[y.Key]; ok {
						myOverride = val
					}
					indexedTasks[y.Key] =
						TaskResponseStruct{
							ID:               y.Key,
							Title:            TruncateShort(y.Fields.Summary, 60),
							CreatedDateTime:  dt,
							Priority:         j.jiraPriorityToGSMPriority(y.Fields.Priority.Name),
							Status:           y.Fields.Status.Name,
							PriorityOverride: myOverride,
							Type:             y.Fields.IssueType.Name,
							Blocked:          false,
						}
				}
				if len(jiraResponse.Issues) == 0 {
					break
				} else {
					queryToCall = fmt.Sprintf("%s&startAt=%d", baseQuery, jiraResponse.MaxResults*page)
				}
			} else {
				fmt.Printf("Failed to get Jira Tasks %s\n", err)
			}
		}
		baseQuery = `jql=assignee%3Dcurrentuser()%20AND%20status%20!%3D%20%22Done%22%20%20AND%20issuelinktype%20%3D%20%22is%20blocked%20by%22&fields=key`
		queryToCall = fmt.Sprintf("%s&startAt=0", baseQuery)
		for page := 1; page < 200; page++ {
			r, err := j.callJiraURI("GET", "search", []byte{}, queryToCall)
			if err == nil {
				defer r.Close()
				_ = json.NewDecoder(r).Decode(&jiraResponse)

				for _, y := range jiraResponse.Issues {
					mike := indexedTasks[y.Key]
					mike.Blocked = true
					indexedTasks[y.Key] = mike
				}
				if len(jiraResponse.Issues) == 0 {
					break
				} else {
					queryToCall = fmt.Sprintf("%s&startAt=%d", baseQuery, jiraResponse.MaxResults*page)
				}
			} else {
				fmt.Printf("Failed to get Jira Tasks %s\n", err)
			}
		}
		for _, v := range indexedTasks {
			j.MyTasks = append(j.MyTasks, v)
		}
		// sort
		sort.SliceStable(j.MyTasks, func(i, k int) bool {
			if j.MyTasks[i].PriorityOverride == j.MyTasks[k].PriorityOverride {
				return j.MyTasks[i].CreatedDateTime.Before(j.MyTasks[k].CreatedDateTime)
			}
			return j.MyTasks[i].PriorityOverride < j.MyTasks[k].PriorityOverride
		})
	}
}
func (j *JiraStruct) callJiraURI(method string, path string, payload []byte, query string) (io.ReadCloser, error) {
	client := &http.Client{
		Timeout: time.Second * 10,
	}
	newpath, _ := url.JoinPath("https://griffith.atlassian.net/rest/api/2/", path)
	if len(query) > 0 {
		newpath = newpath + "?" + query
	}
	req, _ := http.NewRequest(method, newpath, bytes.NewReader(payload))
	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", AppPreferences.JiraUsername, AppPreferences.JiraKey)))))
	req.Header.Set("Content-type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	return resp.Body, err
}
func (j *JiraStruct) jiraPriorityToGSMPriority(priority string) string {
	switch priority {
	case "Highest":
		return "1"
	case "High":
		return "2"
	case "Medium":
		return "3"
	case "Low":
		return "4"
	case "Lowest":
		return "5"
	default:
		return "5"
	}
}
