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

	pageMap := make(map[string]*Page)
	for _, item := range data.Rows {
		if strings.Count(item[2], "/") != 1 {
			title := strings.Split(item[0], os.Getenv("TITLE_SPLIT"))[0]
			if _, ok := pageMap[title]; ok {
				pageMap[title].PV += item[3]
			} else {
				pageMap[title] = &Page{
					Title: title,
					Path:  item[1] + item[2],
					PV:    item[3],
				}
			}
		}
	}

	result := []*Page{}
	for _, item := range pageMap {
		result = append(result, item)
	}

	sort.Slice(result, func(i, j int) bool {
		a, _ := strconv.Atoi(result[i].PV)
		b, _ := strconv.Atoi(result[j].PV)
		return a > b
	})

	return result, nil
}
