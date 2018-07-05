package main

import (
	"context"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
	"github.com/aws/aws-sdk-go/service/kms"
	"gopkg.in/aws/aws-lambda-go.v1/lambda"
)

// Params are the parameters from the Lambda invokation.
type Params struct {
	IsReplace       string `json:"is_replace"`
	EncryptionKeyID string `json:"encryption_key_id"`
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
	PolicyKeyID     string
	EncryptionKeyID string
	Pan             string
}

// RotateEncryptionKeyHandler is a CloudWatch Event handler that rotates the encryption key.
func RotateEncryptionKeyHandler(ctx context.Context, params Params) (string, error) {
	var out string

	config := &aws.Config{
		Region: aws.String(os.Getenv("REGION")),
	}
	sess, err := session.NewSession(config)
	if err != nil {
		return out, err
	}
	dynamodbSvc := dynamodb.New(sess)

	var encryptionKey EncryptionKey

	// convert IsReplace to bool
	isReplace, err := strconv.ParseBool(params.IsReplace)
	if err != nil {
		return out, err
	}

	// check if this is a replacement and ensure a valid EncryptionKeyID if so
	if isReplace {
		result, err := dynamodbSvc.GetItem(&dynamodb.GetItemInput{
			TableName: aws.String("EncryptionKeys"),
			Key: map[string]*dynamodb.AttributeValue{
				"EncryptionKeyID": {
					S: aws.String(params.EncryptionKeyID),
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
		KeyId:   aws.String(*encryptionKeyRes.KeyMetadata.KeyId),
		KeySpec: aws.String("AES_256"),
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
			"EncryptionKeyID": {
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
	if isReplace {
		// get all the IDs of the Tokens to have their EncryptionKeyId replaced and Pan encrypted pan updated
		filt := expression.Name("EncryptionKeyId").Equal(expression.Value(params.EncryptionKeyID))
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
		}, func(result *dynamodb.ScanOutput, isLastPage bool) bool {
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
