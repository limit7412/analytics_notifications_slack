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

	titleSplit := os.Getenv("TITLE_SPLIT")
	pageMap := make(map[string]*Page)
	processed := 0
	for _, propertyId := range strings.Split(os.Getenv("PROPERTY_ID"), ",") {
		id := strings.TrimSpace(propertyId)
		if id == "" {
			continue
		}
		processed++

		data, err := a.service.Properties.RunReport("properties/"+id, runReportRequest).Context(ctx).Do()
		if err != nil {
			return nil, fmt.Errorf("run report for property %s: %w", id, err)
		}

		if err := aggregateRows(pageMap, data.Rows, titleSplit); err != nil {
			return nil, err
		}
	}

	// Treat a missing/empty PROPERTY_ID as a configuration error rather than
	// silently returning an empty (but "successful") ranking.
	if processed == 0 {
		return nil, fmt.Errorf("no valid PROPERTY_ID configured")
	}

	return sortPages(pageMap), nil
}

// aggregateRows folds report rows into pageMap, summing PV for duplicate titles.
// Top-level paths (a single "/") are skipped. Rows missing the expected
// dimensions or metrics are skipped defensively to avoid panics.
func aggregateRows(pageMap map[string]*Page, rows []*analytics.Row, titleSplit string) error {
	for _, row := range rows {
		if row == nil || len(row.DimensionValues) < 3 || len(row.MetricValues) < 1 {
			continue
		}

		pageTitle := row.DimensionValues[0].Value
		hostName := row.DimensionValues[1].Value
		pagePath := row.DimensionValues[2].Value
		screenPageViews := row.MetricValues[0].Value

		if strings.Count(pagePath, "/") == 1 {
			continue
		}

		title := pageTitle
		if titleSplit != "" {
			title = strings.Split(pageTitle, titleSplit)[0]
		}
		pv, err := strconv.Atoi(screenPageViews)
		if err != nil {
			return fmt.Errorf("parse screenPageViews %q: %w", screenPageViews, err)
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

	return nil
}

// sortPages returns the pages sorted by descending PV.
func sortPages(pageMap map[string]*Page) []*Page {
	result := slices.Collect(maps.Values(pageMap))
	slices.SortFunc(result, func(a, b *Page) int {
		return cmp.Compare(b.PV, a.PV)
	})

	return result
}
