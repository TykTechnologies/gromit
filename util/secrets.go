package util

import (
	"context"

	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	"github.com/rs/zerolog/log"
)

// CreateSecret will create a secret with id sid and write the plaintext into it
func CreateSecret(sid string, plaintext string) error {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatal().Err(err).Msg("could not load AWS config")
	}

	svc := secretsmanager.NewFromConfig(cfg)
	input := &secretsmanager.CreateSecretInput{
		Description:  aws.String("created by automation"),
		Name:         aws.String(sid),
		SecretString: aws.String(plaintext),
	}

	result, err := svc.CreateSecret(context.Background(), input)
	log.Trace().Interface("result", result).Msg("from create secret")
	return err
}

// UpdateSecret will write the plaintext into the AWS Secrets Manager secret referred to by sid
func UpdateSecret(sid string, plaintext string) error {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatal().Err(err).Msg("could not load AWS config")
	}

	svc := secretsmanager.NewFromConfig(cfg)
	input := &secretsmanager.UpdateSecretInput{
		SecretId:     aws.String(sid),
		SecretString: aws.String(plaintext),
	}

	result, err := svc.UpdateSecret(context.Background(), input)
	log.Trace().Interface("result", result).Msg("from create secret")
	if err != nil {
		var rne *types.ResourceNotFoundException
		if errors.As(err, &rne) {
			log.Info().Str("secretid", sid).Msg("not found, creating")
			return CreateSecret(sid, plaintext)
		}
		var ise *types.InternalServiceError
		if errors.As(err, &ise) {
			log.Error().Err(err).Msg("ISE from AWS, implement retry if appropriate")
		}
	}
	return err
}
