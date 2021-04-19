package devenv

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/awserr"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/dynamodbiface"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/expression"
	"github.com/rs/zerolog/log"
)

const (
	// STATE is the name of the state attribute in the DB
	STATE = "state"
	// NAME is the name of the name attribute in the DB :)
	NAME = "name"
	// NEW is the state when an env is new
	NEW = "new"
	// PROCESSED is the state when an env has been processed by the runner
	PROCESSED = "processed"
	// DELETED is when at least one of the branches for this env have been deleted
	DELETED = "deleted"
)

// GetDevEnv will get the named env with the supplied name from the DB
func GetDevEnv(svc dynamodbiface.ClientAPI, table string, envName string) (*DevEnv, error) {
	input := &dynamodb.GetItemInput{
		Key: map[string]dynamodb.AttributeValue{
			NAME: {
				S: aws.String(envName),
			},
		},
		TableName: aws.String(table),
	}

	req := svc.GetItemRequest(input)
	result, err := req.Send(context.Background())
	log.Trace().Interface("result", result).Msgf("get for %s", envName)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case dynamodb.ErrCodeResourceNotFoundException:
				log.Error().Err(aerr).Msg("could not find table")
			case dynamodb.ErrCodeRequestLimitExceeded:
				log.Error().Err(aerr).Msg("Resource limit exceeded")
			default:
				log.Error().Err(aerr).Msg("nknown error while looking for env")
			}
		}
		return &DevEnv{}, err
	}
	if result.Item == nil {
		return &DevEnv{}, NotFoundError{envName}
	}

	envMap := make(VersionMap)
	err = dynamodbattribute.UnmarshalMap(result.Item, &envMap)
	if err != nil {
		return &DevEnv{}, err
	}

	state := envMap[STATE]
	delete(envMap, NAME)
	delete(envMap, STATE)
	return &DevEnv{
		Name:     envName,
		versions: envMap,
		state:    state,
		dbClient: svc,
		table:    table,
	}, nil
}

// NewDevEnv returns an unsaved environment but the env is ready to be saved
func NewDevEnv(name string, db dynamodbiface.ClientAPI, table string) *DevEnv {
	return &DevEnv{
		Name:     name,
		state:    NEW,
		dbClient: db,
		table:    table,
		versions: nil,
	}
}

// Save will save the env to the DB, creating the item if needed
func (d *DevEnv) Save() error {
	update := expression.UpdateBuilder{}
	update = update.Set(expression.Name("state"), expression.Value(d.state))
	for k, v := range d.versions {
		update = update.Set(expression.Name(k), expression.Value(v))
	}

	// Create the DynamoDB expression from the Update.
	expr, err := expression.NewBuilder().
		WithUpdate(update).
		Build()
	if err != nil {
		return err
	}

	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeNames: expr.Names(),
		Key: map[string]dynamodb.AttributeValue{
			NAME: {
				S: aws.String(d.Name),
			},
		},
		ExpressionAttributeValues: expr.Values(),
		ReturnValues:              dynamodb.ReturnValueAllNew,
		TableName:                 aws.String(d.table),
		UpdateExpression:          expr.Update(),
	}

	req := d.dbClient.UpdateItemRequest(input)
	result, err := req.Send(context.Background())
	log.Trace().Interface("result", result).Msg("from api")
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case dynamodb.ErrCodeResourceNotFoundException:
				log.Warn().Msg("table not found, while saving env")
				err := createTable(d.dbClient, d.table)
				if err != nil {
					return err
				}
				log.Warn().Msg("retrying saving after creating the table")
				err = d.Save()
				if err != nil {
					return err
				}
			case dynamodb.ErrCodeItemCollectionSizeLimitExceededException:
				log.Error().Err(aerr).Msg("too big for upserts")
			case dynamodb.ErrCodeRequestLimitExceeded:
				log.Error().Err(aerr).Msg("request limit exceeded")
			case dynamodb.ErrCodeInternalServerError:
				log.Error().Err(aerr).Msg("ISE from AWS, please implement retry if appropriate")
			default:
				log.Error().Err(aerr).Msg("unknown error")
			}
		}
		return err
	}
	return nil
}

// Promote all versions to top-level keys.
// This is done so that VersionMap can support any list of repos at run time
func (d *DevEnv) VersionMap() VersionMap {
	versions := make(map[string]string)
	versions["name"] = d.Name
	versions["state"] = d.state
	for k, v := range d.versions {
		versions[k] = v
	}
	return versions
}

func (d *DevEnv) MarkDeleted() {
	d.state = DELETED
}

func (d *DevEnv) MarkNew() {
	d.state = NEW
}

func (d *DevEnv) MarkProcessed() {
	d.state = PROCESSED
}

func (d *DevEnv) MergeVersions(vs VersionMap) error {
	for k, v := range vs {
		d.versions[k] = v
	}
	return d.Save()
}

func (d *DevEnv) SetVersions(vs VersionMap) {
	delete(vs, STATE)
	delete(vs, NAME)
	d.versions = vs
}

func (d *DevEnv) SetVersion(repo string, version string) {
	d.versions[repo] = version
}

// Delete will delete the env if its internal state is devenv.DELETED
func (d *DevEnv) Delete() error {
	if d.state != DELETED {
		return fmt.Errorf("state needs to be %s but is %s", DELETED, d.state)
	}
	input := &dynamodb.DeleteItemInput{
		Key: map[string]dynamodb.AttributeValue{
			NAME: {
				S: aws.String(d.Name),
			},
		},
		TableName: aws.String(d.table),
	}

	req := d.dbClient.DeleteItemRequest(input)
	result, err := req.Send(context.Background())
	log.Trace().Interface("result", result).Msg("from api")
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case dynamodb.ErrCodeResourceNotFoundException:
				log.Warn().Msg("not found, doing nothing as a delete was called.")
			case dynamodb.ErrCodeItemCollectionSizeLimitExceededException:
				log.Error().Err(aerr).Msg("too big for deletes")
			case dynamodb.ErrCodeRequestLimitExceeded:
				log.Error().Err(aerr).Msg("request limit exceeded")
			case dynamodb.ErrCodeInternalServerError:
				log.Error().Err(aerr).Msg("ISE from AWS, please implement retry if appropriate")
			default:
				log.Error().Err(aerr).Msg("unknown error")
			}
		}
		return err
	}
	return nil
}

// GetEnvsByState will fetch all envs in the supplied state from the DB
// Only attribute names matching the list in repos will be fetched
// If any error occurs while retrieving an env, it fails immediately and returns the list of envs that have been retrieved so far
func GetEnvsByState(svc dynamodbiface.ClientAPI, table string, state string, repos []string) ([]DevEnv, error) {
	var envs []DevEnv
	filt := expression.Name(STATE).Equal(expression.Value(state))

	proj := expression.NamesList(expression.Name(NAME))
	for _, r := range repos {
		newProj := proj.AddNames(expression.Name(r))
		proj = newProj
	}

	expr, err := expression.NewBuilder().
		WithFilter(filt).
		WithProjection(proj).
		Build()
	if err != nil {
		fmt.Println(err)
	}

	// Use the built expression to populate the DynamoDB Scan API input parameters.
	input := &dynamodb.ScanInput{
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		FilterExpression:          expr.Filter(),
		ProjectionExpression:      expr.Projection(),
		TableName:                 aws.String(table),
	}

	req := svc.ScanRequest(input)
	result, err := req.Send(context.Background())
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case dynamodb.ErrCodeResourceNotFoundException:
				log.Warn().Msgf("table not found")
			case dynamodb.ErrCodeItemCollectionSizeLimitExceededException:
				log.Error().Err(aerr).Msg("too big for scans")
			case dynamodb.ErrCodeRequestLimitExceeded:
				log.Error().Err(aerr).Msg("request limit exceeded")
			case dynamodb.ErrCodeInternalServerError:
				log.Error().Err(aerr).Msg("ISE from AWS, please implement retry if appropriate")
			default:
				log.Error().Err(aerr).Msg("unknown error")
			}
		}
		return envs, err
	}
	for _, row := range result.Items {
		envMap := make(VersionMap)
		err = dynamodbattribute.UnmarshalMap(row, &envMap)
		if err != nil {
			return envs, err
		}
		state := envMap[STATE]
		name := envMap[NAME]
		delete(envMap, NAME)
		delete(envMap, STATE)
		de := DevEnv{
			Name:     name,
			state:    state,
			dbClient: svc,
			table:    table,
			versions: envMap,
		}
		envs = append(envs, de)
	}
	return envs, nil
}

// EnsureTableExists creates a PAY_PER_REQUEST DynamoDB table. If the
// table already exists, it is not re-created nor is an error raised.
// Will create the table if ResourceNotFound is received
func EnsureTableExists(db dynamodbiface.ClientAPI, table string) error {
	tableDesc := &dynamodb.DescribeTableInput{
		TableName: aws.String(table),
	}
	req := db.DescribeTableRequest(tableDesc)
	result, err := req.Send(context.Background())
	log.Trace().Interface("desctable", result)
	if awserr, ok := err.(awserr.Error); ok {
		if awserr.Code() == "ResourceNotFoundException" {
			log.Warn().Msgf("table %s not found, creating.", table)
			err := createTable(db, table)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// DeleteTable unconditionally deletes the table
func DeleteTable(db dynamodbiface.ClientAPI, table string) error {
	input := &dynamodb.DeleteTableInput{
		TableName: aws.String(table),
	}

	req := db.DeleteTableRequest(input)
	result, err := req.Send(context.Background())
	log.Trace().Interface("result", result).Msgf("result from delete")
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case dynamodb.ErrCodeResourceInUseException:
				log.Error().
					Err(aerr).
					Msgf("cannot delete table %s as it is in use", table)
			case dynamodb.ErrCodeResourceNotFoundException:
				log.Error().
					Err(aerr).
					Msgf("delete called for non-existent table %s", table)
			case dynamodb.ErrCodeRequestLimitExceeded:
				log.Error().
					Err(aerr).
					Msgf("request limit exceeded for table %s", table)
			case dynamodb.ErrCodeInternalServerError:
				log.Error().
					Err(aerr).
					Msg("ISE from AWS, please implement retry if appropriate")
			default:
				log.Error().
					Err(aerr).
					Msgf("error deleting table %s", table)
			}
		}
		return err
	}
	return nil
}

func createTable(db dynamodbiface.ClientAPI, table string) error {
	req := db.CreateTableRequest(&dynamodb.CreateTableInput{
		AttributeDefinitions: []dynamodb.AttributeDefinition{
			{
				AttributeName: aws.String(NAME),
				AttributeType: dynamodb.ScalarAttributeTypeS,
			},
		},
		KeySchema: []dynamodb.KeySchemaElement{
			{
				AttributeName: aws.String(NAME),
				KeyType:       dynamodb.KeyTypeHash,
			},
		},
		BillingMode: "PAY_PER_REQUEST",
		TableName:   aws.String(table),
	})
	result, err := req.Send(context.Background())
	log.Trace().Interface("createtable", result)
	if err != nil {
		return err
	}
	log.Info().Msgf("created table %s, waiting for completion", table)
	tableDesc := &dynamodb.DescribeTableInput{
		TableName: aws.String(table),
	}
	err = db.WaitUntilTableExists(context.Background(), tableDesc)
	log.Info().Msgf("completed creation of table %s", table)
	return err
}
