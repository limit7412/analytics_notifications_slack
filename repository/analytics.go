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
	GetSessions(start string, end string) ([]*Page, error)
}

type analyticsImpl struct {
}

// NewAnalyticsRepository access to analytics
func NewAnalyticsRepository() AnalyticsRepository {
	return &analyticsImpl{}
}

type Page struct {
	Title string
	Path  string
	PV    string
}

func (a *analyticsImpl) getService() (*analytics.Service, error) {
	ctx := context.Background()
	analyticsService, err := analytics.NewService(ctx)
	if err != nil {
		return nil, err
	}

	return analyticsService, nil
}

func (a *analyticsImpl) GetSessions(start string, end string) ([]*Page, error) {
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

	result := []*Page{}
	for _, line := range data.Rows {
		if strings.Count(line[2], "/") != 1 {
			page := &Page{
				Title: line[0],
				Path:  line[1] + line[2],
				PV:    line[3],
			}
			result = append(result, page)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		a, _ := strconv.Atoi(result[i].PV)
		b, _ := strconv.Atoi(result[j].PV)
		return a > b
	})

	return result, nil
}
