package main

import (
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

// TokenizeHandler is an API Gateway handler that tokenizes a credit card.
func TokenizeHandler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// get pan
	// tokenize
	// store in db
	// return token

	return events.APIGatewayProxyResponse{
		Body:       "Tokenize",
		StatusCode: 200,
	}, nil
}

func main() {
	lambda.Start(TokenizeHandler)
}
