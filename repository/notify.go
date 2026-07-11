package repository

import "context"

// NotifyRepository は通知メッセージを webhook へ投稿するリポジトリの
// 共通インターフェース。Slack / Discord をアダプタとして差し替えられる。
type NotifyRepository interface {
	Post(ctx context.Context, webhookURL string, msgs []*Message) error
}

// Message は投稿先サービスに依存しない中立な通知メッセージ。
// リンクは markdown 形式 `[title](url)` で Text に保持し、
// 必要な変換は各アダプタが行う。
type Message struct {
	Fallback string
	// Mention が true の場合、アダプタが投稿先の形式で全体通知
	// (Slack: <!channel>、Discord: @everyone)を付与する。
	Mention bool
	Pretext string
	Title   string
	Text    string
	// Color は `#4286f4` 形式の hex 文字列。
	Color  string
	Footer string
}
