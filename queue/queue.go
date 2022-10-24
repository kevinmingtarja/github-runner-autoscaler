package queue

import "context"

type WorkflowJobQueue interface {
	Poll(ch chan<- *Message)
	SendJob(ctx context.Context, job *WorkflowJob) (*SendMessageOutput, error)
}

type Message struct {
	Body *string
	MessageId *string
}

type SendMessageOutput struct {
	MessageId *string
}

type WorkflowJob struct {
	Id int `json:"id"`
}