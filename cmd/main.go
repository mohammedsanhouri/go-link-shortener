package main

import (
    "context"
    "github.com/aws/aws-lambda-go/events"
    "github.com/aws/aws-lambda-go/lambda"
)

// This function acts as the entry point for AWS Lambda
func Handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
    // Logic: 1. Parse JSON 2. Generate Hash 3. Save to DynamoDB
    return events.APIGatewayProxyResponse{
        StatusCode: 200,
        Body:       "Shortener API is Active",
    }, nil
}

func main() {
    lambda.Start(Handler)
}