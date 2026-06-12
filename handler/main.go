package main

import (
	"context"
	"fmt"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/limit7412/analytics_notifications_slack/repository"
	"github.com/limit7412/analytics_notifications_slack/usecase"
)

// Repositories are initialised lazily and cached across warm Lambda
// invocations to avoid re-establishing connections/credentials every call.
var (
	slackRepo     repository.SlackRepository
	analyticsRepo repository.AnalyticsRepository
)

// Handler is our lambda handler invoked by the `lambda.Start` function call.
// It is triggered on a schedule (EventBridge), so it takes no input payload.
func Handler(ctx context.Context) error {
	if slackRepo == nil {
		slackRepo = repository.NewSlackRepository()
	}

	if analyticsRepo == nil {
		// Use a background context so the cached service is not bound to a
		// single invocation's (cancellable) context.
		repo, err := repository.NewAnalyticsRepository(context.Background())
		if err != nil {
			err = fmt.Errorf("init analytics repository: %w", err)
			// slackRepo is already available, so still surface the failure.
			usecase.NewNotifyUsecase(nil, slackRepo).Error(ctx, err)
			return err
		}
		analyticsRepo = repo
	}

	app := usecase.NewNotifyUsecase(analyticsRepo, slackRepo)
	if err := app.Run(ctx); err != nil {
		app.Error(ctx, err)
		return err
	}

	return nil
}

func main() {
	lambda.Start(Handler)
}
