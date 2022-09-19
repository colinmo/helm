package main

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
		} `json:"fields"`
	} `json:"issues"`
}

type Jira struct {
	Task
	MyTasks []TaskResponseStruct
}

func (j *Jira) Init() {
	//
}

func (j *Jira) Download() {
	if appPreferences.JiraActive && appPreferences.JiraKey > "" {
		activeTaskStatusUpdate(1)
		defer activeTaskStatusUpdate(-1)
		j.MyTasks = []TaskResponseStruct{}
		connectionStatusBox(true, "J")
		var jiraResponse JiraResponseType
		baseQuery := `jql=assignee%3Dcurrentuser()%20AND%20status%20!%3D%20%22Done%22%20order%20by%20priority,created%20asc&fields=summary,created,priority,status`
		queryToCall := fmt.Sprintf("%s&startAt=0", baseQuery)
		for page := 1; page < 200; page++ {
			r, err := j.callJiraURI("GET", "search", []byte{}, queryToCall)
			if err == nil {
				defer r.Close()
				_ = json.NewDecoder(r).Decode(&jiraResponse)

				for _, y := range jiraResponse.Issues {
					dt, _ := time.Parse("2006-01-02T15:04:05.999-0700", y.Fields.CreatedDateTime)
					j.MyTasks = append(
						j.MyTasks,
						TaskResponseStruct{
							ID:               y.Key,
							Title:            TruncateShort(y.Fields.Summary, 60),
							CreatedDateTime:  dt,
							Priority:         j.jiraPriorityToGSMPriority(y.Fields.Priority.Name),
							Status:           y.Fields.Status.Name,
							PriorityOverride: j.jiraPriorityToGSMPriority(y.Fields.Priority.Name),
						},
					)
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
		// sort
		sort.SliceStable(j.MyTasks, func(i, k int) bool {
			if j.MyTasks[i].PriorityOverride == j.MyTasks[k].PriorityOverride {
				return j.MyTasks[i].CreatedDateTime.Before(j.MyTasks[k].CreatedDateTime)
			}
			return j.MyTasks[i].PriorityOverride < j.MyTasks[k].PriorityOverride
		})
		taskWindowRefresh("Jira")
	}
}
func (j *Jira) callJiraURI(method string, path string, payload []byte, query string) (io.ReadCloser, error) {
	client := &http.Client{
		Timeout: time.Second * 10,
	}
	newpath, _ := url.JoinPath("https://griffith.atlassian.net/rest/api/2/", path)
	if len(query) > 0 {
		newpath = newpath + "?" + query
	}
	req, _ := http.NewRequest(method, newpath, bytes.NewReader(payload))
	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", appPreferences.JiraUsername, appPreferences.JiraKey)))))
	req.Header.Set("Content-type", "application/json")

	resp, err := client.Do(req)
	return resp.Body, err
}
func (j *Jira) jiraPriorityToGSMPriority(priority string) string {
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
