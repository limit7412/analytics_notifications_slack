package notify

import "context"

// Poster は通知メッセージを webhook へ投稿する共通インターフェース。
// Slack / Discord をアダプタとして差し替えられる。
type Poster interface {
	Post(ctx context.Context, webhookURL string, msgs []*Message) error
}
