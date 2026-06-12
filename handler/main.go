package main

import (
	"context"
	"log/slog"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/limit7412/analytics_notifications_slack/repository"
	"github.com/limit7412/analytics_notifications_slack/usecase"
)

// Handler is our lambda handler invoked by the `lambda.Start` function call.
// It is triggered on a schedule (EventBridge), so it takes no input payload.
func Handler(ctx context.Context) error {
	slack := repository.NewSlackRepository()

	analytics, err := repository.NewAnalyticsRepository(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to init analytics repository", slog.Any("error", err))
		return err
	}

	app := usecase.NewNotifyUsecase(analytics, slack)
	if err := app.Run(ctx); err != nil {
		app.Error(ctx, err)
		return err
	}

	return nil
}

func main() {
	lambda.Start(Handler)
}
