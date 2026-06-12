package repository

import (
	"cmp"
	"context"
	"fmt"
	"maps"
	"os"
	"slices"
	"strconv"
	"strings"

	analytics "google.golang.org/api/analyticsdata/v1beta"
	"google.golang.org/api/option"
)

type AnalyticsRepository interface {
	GetSessions(ctx context.Context, start string, end string) ([]*Page, error)
}

type analyticsImpl struct {
	service *analytics.Service
}

// NewAnalyticsRepository access to analytics
func NewAnalyticsRepository(ctx context.Context) (AnalyticsRepository, error) {
	service, err := analytics.NewService(ctx, option.WithCredentialsFile("./secret.json"))
	if err != nil {
		return nil, fmt.Errorf("create analytics service: %w", err)
	}

	return &analyticsImpl{service: service}, nil
}

type Page struct {
	Title string
	Path  string
	PV    int
}

func (a *analyticsImpl) GetSessions(ctx context.Context, start string, end string) ([]*Page, error) {
	runReportRequest := &analytics.RunReportRequest{
		DateRanges: []*analytics.DateRange{
			{StartDate: start, EndDate: end},
		},
		Dimensions: []*analytics.Dimension{
			{Name: "pageTitle"},
			{Name: "hostName"},
			{Name: "pagePath"},
		},
		Metrics: []*analytics.Metric{
			{Name: "screenPageViews"},
		},
	}

	pageMap := make(map[string]*Page)
	for _, propertyId := range strings.Split(os.Getenv("PROPERTY_ID"), ",") {
		data, err := a.service.Properties.RunReport("properties/"+propertyId, runReportRequest).Context(ctx).Do()
		if err != nil {
			return nil, fmt.Errorf("run report for property %s: %w", propertyId, err)
		}

		for _, row := range data.Rows {
			pageTitle := row.DimensionValues[0].Value
			hostName := row.DimensionValues[1].Value
			pagePath := row.DimensionValues[2].Value
			screenPageViews := row.MetricValues[0].Value

			if strings.Count(pagePath, "/") == 1 {
				continue
			}

			title := strings.Split(pageTitle, os.Getenv("TITLE_SPLIT"))[0]
			pv, err := strconv.Atoi(screenPageViews)
			if err != nil {
				return nil, fmt.Errorf("parse screenPageViews %q: %w", screenPageViews, err)
			}
			if _, ok := pageMap[title]; ok {
				pageMap[title].PV += pv
			} else {
				pageMap[title] = &Page{
					Title: title,
					Path:  hostName + pagePath,
					PV:    pv,
				}
			}
		}
	}

	result := slices.Collect(maps.Values(pageMap))
	slices.SortFunc(result, func(a, b *Page) int {
		return cmp.Compare(b.PV, a.PV)
	})

	return result, nil
}
