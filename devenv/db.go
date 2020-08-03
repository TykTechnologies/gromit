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
)

// DevEnv is a tyk env on the dev env. This is not a type because
// changes in repos lists will require a change in the type since this
// type would contain a list of repos. By using a map, we trade type
// checking of the state for flexibility in adding and removing repos.
type DevEnv map[string]interface{}

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

// GetEnv will get the named env using the supplied list of repos to
// construct the projection expression. This method returns *DevEnv,
// not a dynamodb return type
func GetEnv(db dynamodbiface.ClientAPI, table string, env string) (*DevEnv, error) {
	input := &dynamodb.GetItemInput{
		Key: map[string]dynamodb.AttributeValue{
			NAME: {
				S: aws.String(env),
			},
		},
		TableName: aws.String(table),
	}

	req := db.GetItemRequest(input)
	result, err := req.Send(context.Background())
	log.Trace().Interface("result", result).Msgf("get for %s", env)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case dynamodb.ErrCodeResourceNotFoundException:
				log.Error().
					Err(aerr).
					Msgf("Could not find table %s when looking for %s", table, env)
			case dynamodb.ErrCodeRequestLimitExceeded:
				log.Error().
					Err(aerr).
					Msgf("Resource limit exceeded for table %s", table)
			default:
				log.Error().
					Err(aerr).
					Msgf("Unknown error while looking for %s in %s", env, table)
			}
		}
		return &DevEnv{}, err
	}
	if result.Item == nil {
		return &DevEnv{}, NotFoundError{env}
	}

	de := make(DevEnv)
	err = dynamodbattribute.UnmarshalMap(result.Item, &de)
	if err != nil {
		return &de, err
	}

	return &de, nil
}

// InsertEnv will error if the the env already exists
func InsertEnv(db dynamodbiface.ClientAPI, table string, env string, stateMap DevEnv) error {
	// Remove key elements from the map as updates will fail
	delete(stateMap, NAME)
	stateMap[STATE] = NEW

	// An env with the "name" key from state should not already exist
	cond := expression.AttributeNotExists(expression.Name(NAME))

	update := expression.UpdateBuilder{}
	for k, v := range stateMap {
		newUpdate := update.Set(expression.Name(k), expression.Value(v))
		update = newUpdate
	}

	expr, err := expression.NewBuilder().
		WithCondition(cond).
		WithUpdate(update).
		Build()
	if err != nil {
		return err
	}

	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeNames: expr.Names(),
		Key: map[string]dynamodb.AttributeValue{
			NAME: {
				S: aws.String(env),
			},
		},
		ConditionExpression:       expr.Condition(),
		ExpressionAttributeValues: expr.Values(),
		ReturnValues:              dynamodb.ReturnValueAllNew,
		TableName:                 aws.String(table),
		UpdateExpression:          expr.Update(),
	}
	// input := &dynamodb.PutItemInput{
	// 	Item:                   insert,
	// 	ConditionExpression:    expr.Condition(),
	// 	ReturnConsumedCapacity: dynamodb.ReturnConsumedCapacityTotal,
	// 	TableName:              aws.String(table),
	// }

	req := db.UpdateItemRequest(input)
	result, err := req.Send(context.Background())
	log.Trace().Interface("result", result).Msgf("result from inserting %s", env)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case dynamodb.ErrCodeConditionalCheckFailedException:
				log.Error().
					Err(aerr).
					Msgf("env %s already exists", env)
				return ExistsError{env}
			case dynamodb.ErrCodeResourceNotFoundException:
				log.Warn().Msgf("table %s not found, creating.", table)
				err := createTable(db, table)
				if err != nil {
					return err
				}
				log.Warn().Msgf("retrying to insert the given values")
				err = InsertEnv(db, table, env, stateMap)
				if err != nil {
					return err
				}
			case dynamodb.ErrCodeItemCollectionSizeLimitExceededException:
				log.Error().
					Err(aerr).
					Msgf("table %s too big for inserts", table)
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
					Msgf("error inserting %s into %s", env, table)
			}
		}
		return err
	}
	return nil
}

// UpsertEnv will blindly update the given env
// If env is not found, it will be created with the given state
func UpsertEnv(db dynamodbiface.ClientAPI, table string, env string, stateMap DevEnv) error {
	// Remove key elements from the map as updates will fail
	delete(stateMap, NAME)
	// Reset the state so that the runner will pick it up
	stateMap[STATE] = NEW

	update := expression.UpdateBuilder{}
	for k, v := range stateMap {
		newUpdate := update.Set(expression.Name(k), expression.Value(v))
		update = newUpdate
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
				S: aws.String(env),
			},
		},
		ExpressionAttributeValues: expr.Values(),
		ReturnValues:              dynamodb.ReturnValueAllNew,
		TableName:                 aws.String(table),
		UpdateExpression:          expr.Update(),
	}

	req := db.UpdateItemRequest(input)
	result, err := req.Send(context.Background())
	log.Trace().Interface("result", result).Msgf("result from upserting %s", env)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case dynamodb.ErrCodeResourceNotFoundException:
				log.Warn().Msgf("table %s not found, creating.", table)
				err := createTable(db, table)
				if err != nil {
					return err
				}
				log.Warn().Msgf("retrying to upsert the given values")
				err = UpsertEnv(db, table, env, stateMap)
				if err != nil {
					return err
				}
			case dynamodb.ErrCodeItemCollectionSizeLimitExceededException:
				log.Error().Err(aerr).Msgf("table %s too big for upserts", table)
			case dynamodb.ErrCodeRequestLimitExceeded:
				log.Error().Err(aerr).Msgf("request limit exceeded for table %s", table)
			case dynamodb.ErrCodeInternalServerError:
				log.Error().Err(aerr).Msg("ISE from AWS, please implement retry if appropriate")
			default:
				log.Error().Err(aerr).Msgf("error inserting %s into %s", env, table)
			}
		}
		return err
	}
	return nil
}

// DeleteEnv will blindly delete the given env
func DeleteEnv(db dynamodbiface.ClientAPI, table string, env string) error {
	input := &dynamodb.DeleteItemInput{
		Key: map[string]dynamodb.AttributeValue{
			NAME: {
				S: aws.String(env),
			},
		},
		TableName: aws.String(table),
	}

	req := db.DeleteItemRequest(input)
	result, err := req.Send(context.Background())
	log.Trace().Interface("result", result).Msgf("result from deleting %s", env)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case dynamodb.ErrCodeResourceNotFoundException:
				log.Warn().Msgf("env %s not found, doing nothing as a delete was called.", table)
			case dynamodb.ErrCodeRequestLimitExceeded:
				log.Error().Err(aerr).Msgf("request limit exceeded for table %s", table)
			case dynamodb.ErrCodeInternalServerError:
				log.Error().Err(aerr).Msg("ISE from AWS, please implement retry if appropriate")
			default:
				log.Error().Err(aerr).Msgf("error deleting %s from %s", env, table)
			}
		}
		return err
	}
	return nil
}

// GetNewEnvs will fetch all envs with state==NEW from the DB
// Only attribute names matching the list in repos will be fetched
func GetNewEnvs(db dynamodbiface.ClientAPI, table string, repos []string) ([]DevEnv, error) {
	filt := expression.Name(STATE).Equal(expression.Value(NEW))

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

	req := db.ScanRequest(input)
	result, err := req.Send(context.Background())
	var envs []DevEnv
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case dynamodb.ErrCodeResourceNotFoundException:
				log.Warn().Msgf("table %s not found", table)
			case dynamodb.ErrCodeRequestLimitExceeded:
				log.Error().Err(aerr).Msgf("request limit exceeded for table %s", table)
			case dynamodb.ErrCodeInternalServerError:
				log.Error().Err(aerr).Msg("ISE from AWS, please implement retry if appropriate")
			default:
				log.Error().Err(aerr).Msgf("error getting new envs from table %s", table)
			}
		}
		return envs, err
	}
	for _, row := range result.Items {
		de := make(DevEnv)
		err = dynamodbattribute.UnmarshalMap(row, &de)
		if err != nil {
			return envs, err
		}
		envs = append(envs, de)
	}
	return envs, nil
}
