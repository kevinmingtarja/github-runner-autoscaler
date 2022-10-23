package main

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"example.com/github-runner-autoscaler/queue"
	"fmt"
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
	for {
		out, err := q.ReceiveMessage(context.Background(), &sqs.ReceiveMessageInput{
			QueueUrl:            &q.url,
			MaxNumberOfMessages: SqsMaxNumberOfMessaged,
			WaitTimeSeconds:     SqsMaxWaitTimeSeconds,
		})

		if err != nil {
			log.Fatalf("failed to fetch sqs message %v", err)
		}

		for _, message := range out.Messages {
			fmt.Println("RECV", message.MessageId)
			ch <- &queue.Message{Body: message.Body, MessageId: message.MessageId}
		}
	}
}

func (q *sqsQueue) SendJob(ctx context.Context, job *queue.Job) (*queue.SendMessageOutput, error) {
	b, err := json.Marshal(*job)
	if err != nil {
		return nil, err
	}
	hash := md5.Sum(b)
	out, err := q.SendMessage(
		ctx,
		&sqs.SendMessageInput{
			MessageBody: aws.String(string(b)),
			QueueUrl: &q.url,
			DelaySeconds: 30,
			MessageGroupId: aws.String(MessageGroupId),
			MessageDeduplicationId: aws.String(hex.EncodeToString(hash[:])),
		},
	)
	if err != nil {
		return nil, err
	}
	return &queue.SendMessageOutput{MessageId: out.MessageId}, nil
}
