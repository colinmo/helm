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

/*
type JiraDifferences struct {
	LocalFile     string
	LocalUpdated  time.Time
	RemoteFile    string
	RemoteUpdated time.Time
	Differences   string
}

var JiraDateTimeStampFormat string = "2006-01-02T13:04:05.999+0700"

// This pulls down the JIRA tasks assigned to the user and compares them
// to a local directory copy in Markdown. Differences are displayed for the
// user to action
func (j *Jira) CompareLocalToRemote() []JiraDifferences {
	localMarkdowns, newMarkdowns := j.ProcessLocalFiles()
	remoteMarkdowns := j.DownloadFull()
	// Compare
	return j.DifferenceObjects(&localMarkdowns, &newMarkdowns, &remoteMarkdowns)
}

func (j *Jira) DifferenceObjects(localMarkdowns *map[string]MarkdownTask, newMarkdowns *[]MarkdownTask, remoteMarkdowns *map[string]MarkdownTask) []JiraDifferences {
	differences := []JiraDifferences{}
	for _, x := range *newMarkdowns {
		dt, _ := time.Parse(JiraDateTimeStampFormat, x.Metadata.Updated)
		differences = append(
			differences,
			JiraDifferences{
				x.Filename,
				dt,
				"",
				time.Time{},
				"New Local File",
			},
		)
	}
	for i, x := range *localMarkdowns {
		diff :=
			j.CompareLocalToRemoteSingle(
				x.Filename,
				x.Body,
				x.Metadata.Updated,
				(*remoteMarkdowns)[i].Body,
				(*remoteMarkdowns)[i].Metadata.Updated,
			)
		if diff.Differences != "" {
			differences = append(
				differences,
				diff,
			)
		}
		delete((*remoteMarkdowns), i)
	}
	for _, x := range *remoteMarkdowns {
		dt, _ := time.Parse(JiraDateTimeStampFormat, x.Metadata.Updated)
		differences = append(
			differences,
			JiraDifferences{
				"",
				time.Time{},
				"Remote " + x.JiraID,
				dt,
				"New Remote File",
			},
		)
	}
	return differences
}

func (j *Jira) CompareLocalToRemoteSingle(localFile string, localContent string, localUpdated string, remoteContent string, remoteUpdated string) JiraDifferences {
	edits := myers.ComputeEdits(span.URIFromPath(localFile), localContent, remoteContent)
	diff := fmt.Sprint(gotextdiff.ToUnified(localFile, "Remote", localContent, edits))
	GetFileModTime(localFile).Format(JiraDateTimeStampFormat)

	dt1, _ := time.Parse(JiraDateTimeStampFormat, localUpdated)
	dt2, _ := time.Parse(JiraDateTimeStampFormat, remoteUpdated)
	// @todo: comment compare
	return JiraDifferences{
		localFile,
		dt1,
		"Remote",
		dt2,
		diff,
	}
}

type JiraTaskCompareActions struct {
	LocalFile  string
	RemoteJira string
	Action     string
}

// This performs the actions the user requests
func (j *Jira) ApplyTaskChanges(actions []string) {
	// Update local files to the contents of remote
	// Send changes to Jira
}

type MarkdownTask struct {
	JiraID   string
	Filename string
	Body     string
	Comments []string
	Metadata JiraMetadata
}

type JiraMetadata struct {
	Name            string   `yaml:"Name"`
	Sponsor         string   `yaml:"Sponsor"`
	Status          string   `yaml:"Status"`
	Priority        string   `yaml:"Priority"`
	Client          string   `yaml:"Client"`
	SecurityContact string   `yaml:"SecurityContact"`
	JiraKey         string   `yaml:"JiraKey"`
	Updated         string   `yaml:"JiraUpdated"`
	Inactive        string   `yaml:"Inactive"`
	Epic            string   `yaml:"Epic"`
	Tags            []string `yaml:"Tags"`
}

func GetFileModTime(filename string) time.Time {
	mep, e := os.Stat(filename)
	if e == nil {
		return mep.ModTime()
	}
	return time.Time{}
}
func (j *Jira) ProcessLocalFiles() (map[string]MarkdownTask, []MarkdownTask) {
	// Get all local tasks
	files := []string{}
	file, err := os.Open(appPreferences.JiraProjectHome)
	if err != nil {
		fmt.Printf("Err load")
	}
	defer file.Close()
	names, err := file.Readdirnames(0)
	if err != nil {
		fmt.Printf("Err read")
	}
	for _, name := range names {
		if len(name) > 3 && name[len(name)-3:] == ".md" {
			files = append(files, filepath.Join(appPreferences.JiraProjectHome, name))
		}
	}
	foundWithJira := map[string]MarkdownTask{}
	foundWithoutJira := []MarkdownTask{}
	for _, filename := range files {
		me := j.convertFileToData(filename)
		me.Metadata.Updated = GetFileModTime(filename).Format(JiraDateTimeStampFormat)
		if me.JiraID == "" {
			foundWithoutJira = append(foundWithoutJira, me)
		} else {
			foundWithJira[me.JiraID] = me
		}
	}
	return foundWithJira, foundWithoutJira
}

func (j *Jira) convertFileToData(filename string) MarkdownTask {
	v := JiraMetadata{}
	var comments2 []string
	var description_wiki []byte
	var wikiContent string

	data, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatal(err)
	}
	mep := strings.Split(string(data), "---\n")
	if len(mep) < 3 {
		return MarkdownTask{}
	}
	mep[1] = strings.Trim(mep[1], "\n")
	err = yaml.Unmarshal([]byte(mep[1]), &v)
	if err != nil {
		log.Fatal(err)
	}
	content := strings.Join(mep[2:], "---\n")

	sections := strings.Split(content, "\n## ")
	for i, s := range sections {
		if len(s) > 7 && strings.ToLower(s[0:8]) == "updates\n" {
			// Remove this from the main sections
			sections = append(sections[:i], sections[i+1:]...)
			// Comments
			comments2 = strings.Split(s, "\n### ")[1:]
			for ii, cc := range comments2 {
				cmd := exec.Command("pandoc", "-f", "markdown", "-t", "jira", "-")
				stdin, err := cmd.StdinPipe()
				if err != nil {
					log.Fatal(err)
				}
				go func() {
					defer stdin.Close()
					io.WriteString(stdin, cc)
				}()
				out, err := cmd.CombinedOutput()
				if err != nil {
					log.Fatal(err)
				}
				comments2[ii] = string(out)
			}
		}
	}
	wikiContent = strings.Join(sections, "\n## ")

	cmd := exec.Command("pandoc", "-f", "markdown", "-t", "jira", "-")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		defer stdin.Close()
		io.WriteString(stdin, wikiContent)
	}()
	description_wiki, err = cmd.CombinedOutput()
	if err != nil {
		log.Fatal(err)
	}
	if v.Name == "" && strings.Contains(strings.Split("\n", content)[0], "-") {
		v.Name = strings.Split("-", strings.Split("\n", content)[0])[1]
	} else if v.Name == "" {
		v.Name = strings.Split("\n", content)[0]
	}
	if !strings.Contains(v.Priority, "Deferred") {
		if val, ok := validPriorities[strings.ToUpper(v.Priority)]; ok {
			v.Priority = val
		} else {
			v.Priority = "Low"
		}
	} else {
		v.Priority = "Lowest"
	}

	x1 := MarkdownTask{
		JiraID:   "",
		Filename: filename,
		Body:     string(description_wiki),
		Comments: comments2,
		Metadata: v,
	}
	if v.JiraKey != "" {
		x1.JiraID = v.JiraKey
	}
	return x1
}

type JiraExtendedResponseType struct {
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
			Parent struct {
				Key string `json:"key"`
			} `json:"parent"`
			CreatedDateTime string `json:"created"`
			Status          struct {
				Name    string `json:"name"`
				IconURL string `json:"iconUrl"`
			} `json:"status"`
			UpdatedDateTime string `json:"updated"`
			Description     string `json:"description"`
			Progress        struct {
				Progress int `json:"progress"`
				Total    int `json:"total"`
			} `json:"progress"`
		} `json:"fields"`
	} `json:"issues"`
}

func (j *Jira) DownloadFull() map[string]MarkdownTask {
	toReturn := map[string]MarkdownTask{}
	var jiraResponse JiraExtendedResponseType
	baseQuery := `jql=assignee%3Dcurrentuser()`
	queryToCall := fmt.Sprintf("%s&startAt=0", baseQuery)
	for page := 1; page < 200; page++ {
		r, err := j.callJiraURI("GET", "search", []byte{}, queryToCall)
		if err == nil {
			defer r.Close()
			_ = json.NewDecoder(r).Decode(&jiraResponse)

			for _, y := range jiraResponse.Issues {
				toReturn[y.ID] = MarkdownTask{
					JiraID:   y.ID,
					Body:     y.Fields.Description,
					Comments: []string{},
					Metadata: JiraMetadata{
						Name:            y.Fields.Summary,
						Sponsor:         "",
						Status:          y.Fields.Status.Name,
						Priority:        validPriorities[strings.ToUpper(y.Fields.Priority.Name)],
						Client:          "",
						SecurityContact: "",
						JiraKey:         y.ID,
						Updated:         y.Fields.UpdatedDateTime,
						Inactive:        "",
						Epic:            y.Fields.Parent.Key,
						Tags:            []string{},
					},
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
	return toReturn
}

var validPriorities = map[string]string{
	"HIGHEST": "Highest",
	"HIGH":    "High",
	"MEDIUM":  "Medium",
	"LOW":     "Low",
	"LOWEST":  "Lowest",
}
*/
