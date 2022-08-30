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

func GetJira() {
	if appPreferences.JiraActive && appPreferences.JiraKey > "" {
		activeTaskStatusUpdate(1)
		defer activeTaskStatusUpdate(-1)
		AppStatus.MyTasksFromJira = [][]string{}
		connectionStatusBox(true, "J")
		var jiraResponse JiraResponseType
		baseQuery := "jql=assignee=currentuser()+and+status+not+in+(Done)+order+by+priority,+created+asc&fields=status,summary,created,priority&orderBy=priority,created"
		queryToCall := fmt.Sprintf("%s&startAt=0", baseQuery)
		for page := 1; page < 200; page++ {
			r, err := callJiraURI("GET", "search", []byte{}, queryToCall)
			if err == nil {
				defer r.Close()
				_ = json.NewDecoder(r).Decode(&jiraResponse)

				for _, y := range jiraResponse.Issues {
					AppStatus.MyTasksFromJira = append(
						AppStatus.MyTasksFromJira,
						[]string{
							y.Key,
							TruncateShort(y.Fields.Summary, 60),
							y.Fields.CreatedDateTime,
							jiraPriorityToGSMPriority(y.Fields.Priority.Name),
							y.Fields.Status.Name,
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
		sort.SliceStable(AppStatus.MyTasksFromPlanner, func(i, j int) bool {
			if AppStatus.MyTasksFromPlanner[i][3] == AppStatus.MyTasksFromPlanner[j][3] {
				return AppStatus.MyTasksFromPlanner[i][2] < AppStatus.MyTasksFromPlanner[j][2]
			}
			return AppStatus.MyTasksFromPlanner[i][3] < AppStatus.MyTasksFromPlanner[j][3]
		})
		taskWindowRefresh("Jira")
	}
}
func callJiraURI(method string, path string, payload []byte, query string) (io.ReadCloser, error) {
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
func jiraPriorityToGSMPriority(priority string) string {
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
