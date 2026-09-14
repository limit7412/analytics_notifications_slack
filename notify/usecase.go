package notify

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/limit7412/analytics_notifications_slack/analytics"
	"golang.org/x/sync/errgroup"
)

type Usecase interface {
	Run(ctx context.Context) error
}

type usecaseImpl struct {
	analytics analytics.Repository
	poster    Poster
}

// NewUsecase は分析結果を通知先(Slack / Discord)へ通知するユースケースを生成する
func NewUsecase(analytics analytics.Repository, poster Poster) Usecase {
	return &usecaseImpl{
		analytics: analytics,
		poster:    poster,
	}
}

func (n *usecaseImpl) Run(ctx context.Context) error {
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
	rankings := make([]*Message, len(ranges))
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

	msgs := make([]*Message, 0, len(ranges)+1)
	msgs = append(msgs, &Message{
		Fallback: os.Getenv("SUCCESS_FALLBACK"),
		Pretext:  os.Getenv("SUCCESS_FALLBACK"),
	})
	msgs = append(msgs, rankings...)

	if err := n.poster.Post(ctx, os.Getenv("SUCCESS_WEBHOOK_URL"), msgs); err != nil {
		return fmt.Errorf("post notification: %w", err)
	}

	return nil
}

func (n *usecaseImpl) createRankingData(title string, color string, data []*analytics.Page) *Message {
	text := []string{}
	for i, item := range data {
		if i >= 5 {
			break
		}
		if item == nil {
			continue
		}
		// リンクは中立表現の markdown 形式で保持し、変換は各アダプタに任せる。
		text = append(text, fmt.Sprintf("[%d] [%s](https://%s): %dpv", i+1, sanitizeLinkTitle(item.Title), sanitizeLinkURL(item.Path), item.PV))
	}

	return &Message{
		Title: title,
		Text:  strings.Join(text, "\n"),
		Color: color,
	}
}

// sanitizeLinkTitle は markdown リンク `[title](url)` や Slack mrkdwn `<url|title>` を
// 壊す文字を、見た目の近い全角文字(U+FF3B/FF3D/FF1C/FF1E/FF5C)へ置き換える。
var sanitizeLinkTitle = strings.NewReplacer(
	"[", "［", "]", "］",
	"<", "＜", ">", "＞",
	"|", "｜",
).Replace

// sanitizeLinkURL はリンク URL の解釈を途中で打ち切る文字をパーセントエンコードする。
var sanitizeLinkURL = strings.NewReplacer(
	"(", "%28", ")", "%29",
	"<", "%3C", ">", "%3E",
	"|", "%7C", " ", "%20",
).Replace
