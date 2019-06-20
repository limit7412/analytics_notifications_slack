package adapter

import (
	"context"
	"os"

	"google.golang.org/api/analytics/v3"
)

type AnalyticsAdapter interface {
	GetSessions(start string, end string) (*analytics.GaData, error)
	GetService() (*analytics.Service, error)
}

type analyticsImpl struct {
}

// NewAnalyticsImplAdapter access to analytics
func NewAnalyticsImplAdapter() AnalyticsAdapter {
	return &analyticsImpl{}
}

func (a *analyticsImpl) GetSessions(start string, end string) (*analytics.GaData, error) {
	service, err := a.GetService()
	if err != nil {
		return nil, err
	}

	result, err := service.Data.Ga.
		Get("ga:"+os.Getenv("PROFILE_ID"), start, end, "ga:sessions").
		Dimensions("ga:pageTitle,ga:pagePath").
		Do()
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (a *analyticsImpl) GetService() (*analytics.Service, error) {
	ctx := context.Background()
	analyticsService, err := analytics.NewService(ctx)
	if err != nil {
		return nil, err
	}

	return analyticsService, nil
}
