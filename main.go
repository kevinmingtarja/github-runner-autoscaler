package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

const (
	StatusRequested = "requested"
)

func main() {
	http.HandleFunc("/webhook", webhookHandler)

	fmt.Printf("Starting server at port 8080\n")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

//type workflowRunEvent struct {
//	Action string `json:"action"`
//	WorkflowRun workflowRun `json:"workflow_run"`
//}
//
//type workflowRun struct {
//	CancelUrl string `json:"cancel_url"`
//	Event string `json:"event"`
//	CreatedAt string `json:"created_at"`
//	RerunUrl string `json:"rerun_url"`
//	Status string `json:"status"`
//}

type workflowJobEvent struct {
	Action string `json:"action"`
	WorkflowJob workflowJob `json:"workflow_job"`
}

type workflowJob struct {
	RunUrl string `json:"run_url"`
	Conclusion string `json:"conclusion"`
	StartedAt string `json:"started_at"`
	CompletedAt string `json:"completed_at"`
	Labels []string `json:"labels"`
	RunnerId int `json:"runner_id"`
	RunnerName string `json:"runner_name"`
}

func webhookHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r.Body)
	var wr workflowJobEvent

	err := json.NewDecoder(r.Body).Decode(&wr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fmt.Println(wr)
	if wr.Action == StatusRequested {
		fmt.Println("REQUESTED")
	}

	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte("Success"))
	if err != nil {
		return
	}
}