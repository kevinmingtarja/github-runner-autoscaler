package queue

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

type WorkflowJobQueue interface {
	//Poll(ch chan<- *Message)
	SendJob(ctx context.Context, job *WorkflowJob) (*SendMessageOutput, error)
	ReceiveJob(ctx context.Context) ([]types.Message, error)
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