package tasks

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strings"
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
	newpath, _ := url.JoinPath("https://griffith.atlassian.net/rest/api/3/", path)
	if len(query) > 0 {
		newpath = newpath + "?" + query
	}
	req, _ := http.NewRequest(method, newpath, bytes.NewReader(payload))
	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", AppPreferences.JiraUsername, AppPreferences.JiraKey)))))
	req.Header.Set("Content-Type", "application/json")

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

type TeamsSearchResponse struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	State       string `json:"state"`
}

func (j *JiraStruct) TeamsLookup() (map[string]string, []string) {
	ActiveTaskStatusUpdate(1)
	defer ActiveTaskStatusUpdate(-1)
	ConnectionStatusBox(true, "J")
	baseQuery := `organizationId=j084dc2b-38b8-1dj7-j0bj-kb330a12d358`
	queryToCall := fmt.Sprintf("%s&startAt=0", baseQuery)
	foundTeams := []TeamsSearchResponse{}
	bigFoundTeams := map[string]string{}
	values := []string{}
	r, err := j.callJiraURI("GET", "../../../gateway/api/v3/teams/search", []byte{}, queryToCall)
	if err == nil {
		defer r.Close()
		_ = json.NewDecoder(r).Decode(&foundTeams)
	}
	for _, x := range foundTeams {
		if x.State == "ACTIVE" {
			bigFoundTeams[x.DisplayName] = strings.Split(x.ID, "/")[1]
			values = append(values, x.DisplayName)
		}
	}
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	return bigFoundTeams, values
}

type JiraPersonSearchResponse struct {
	ID          string `json:"accountId"`
	DisplayName string `json:"displayName"`
}

func (j *JiraStruct) PersonLookup(searchstring string) (map[string]string, []string) {
	ActiveTaskStatusUpdate(1)
	defer ActiveTaskStatusUpdate(-1)
	ConnectionStatusBox(true, "J")
	baseQuery := `query=` + url.QueryEscape(searchstring)
	found := []JiraPersonSearchResponse{}
	bigFound := map[string]string{}
	values := []string{}
	r, err := j.callJiraURI("GET", "/user/search", []byte{}, baseQuery)
	if err == nil {
		defer r.Close()
		_ = json.NewDecoder(r).Decode(&found)
	}
	for _, x := range found {
		bigFound[x.DisplayName] = x.ID
		values = append(values, x.DisplayName)
	}
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	return bigFound, values
}

type JiraSimpleSearchResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (j *JiraStruct) ProjectLookup(searchstring string) (map[string]string, []string) {
	ActiveTaskStatusUpdate(1)
	defer ActiveTaskStatusUpdate(-1)
	ConnectionStatusBox(true, "J")
	baseQuery := `query=` + searchstring
	found := []JiraSimpleSearchResponse{}
	bigFound := map[string]string{}
	values := []string{}
	r, err := j.callJiraURI("GET", "/project", []byte{}, baseQuery)
	if err == nil {
		defer r.Close()
		_ = json.NewDecoder(r).Decode(&found)
	}
	for _, x := range found {
		bigFound[x.Name] = x.ID
		values = append(values, x.Name)
	}
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	return bigFound, values
}

func (j *JiraStruct) GetMyId() string {
	ActiveTaskStatusUpdate(1)
	defer ActiveTaskStatusUpdate(-1)
	ConnectionStatusBox(true, "J")
	found := JiraPersonSearchResponse{}
	r, err := j.callJiraURI("GET", "myself", []byte{}, "")
	if err == nil {
		defer r.Close()
		json.NewDecoder(r).Decode(&found)
	} else {
		log.Fatal("Didn't work")
	}
	return found.ID
}

type JiraIssuePickerResponseType struct {
	Sections []struct {
		Subject string `json:"sub"`
		Issues  []struct {
			ID      string `json:"id"`
			Key     string `json:"key"`
			Summary string `json:"summaryText"`
		} `json:"issues"`
	} `json:"sections"`
}

func (j *JiraStruct) RelatedIssuesLookupByType(issuetype, query string) []TaskResponseStruct {
	ActiveTaskStatusUpdate(1)
	defer ActiveTaskStatusUpdate(-1)
	foundTasks := []TaskResponseStruct{}
	indexedTasks := map[string]TaskResponseStruct{}
	ConnectionStatusBox(true, "J")
	var jiraResponse JiraIssuePickerResponseType
	baseQuery := `currentJQL=` + url.QueryEscape(fmt.Sprintf(`project=RSD and issueType=%s and status!=Done`, issuetype)) + `&query=` + url.QueryEscape(query)
	r, err := j.callJiraURI("GET", "issue/picker", []byte{}, baseQuery)
	if err == nil {
		defer r.Close()
		_ = json.NewDecoder(r).Decode(&jiraResponse)

		lastSections := len(jiraResponse.Sections) - 1

		for _, y := range jiraResponse.Sections[lastSections].Issues {
			indexedTasks[y.Key] =
				TaskResponseStruct{
					ID:    y.Key,
					Title: TruncateShort(y.Summary, 60),
				}
		}
	} else {
		fmt.Printf("Failed to get Jira Tasks %s\n", err)
	}
	for _, v := range indexedTasks {
		foundTasks = append(foundTasks, v)
	}
	// sort
	sort.SliceStable(foundTasks, func(i, k int) bool {
		return foundTasks[i].Title < foundTasks[k].Title
	})
	return foundTasks
}

type JiraIssueCreatedResponseType struct {
	ID         string `json:"id"`
	Self       string `json:"self"`
	Transition struct {
		Status          int `json:"status"`
		ErrorCollection struct {
			ErrorMessages []string `json:"errorMessages"`
		} `json:"errorCollection"`
	} `json:"transition"`
}

func (j *JiraStruct) CreateTask(jsonstring []byte) (string, string, error) {
	r, err := j.callJiraURI("POST", "/issue", jsonstring, "")
	saveResult := JiraIssueCreatedResponseType{}
	if err == nil {
		defer r.Close()

		//_ = json.NewDecoder(r).Decode(&saveResult)
		mep, _ := io.ReadAll(r)
		json.Unmarshal(mep, &saveResult)
		if saveResult.Transition.Status < 200 || saveResult.Transition.Status > 299 {
			x, _ := json.Marshal(saveResult)
			return "", "", fmt.Errorf("%d: %s\n%s", saveResult.Transition.Status, x, mep)
		}
	}
	return saveResult.ID, saveResult.Self, nil
}
func (j *JiraStruct) MyIcon() []byte {
	return []byte(`<?xml version="1.0" encoding="UTF-8"?>
	<svg width="800px" height="800px" viewBox="0 0 256 256" version="1.1" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" preserveAspectRatio="xMidYMid">
		<defs>
			<linearGradient x1="98.0308675%" y1="0.160599572%" x2="58.8877062%" y2="40.7655246%" id="linearGradient-1">
				<stop stop-color="#0052CC" offset="18%"></stop>
				<stop stop-color="#2684FF" offset="100%"></stop>
			</linearGradient>
			<linearGradient x1="100.665247%" y1="0.45503212%" x2="55.4018095%" y2="44.7269807%" id="linearGradient-2">
				<stop stop-color="#0052CC" offset="18%"></stop>
				<stop stop-color="#2684FF" offset="100%"></stop>
			</linearGradient>
		</defs>
		<g>
			<path d="M244.657778,0 L121.706667,0 C121.706667,14.7201046 127.554205,28.837312 137.962891,39.2459977 C148.371577,49.6546835 162.488784,55.5022222 177.208889,55.5022222 L199.857778,55.5022222 L199.857778,77.3688889 C199.877391,107.994155 224.699178,132.815943 255.324444,132.835556 L255.324444,10.6666667 C255.324444,4.77562934 250.548815,3.60722001e-16 244.657778,0 Z" fill="#2684FF"></path>
			<path d="M183.822222,61.2622222 L60.8711111,61.2622222 C60.8907238,91.8874888 85.7125112,116.709276 116.337778,116.728889 L138.986667,116.728889 L138.986667,138.666667 C139.025905,169.291923 163.863607,194.097803 194.488889,194.097778 L194.488889,71.9288889 C194.488889,66.0378516 189.71326,61.2622222 183.822222,61.2622222 Z" fill="url(#linearGradient-1)"></path>
			<path d="M122.951111,122.488889 L0,122.488889 C3.75391362e-15,153.14192 24.8491913,177.991111 55.5022222,177.991111 L78.2222222,177.991111 L78.2222222,199.857778 C78.241767,230.45532 103.020285,255.265647 133.617778,255.324444 L133.617778,133.155556 C133.617778,127.264518 128.842148,122.488889 122.951111,122.488889 Z" fill="url(#linearGradient-2)"></path>
		</g>
	</svg>`)
}
