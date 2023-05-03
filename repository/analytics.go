package repository

import (
	"context"
	"os"
	"sort"
	"strconv"
	"strings"

	analytics "google.golang.org/api/analyticsdata/v1beta"
	"google.golang.org/api/option"
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
	PV    int
}

func (a *analyticsImpl) getService() (*analytics.Service, error) {
	ctx := context.Background()
	analyticsService, err := analytics.NewService(ctx, option.WithCredentialsFile("./credentials.json"))
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

	runReportRequest := &analytics.RunReportRequest{
		DateRanges: []*analytics.DateRange{
			{StartDate: start, EndDate: end},
		},
		Dimensions: []*analytics.Dimension{
			{Name: "pageTitle"},
			{Name: "hostname"},
			{Name: "pagePath"},
		},
		// Metrics: []*analytics.Metric{
		// 	{Name: "active1DayUsers"},
		// 	{Name: "active7DayUsers"},
		// 	{Name: "active28DayUsers"},
		// },
	}

	// data, err := service.Data.Ga.
	// 	Get("ga:"+os.Getenv("PROFILE_ID"), start, end, "ga:sessions").
	// 	Dimensions("ga:pageTitle,ga:hostname,ga:pagePath").
	// 	Do()
	// if err != nil {
	// 	return nil, err
	// }

	pageMap := make(map[string]*Page)
	for _, propertyId := range strings.Split(os.Getenv("PROPERTY_ID"), ",") {
		data, err := service.Properties.RunReport("properties/"+propertyId, runReportRequest).Do()
		if err != nil {
			return nil, err
		}

		for _, item := range data.Rows {
			if strings.Count(item[2], "/") != 1 {
				title := strings.Split(item[0], os.Getenv("TITLE_SPLIT"))[0]
				pv, err := strconv.Atoi(item[3])
				if err != nil {
					return nil, err
				}
				if _, ok := pageMap[title]; ok {
					pageMap[title].PV += pv
				} else {
					pageMap[title] = &Page{
						Title: title,
						Path:  item[1] + item[2],
						PV:    pv,
					}
				}
			}
		}
	}

	result := []*Page{}
	for _, item := range pageMap {
		result = append(result, item)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].PV > result[j].PV
	})

	return result, nil
}
