# AWS Serverless Link Shortener (Golang)

## Project Goal
To build a high-scale, low-latency URL shortening service using a serverless architecture that minimizes costs and eliminates server management.

##  Tech Stack
- **Backend:** Go (Golang) 1.22+
- **Database:** Amazon DynamoDB (NoSQL)
- **Deployment:** GitHub Actions & Terraform
- **API Management:** Amazon API Gateway
- **CDN:** Amazon CloudFront

##  Architecture


1. User sends a URL to the API Gateway.
2. API Gateway triggers a **Go Lambda Function**.
3. Lambda generates a unique ID, stores it in **DynamoDB**, and returns the short link.
4. When a user clicks the short link, another Lambda retrieves the URL and performs a **301 Redirect**.