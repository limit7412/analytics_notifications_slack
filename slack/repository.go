package slack

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"

	"github.com/limit7412/analytics_notifications_slack/notify"
	"github.com/limit7412/analytics_notifications_slack/webhook"
)

type repositoryImpl struct {
	webhook *webhook.Client
}

// NewRepository は Slack へ投稿するリポジトリを生成する
func NewRepository() notify.Poster {
	return &repositoryImpl{
		webhook: webhook.New("slack"),
	}
}

type attachment struct {
	Fallback string `json:"fallback"`
	Pretext  string `json:"pretext"`
	Title    string `json:"title"`
	Text     string `json:"text"`
	Color    string `json:"color"`
	Footer   string `json:"footer"`
}

type slackPayload struct {
	Attachments []*attachment `json:"attachments"`
}

// markdownLink は中立表現のリンク `[title](url)` にマッチする。
var markdownLink = regexp.MustCompile(`\[([^\]]*)\]\(([^)]+)\)`)

// toMrkdwnLinks は `[title](url)` を Slack mrkdwn 形式 `<url|title>` へ変換する。
func toMrkdwnLinks(text string) string {
	return markdownLink.ReplaceAllString(text, "<$2|$1>")
}

func (a *repositoryImpl) Post(ctx context.Context, webhookURL string, msgs []*notify.Message) error {
	attachments := make([]*attachment, 0, len(msgs))
	for _, msg := range msgs {
		if msg == nil {
			continue
		}
		pretext := msg.Pretext
		if msg.Mention {
			pretext = "<!channel> " + pretext
		}
		attachments = append(attachments, &attachment{
			Fallback: msg.Fallback,
			Pretext:  pretext,
			Title:    msg.Title,
			Text:     toMrkdwnLinks(msg.Text),
			Color:    msg.Color,
			Footer:   msg.Footer,
		})
	}

	params, err := json.Marshal(slackPayload{
		Attachments: attachments,
	})
	if err != nil {
		return fmt.Errorf("marshal slack payload: %w", err)
	}

	// Slack の Incoming Webhook には payload パラメータに JSON を入れた
	// フォーム形式で送る。
	body := url.Values{"payload": {string(params)}}
	return a.webhook.Post(ctx, webhookURL, "application/x-www-form-urlencoded", []byte(body.Encode()))
}
