package devenv

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecr/ecriface"
	"github.com/rs/zerolog/log"
)

// GetECRState returns master as the tree-ish if no env was found
func GetECRState(svc ecriface.ClientAPI, registry string, envName string, repos []string) (VersionMap, error) {
	var versionMap = make(VersionMap)

	for _, repo := range repos {
		tag, err := getExistingTag(svc, registry, repo, envName)
		versionMap[repo] = tag

		if err != nil {
			if _, ok := err.(NotFoundError); ok {
				log.Debug().Msgf("%s: %s", envName, err)
				versionMap[repo] = "master"
			} else {
				return versionMap, err
			}
		}
	}
	return versionMap, nil
}

func getExistingTag(svc ecriface.ClientAPI, registry string, repo string, tag string) (string, error) {
	input := &ecr.DescribeImagesInput{
		RegistryId:     aws.String(registry),
		RepositoryName: aws.String(repo),
		MaxResults:     aws.Int64(1000),
	}

	req := svc.DescribeImagesRequest(input)
	result, err := req.Send(context.Background())
	log.Trace().Interface("result", result).Msgf("images for %s", repo)
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
