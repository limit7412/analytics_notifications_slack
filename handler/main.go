package main

import (
	"context"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/limit7412/analytics_notifications_slack/usecase"
)

// Response is of type APIGatewayProxyResponse since we're leveraging the
// AWS Lambda Proxy Request functionality (default behavior)
//
// https://serverless.com/framework/docs/providers/aws/events/apigateway/#lambda-proxy-integration
type Response events.CloudWatchEvent

// Handler is our lambda handler invoked by the `lambda.Start` function call
func Handler(ctx context.Context) (Response, error) {
	app := usecase.NewNotifyUsecase()
	err := app.Run()
	if err != nil {
		app.Error(err)
	}

	return Response{}, nil
}

func main() {
	lambda.Start(Handler)
}
