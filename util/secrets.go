package util

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/awserr"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/rs/zerolog/log"
)

// CreateSecret will create a secret with id sid and write the plaintext into it
func CreateSecret(sid string, plaintext string) error {
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("could not load AWS config")
	}

	svc := secretsmanager.New(cfg)
	input := &secretsmanager.CreateSecretInput{
		Description:        aws.String("created by automation"),
		Name:               aws.String(sid),
		SecretString:       aws.String(plaintext),
	}

	req := svc.CreateSecretRequest(input)
	result, err := req.Send(context.Background())
	log.Trace().Interface("result", result).Msg("from create secret")
	return err
}

// UpdateSecret will write the plaintext into the AWS Secrets Manager secret referred to by sid
func UpdateSecret(sid string, plaintext string) error {
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("could not load AWS config")
	}

	svc := secretsmanager.New(cfg)
	input := &secretsmanager.UpdateSecretInput {
		SecretId:     aws.String(sid),
			SecretString: aws.String(plaintext),
		}

	req := svc.UpdateSecretRequest(input)
	result, err := req.Send(context.Background())
	log.Trace().Interface("result", result).Msg("from create secret")
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case secretsmanager.ErrCodeResourceNotFoundException:
				log.Info().Str("secretid", sid).Msg("not found, creating")
				return CreateSecret(sid, plaintext)
			case secretsmanager.ErrCodeInternalServiceError:
				log.Error().Err(err).Msg("ISE from AWS, implement retry if appropriate")
			}
		}
	}
	return err
}
