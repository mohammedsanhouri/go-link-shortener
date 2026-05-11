package main

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"os"
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
var tableName = os.Getenv("TABLE_NAME")

func init() {
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
	// CORS Headers - Set to "*" for now so you can test locally or from S3 later.
	// You can change this to your specific S3 URL once the bucket is created.
	headers := map[string]string{
		"Access-Control-Allow-Origin":  "*",
		"Access-Control-Allow-Methods": "POST, GET, OPTIONS",
		"Access-Control-Allow-Headers": "Content-Type",
	}

	// 1. HANDLE CORS PRE-FLIGHT (OPTIONS)
	if request.HTTPMethod == "OPTIONS" {
		return events.APIGatewayProxyResponse{
			StatusCode: 200,
			Headers:    headers,
			Body:       "",
		}, nil
	}

	// 2. PATH 1: Create a Short Link (POST)
	if request.HTTPMethod == "POST" {
		var item URLItem
		err := json.Unmarshal([]byte(request.Body), &item)
		if err != nil {
			return events.APIGatewayProxyResponse{StatusCode: 400, Body: "Invalid JSON", Headers: headers}, nil
		}

		item.ShortID = GenerateShortID(item.LongURL)

		av, _ := attributevalue.MarshalMap(item)
		_, err = dbClient.PutItem(ctx, &dynamodb.PutItemInput{
			TableName: aws.String(tableName),
			Item:      av,
		})

		if err != nil {
			return events.APIGatewayProxyResponse{StatusCode: 500, Body: "Database Error", Headers: headers}, nil
		}

		// Add JSON content type to our existing CORS headers
		headers["Content-Type"] = "application/json"

		return events.APIGatewayProxyResponse{
			StatusCode: 201,
			Headers:    headers,
			Body:       "{\"short_url\": \"" + item.ShortID + "\"}",
		}, nil
	}

	// 3. PATH 2: Redirect a User (GET)
	if request.HTTPMethod == "GET" {
		shortID := request.PathParameters["id"]

		result, err := dbClient.GetItem(ctx, &dynamodb.GetItemInput{
			TableName: aws.String(tableName),
			Key: map[string]types.AttributeValue{
				"ShortID": &types.AttributeValueMemberS{Value: shortID},
			},
		})

		if err != nil || result.Item == nil {
			return events.APIGatewayProxyResponse{StatusCode: 404, Body: "URL not found", Headers: headers}, nil
		}

		var foundItem URLItem
		attributevalue.UnmarshalMap(result.Item, &foundItem)

		// Overwrite headers for redirection while keeping CORS
		headers["Location"] = foundItem.LongURL

		return events.APIGatewayProxyResponse{
			StatusCode: 301,
			Headers:    headers,
		}, nil
	}

	return events.APIGatewayProxyResponse{StatusCode: 405, Body: "Method Not Allowed", Headers: headers}, nil
}

func main() {
	lambda.Start(Handler)
}
