package worker

import (
	"context"
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
	"time"
)

const (
	githubRepoOwner = "kevinmingtarja"
	githubRepoName  = "dgraph"
	ec2NamePrefix   = "gh-runner-"
	ec2NameLength   = 18
)

type Worker struct {
	gh   *github.Client
	ec2c *ec2.Client
	ssmc *ssm.Client
	ec2Args *ec2Args
}

type ec2Args struct {
	kmsKeyId              string
	iamInstanceProfileArn string
	amiName                 string
	ec2InstanceType       ec2Types.InstanceType
	ec2KeyName            string
	ec2SecurityGroup      string
}

// TO-DO: Set disk size to 30gb
func (w *Worker) ScaleUp() error {
	log.Println("Scaling up runners...")
	ctx := context.Background()

	token, err := w.createRunnerAuthToken(ctx)
	if err != nil {
		return err
	}

	runnerName := createEc2InstanceName()

	err = w.storeToken(ctx, &runnerName, token)
	if err != nil {
		return err
	}

	err = w.createNewRunner(ctx, &runnerName)
	if err != nil {
		return err
	}

	return nil
}

func Setup(accessToken string) (*Worker, error) {
	log.Println("Setting up gh runner scaling manager")

	gh := setupGithubClient(accessToken)
	ssmc, err := setupSsm()
	if err != nil {
		return nil, err
	}
	ec2c, err := setupEc2()
	if err != nil {
		return nil, err
	}
	return &Worker{gh: gh, ssmc: ssmc, ec2c: ec2c}, nil
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

func (w *Worker) createRunnerAuthToken(ctx context.Context) (*string, error) {
	log.Println("Attempting to create a new token for the new runner")
	token, _, err := w.gh.Actions.CreateRegistrationToken(ctx, githubRepoOwner, githubRepoName)
	if err != nil {
		return nil, err
	}
	log.Println("Successfully created token")
	return token.Token, nil
}

// Stores token as a SecureString in AWS Systems Manager Parameter Store
func (w *Worker) storeToken(ctx context.Context, runnerName *string, token *string) error {
	log.Println("Attempting to store token in aws ssm parameter store")
	_, err := w.ssmc.PutParameter(ctx, &ssm.PutParameterInput{Name: runnerName, Value: token, KeyId: &w.ec2Args.kmsKeyId, Type: ssmTypes.ParameterTypeSecureString})
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

func (w *Worker) createNewRunner(ctx context.Context, name *string) error {
	log.Println("Attempting to create new ec2 instance")
	amiId, err := getAmiId()
	if err != nil {
		return err
	}
	instance, err := w.ec2c.RunInstances(
		ctx,
		&ec2.RunInstancesInput{
			MinCount:     aws.Int32(1),
			MaxCount:     aws.Int32(1),
			ImageId:      aws.String(amiId),
			InstanceType: w.ec2Args.ec2InstanceType,
			IamInstanceProfile: &ec2Types.IamInstanceProfileSpecification{
				Arn: aws.String(w.ec2Args.iamInstanceProfileArn),
			},
			TagSpecifications: []ec2Types.TagSpecification{{
				ResourceType: ec2Types.ResourceTypeInstance,
				Tags:         []ec2Types.Tag{{Key: aws.String("Name"), Value: name}}},
			},
			SecurityGroups: []string{w.ec2Args.ec2SecurityGroup},
			KeyName:        aws.String(w.ec2Args.ec2KeyName),
		},
	)
	if err != nil {
		log.Println("Failed to create new ec2 instance")
		return err
	}
	log.Printf("Created instance %+v", instance)
	return nil
}

func (w *Worker) getAmiId(name string) (string, error) {

}

// Generates a name of ec2NamePrefix + random string
func createEc2InstanceName() string {
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, ec2NameLength)
	rand.Read(b)
	return fmt.Sprintf("%s%x", ec2NamePrefix, b)[:ec2NameLength]
}

func (m *Worker) handleScaleDown() {
	// use https://docs.github.com/en/rest/actions/self-hosted-runners#list-self-hosted-runners-for-a-repository
	// to query runners and their status
}