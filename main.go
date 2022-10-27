package main

import (
	"example.com/github-runner-autoscaler/worker"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/joho/godotenv"
	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	"log"
	"os"
)

var (
	arch = pflag.StringP("arch", "", "amd",
		"architecture of the machine (amd or arm)")
	kmsKeyId = pflag.StringP("kms-key-id", "", "amd",
		"id of the AWS KMS key used to encrypt secrets")
	iamInstanceProfileArn = pflag.StringP("iam-arn", "", "amd",
		"id of the AWS KMS key used to encrypt secrets")
	amiId                 string
	ec2InstanceType       ec2Types.InstanceType
	ec2KeyName            string
	ec2SecurityGroup      string


	baseDir = pflag.StringP("base", "", "../",
		"Base dir for Dgraph")
	runPkg = pflag.StringP("pkg", "p", "",
		"Only run tests for this package")
	runTest = pflag.StringP("test", "t", "",
		"Only run this test")
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("%s\n", err)
	}
}

func run() error {
	log.Println("Loading environment variables")
	err := godotenv.Load(".env")
	if err != nil {
		return errors.Wrap(err, "environment variables")
	}

	w, err := worker.Setup(os.Getenv("GITHUB_TOKEN"))
	if err != nil {
		return errors.Wrap(err, "setup scaling worker")
	}

	err = w.ScaleUp()
	if err != nil {
		return errors.Wrap(err, "scale up")
	}

	return nil
}

type workflowJobEvent struct {
	Action      string      `json:"action"`
	WorkflowJob workflowJob `json:"workflow_job"`
}

type workflowJob struct {
	Id         int      `json:"id"`
	Name       string   `json:"name"`
	Url        string   `json:"url"`
	StartedAt  string   `json:"started_at"`
	Labels     []string `json:"labels"`
	Conclusion string   `json:"conclusion"`
	RunnerId   int      `json:"runner_id"`
	RunnerName string   `json:"runner_name"`
}