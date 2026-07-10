package usecase

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/limit7412/analytics_notifications_slack/repository"
)

type fakeAnalytics struct {
	pages []*repository.Page
	err   error
	calls atomic.Int64
}

func (f *fakeAnalytics) GetSessions(_ context.Context, _ string, _ string) ([]*repository.Page, error) {
	f.calls.Add(1)
	if f.err != nil {
		return nil, f.err
	}
	return f.pages, nil
}

type fakeNotify struct {
	posts    [][]*repository.Message
	paths    []string
	postErrs []error // Post 呼び出し時点の ctx.Err()
	err      error
}

func (f *fakeNotify) Post(ctx context.Context, webhookURL string, msgs []*repository.Message) error {
	f.paths = append(f.paths, webhookURL)
	f.posts = append(f.posts, msgs)
	f.postErrs = append(f.postErrs, ctx.Err())
	return f.err
}

func TestCreateRankingData(t *testing.T) {
	n := &notifyImpl{}
	pages := make([]*repository.Page, 0, 7)
	for i := 0; i < 7; i++ {
		pages = append(pages, &repository.Page{Title: "t", Path: "h/p", PV: i})
	}

	msg := n.createRankingData("ランキング", "#fff", pages)

	if msg.Title != "ランキング" || msg.Color != "#fff" {
		t.Errorf("unexpected title/color: %+v", msg)
	}
	// 上位5件のみが描画される。
	if lines := strings.Count(msg.Text, "\n") + 1; lines != 5 {
		t.Errorf("got %d lines, want 5", lines)
	}
	// リンクは中立表現の markdown 形式で保持される。
	if !strings.Contains(msg.Text, "[1] [t](https://h/p): 0pv") {
		t.Errorf("unexpected first line: %q", msg.Text)
	}
}

func TestCreateRankingDataFewerThanFive(t *testing.T) {
	n := &notifyImpl{}
	pages := []*repository.Page{{Title: "a", Path: "h/a", PV: 3}}

	msg := n.createRankingData("t", "#000", pages)

	if msg.Text != "[1] [a](https://h/a): 3pv" {
		t.Errorf("unexpected text: %q", msg.Text)
	}
}

func TestRunSuccess(t *testing.T) {
	t.Setenv("SUCCESS_WEBHOOK_URL", "https://hooks.example/success")
	t.Setenv("SUCCESS_FALLBACK", "ok")

	analytics := &fakeAnalytics{pages: []*repository.Page{{Title: "a", Path: "h/a", PV: 1}}}
	notify := &fakeNotify{}
	n := NewNotifyUsecase(analytics, notify)

	if err := n.Run(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 期間(今日・今月・累計)ごとに1回ずつ、並列に呼び出される。
	if got := analytics.calls.Load(); got != 3 {
		t.Errorf("GetSessions called %d times, want 3", got)
	}
	if len(notify.posts) != 1 {
		t.Fatalf("notify posted %d times, want 1", len(notify.posts))
	}
	if notify.paths[0] != "https://hooks.example/success" {
		t.Errorf("posted to %q", notify.paths[0])
	}
	// 成功通知には、errgroup の Wait 後にキャンセルされる派生 ctx ではなく
	// 有効な ctx が渡されなければならない。
	if notify.postErrs[0] != nil {
		t.Errorf("notify.Post received a cancelled context: %v", notify.postErrs[0])
	}
	// 先頭のフォールバック + 3つのランキングが順番通りに並ぶ。
	if got := len(notify.posts[0]); got != 4 {
		t.Fatalf("messages = %d, want 4", got)
	}
	wantTitles := []string{"", "今日のpv数ランキング", "今月のpv数ランキング", "累計pv数ランキング"}
	for i, want := range wantTitles {
		if got := notify.posts[0][i].Title; got != want {
			t.Errorf("message %d title = %q, want %q", i, got, want)
		}
	}
}

func TestRunAnalyticsError(t *testing.T) {
	wantErr := errors.New("boom")
	analytics := &fakeAnalytics{err: wantErr}
	notify := &fakeNotify{}
	n := NewNotifyUsecase(analytics, notify)

	err := n.Run(context.Background())
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want wrap of %v", err, wantErr)
	}
	if len(notify.posts) != 0 {
		t.Errorf("notify should not be posted on analytics failure")
	}
}

func TestError(t *testing.T) {
	t.Setenv("FAILD_WEBHOOK_URL", "https://hooks.example/fail")
	t.Setenv("FAILD_FALLBACK", "failed")

	notify := &fakeNotify{}
	n := NewNotifyUsecase(&fakeAnalytics{}, notify)

	// 既にキャンセル済みのコンテキストでも通知は送られなければならない。
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	n.Error(ctx, errors.New("something broke"))

	if len(notify.posts) != 1 {
		t.Fatalf("notify posted %d times, want 1", len(notify.posts))
	}
	if notify.paths[0] != "https://hooks.example/fail" {
		t.Errorf("posted to %q", notify.paths[0])
	}
	msg := notify.posts[0][0]
	if msg.Title != "something broke" {
		t.Errorf("title = %q", msg.Title)
	}
	// メンションの形式変換はアダプタに任せるため、usecase はフラグのみ立てる。
	if !msg.Mention {
		t.Errorf("mention flag should be set")
	}
	if msg.Pretext != "failed" {
		t.Errorf("pretext = %q, want %q", msg.Pretext, "failed")
	}
}
