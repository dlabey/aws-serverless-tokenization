package main

import (
	"encoding/json"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/kms"
	"gopkg.in/aws/aws-lambda-go.v1/events"
	"gopkg.in/aws/aws-lambda-go.v1/lambda"
)

// Detail is the contents of the CloudWatch event constant.
type Detail struct {
	EncryptionKeyID string
}

// EncryptionKey is the encryption key from DynamoDB.
type EncryptionKey struct {
	EncryptionKeyID string
	DataKey         []byte
	IsCurrent       bool
	RotatedAt       string
}

// Token is the token from DynamoDB.
type Token struct {
	Token           string
	PolicyKeyId     string
	EncryptionKeyId string
	Pan             string
}

// RotateEncryptionKeyHandler is a CloudWatch Event handler that rotates the encryption key.
func RotateEncryptionKeyHandler(event events.CloudWatchEvent) (string, error) {
	var out string

	config := &aws.Config{
		Region: aws.String(os.Getenv("REGION")),
	}
	sess, err := session.NewSession(config)
	if err != nil {
		return out, err
	}
	dynamodbSvc := dynamodb.New(sess)

	// TODO: check if there is a replace constant
	// TODO: if there is a replace constant ensure there is a valid EncryptionKeyId to replace
	// TODO: if there is a valid EncryptionKeyId to replace continue
	var isReplace bool
	var detail Detail
	var encryptionKey EncryptionKey

	if isReplace {
		err := json.Unmarshal(event.Detail, &detail)
		if err != nil {
			return out, err
		}

		result, err := dynamodbSvc.GetItem(&dynamodb.GetItemInput{
			TableName: aws.String("EncryptionKeys"),
			Key: map[string]*dynamodb.AttributeValue{
				"EncryptionKeyID": {
					S: aws.String(detail.EncryptionKeyID),
				},
			},
		})
		if err != nil {
			return out, err
		}

		err = dynamodbattribute.UnmarshalMap(result.Item, &encryptionKey)
		if err != nil {
			return out, err
		}
	}

	kmsSvc := kms.New(sess)

	// create a new encryption key
	encryptionKeyRes, err := kmsSvc.CreateKey(&kms.CreateKeyInput{})
	if err != nil {
		return out, err
	}

	// generate the encrypted data key for the new encryption key
	dataKeyInput := &kms.GenerateDataKeyInput{
		KeyId: aws.String(*encryptionKeyRes.KeyMetadata.KeyId),
	}
	dataKeyRes, err := kmsSvc.GenerateDataKey(dataKeyInput)
	if err != nil {
		return out, err
	}

	var isCurrent bool

	// if this is a replace set to is current based on valid EncryptionKeyId
	if isReplace {
		isCurrent = encryptionKey.IsCurrent
	} else {
		isCurrent = true
	}

	// save encryption key reference in database
	_, err = dynamodbSvc.PutItem(&dynamodb.PutItemInput{
		TableName: aws.String("EncryptionKeys"),
		Item: map[string]*dynamodb.AttributeValue{
			"EncryptionKeyId": {
				S: aws.String(*encryptionKeyRes.KeyMetadata.KeyId),
			},
			"DataKey": {
				B: dataKeyRes.CiphertextBlob,
			},
			"IsCurrent": {
				BOOL: aws.Bool(isCurrent),
			},
			"RotatedAt": {
				S: aws.String(time.Now().UTC().Format(time.RFC3339)),
			},
		},
	})
	if err == nil {
		out = *encryptionKeyRes.KeyMetadata.KeyId
	}

	// if this is a replace do the batch update
	if isReplace == true {
		// get all the IDs of the Tokens to have their EncryptionKeyId replaced and Pan encrypted pan updated
		filt := expression.Name("EncryptionKeyId").Equal(expression.Value(detail.EncryptionKeyID))
		proj := expression.NamesList(expression.Name("Token"))
		expr, err := expression.NewBuilder().WithFilter(filt).WithProjection(proj).Build()
		if err != nil {
			return out, err
		}

		var scanErr error

		err = dynamodbSvc.ScanPages(&dynamodb.ScanInput{
			ExpressionAttributeNames:  expr.Names(),
			ExpressionAttributeValues: expr.Values(),
			FilterExpression:          expr.Filter(),
			ProjectionExpression:      expr.Projection(),
			TableName:                 aws.String("EncryptionKeys"),
		},
			func(result dynamodb.ScanOutput, isLastPage bool) bool {
				for _, i := range result.Items {
					token := Token{}
					err = dynamodbattribute.UnmarshalMap(i, &token)
					if err != nil {
						scanErr = err
						break
					}

					// put Token into SQS/Lambda for update and ingestion
					// put Token and EncryptionKeyID
				}
				if scanErr != nil {
					return false
				}

				return !isLastPage
			})
		if err != nil {
			return out, err
		}
		if scanErr != nil {
			return out, scanErr
			1
		}

		// start a lambda step function state machine that batch updates the tokens
		// which match the validEncryptionKeyId to be replaced
		// * batch update the records with parallel lambdas
		// * if all are successful good
		// * if any fail send notification to SNS topic
		// * prior key is kept around if any fail, otherwise deleted
	}

	return out, err
}

func main() {
	lambda.Start(RotateEncryptionKeyHandler)
}
