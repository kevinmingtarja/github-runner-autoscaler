package main

import (
	"context"
	"encoding/json"
	"example.com/github-runner-autoscaler/queue"
	"example.com/github-runner-autoscaler/runnerscaling"
	"github.com/joho/godotenv"
	"github.com/pkg/errors"
	"log"
	"net/http"
	"os"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("%s\n", err)
	}
}

func run() error {
	log.Println("Loading environment variables")
	err := godotenv.Load(".env")
	if err != nil {
		return errors.Wrap(err, "environment variables")
	}

	q, err := setupSqsQueue(os.Getenv("QUEUE_URL"))
	if err != nil {
		return errors.Wrap(err, "setup queue")
	}

	errs := make(chan error, 1)

	m, err := runnerscaling.SetupManager(os.Getenv("GITHUB_TOKEN"))
	if err != nil {
		return errors.Wrap(err, "setup gh runner scaling manager")
	}
	m.RegisterQueue(q)
	go func() {
		errs <- m.ListenAndHandleScaleUp()
	}()

	srv := initServer(q)
	go func() {
		log.Println("Starting server at port 8080")
		errs <- http.ListenAndServe(":8080", srv)
	}()

	if err := <-errs; err != nil {
		return err
	}

	return nil
}

type workflowJobEvent struct {
	Action      runnerscaling.Status `json:"action"`
	WorkflowJob queue.WorkflowJob    `json:"workflow_job"`
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

	log.Printf("Receive workflow job: '%+v'\n", e)
	if e.Action == runnerscaling.StatusQueued {
		msg, err := s.q.SendJob(ctx, &e.WorkflowJob)
		if err != nil {
			log.Println(err)
			return
		}
		log.Printf("Successfully sent workflow job %d in message %s\n", e.WorkflowJob.Id, *msg.MessageId)
	}

	log.Println("Finish handling request")
}
