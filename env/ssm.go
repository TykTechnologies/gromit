package env

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	ssmtypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/rs/zerolog/log"
)

// PutSecureParameter will store the given string in the supplied path as a SecureString.
// The parameter will be created if needed.
func (c *Client) StoreLicense(value, path, keyid string) error {
	ssmc := ssm.NewFromConfig(c.cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	op, err := ssmc.PutParameter(ctx, &ssm.PutParameterInput{
		Name:      aws.String(path),
		Value:     aws.String(value),
		Type:      ssmtypes.ParameterTypeSecureString,
		KeyId:     aws.String(keyid),
		Overwrite: aws.Bool(true),
		Tier:      ssmtypes.ParameterTierStandard,
	})
	log.Trace().Interface("ssmo", op).Msg("PutParameter")
	return err
}
