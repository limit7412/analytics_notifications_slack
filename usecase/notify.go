package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/limit7412/analytics_notifications_slack/repository"
)

type NotifyUsecase interface {
	Run(ctx context.Context) error
	Error(ctx context.Context, err error)
}

type notifyImpl struct {
	analytics repository.AnalyticsRepository
	slack     repository.SlackRepository
}

// NewNotifyUsecase notification analytics to slack
func NewNotifyUsecase(analytics repository.AnalyticsRepository, slack repository.SlackRepository) NotifyUsecase {
	return &notifyImpl{
		analytics: analytics,
		slack:     slack,
	}
}

func (n *notifyImpl) Run(ctx context.Context) error {
	now := time.Now()
	today := now.Format("2006-01-02")
	month := now.AddDate(0, 0, -(now.Day() - 1)).Format("2006-01-02")

	posts := []*repository.Post{
		{
			Fallback: os.Getenv("SUCCESS_FALLBACK"),
			Pretext:  os.Getenv("SUCCESS_FALLBACK"),
		},
	}

	ranges := []struct {
		title string
		color string
		start string
		end   string
	}{
		{"今日のpv数ランキング", "#4286f4", today, today},
		{"今月のpv数ランキング", "#dbe031", month, today},
		{"累計pv数ランキング", "#41a300", "2015-08-14", today},
	}

	for _, r := range ranges {
		data, err := n.analytics.GetSessions(ctx, r.start, r.end)
		if err != nil {
			return fmt.Errorf("get sessions (%s): %w", r.title, err)
		}
		posts = append(posts, n.createRankingData(r.title, r.color, data))
	}

	if err := n.slack.Post(ctx, os.Getenv("SUCCESS_WEBHOOK_URL"), posts); err != nil {
		return fmt.Errorf("post to slack: %w", err)
	}

	return nil
}

func (n *notifyImpl) Error(ctx context.Context, err error) {
	slog.ErrorContext(ctx, "notify failed", slog.Any("error", err))

	posts := []*repository.Post{
		{
			Fallback: os.Getenv("FAILD_FALLBACK"),
			Pretext:  "<@" + os.Getenv("SLACK_ID") + "> " + os.Getenv("FAILD_FALLBACK"),
			Title:    err.Error(),
			Color:    "#EB4646",
			Footer:   "analytics_notifications_slack",
		},
	}

	if postErr := n.slack.Post(ctx, os.Getenv("FAILD_WEBHOOK_URL"), posts); postErr != nil {
		slog.ErrorContext(ctx, "failed to post error to slack", slog.Any("error", postErr))
	}
}

func (n *notifyImpl) createRankingData(title string, color string, data []*repository.Page) *repository.Post {
	text := []string{}
	for i, item := range data {
		if i >= 5 {
			break
		}
		text = append(text, fmt.Sprintf("[%d] <https://%s|%s>: %dpv", i+1, item.Path, item.Title, item.PV))
	}

	return &repository.Post{
		Title: title,
		Text:  strings.Join(text, "\n"),
		Color: color,
	}
}
