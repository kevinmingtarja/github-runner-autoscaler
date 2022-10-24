package main

import (
	"example.com/github-runner-autoscaler/queue"
	"log"
	"net/http"
)

type server struct {
	router *http.ServeMux
	q      queue.WorkflowJobQueue
}

func initServer(q queue.WorkflowJobQueue) *server {
	srv := &server{
		router: http.DefaultServeMux,
		q:      q,
	}
	srv.routes()
	return srv
}

func (s *server) routes() {
	log.Println("Registering server function handlers")
	// HandleFunc add handlers to DefaultServeMux
	http.HandleFunc("/webhook", s.handleWebhookEvent)
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}
