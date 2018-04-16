package main

import (
	"gopkg.in/aws/aws-lambda-go.v1/events"
	"gopkg.in/aws/aws-lambda-go.v1/lambda"
)

func Handler(event events.CloudWatchEvent) error {
	return nil
}

func main() {
	lambda.Start(Handler)
}
