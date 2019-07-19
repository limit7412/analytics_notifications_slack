package usecase

import (
	"fmt"
	"os"
	"strings"

	"github.com/limit7412/analytics_notifications_slack/repository"
)

type NotifyUsecase interface {
	Run() error
	Error(err error)
	CreateRankingData(title string, color string, data [][]string) repository.Post
	PostToSlack(post []repository.Post) error
}

type notifyImpl struct {
}

// NewNotifyUsecase notification analytics to slack
func NewNotifyUsecase() NotifyUsecase {
	return &notifyImpl{}
}

func (n *notifyImpl) Run() error {
	post := []repository.Post{}
	post = append(post, repository.Post{
		Fallback: os.Getenv("SUCCESS_FALLBACK"),
		Pretext:  os.Getenv("SUCCESS_FALLBACK"),
	})

	adp := repository.NewAnalyticsRepository()
	data, err := adp.GetSessions("today", "today")
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
	slack := repository.NewSlackRepository(os.Getenv("FAILD_WEBHOOK_URL"))

	post := []repository.Post{}
	post = append(post, repository.Post{
		Fallback: os.Getenv("FAILD_FALLBACK"),
		Pretext:  "<@" + os.Getenv("SLACK_ID") + "> " + os.Getenv("FAILD_FALLBACK"),
		Title:    err.Error(),
		Color:    "#EB4646",
		Footer:   "analytics_notifications_slack",
	})

	_ = slack.Post(post)
	fmt.Print(err)
}

func (n *notifyImpl) CreateRankingData(title string, color string, data [][]string) repository.Post {
	text := []string{}
	for i, line := range data {
		if i >= 5 {
			break
		}
		text = append(text, fmt.Sprintf("[%d] %s: %spv", i+1, line[0], line[2]))
	}
	post := repository.Post{
		Title: title,
		Text:  strings.Join(text, "\n"),
		Color: color,
	}

	return post
}

func (n *notifyImpl) PostToSlack(post []repository.Post) error {
	slack := repository.NewSlackRepository(os.Getenv("SUCCESS_WEBHOOK_URL"))
	err := slack.Post(post)
	if err != nil {
		return err
	}

	return nil
}
