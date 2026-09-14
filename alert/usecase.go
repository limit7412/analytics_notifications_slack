package alert

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/limit7412/analytics_notifications_slack/notify"
)

// Usecase は処理の失敗を通知先(Slack / Discord)へ知らせるユースケース。
// 集計に依存しないため、analytics の初期化に失敗した経路からも使える。
type Usecase interface {
	Notify(ctx context.Context, err error)
}

type usecaseImpl struct {
	poster notify.Poster
}

// NewUsecase は失敗通知のユースケースを生成する
func NewUsecase(poster notify.Poster) Usecase {
	return &usecaseImpl{poster: poster}
}

func (u *usecaseImpl) Notify(ctx context.Context, err error) {
	slog.ErrorContext(ctx, "notify failed", slog.Any("error", err))

	msgs := []*notify.Message{
		{
			Fallback: os.Getenv("FAILD_FALLBACK"),
			Mention:  true,
			Pretext:  os.Getenv("FAILD_FALLBACK"),
			Title:    err.Error(),
			Color:    "#EB4646",
			Footer:   "analytics_notifications_slack",
		},
	}

	// 渡された ctx が既にキャンセル済み(例: Lambda タイムアウト)でも失敗通知を
	// 届けられるよう、キャンセルを切り離したコンテキストを使う。
	notifyCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()

	if postErr := u.poster.Post(notifyCtx, os.Getenv("FAILD_WEBHOOK_URL"), msgs); postErr != nil {
		slog.ErrorContext(ctx, "failed to post error notification", slog.Any("error", postErr))
	}
}
