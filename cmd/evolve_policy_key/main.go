package main

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/kms"
)

// Params are the parameters from the Lambda invokation.
type Params struct {
}

// EvolvePolicyHandler is a CloudWatch even handler that envolves the policy key.
func EvolvePolicyHandler(kmsSvc kms.Client, dynamodbSvc dynamodb.Client) func(params Params) (string, error) {
	return func(params Params) (string, error) {
		var out string

		// create a new policy key
		policyKeyRes, err := kmsSvc.CreateKey(context.TODO(), &kms.CreateKeyInput{})
		if err != nil {
			return out, err
		}

		// generate the encrypted data key for the new policy key
		dataKeyInput := &kms.GenerateDataKeyInput{
			KeyId:   aws.String(*policyKeyRes.KeyMetadata.KeyId),
			KeySpec: "AES_256",
		}
		dataKeyRes, err := kmsSvc.GenerateDataKey(context.TODO(), dataKeyInput)
		if err != nil {
			return out, err
		}

		// save policy reference in database
		policyKeyId, err := attributevalue.Marshal(*policyKeyRes.KeyMetadata.KeyId)
		if err != nil {
			panic(err)
		}
		dataKey, err := attributevalue.Marshal(dataKeyRes.CiphertextBlob)
		if err != nil {
			panic(err)
		}
		createdAt, err := attributevalue.Marshal(time.Now().UTC().Format(time.RFC3339))
		if err != nil {
			panic(err)
		}
		policyKeyInput := &dynamodb.PutItemInput{
			TableName: aws.String("PolicyKeys"),
			Item: map[string]types.AttributeValue{
				"PolicyKeyID": policyKeyId,
				"DataKey": dataKey,
				"CreatedAt": createdAt,
			},

		}
		_, err = dynamodbSvc.PutItem(context.TODO(), policyKeyInput)
		if err == nil {
			out = *policyKeyRes.KeyMetadata.KeyId
		}

		return out, err
	}
}

func main() {
	cfg := aws.Config{
		Region: os.Getenv("REGION"),
	}
	kmsSvc := kms.NewFromConfig(cfg)
	dynamodbSvc := dynamodb.NewFromConfig(cfg)

	lambda.Start(EvolvePolicyHandler(*kmsSvc, *dynamodbSvc))
}
