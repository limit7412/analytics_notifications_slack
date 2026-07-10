package repository

import (
	"context"
	"os"
)

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
	// Mention が true の場合、アダプタが投稿先の形式でメンションを付与する。
	// メンション先の ID は MENTION_ID(未設定時は SLACK_ID)から解決する。
	Mention bool
	Pretext string
	Title   string
	Text    string
	// Color は `#4286f4` 形式の hex 文字列。
	Color  string
	Footer string
}

// mentionID はメンション先のユーザー ID を返す。
// Slack と Discord で ID 体系が異なるため MENTION_ID に一般化しつつ、
// 後方互換として未設定時は SLACK_ID にフォールバックする。
func mentionID() string {
	if id := os.Getenv("MENTION_ID"); id != "" {
		return id
	}
	return os.Getenv("SLACK_ID")
}
