package main

import (
	"context"
	"fmt"
	"github.com/google/go-github/v48/github"
	"golang.org/x/oauth2"
	"os"
)

func handleScaleUp() {
	ctx := context.Background()
	client := newGithubClient()
	var jobId int64 = 123
	isQueued, err := isJobQueued(ctx, client, jobId)
	if err != nil {
		return
	}
	fmt.Println(isQueued)
}

func newGithubClient() *github.Client {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GITHUB_TOKEN")},
	)
	tc := oauth2.NewClient(ctx, ts)

	return github.NewClient(tc)
}

func getWorkflowJobByID(ctx context.Context, client *github.Client, jobId int64) (*github.WorkflowJob, error) {
	jobForWorkflowRun, _, err := client.Actions.GetWorkflowJobByID(ctx, "kevinmingtarja", "dgraph", jobId)
	return jobForWorkflowRun, err
}

func isJobQueued(ctx context.Context, client *github.Client, jobId int64) (bool, error) {
	job, err := getWorkflowJobByID(ctx, client, jobId)
	if err != nil {
		return false, err
	}
	return *job.Status == StatusQueued, nil
}
