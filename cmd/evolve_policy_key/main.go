package main

import (
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/kms"
	"gopkg.in/aws/aws-lambda-go.v1/events"
	"gopkg.in/aws/aws-lambda-go.v1/lambda"
)

// KMSAPI is a subset of kmsiface.KMSAPI
type KMSAPI interface {
	CreateKey(input *kms.CreateKeyInput) (*kms.CreateKeyOutput, error)
	GenerateDataKey(input *kms.GenerateDataKeyInput) (*kms.GenerateDataKeyOutput, error)
}

// DynamoDBAPI is a subset of dynamodbiface.DynamoDBAPI
type DynamoDBAPI interface {
	PutItem(input *dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error)
}

// EvolvePolicyHandler is a CloudWatch even handler that envolves the policy key.
func EvolvePolicyHandler(kmsSvc KMSAPI, dynamodbSvc DynamoDBAPI) func(event events.CloudWatchEvent) (string, error) {
	return func(event events.CloudWatchEvent) (string, error) {
		var out string

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
}

func main() {
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(os.Getenv("REGION")),
	}))

	kmsSvc := kms.New(sess)
	dynamodbSvc := dynamodb.New(sess)

	lambda.Start(EvolvePolicyHandler(kmsSvc, dynamodbSvc))
}
