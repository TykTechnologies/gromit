package devenv

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/dynamodbiface"
	"github.com/aws/aws-sdk-go-v2/service/ec2/ec2iface"
	"github.com/aws/aws-sdk-go-v2/service/ecs/ecsiface"
	"github.com/aws/aws-sdk-go-v2/service/route53/route53iface"
	"github.com/rs/zerolog"
)

// VersionMap maps repos to any tree-ish in git
type VersionMap map[string]string

// DevEnv represents the known (or desired) state of an environment based on the state in DynamoDB.
// This type is concerned with management of the DynamoDB item representing the state of an environment named DevEnv.Name
type DevEnv struct {
	Name     string `json:"name"`
	state    string
	versions VersionMap
	dbClient dynamodbiface.ClientAPI
	table    string
	aws      aws.Config
}

// GromitTask is used inside GromitCluster
type GromitTask struct {
	Name string
	IP   string
}

// GromitCluster represents an ECS cluster running an environment that was spun up by DevEnv.Sow()
// It represents the runtime state (ECR, ECS, R53) of an environment and is intended to encapsulate the AWS implementation specific details
// All its methods are read-only except for SyncDNS which will update the public Route53 entries for a developer environment
type GromitCluster struct {
	Name      string
	Region    string
	r53Client route53iface.ClientAPI
	ecsClient ecsiface.ClientAPI
	ec2Client ec2iface.ClientAPI
	aws       aws.Config
	tasks     []GromitTask
	log       zerolog.Logger
}

type baseError struct {
	Thing string
}

// NotFoundError is used to distinguish between other errors and this expected error
// in getEnv and elsewhere
type NotFoundError baseError

func (e NotFoundError) Error() string { return "does not exist: " + e.Thing }

// ExistsError is used when the environment exists but was updated via
// a method that is not idempotent
type ExistsError baseError

func (e ExistsError) Error() string { return "already exists: " + e.Thing }
