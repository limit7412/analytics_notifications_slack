package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type slackImpl struct {
	client *http.Client
}

// NewSlackRepository は Slack へ投稿するリポジトリを生成する
func NewSlackRepository() NotifyRepository {
	return &slackImpl{
		client: &http.Client{Timeout: 10 * time.Second},
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

func (a *slackImpl) Post(ctx context.Context, webhookURL string, msgs []*Message) error {
	attachments := make([]*attachment, 0, len(msgs))
	for _, msg := range msgs {
		if msg == nil {
			continue
		}
		pretext := msg.Pretext
		// ID 未設定時は壊れたメンション `<@>` を出力しない。
		if id := mentionID(); msg.Mention && id != "" {
			pretext = "<@" + id + "> " + pretext
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

	body := url.Values{"payload": {string(params)}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, strings.NewReader(body.Encode()))
	if err != nil {
		return fmt.Errorf("create slack request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("post to slack: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		resBody, _ := io.ReadAll(io.LimitReader(res.Body, 1024))
		return fmt.Errorf("slack returned status %d: %s", res.StatusCode, strings.TrimSpace(string(resBody)))
	}
	_, _ = io.Copy(io.Discard, res.Body)

	return nil
}
