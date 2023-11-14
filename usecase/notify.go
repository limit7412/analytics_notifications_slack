package usecase

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/limit7412/analytics_notifications_slack/repository"
)

type NotifyUsecase interface {
	Run() error
	Error(err error)
}

type notifyImpl struct {
}

// NewNotifyUsecase notification analytics to slack
func NewNotifyUsecase() NotifyUsecase {
	return &notifyImpl{}
}

func (n *notifyImpl) Run() error {
	post := []*repository.Post{}
	post = append(post, &repository.Post{
		Fallback: os.Getenv("SUCCESS_FALLBACK"),
		Pretext:  os.Getenv("SUCCESS_FALLBACK"),
	})

	today := time.Now().Format("2006-01-02")

	adp := repository.NewAnalyticsRepository()
	data, err := adp.GetSessions(today, today)
	if err != nil {
		return err
	}
	line := n.createRankingData("今日のpv数ランキング", "#4286f4", data)
	post = append(post, line)

	month := time.Now().AddDate(0, -(time.Now().Day() - 1), 0).Format("2006-01-02")
	data, err = adp.GetSessions(month, today)
	if err != nil {
		return err
	}
	line = n.createRankingData("今月のpv数ランキング", "#dbe031", data)
	post = append(post, line)

	data, err = adp.GetSessions("2015-08-14", today)
	if err != nil {
		return err
	}
	line = n.createRankingData("累計pv数ランキング", "#41a300", data)
	post = append(post, line)

	slack := repository.NewSlackRepository()
	err = slack.Post(os.Getenv("SUCCESS_WEBHOOK_URL"), post)
	if err != nil {
		return err
	}

	return nil
}

func (n *notifyImpl) Error(err error) {
	slack := repository.NewSlackRepository()

	post := []*repository.Post{}
	post = append(post, &repository.Post{
		Fallback: os.Getenv("FAILD_FALLBACK"),
		Pretext:  "<@" + os.Getenv("SLACK_ID") + "> " + os.Getenv("FAILD_FALLBACK"),
		Title:    err.Error(),
		Color:    "#EB4646",
		Footer:   "analytics_notifications_slack",
	})

	_ = slack.Post(os.Getenv("FAILD_WEBHOOK_URL"), post)
	fmt.Print(err)
}

func (n *notifyImpl) createRankingData(title string, color string, data []*repository.Page) *repository.Post {
	text := []string{}
	for i, item := range data {
		if i >= 5 {
			break
		}
		text = append(text, fmt.Sprintf("[%d] <https://%s|%s>: %dpv", i+1, item.Path, item.Title, item.PV))
	}
	post := &repository.Post{
		Title: title,
		Text:  strings.Join(text, "\n"),
		Color: color,
	}

	return post
}
