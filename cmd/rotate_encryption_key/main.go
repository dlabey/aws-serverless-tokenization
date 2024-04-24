package main

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	kmsTypes "github.com/aws/aws-sdk-go-v2/service/kms/types"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
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

	cfg := aws.Config{
		Region: os.Getenv("REGION"),
	}
	dynamodbSvc := dynamodb.NewFromConfig(cfg)

	var encryptionKey EncryptionKey

	// convert IsReplace to bool
	isReplace, err := strconv.ParseBool(params.IsReplace)
	if err != nil {
		return out, err
	}

	// check if this is a replacement and ensure a valid EncryptionKeyID if so
	if isReplace {
		encryptionKeyID, err := attributevalue.Marshal(time.Now().UTC().Format(time.RFC3339))
		result, err := dynamodbSvc.GetItem(context.TODO(), &dynamodb.GetItemInput{
			TableName: aws.String("EncryptionKeys"),
			Key: map[string]dynamodbTypes.AttributeValue{
				"EncryptionKeyID": encryptionKeyID,
			},
		})
		if err != nil {
			return out, err
		}

		err = attributevalue.UnmarshalMap(result.Item, &encryptionKey)
		if err != nil {
			return out, err
		}
	}

	kmsSvc := kms.NewFromConfig(cfg)

	// create a new encryption key
	encryptionKeyRes, err := kmsSvc.CreateKey(context.TODO(), &kms.CreateKeyInput{})
	if err != nil {
		return out, err
	}

	// generate the encrypted data key for the new encryption key
	dataKeyInput := &kms.GenerateDataKeyInput{
		KeyId:   aws.String(*encryptionKeyRes.KeyMetadata.KeyId),
		KeySpec: kmsTypes.DataKeySpecAes256,
	}
	dataKeyRes, err := kmsSvc.GenerateDataKey(context.TODO(), dataKeyInput)
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
	encryptionKeyID, err := attributevalue.Marshal(*encryptionKeyRes.KeyMetadata.KeyId)
	if err != nil {
		return out, err
	}
	dataKey, err := attributevalue.Marshal(dataKeyRes.CiphertextBlob)
	if err != nil {
		return out, err
	}
	isCurrentVal, err := attributevalue.Marshal(isCurrent)
	if err != nil {
		return out, err
	}
	rotatedAt, err := attributevalue.Marshal(time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		return out, err
	}

	_, err = dynamodbSvc.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: aws.String("EncryptionKeys"),
		Item: map[string]dynamodbTypes.AttributeValue{
			"EncryptionKeyID": encryptionKeyID,
			"DataKey": dataKey,
			"IsCurrent": isCurrentVal,
			"RotatedAt": rotatedAt,
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

		scanPaginator := dynamodb.NewScanPaginator(dynamodbSvc, &dynamodb.ScanInput{
			ExpressionAttributeNames:  expr.Names(),
			ExpressionAttributeValues: expr.Values(),
			FilterExpression:          expr.Filter(),
			ProjectionExpression:      expr.Projection(),
			TableName:                 aws.String("EncryptionKeys"),
		})
		for scanPaginator.HasMorePages() {
			scanOut, err := scanPaginator.NextPage(context.TODO())
			if err != nil {
				return out, err
			}

			for _, i := range scanOut.Items {
				token := Token{}
				err = attributevalue.UnmarshalMap(i, &token)
				if err != nil {
					scanErr = err
					break
				}

				// put Token into SQS/Lambda for update and ingestion
				// put Token and EncryptionKeyID
			}
			if scanErr != nil {
				break
			}
		}
		if err != nil {
			return out, err
		}
		if scanErr != nil {
			return out, scanErr
		}

		// send the ids of the tokens that need to be rotated to an sns topic
		// subscribe the lambda to the sns topic to rotate the keys
		// any failed notify another sns topic if retry does not work
		// put any that did not work into dead letter sns, keep track of retry on token record, reset if successful
		// the retries go to a sqs which is polled for dead letter queue
	}

	return out, err
}

func main() {
	lambda.Start(RotateEncryptionKeyHandler)
}
