package main

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/stretchr/testify/mock"
	"gopkg.in/aws/aws-lambda-go.v1/events"
)

type MockKMS struct {
	mock.Mock
}

func (mock *MockKMS) CreateKey(input *kms.CreateKeyInput) (*kms.CreateKeyOutput, error) {
	args := mock.Called(input)

	return args.Get(0).(*kms.CreateKeyOutput), args.Error(1)
}

func (mock *MockKMS) GenerateDataKey(input *kms.GenerateDataKeyInput) (*kms.GenerateDataKeyOutput, error) {
	args := mock.Called(input)

	return args.Get(0).(*kms.GenerateDataKeyOutput), args.Error(1)
}

type MockDynamoDb struct {
	mock.Mock
}

func (mock *MockDynamoDb) PutItem(input *dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error) {
	args := mock.Called(input)

	return args.Get(0).(*dynamodb.PutItemOutput), args.Error(1)
}

func TestHandler(test *testing.T) {
	kmsSvcMock := new(MockKMS)
	dynamodbSvcMock := new(MockDynamoDb)

	evt := events.CloudWatchEvent{}

	kmsSvcMock.On("CreateKey", mock.Anything).Return(&kms.CreateKeyOutput{
		KeyMetadata: &kms.KeyMetadata{
			KeyId: aws.String("test"),
		},
	}, nil)
	kmsSvcMock.On("GenerateDataKey", mock.Anything).Return(&kms.GenerateDataKeyOutput{
		CiphertextBlob: []byte("test"),
	}, nil)
	dynamodbSvcMock.On("PutItem", mock.Anything).Return(&dynamodb.PutItemOutput{}, nil)

	out, err := EvolvePolicyHandler(kmsSvcMock, dynamodbSvcMock)(evt)

	fmt.Println(out)
	fmt.Println(err)

	kmsSvcMock.AssertExpectations(test)
	dynamodbSvcMock.AssertExpectations(test)
}
