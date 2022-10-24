package runnerscaling

import (
	"context"
	"example.com/github-runner-autoscaler/queue"
	"github.com/google/go-github/v48/github"
	"golang.org/x/oauth2"
)

const (
	githubRepoOwner = "kevinmingtarja"
	githubRepoName  = "dgraph"
	StatusQueued = "queued"
)

type Manager struct {
	gh *github.Client
	q queue.WorkflowJobQueue
}

func SetupManager(accessToken string) *Manager {
	gh := newGithubClient(accessToken)
	return &Manager{gh: gh}
}

func (m *Manager) RegisterQueue(q queue.WorkflowJobQueue) {
	m.q = q
}

func (m *Manager) ListenAndHandleScaleUp() {

}

func handleScaleUp() {
	//ctx := context.Background()
	//var jobId int64 = 123
	//isQueued, err := isJobQueued(ctx, client, jobId)
	//if err != nil {
	//	return
	//}
	//fmt.Println(isQueued)
}

func newGithubClient(accessToken string) *github.Client {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: accessToken},
	)
	tc := oauth2.NewClient(ctx, ts)
	return github.NewClient(tc)
}

func getWorkflowJobByID(ctx context.Context, client *github.Client, jobId int64) (*github.WorkflowJob, error) {
	jobForWorkflowRun, _, err := client.Actions.GetWorkflowJobByID(ctx, githubRepoOwner, githubRepoName, jobId)
	return jobForWorkflowRun, err
}

func isJobQueued(ctx context.Context, client *github.Client, jobId int64) (bool, error) {
	job, err := getWorkflowJobByID(ctx, client, jobId)
	if err != nil {
		return false, err
	}
	return *job.Status == StatusQueued, nil
}
