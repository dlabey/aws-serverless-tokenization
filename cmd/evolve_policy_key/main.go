package main

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/kms"
	"gopkg.in/aws/aws-lambda-go.v1/events"
	"gopkg.in/aws/aws-lambda-go.v1/lambda"
)

// AWS Clients that can be mocked for testing
var (
	KMS      = NewKMS()
	DynamoDB = NewDynamoDB()

	sess = session.Must(session.NewSession())
)

// KMSAPI is a subset of kmsiface.KMSAPI
type KMSAPI interface {
	CreateKey(input *kms.CreateKeyInput) (*kms.CreateKeyOutput, error)
	GenerateDataKey(input *kms.GenerateDataKeyInput) (*kms.GenerateDataKeyOutput, error)
}

// NewKMS is a mockable KMS client
func NewKMS() KMSAPI {
	return kms.New(sess)
}

// DynamoDBAPI is a subset of dynamodbiface.DynamoDBAPI
type DynamoDBAPI interface {
	PutItem(input *dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error)
}

// NewDynamoDB is a mockable DynamoDB client
func NewDynamoDB() DynamoDBAPI {
	return dynamodb.New(sess)
}

// EvolvePolicyHandler is a CloudWatch even handler that envolves the policy key.
func EvolvePolicyHandler(event events.CloudWatchEvent) (string, error) {
	var out string

	kmsSvc := NewKMS()
	dynamodbSvc := NewDynamoDB()

	// create a new policy key
	policyKeyRes, err := kmsSvc.CreateKey(&kms.CreateKeyInput{})
	if err != nil {
		return out, err
	}

	// generate the encrypted data key for the new policy key
	dataKeyInput := &kms.GenerateDataKeyInput{
		KeyId:   aws.String(*policyKeyRes.KeyMetadata.KeyId),
		KeySpec: aws.String("AES_256"),
	}
	dataKeyRes, err := kmsSvc.GenerateDataKey(dataKeyInput)
	if err != nil {
		return out, err
	}

	// save policy reference in database
	policyKeyInput := &dynamodb.PutItemInput{
		TableName: aws.String("PolicyKeys"),
		Item: map[string]*dynamodb.AttributeValue{
			"PolicyKeyID": {
				S: aws.String(*policyKeyRes.KeyMetadata.KeyId),
			},
			"DataKey": {
				B: dataKeyRes.CiphertextBlob,
			},
			"CreatedAt": {
				S: aws.String(time.Now().UTC().Format(time.RFC3339)),
			},
		},
	}
	_, err = dynamodbSvc.PutItem(policyKeyInput)
	if err == nil {
		out = *policyKeyRes.KeyMetadata.KeyId
	}

	return out, err
}

func main() {
	lambda.Start(EvolvePolicyHandler)
}
