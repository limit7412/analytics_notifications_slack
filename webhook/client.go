// Package webhook は webhook URL へ HTTP POST する共通クライアントを提供する。
//
// Slack と Discord のアダプタは投稿の組み立て方こそ違うが、「webhook URL へ
// 本文を POST し、2xx 以外を失敗として応答本文をエラーに含める」点は同じで、
// この部分が二重に書かれていた。片方だけ直すと挙動がずれるため、送信そのものは
// ここにまとめ、アダプタはペイロードの組み立てと Content-Type の指定に専念する。
package webhook

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// エラーメッセージに含める応答本文の最大バイト数。
const maxErrorBodyLen = 1024

// Client は webhook へ POST するクライアント。
type Client struct {
	// service はエラーメッセージに出す送信先の名前(slack / discord)。
	service string
	client  *http.Client
}

// New は service 向けのクライアントを生成する。
// タイムアウトは Lambda の実行時間を食い潰さないよう 10 秒に固定する。
func New(service string) *Client {
	return &Client{
		service: service,
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

// Post は body を contentType として webhookURL へ POST する。
// 2xx 以外の応答は失敗とみなし、ステータスと応答本文の先頭をエラーに含める。
func (c *Client) Post(ctx context.Context, webhookURL string, contentType string, body []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create %s request: %w", c.service, err)
	}
	req.Header.Set("Content-Type", contentType)

	res, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("post to %s: %w", c.service, err)
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		resBody, _ := io.ReadAll(io.LimitReader(res.Body, maxErrorBodyLen))
		return fmt.Errorf("%s returned status %d: %s", c.service, res.StatusCode, strings.TrimSpace(string(resBody)))
	}
	// コネクションを再利用できるよう、成功時も本文を読み切る。
	_, _ = io.Copy(io.Discard, res.Body)

	return nil
}
