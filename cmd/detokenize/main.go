package main

import (
	"gopkg.in/aws/aws-lambda-go.v1/events"
	"gopkg.in/aws/aws-lambda-go.v1/lambda"
)

// DetokenizeHandler is An API Gateway handler that detokenizes a token into a credit card.
func DetokenizeHandler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// get token
	// get encryption key
	// detokenize

	return events.APIGatewayProxyResponse{
		Body:       "Detokenize",
		StatusCode: 200,
	}, nil
}

func main() {
	lambda.Start(DetokenizeHandler)
}
