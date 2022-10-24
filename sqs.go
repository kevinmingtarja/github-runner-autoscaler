package main

import (
	"context"
	"encoding/json"
	"example.com/github-runner-autoscaler/queue"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"log"
)

const (
	SqsMaxWaitTimeSeconds  = 20
	SqsMaxNumberOfMessaged = 10
	MessageGroupId         = "jobs-queue"
)

type sqsQueue struct {
	*sqs.Client
	url string
}

func setupSqsQueue(url string) (*sqsQueue, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, err
	}

	return &sqsQueue{sqs.NewFromConfig(cfg), url}, nil
}

func (q *sqsQueue) Poll(ch chan<- *queue.Message) {
	log.Println("Polling SQS queue")
	for {
		out, err := q.ReceiveMessage(context.Background(), &sqs.ReceiveMessageInput{
			QueueUrl:            &q.url,
			MaxNumberOfMessages: SqsMaxNumberOfMessaged,
			WaitTimeSeconds:     SqsMaxWaitTimeSeconds,
		})

		if err != nil {
			log.Fatalf("Failed to fetch sqs message %v", err)
		}

		for _, message := range out.Messages {
			log.Printf("Received message with id %s\n", *message.MessageId)
			ch <- &queue.Message{Body: message.Body, MessageId: message.MessageId}
		}
	}
}

func (q *sqsQueue) SendJob(ctx context.Context, job *queue.WorkflowJob) (*queue.SendMessageOutput, error) {
	log.Printf("Sending workflow job %d to SQS", job.Id)
	b, err := json.Marshal(*job)
	if err != nil {
		return nil, err
	}
	out, err := q.SendMessage(
		ctx,
		&sqs.SendMessageInput{
			MessageBody: aws.String(string(b)),
			QueueUrl: &q.url,
			MessageGroupId: aws.String(MessageGroupId),
		},
	)
	if err != nil {
		return nil, err
	}
	return &queue.SendMessageOutput{MessageId: out.MessageId}, nil
}
