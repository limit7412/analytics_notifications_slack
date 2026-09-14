package analytics

import (
	"cmp"
	"context"
	"fmt"
	"maps"
	"os"
	"slices"
	"strconv"
	"strings"

	analyticsdata "google.golang.org/api/analyticsdata/v1beta"
	"google.golang.org/api/option"
)

type Repository interface {
	GetSessions(ctx context.Context, start string, end string) ([]*Page, error)
}

type repositoryImpl struct {
	service *analyticsdata.Service
}

// NewRepository は Google アナリティクスへアクセスするリポジトリを生成する
func NewRepository(ctx context.Context) (Repository, error) {
	service, err := analyticsdata.NewService(ctx, option.WithCredentialsFile("./secret.json"))
	if err != nil {
		return nil, fmt.Errorf("create analytics service: %w", err)
	}

	return &repositoryImpl{service: service}, nil
}

type Page struct {
	Title string
	Path  string
	PV    int
}

func (a *repositoryImpl) GetSessions(ctx context.Context, start string, end string) ([]*Page, error) {
	runReportRequest := &analyticsdata.RunReportRequest{
		DateRanges: []*analyticsdata.DateRange{
			{StartDate: start, EndDate: end},
		},
		Dimensions: []*analyticsdata.Dimension{
			{Name: "pageTitle"},
			{Name: "hostName"},
			{Name: "pagePath"},
		},
		Metrics: []*analyticsdata.Metric{
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

	// PROPERTY_ID が未設定/空の場合、空の(だが"成功"扱いの)ランキングを黙って
	// 返すのではなく、設定不備のエラーとして扱う。
	if processed == 0 {
		return nil, fmt.Errorf("no valid PROPERTY_ID configured")
	}

	return sortPages(pageMap), nil
}

// aggregateRows はレポートの行を pageMap に集約し、同一タイトルの PV を合算する。
// トップレベルのパス(スラッシュ1つ)はスキップする。想定するディメンション/
// メトリクスを欠く行は、panic を避けるため防御的にスキップする。
func aggregateRows(pageMap map[string]*Page, rows []*analyticsdata.Row, titleSplit string) error {
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

// sortPages はページを PV の降順に並べて返す。
func sortPages(pageMap map[string]*Page) []*Page {
	result := slices.Collect(maps.Values(pageMap))
	slices.SortFunc(result, func(a, b *Page) int {
		return cmp.Compare(b.PV, a.PV)
	})

	return result
}
