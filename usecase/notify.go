package usecase

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/limit7412/analytics_notifications_slack/adapter"
	"google.golang.org/api/analytics/v3"
)

type NotifyUsecase interface {
	Run() error
	Error(err error)
	GetTodaySessions() (*analytics.GaData, error)
	CreateRankingData(title string, color string, data *analytics.GaData) adapter.Post
	PostToSlack(post []adapter.Post) error
}

type notifyImpl struct {
}

// NewNotifyUsecase notification analytics to slack
func NewNotifyUsecase() NotifyUsecase {
	return &notifyImpl{}
}

func (n *notifyImpl) Run() error {
	post := []adapter.Post{}
	post = append(post, adapter.Post{
		Fallback: os.Getenv("SUCCESS_FALLBACK"),
		Pretext:  os.Getenv("SUCCESS_FALLBACK"),
	})

	data, err := n.GetTodaySessions()
	if err != nil {
		return err
	}
	line := n.CreateRankingData("今日のpv数ランキング", "#4286f4", data)
	post = append(post, line)

	err = n.PostToSlack(post)
	if err != nil {
		return err
	}

	return nil
}

func (n *notifyImpl) Error(err error) {
	slack := adapter.NewSlackAdapter()

	post := []adapter.Post{}
	post = append(post, adapter.Post{
		Text:   err.Error(),
		Color:  "#EB4646",
		Footer: "analytics_notifications_slack",
	})

	_ = slack.Post(post)
	fmt.Print(err)
}

func (n *notifyImpl) GetTodaySessions() (*analytics.GaData, error) {
	adp := adapter.NewAnalyticsAdapter()
	result, err := adp.GetSessions("today", "today")
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (n *notifyImpl) CreateRankingData(title string, color string, data *analytics.GaData) adapter.Post {
	rows := data.Rows
	sort.Slice(rows, func(i, j int) bool {
		a, _ := strconv.Atoi(rows[i][2])
		b, _ := strconv.Atoi(rows[j][2])
		return a > b
	})

	text := []string{}
	for i, line := range rows {
		if i >= 5 {
			break
		}
		text = append(text, fmt.Sprintf("[%d] %s: %spv", i+1, line[0], line[2]))
	}
	post := adapter.Post{
		Title: title,
		Text:  strings.Join(text, "\n"),
		Color: color,
	}

	return post
}

func (n *notifyImpl) PostToSlack(post []adapter.Post) error {
	slack := adapter.NewSlackAdapter()
	err := slack.Post(post)
	if err != nil {
		return err
	}

	return nil
}
