package main

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// URLItem defines how our data looks in Go and DynamoDB
type URLItem struct {
	ShortID string `json:"short_id" dynamodbav:"ShortID"`
	LongURL string `json:"long_url"  dynamodbav:"LongURL"`
}

var dbClient *dynamodb.Client

const tableName = "ShortUrls"

func init() {
	// init() runs once when the Lambda container starts (Optimization)
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		panic("Configuration error: " + err.Error())
	}
	dbClient = dynamodb.NewFromConfig(cfg)
}

func GenerateShortID(longURL string) string {
	hash := sha256.Sum256([]byte(longURL))
	encoded := base64.URLEncoding.EncodeToString(hash[:6])
	return strings.ReplaceAll(encoded, "_", "a")
}

func Handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// PATH 1: Create a Short Link (POST)
	if request.HTTPMethod == "POST" {
		var item URLItem
		err := json.Unmarshal([]byte(request.Body), &item)
		if err != nil {
			return events.APIGatewayProxyResponse{StatusCode: 400, Body: "Invalid JSON"}, nil
		}

		item.ShortID = GenerateShortID(item.LongURL)

		// Save to DynamoDB
		av, _ := attributevalue.MarshalMap(item)
		_, err = dbClient.PutItem(ctx, &dynamodb.PutItemInput{
			TableName: aws.String(tableName),
			Item:      av,
		})

		if err != nil {
			return events.APIGatewayProxyResponse{StatusCode: 500, Body: "Database Error"}, nil
		}

		return events.APIGatewayProxyResponse{
			StatusCode: 201,
			Headers:    map[string]string{"Content-Type": "application/json"},
			Body:       "{\"short_url\": \"" + item.ShortID + "\"}",
		}, nil
	}

	// PATH 2: Redirect a User (GET)
	if request.HTTPMethod == "GET" {
		// API Gateway passes the {id} in PathParameters
		shortID := request.PathParameters["id"]

		result, err := dbClient.GetItem(ctx, &dynamodb.GetItemInput{
			TableName: aws.String(tableName),
			Key: map[string]types.AttributeValue{
				"ShortID": &types.AttributeValueMemberS{Value: shortID},
			},
		})

		if err != nil || result.Item == nil {
			return events.APIGatewayProxyResponse{StatusCode: 404, Body: "URL not found"}, nil
		}

		var foundItem URLItem
		attributevalue.UnmarshalMap(result.Item, &foundItem)

		// This is the "Magic" - HTTP 301 tells the browser to go elsewhere
		return events.APIGatewayProxyResponse{
			StatusCode: 301,
			Headers:    map[string]string{"Location": foundItem.LongURL},
		}, nil
	}

	return events.APIGatewayProxyResponse{StatusCode: 405, Body: "Method Not Allowed"}, nil
}

func main() {
	lambda.Start(Handler)
}
