package devenv

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/dynamodbiface"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecr/ecriface"
	"github.com/rs/zerolog/log"
)

// EnvState holds the branch name for an environment
type EnvState map[string]string

// notFoundError is internal to this package
type notFoundError struct {
	Tag string
}

func (e *notFoundError) Error() string { return "not found for " + e.Tag }

// UpsertNewBuild will create or update the item for a given env
func UpsertNewBuild(db dynamodbiface.ClientAPI, tableName string, env string, state EnvState) error {
	update := expression.Set(
		expression.Name("tyk"),
		expression.Value(state["tyk"]),
	).Set(
		expression.Name("tyk-analytics"),
		expression.Value(state["tyk-analytics"]),
	).Set(
		expression.Name("tyk-pump"),
		expression.Value(state["tyk-pump"]),
	)

	// Create the DynamoDB expression from the Update.
	expr, err := expression.NewBuilder().
		WithUpdate(update).
		Build()

	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeNames: expr.Names(),
		Key: map[string]dynamodb.AttributeValue{
			"env": {
				S: aws.String(env),
			},
		},
		ExpressionAttributeValues: expr.Values(),
		ReturnValues:              dynamodb.ReturnValueAllNew,
		TableName:                 aws.String(tableName),
		UpdateExpression:          expr.Update(),
	}

	req := db.UpdateItemRequest(input)
	result, err := req.Send(context.Background())
	log.Trace().Interface("upsertnewbuild", result)
	if err != nil {
		return err
	}
	return nil
}

// GetEnvState returns master as the ref if no env was found
func GetEnvState(svc ecriface.ClientAPI, registry string, env string, repos []string) (EnvState, error) {
	var state = make(EnvState)
	state["name"] = env

	for _, repo := range repos {
		tag, err := getExistingTag(svc, registry, repo, env)
		state[repo] = tag

		if err != nil {
			if _, ok := err.(*notFoundError); ok {
				log.Debug().Msgf("%s: %s", env, err)
				state[repo] = "master"
			} else {
				return state, err
			}
		}
	}
	return state, nil
}

func getExistingTag(svc ecriface.ClientAPI, registry string, repo string, tag string) (string, error) {
	input := &ecr.DescribeImagesInput{
		RegistryId:     aws.String(registry),
		RepositoryName: aws.String(repo),
		MaxResults:     func() *int64 { i := int64(1000); return &i }(),
	}

	req := svc.DescribeImagesRequest(input)
	result, err := req.Send(context.Background())
	log.Trace().Interface("netifaces", result).Msgf("images for %s", repo)
	if err != nil {
		return "", err
	}
	if result.NextToken != nil {
		log.Warn().Msg("More than 1000 images, implement pagination.")
	}
	for _, image := range result.ImageDetails {
		for _, t := range image.ImageTags {
			if t == tag {
				return t, nil
			}
		}
	}

	return "", &notFoundError{repo}
}
