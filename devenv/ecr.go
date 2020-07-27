package devenv

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecr/ecriface"
	"github.com/rs/zerolog/log"
)

// EnvState holds the branch name for an environment
type EnvState map[string]string

// GetEnvState returns master as the ref if no env was found
func GetEnvState(svc ecriface.ClientAPI, registry string, env string, repos []string) (EnvState, error) {
	var state = make(EnvState)
	state["name"] = env

	for _, repo := range repos {
		tag, err := getExistingTag(svc, registry, repo, env)
		state[repo] = tag

		if err != nil {
			if _, ok := err.(NotFoundError); ok {
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

	return "", NotFoundError{repo}
}
