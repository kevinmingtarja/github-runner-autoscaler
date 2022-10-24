package main

import (
	"context"
	"encoding/json"
	"example.com/github-runner-autoscaler/queue"
	"fmt"
	"github.com/joho/godotenv"
	"github.com/pkg/errors"
	"log"
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
	}
}

func run() error {
	log.Println("Loading environment variables")
	err := godotenv.Load(".env")
	if err != nil {
		return errors.Wrap(err, "environment variables")
	}

	log.Println("Setting up connection with SQS queue")
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

	log.Println("Starting server at port 8080")
	return http.ListenAndServe(":8080", srv)
}

type workflowJobEvent struct {
	Action      string      `json:"action"`
	WorkflowJob workflowJob `json:"workflow_job"`
}

type workflowJob struct {
	Id          int      `json:"id"`
	RunUrl      string   `json:"run_url"`
	Conclusion  string   `json:"conclusion"`
	StartedAt   string   `json:"started_at"`
	CompletedAt string   `json:"completed_at"`
	Labels      []string `json:"labels"`
	RunnerId    int      `json:"runner_id"`
	RunnerName  string   `json:"runner_name"`
}

func (s *server) handleWebhookEvent(w http.ResponseWriter, r *http.Request) {
	// TO-DO: Add better request logging
	log.Printf("Handling request: %s %s\n", r.Method, r.URL.String())
	ctx := context.Background()

	var e workflowJobEvent
	err := json.NewDecoder(r.Body).Decode(&e)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("Receive workflow job of action '%s'\n", e.Action)
	if e.Action == StatusQueued {
		msg, err := s.q.SendJob(ctx, &queue.WorkflowJob{Id: e.WorkflowJob.Id})
		if err != nil {
			log.Println(err)
			return
		}
		log.Printf("Successfully sent workflow job %d in message %s\n", e.WorkflowJob.Id, *msg.MessageId)
	}

	log.Println("Finish handling request")
}
