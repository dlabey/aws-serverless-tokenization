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

func Handler(event events.CloudWatchEvent) (string, error) {
	// TODO: check if there is a replace constant
	// TODO: if there is a replace constant ensure there is a valid EncryptionKeyId to replace
	// TODO: if there is a valid EncryptionKeyId to replace continue
	var isReplace bool
	var encryptionKeyId string

	if isReplace {
		// TODO: get and verify record to be replaced
	}

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
	// if this is a replace set to is current based on validEncryptionKeyId
	if isReplace {
		// TODO: set based on current state of validEncryptionKeyId
		isCurrent = false
	} else {
		isCurrent = true
	}

	// save encryption key reference in database
	// if this is a replace do not set IsCurrent to true right away
	encryptionKeyInput := &dynamodb.PutItemInput{
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
	}
	_, err = dynamodbSvc.PutItem(encryptionKeyInput)
	if err == nil {
		out = *encryptionKeyRes.KeyMetadata.KeyId
	}

	// if this is a replace do the batch update
	if isReplace == true {
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
	lambda.Start(Handler)
}
