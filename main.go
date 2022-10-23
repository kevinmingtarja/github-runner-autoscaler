package main

import (
	"context"
	"encoding/json"
	"example.com/github-runner-autoscaler/queue"
	"fmt"
	"github.com/joho/godotenv"
	"github.com/pkg/errors"
	"net/http"
	"os"
)

const (
	StatusQueued = "queued"
)

func main() {
	if err := run(); err != nil {
	fmt.Fprintf(os.Stderr, "%s\n", err)
	os.Exit(1)
}}

func run() error {
	err := godotenv.Load(".env")
	if err != nil {
		return errors.Wrap(err, "environment variables")
	}

	q, err := setupSqsQueue(os.Getenv("QUEUE_URL"))
	if err != nil {
		return errors.Wrap(err, "setup queue")
	}
	sqsCh := make(chan *queue.Message)
	go q.Poll(sqsCh)

	//go func() {
	//	msg := <- sqsCh
	//	// handle scale up
	//}()

	srv := initServer(q)

	fmt.Printf("Starting server at port 8080\n")
	return http.ListenAndServe(":8080", srv)
}



type workflowJobEvent struct {
	Action string `json:"action"`
	WorkflowJob workflowJob `json:"workflow_job"`
}

type workflowJob struct {
	Id int `json:"id"`
	RunUrl string `json:"run_url"`
	Conclusion string `json:"conclusion"`
	StartedAt string `json:"started_at"`
	CompletedAt string `json:"completed_at"`
	Labels []string `json:"labels"`
	RunnerId int `json:"runner_id"`
	RunnerName string `json:"runner_name"`
}

func (s *server) handleWebhookEvent(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	var e workflowJobEvent
	err := json.NewDecoder(r.Body).Decode(&e)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fmt.Println(e)
	if e.Action == StatusQueued {
		fmt.Println(StatusQueued)
		msg, err := s.q.SendJob(ctx, &queue.Job{JobId: e.WorkflowJob.Id})
		if err != nil {
			return
		}
		fmt.Println(msg)
	}

	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte("Success"))
	if err != nil {
		return
	}
}