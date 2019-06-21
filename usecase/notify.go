package usecase

import (
	"github.com/limit7412/analytics_notifications_slack/adapter"
	"google.golang.org/api/analytics/v3"
)

type NotifyUsecase interface {
	Run() error
	Error(err error)
	GetTodaySessions() (*analytics.GaData, error)
	ParseSessionsData(data *analytics.GaData) *adapter.Post
	PostToSlack(post *adapter.Post) error
}

type notifyImpl struct {
}

// NewNotifyUsecase notification analytics to slack
func NewNotifyUsecase() NotifyUsecase {
	return &notifyImpl{}
}

func (n *notifyImpl) Run() error {
	data, err := n.GetTodaySessions()
	if err != nil {
		return err
	}

	post := n.ParseSessionsData(data)
	err = n.PostToSlack(post)
	if err != nil {
		return err
	}

	return nil
}

func (n *notifyImpl) Error(err error) {
	slack := adapter.NewSlackAdapter()
	post := &adapter.Post{
		Text:   err.Error(),
		Color:  "#EB4646",
		Footer: "analytics_notifications_slack",
	}

	_ = slack.Post(post)
}

func (n *notifyImpl) GetTodaySessions() (*analytics.GaData, error) {
	adp := adapter.NewAnalyticsAdapter()
	result, err := adp.GetSessions("today", "today")
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (n *notifyImpl) ParseSessionsData(data *analytics.GaData) *adapter.Post {
	post := &adapter.Post{
		Text:   "test",
		Footer: "analytics_notifications_slack",
	}

	return post
}

func (n *notifyImpl) PostToSlack(post *adapter.Post) error {
	slack := adapter.NewSlackAdapter()
	err := slack.Post(post)
	if err != nil {
		return err
	}

	return nil
}
