package main

import (
	"testing"

	"github.com/stretchr/testify/mock"
	"gopkg.in/aws/aws-lambda-go.v1/events"
)

type Mock struct {
	mock.Mock
}

func TestHandler(t *testing.T) {
	sessionNew = func() (*Session, error) { return new(Mock) }
	kmsNew = func() *KMS { return new(Mock) }
	dynamoNew = func *DynamoDB { return new(Mock) }

	evt := events.CloudWatchEvent{}

	EvolvePolicyHandler(evt)
}
