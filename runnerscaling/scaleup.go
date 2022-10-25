package runnerscaling

import (
	"context"
	"encoding/base64"
	"example.com/github-runner-autoscaler/queue"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	ssmTypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/google/go-github/v48/github"
	"golang.org/x/oauth2"
	"log"
	"math/rand"
	"os"
	"time"
)

type Status string

// Enum values for Status
const (
	StatusQueued Status = "queued"
)

const (
	githubRepoOwner = "kevinmingtarja"
	githubRepoName  = "dgraph"
	ec2NamePrefix   = "gh-runner-"
	ec2NameLength   = 18
	ec2UserData     = `
#cloud-boothook
#!/bin/bash
chmod +x /home/ubuntu/init-runner.sh
sh /home/ubuntu/init-runner.sh
`
	MaxRunners = 4
)

var (
	kmsKeyId              string
	iamInstanceProfileArn string
)

type Manager struct {
	gh   *github.Client
	q    queue.WorkflowJobQueue
	ssmc *ssm.Client
	ec2c *ec2.Client
}

func SetupManager(accessToken string) (*Manager, error) {
	log.Println("Setting up gh runner scaling manager")
	kmsKeyId = os.Getenv("KMS_KEY_ID")
	iamInstanceProfileArn = os.Getenv("IAM_ARN")

	gh := setupGithubClient(accessToken)
	ssmc, err := setupSsm()
	if err != nil {
		return nil, err
	}
	ec2c, err := setupEc2()
	if err != nil {
		return nil, err
	}
	return &Manager{gh: gh, ssmc: ssmc, ec2c: ec2c}, nil
}

func (m *Manager) RegisterQueue(q queue.WorkflowJobQueue) {
	m.q = q
}

func (m *Manager) ListenAndHandleScaleUp() error {
	ctx := context.Background()
	errs := make(chan error, 1)

	for {
		select {
		case err := <-errs:
			return err
		default:
			messages, err := m.q.ReceiveMessages(ctx)
			if err != nil {
				// TO-DO: Improve error handling
				log.Fatalf("Failed to fetch sqs message %v", err)
			}

			for _, msg := range messages {
				msg := msg
				go func() {
					err := m.handleScaleUp(ctx, msg)
					if err != nil {
						errs <- err
					}
				}()
			}
		}
	}
}

func (m *Manager) handleScaleUp(ctx context.Context, msg queue.Message) error {
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
		return nil
	}

	n, runners, err := m.listCurrentRunners(ctx)
	if err != nil {
		// TO-DO: Improve error handling
		return err
	}
	log.Printf("Current runners: %d out of %d", n, MaxRunners)
	log.Println(runners)

	if n >= MaxRunners {
		log.Println("Max number of runners reached. No runners will be created")
		return nil
	}

	token, err := m.createRunnerAuthToken(ctx)
	if err != nil {
		// TO-DO: Improve error handling
		return err
	}

	runnerName := createEc2InstanceName()

	err = m.storeToken(ctx, &runnerName, token)
	if err != nil {
		// TO-DO: Improve error handling
		return err
	}
	defer m.deleteToken(ctx, &runnerName)

	err = m.createNewRunner(ctx, &runnerName)
	if err != nil {
		// TO-DO: Improve error handling
		return err
	}

	err = m.q.MarkMessageAsDone(ctx, msg)
	if err != nil {
		return err
	}

	return nil
}

func setupGithubClient(accessToken string) *github.Client {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: accessToken},
	)
	tc := oauth2.NewClient(ctx, ts)
	return github.NewClient(tc)
}

func setupSsm() (*ssm.Client, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, err
	}
	return ssm.NewFromConfig(cfg), nil
}

func setupEc2() (*ec2.Client, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, err
	}
	return ec2.NewFromConfig(cfg), nil
}

func (m *Manager) createRunnerAuthToken(ctx context.Context) (*string, error) {
	log.Println("Attempting to create a new token for the new runner")
	token, _, err := m.gh.Actions.CreateRegistrationToken(ctx, githubRepoOwner, githubRepoName)
	if err != nil {
		return nil, err
	}
	log.Println("Successfully created token")
	return token.Token, nil
}

// Stores token as a SecureString in AWS Systems Manager Parameter Store
func (m *Manager) storeToken(ctx context.Context, runnerName *string, token *string) error {
	log.Println("Attempting to store token in aws ssm parameter store")
	_, err := m.ssmc.PutParameter(ctx, &ssm.PutParameterInput{Name: runnerName, Value: token, KeyId: &kmsKeyId, Type: ssmTypes.ParameterTypeSecureString})
	if err != nil {
		return err
	}
	log.Println("Successfully stored token")
	return nil
}

//type ec2Instance struct {
//	ImageId *string
//	InstanceId *string
//	InstanceLifecycle ec2Types.InstanceLifecycleType
//	InstanceType ec2Types.InstanceType
//	KeyName *string
//	LaunchTime *time.Time
//	PublicDnsName *string
//	State *ec2Types.InstanceState
//	Tags []ec2Types.Tag
//}

func (m *Manager) listCurrentRunners(ctx context.Context) (int, []ec2Types.Instance, error) {
	// TO-DO: Filter with a special tag instead of name
	filters := []ec2Types.Filter{
		{
			Name:   aws.String("tag:Name"),
			Values: []string{fmt.Sprintf("%s*", ec2NamePrefix)},
		},
	}
	res, err := m.ec2c.DescribeInstances(ctx, &ec2.DescribeInstancesInput{Filters: filters})
	if err != nil {
		return 0, nil, err
	}

	n := 0
	instances := make([]ec2Types.Instance, 0)
	for _, r := range res.Reservations {
		for _, i := range r.Instances {
			instances = append(instances, i)
			n++
		}
	}
	return n, instances, nil
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
	return Status(*job.Status) == StatusQueued, nil
}

func (m *Manager) createNewRunner(ctx context.Context, name *string) error {
	log.Println("Attempting to create new ec2 instance")
	instance, err := m.ec2c.RunInstances(
		ctx,
		&ec2.RunInstancesInput{
			MinCount:     aws.Int32(1),
			MaxCount:     aws.Int32(1),
			ImageId:      aws.String("ami-0b3cf9d25a3c43687"),
			InstanceType: ec2Types.InstanceTypeM6a4xlarge, // TO-DO
			IamInstanceProfile: &ec2Types.IamInstanceProfileSpecification{
				Arn: aws.String(iamInstanceProfileArn),
			},
			UserData: aws.String(base64.StdEncoding.EncodeToString([]byte(ec2UserData))),
			TagSpecifications: []ec2Types.TagSpecification{{
				ResourceType: ec2Types.ResourceTypeInstance,
				Tags:         []ec2Types.Tag{{Key: aws.String("Name"), Value: name}}},
			},
		},
	)
	if err != nil {
		log.Println("Failed to create new ec2 instance")
		return err
	}
	log.Printf("Created instance %+v", instance)
	return nil
}

func (m *Manager) deleteToken(ctx context.Context, runnerName *string) error {
	log.Println("Attempting to delete token from aws ssm parameter store")
	_, err := m.ssmc.DeleteParameter(ctx, &ssm.DeleteParameterInput{Name: runnerName})
	if err != nil {
		return err
	}
	log.Println("Successfully deleted token")
	return nil
}

// Generates a name of ec2NamePrefix + random string
func createEc2InstanceName() string {
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, ec2NameLength)
	rand.Read(b)
	return fmt.Sprintf("%s%x", ec2NamePrefix, b)[:ec2NameLength]
}
