package main

import (
	"gopkg.in/aws/aws-lambda-go.v1/events"
	"gopkg.in/aws/aws-lambda-go.v1/lambda"
)

// TokenizeHandler is an API Gateway handler that tokenizes a credit card.
func TokenizeHandler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return events.APIGatewayProxyResponse{
		Body:       "Tokenize",
		StatusCode: 200,
	}, nil
}

func main() {
	lambda.Start(TokenizeHandler)
}
