package devenv

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecr/ecriface"
	"github.com/rs/zerolog/log"
)

// GetECRState returns master as the ref if no env was found.
// Using DevEnv here to model the state of the repositories, the actual repos
// being used is abstracted away. Hopefully, this will turn out to be the right choice.
func GetECRState(svc ecriface.ClientAPI, registry string, env string, repos []string) (DevEnv, error) {
	var state = make(DevEnv)

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
		MaxResults:     aws.Int64(1000),
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
