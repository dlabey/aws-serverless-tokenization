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

// EvolvePolicyHandler is a CloudWatch even handler that envolves the policy key.
func EvolvePolicyHandler(event events.CloudWatchEvent) (string, error) {
	var out string

	config := &aws.Config{
		Region: aws.String(os.Getenv("REGION")),
	}
	sess, err := session.NewSession(config)
	if err != nil {
		return out, err
	}
	kmsSvc := kms.New(sess)
	dynamodbSvc := dynamodb.New(sess)

	// create a new policy key
	policyKeyRes, err := kmsSvc.CreateKey(&kms.CreateKeyInput{})
	if err != nil {
		return out, err
	}

	// generate the encrypted data key for the new policy key
	dataKeyInput := &kms.GenerateDataKeyInput{
		KeyId: aws.String(*policyKeyRes.KeyMetadata.KeyId),
	}
	dataKeyRes, err := kmsSvc.GenerateDataKey(dataKeyInput)
	if err != nil {
		return out, err
	}

	// save policy reference in database
	policyKeyInput := &dynamodb.PutItemInput{
		TableName: aws.String("PolicyKeys"),
		Item: map[string]*dynamodb.AttributeValue{
			"PolicyKeyId": {
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