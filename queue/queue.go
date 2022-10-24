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
	Id *string
	WorkflowJob WorkflowJob
	ReceiptHandle *string
}

type SendMessageOutput struct {
	MessageId *string
}

type WorkflowJob struct {
	Id int `json:"id"`
}