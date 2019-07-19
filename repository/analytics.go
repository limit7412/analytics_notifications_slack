package repository

import (
	"context"
	"os"
	"sort"
	"strconv"
	"strings"

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
		Dimensions("ga:pageTitle,ga:hostname,ga:pagePath").
		Do()
	if err != nil {
		return nil, err
	}

	result := [][]string{}
	for _, line := range data.Rows {
		if strings.Count(line[2], "/") != 1 {
			result = append(result, line)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		a, _ := strconv.Atoi(result[i][3])
		b, _ := strconv.Atoi(result[j][3])
		return a > b
	})

	return result, nil
}
