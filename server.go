package main

import (
	"example.com/github-runner-autoscaler/queue"
	"net/http"
)

type server struct {
	router *http.ServeMux
	q queue.JobQueue
}

func initServer(q queue.JobQueue) *server {
	srv := &server{
		router: http.DefaultServeMux,
		q: q,
	}
	srv.routes()
	return srv
}

func (s *server) routes() {
	// HandleFunc add handlers to DefaultServeMux
	http.HandleFunc("/webhook", s.handleWebhookEvent)
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}