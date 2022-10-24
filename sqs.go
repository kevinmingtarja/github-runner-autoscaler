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

func (q *sqsQueue) ReceiveJobs(ctx context.Context) ([]queue.WorkflowJob, error) {
	waitTimeSeconds := SqsMaxWaitTimeSeconds
	log.Printf("Polling SQS for %d seconds\n", waitTimeSeconds)
	out, err := q.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:            &q.url,
		MaxNumberOfMessages: SqsMaxNumberOfMessaged,
		WaitTimeSeconds: int32(waitTimeSeconds), // Long polling to reduce network calls
	})
	if err != nil {
		return nil, err
	}
	log.Printf("Received %d message(s)\n", len(out.Messages))

	jobs := make([]queue.WorkflowJob, len(out.Messages))
	var wj queue.WorkflowJob
	for i, m := range out.Messages {
		err := json.Unmarshal([]byte(*m.Body), &wj)
		if err != nil {
			return nil, err
		}
		jobs[i] = queue.WorkflowJob{Id: wj.Id}
		log.Printf("Received: Workflow job %d\n", wj.Id)
	}

	return jobs, nil
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
			MessageBody:    aws.String(string(b)),
			QueueUrl:       &q.url,
			MessageGroupId: aws.String(MessageGroupId),
		},
	)
	if err != nil {
		return nil, err
	}
	return &queue.SendMessageOutput{MessageId: out.MessageId}, nil
}
