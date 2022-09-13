package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

func authenticateToDY(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	if query.Get("code") != "" {
		fmt.Printf("QUERY %s\n", query.Get("code"))
	}
}

type MyIndividualProjectTasks struct {
	ID              string  `json:"msdyn_projecttaskid"`
	StartDateTime   string  `json:"msdyn_scheduledstart"`
	Title           string  `json:"msdyn_subject"`
	Priority        int     `json:"msdyn_priority"`
	Effort          float32 `json:"msdyn_effort"`
	EffortRemaining float32 `json:"msdyn_effortremaining"`
	StatusCode      int     `json:"statuscode"`
	ParentTitle     string  `json:"a_806d6ce68e9fe51180e900155db8700e_x002e_msdyn_subject"`
	ParentID        string  `json:"_msdyn_project_value"`
}

type MyProjectTasksResponse struct {
	Value []MyIndividualProjectTasks `json:"value"`
}

func DownloadDynamics() {

	activeTaskStatusUpdate(1)
	defer activeTaskStatusUpdate(-1)

	// * @todo Add Sort
	AppStatus.MyTasksFromDynamics = []TaskResponseStruct{}
	var teamResponse MyProjectTasksResponse
	//		for page := 1; page < 200; page++ {
	r, err := GetDynamics()
	if err == nil {
		defer r.Close()
		_ = json.NewDecoder(r).Decode(&teamResponse)
		for _, y := range teamResponse.Value {
			percentComplete := (y.Effort - y.EffortRemaining) / y.Effort * 100
			if percentComplete < 100 {
				row := TaskResponseStruct{
					ID:               y.ID,
					ParentID:         y.ParentID,
					ParentTitle:      y.ParentTitle,
					Title:            y.Title,
					Priority:         fmt.Sprintf("%d", y.Priority),
					PriorityOverride: fmt.Sprintf("%d", y.Priority),
					Status:           fmt.Sprintf("%d", y.StatusCode),
				}
				row.CreatedDateTime, _ = time.Parse("2006-01-02T15:04:05Z", y.StartDateTime)
				switch y.StatusCode {
				case 0:
					row.Status = "Not started(0)"
				case 1:
					row.Status = "In progress (1)"
				}
				//if val, ok := priorityOverrides.Dynamics[row.ID]; ok {
				//	row.PriorityOverride = val
				//}
				AppStatus.MyTasksFromDynamics = append(
					AppStatus.MyTasksFromDynamics,
					row,
				)
			}
		}
		fmt.Printf("%v\n", AppStatus.MyTasksFromDynamics)
	}
	//		}
}

func GetDynamics() (io.ReadCloser, error) {
	client := &http.Client{
		Timeout: time.Second * 10,
	}
	req, _ := http.NewRequest("GET", `https://orgb972b9ec.api.crm6.dynamics.com/api/data/v9.2/msdyn_projecttasks?savedQuery=4792d21c-d6b4-e511-80e4-00155db8d81d&%24select=_msdyn_project_value,msdyn_subject,_ownerid_value,msdyn_priority,msdyn_project,msdyn_parenttask`, bytes.NewReader([]byte{}))
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", appPreferences.DynamicsKey))
	req.Header.Set("Content-type", "application/json")

	resp, err := client.Do(req)
	return resp.Body, err
}
