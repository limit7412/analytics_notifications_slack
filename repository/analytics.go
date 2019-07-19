package repository

import (
	"context"
	"os"
	"sort"
	"strconv"

	"google.golang.org/api/analytics/v3"
)

type AnalyticsRepository interface {
	GetSessions(start string, end string) ([][]string, error)
}

type analyticsImpl struct {
}

// NewAnalyticsRepository access to analytics
func NewAnalyticsRepository() AnalyticsRepository {
	return &analyticsImpl{}
}

func (a *analyticsImpl) getService() (*analytics.Service, error) {
	ctx := context.Background()
	analyticsService, err := analytics.NewService(ctx)
	if err != nil {
		return nil, err
	}

	return analyticsService, nil
}

func (a *analyticsImpl) GetSessions(start string, end string) ([][]string, error) {
	service, err := a.getService()
	if err != nil {
		return nil, err
	}

	data, err := service.Data.Ga.
		Get("ga:"+os.Getenv("PROFILE_ID"), start, end, "ga:sessions").
		Dimensions("ga:pageTitle,ga:pagePath").
		Do()
	if err != nil {
		return nil, err
	}

	result := data.Rows
	sort.Slice(result, func(i, j int) bool {
		a, _ := strconv.Atoi(result[i][2])
		b, _ := strconv.Atoi(result[j][2])
		return a > b
	})

	return result, nil
}
