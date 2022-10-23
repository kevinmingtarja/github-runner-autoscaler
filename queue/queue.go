package queue

import "context"

type JobQueue interface {
	Poll(ch chan<- *Message)
	SendJob(ctx context.Context, job *Job) (*SendMessageOutput, error)
}

type Message struct {
	Body *string
	MessageId *string
}

type SendMessageOutput struct {
	MessageId *string
}

type Job struct {
	JobId int `json:"job_id"`
}