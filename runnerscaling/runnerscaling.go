package runnerscaling

import (
	"context"
	"example.com/github-runner-autoscaler/queue"
	"github.com/google/go-github/v48/github"
	"golang.org/x/oauth2"
	"log"
)

const (
	githubRepoOwner = "kevinmingtarja"
	githubRepoName  = "dgraph"
	StatusQueued = "queued"
	MaxRunners = 4
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
	ctx := context.Background()
	for {
		messages, err := m.q.ReceiveMessages(ctx)
		if err != nil {
			// TO-DO: Improve error handling
			log.Fatalf("Failed to fetch sqs message %v", err)
		}

		for _, msg := range messages {
			go m.handleScaleUp(ctx, msg)
		}
	}
}

func (m *Manager) handleScaleUp(ctx context.Context, msg queue.Message) {
	workflowJobId := msg.WorkflowJob.Id
	log.Printf("Processing job %d\n", workflowJobId)
	isQueued, err := m.isJobQueued(ctx, int64(workflowJobId))
	if err != nil {
		log.Println(err)
		log.Fatalln("Hint: queue.Message schema might have changed while there is still messages with the old schema in the queue")
	}

	if !isQueued {
		// Even if we receive duplicate messages from the queue, this will prevent us from processing it more than once
		log.Printf("Job %d is no longer queued in github, no runners will be created.\n", workflowJobId)
		return
	}

	log.Printf("Current runners: ... out of MaxRunners")

	log.Printf("Creating a new runner")

	log.Printf("Runner created")

	err = m.q.MarkMessageAsDone(ctx, msg)
	if err != nil {
		return
	}
}

func newGithubClient(accessToken string) *github.Client {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: accessToken},
	)
	tc := oauth2.NewClient(ctx, ts)
	return github.NewClient(tc)
}

func (m *Manager) getWorkflowJobByID(ctx context.Context, jobId int64) (*github.WorkflowJob, error) {
	jobForWorkflowRun, _, err := m.gh.Actions.GetWorkflowJobByID(ctx, githubRepoOwner, githubRepoName, jobId)
	return jobForWorkflowRun, err
}

func (m *Manager) isJobQueued(ctx context.Context, jobId int64) (bool, error) {
	job, err := m.getWorkflowJobByID(ctx, jobId)
	if err != nil {
		return false, err
	}
	return *job.Status == StatusQueued, nil
}
