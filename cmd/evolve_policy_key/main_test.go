package main

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/stretchr/testify/mock"
	"gopkg.in/aws/aws-lambda-go.v1/events"
)

type MockKMS struct {
	mock.Mock
}

var createKeyOutput = &kms.CreateKeyOutput{}

func (mock *MockKMS) CreateKey(input *kms.CreateKeyInput) (*kms.CreateKeyOutput, error) {
	return createKeyOutput, nil
}

var generateDataKeyOutput = &kms.GenerateDataKeyOutput{}

func (mock *MockKMS) GenerateDataKey(input *kms.GenerateDataKeyInput) (*kms.GenerateDataKeyOutput, error) {
	return generateDataKeyOutput, nil
}

type MockDynamoDb struct {
	mock.Mock
}

var putItemOutput = &dynamodb.PutItemOutput{}

func (mock *MockDynamoDb) PutItem(input *dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error) {
	return putItemOutput, nil
}

func TestHandler(test *testing.T) {
	oldKMS := KMS
	oldDynamoDB := DynamoDB

	defer func() { KMS = oldKMS }()
	defer func() { DynamoDB = oldDynamoDB }()

	KMS := new(MockKMS)
	DynamoDB := new(MockDynamoDb)

	evt := events.CloudWatchEvent{}

	KMS.On("CreateKey", mock.Anything).Return(createKeyOutput, nil)
	KMS.On("GenerateDataKey", mock.Anything).Return(generateDataKeyOutput, nil)
	DynamoDB.On("PutItem", mock.Anything).Return(putItemOutput, nil)

	out, err := EvolvePolicyHandler(evt)

	fmt.Println(out)

	fmt.Println(err)

	KMS.AssertExpectations(test)
	DynamoDB.AssertExpectations(test)
}
