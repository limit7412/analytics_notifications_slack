package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type SlackRepository interface {
	Post(ctx context.Context, path string, msg []*Post) error
}

type slackImpl struct {
	client *http.Client
}

// NewSlackRepository は Slack へアクセスするリポジトリを生成する
func NewSlackRepository() SlackRepository {
	return &slackImpl{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

type Post struct {
	Fallback string `json:"fallback"`
	Pretext  string `json:"pretext"`
	Title    string `json:"title"`
	Text     string `json:"text"`
	Color    string `json:"color"`
	Footer   string `json:"footer"`
}

type payload struct {
	Attachments []*Post `json:"attachments"`
}

func (a *slackImpl) Post(ctx context.Context, path string, msg []*Post) error {
	params, err := json.Marshal(payload{
		Attachments: msg,
	})
	if err != nil {
		return fmt.Errorf("marshal slack payload: %w", err)
	}

	body := url.Values{"payload": {string(params)}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, path, strings.NewReader(body.Encode()))
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
