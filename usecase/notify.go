package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/limit7412/analytics_notifications_slack/repository"
	"golang.org/x/sync/errgroup"
)

type NotifyUsecase interface {
	Run(ctx context.Context) error
	Error(ctx context.Context, err error)
}

type notifyImpl struct {
	analytics repository.AnalyticsRepository
	notify    repository.NotifyRepository
}

// NewNotifyUsecase は分析結果を通知先(Slack / Discord)へ通知するユースケースを生成する
func NewNotifyUsecase(analytics repository.AnalyticsRepository, notify repository.NotifyRepository) NotifyUsecase {
	return &notifyImpl{
		analytics: analytics,
		notify:    notify,
	}
}

func (n *notifyImpl) Run(ctx context.Context) error {
	now := time.Now()
	today := now.Format("2006-01-02")
	month := now.AddDate(0, 0, -(now.Day() - 1)).Format("2006-01-02")

	ranges := []struct {
		title string
		color string
		start string
		end   string
	}{
		{"今日のpv数ランキング", "#4286f4", today, today},
		{"今月のpv数ランキング", "#dbe031", month, today},
		{"累計pv数ランキング", "#41a300", "2015-08-14", today},
	}

	// 各期間を並列に取得する。結果はインデックスで格納し、元の順序を保持する。
	// errgroup の派生 ctx (gctx) は Wait 後にキャンセルされるため取得処理にのみ使い、
	// 成功通知には元の ctx を使う。
	rankings := make([]*repository.Message, len(ranges))
	g, gctx := errgroup.WithContext(ctx)
	for i, r := range ranges {
		g.Go(func() error {
			data, err := n.analytics.GetSessions(gctx, r.start, r.end)
			if err != nil {
				return fmt.Errorf("get sessions (%s): %w", r.title, err)
			}
			rankings[i] = n.createRankingData(r.title, r.color, data)
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return err
	}

	msgs := make([]*repository.Message, 0, len(ranges)+1)
	msgs = append(msgs, &repository.Message{
		Fallback: os.Getenv("SUCCESS_FALLBACK"),
		Pretext:  os.Getenv("SUCCESS_FALLBACK"),
	})
	msgs = append(msgs, rankings...)

	if err := n.notify.Post(ctx, os.Getenv("SUCCESS_WEBHOOK_URL"), msgs); err != nil {
		return fmt.Errorf("post notification: %w", err)
	}

	return nil
}

func (n *notifyImpl) Error(ctx context.Context, err error) {
	slog.ErrorContext(ctx, "notify failed", slog.Any("error", err))

	msgs := []*repository.Message{
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

	if postErr := n.notify.Post(notifyCtx, os.Getenv("FAILD_WEBHOOK_URL"), msgs); postErr != nil {
		slog.ErrorContext(ctx, "failed to post error notification", slog.Any("error", postErr))
	}
}

func (n *notifyImpl) createRankingData(title string, color string, data []*repository.Page) *repository.Message {
	text := []string{}
	for i, item := range data {
		if i >= 5 {
			break
		}
		if item == nil {
			continue
		}
		// リンクは中立表現の markdown 形式で保持し、変換は各アダプタに任せる。
		text = append(text, fmt.Sprintf("[%d] [%s](https://%s): %dpv", i+1, item.Title, item.Path, item.PV))
	}

	return &repository.Message{
		Title: title,
		Text:  strings.Join(text, "\n"),
		Color: color,
	}
}
