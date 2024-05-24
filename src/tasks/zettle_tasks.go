package tasks

import (
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"golang.org/x/sync/singleflight"
	"gopkg.in/yaml.v3"
)

type ZettleStruct struct {
	Task
	MyTasks        []TaskResponseStruct
	StatusCallback func(bool, string)
	G              *singleflight.Group
}

var zetLock sync.Mutex
var lookupRegex *regexp.Regexp = regexp.MustCompile(`(?m)^\s*\*\s+\[ \]`)

func (zet *ZettleStruct) Init() {
	zet.G = &singleflight.Group{}
}

func (zet *ZettleStruct) Download(dir string) {
	ActiveTaskStatusUpdate(1)
	defer ActiveTaskStatusUpdate(-1)
	zetLock.Lock()
	defer zetLock.Unlock()
	TaskWindowRefresh("Zettle")
	ConnectionStatusBox(false, "Z")
	zet.MyTasks = []TaskResponseStruct{}
	filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err == nil {
			name := d.Name()
			if d.Name() == "_test" {
				return filepath.SkipDir
			}
			if len(name) > 3 && name[len(name)-3:] == ".md" {
				f, err := os.ReadFile(path)
				if err == nil {
					if lookupRegex.Match(f) {
						zet.MyTasks = append(zet.MyTasks, fileToTask(path))
					}
				} else {
					log.Fatalf("Can't read %v\n", err)
				}
			}
		}
		return nil
	})
	sort.SliceStable(zet.MyTasks, func(i, k int) bool {
		if zet.MyTasks[i].PriorityOverride == zet.MyTasks[k].PriorityOverride {
			return zet.MyTasks[i].CreatedDateTime.Before(zet.MyTasks[k].CreatedDateTime)
		}
		return zet.MyTasks[i].PriorityOverride < zet.MyTasks[k].PriorityOverride
	})
	ConnectionStatusBox(true, "Z")
}

type ZettleYaml struct {
	Name           string `yaml:"Name"`
	Sponsor        string `yaml:"Sponsor"`
	Status         string `yaml:"Status"`
	Priority       string `yaml:"Priority"`
	JiraKey        string `yaml:"JiraKey"`
	JiraLink       string `yaml:"JiraLink"`
	JiraUpdated    string
	Tags           []string
	ServiceNowKey  string
	ServiceNowLink string `yaml:"ServiceNowLink"`
}

func fileToTask(filename string) TaskResponseStruct {
	dat, _ := os.ReadFile(filename)
	fi, _ := os.Stat(filename)
	x := strings.SplitAfterN(string(dat), "---", 3)
	bob := TaskResponseStruct{
		Title:           filepath.Join(filepath.Base(filepath.Dir(filename)), filepath.Base(filename)),
		ID:              filename,
		BusObRecId:      filename,
		Priority:        " ",
		Status:          " ",
		CreatedDateTime: FileCreateTime(fi),
		Links:           []string{},
	}
	if len(x) > 1 {
		c := &ZettleYaml{}
		err := yaml.Unmarshal([]byte(x[1]), c)
		if err == nil {
			bob.Priority = c.Priority
			switch c.Priority {
			case "Highest":
				bob.Priority = "1"
			case "High":
				bob.Priority = "2"
			case "Medium":
				bob.Priority = "3"
			case "Low":
				bob.Priority = "4"
			case "Lowest":
				bob.Priority = "5"
			}
			bob.PriorityOverride = bob.Priority
			bob.Status = c.Status
			bob.Title = c.Name
			/*
				switch c.Status {
				case "In Progress":
					bob.Status = "2"
				case "New":
					bob.Status = "1"
				case "On Hold":
					bob.Status = "3"
				case "Resolved":
					bob.Status = "6"
				case "Cancelled":
					bob.Status = "8"
				}
			*/
			if len(c.JiraLink) > 0 {
				bob.Links = append(bob.Links, c.JiraLink)
			}
			if len(c.ServiceNowLink) > 0 {
				bob.Links = append(bob.Links, c.ServiceNowLink)
			}
		}
	}
	return bob
}
