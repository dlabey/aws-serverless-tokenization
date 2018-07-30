package main

import (
	"context"

	"gopkg.in/aws/aws-lambda-go.v1/lambda"
)

// Params are the parameters from the Lambda invokation.
type Params struct {
}

// Retokenize is used when rotating encryption keys or retrying failed rotating encryption key items.
func Retokenize(ctx context.Context, params Params) (string, error) {
}

func main() {
	lambda.Start(Retokenize)
}
