package queue

import (
	"context"
)

type WorkflowJobQueue interface {
	SendJob(ctx context.Context, job *WorkflowJob) (*SendMessageOutput, error)
	ReceiveMessages(ctx context.Context) ([]Message, error)
	MarkMessageAsDone(ctx context.Context, msg Message) error
}

type Message struct {
	Id            *string
	WorkflowJob   WorkflowJob
	ReceiptHandle *string
}

type SendMessageOutput struct {
	MessageId *string
}

type WorkflowJob struct {
	Id         int      `json:"id"`
	Name       string   `json:"name"`
	Url        string   `json:"url"`
	StartedAt  string   `json:"started_at"`
	Labels     []string `json:"labels"`
	Conclusion string   `json:"conclusion"`
	RunnerId   int      `json:"runner_id"`
	RunnerName string   `json:"runner_name"`
}
