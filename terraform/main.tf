# 1. Define the DynamoDB Table
resource "aws_dynamodb_table" "url_table" {
  name           = "ShortUrls"
  billing_mode   = "PAY_PER_REQUEST" # You only pay when you use it (Free Tier)
  hash_key       = "ShortID"

  attribute {
    name = "ShortID"
    type = "S" # S stands for String
  }
}

# 2. Define the IAM Role (Permissions)
# This gives your Go code permission to "talk" to the database
resource "aws_iam_role" "lambda_role" {
  name = "url_shortener_role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action = "sts:AssumeRole"
      Effect = "Allow"
      Principal = { Service = "lambda.amazonaws.com" }
    }]
  })
}

# 3. Attach the DynamoDB Policy to the Role
resource "aws_iam_role_policy" "dynamo_policy" {
  name = "lambda_dynamo_policy"
  role = aws_iam_role.lambda_role.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action   = ["dynamodb:PutItem", "dynamodb:GetItem"]
      Effect   = "Allow"
      Resource = aws_dynamodb_table.url_table.arn
    }]
  })
}

resource "aws_lambda_function" "url_shortener" {
  filename      = "../lambda.zip"      # Path to the zip file you created
  function_name = "URLShortener"
  role          = aws_iam_role.lambda_role.arn
  handler       = "bootstrap"           # Required for Go in the latest runtime
  runtime       = "provided.al2023"     # The fastest runtime for Go

  # This tells AWS to look for the "bootstrap" file inside your zip
  publish = true 

  environment {
    variables = {
      TABLE_NAME = aws_dynamodb_table.url_table.name
    }
  }
}

# The "Entrance" to your Microservice
resource "aws_apigatewayv2_api" "lambda_api" {
  name          = "v2-url-shortener-api"
  protocol_type = "HTTP"
}

resource "aws_apigatewayv2_stage" "lambda_stage" {
  api_id      = aws_apigatewayv2_api.lambda_api.id
  name        = "$default"
  auto_deploy = true
}

resource "aws_apigatewayv2_integration" "lambda_integration" {
  api_id           = aws_apigatewayv2_api.lambda_api.id
  integration_type = "AWS_PROXY"
  integration_uri  = aws_lambda_function.url_shortener.invoke_arn
}

# The Routes (GET and POST)
resource "aws_apigatewayv2_route" "post_route" {
  api_id    = aws_apigatewayv2_api.lambda_api.id
  route_key = "POST /shorten"
  target    = "integrations/${aws_apigatewayv2_integration.lambda_integration.id}"
}

resource "aws_apigatewayv2_route" "get_route" {
  api_id    = aws_apigatewayv2_api.lambda_api.id
  route_key = "GET /{id}"
  target    = "integrations/${aws_apigatewayv2_integration.lambda_integration.id}"
}

# Permission for the API to talk to your Lambda
resource "aws_lambda_permission" "api_gw" {
  statement_id  = "AllowExecutionFromAPIGateway"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.url_shortener.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_apigatewayv2_api.lambda_api.execution_arn}/*/*"
}

# This will print your URL in the terminal!
output "base_url" {
  value = aws_apigatewayv2_api.lambda_api.api_endpoint
}

# 4. Allow Lambda to write logs (Crucial for debugging)
resource "aws_iam_role_policy_attachment" "lambda_logs" {
  role       = aws_iam_role.lambda_role.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}