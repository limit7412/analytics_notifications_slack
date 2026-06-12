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

type fakeSlack struct {
	posts    [][]*repository.Post
	paths    []string
	postErrs []error // Post 呼び出し時点の ctx.Err()
	err      error
}

func (f *fakeSlack) Post(ctx context.Context, path string, msg []*repository.Post) error {
	f.paths = append(f.paths, path)
	f.posts = append(f.posts, msg)
	f.postErrs = append(f.postErrs, ctx.Err())
	return f.err
}

func TestCreateRankingData(t *testing.T) {
	n := &notifyImpl{}
	pages := make([]*repository.Page, 0, 7)
	for i := 0; i < 7; i++ {
		pages = append(pages, &repository.Page{Title: "t", Path: "h/p", PV: i})
	}

	post := n.createRankingData("ランキング", "#fff", pages)

	if post.Title != "ランキング" || post.Color != "#fff" {
		t.Errorf("unexpected title/color: %+v", post)
	}
	// 上位5件のみが描画される。
	if lines := strings.Count(post.Text, "\n") + 1; lines != 5 {
		t.Errorf("got %d lines, want 5", lines)
	}
	if !strings.Contains(post.Text, "[1] <https://h/p|t>: 0pv") {
		t.Errorf("unexpected first line: %q", post.Text)
	}
}

func TestCreateRankingDataFewerThanFive(t *testing.T) {
	n := &notifyImpl{}
	pages := []*repository.Page{{Title: "a", Path: "h/a", PV: 3}}

	post := n.createRankingData("t", "#000", pages)

	if post.Text != "[1] <https://h/a|a>: 3pv" {
		t.Errorf("unexpected text: %q", post.Text)
	}
}

func TestRunSuccess(t *testing.T) {
	t.Setenv("SUCCESS_WEBHOOK_URL", "https://hooks.example/success")
	t.Setenv("SUCCESS_FALLBACK", "ok")

	analytics := &fakeAnalytics{pages: []*repository.Page{{Title: "a", Path: "h/a", PV: 1}}}
	slack := &fakeSlack{}
	n := NewNotifyUsecase(analytics, slack)

	if err := n.Run(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 期間(今日・今月・累計)ごとに1回ずつ、並列に呼び出される。
	if got := analytics.calls.Load(); got != 3 {
		t.Errorf("GetSessions called %d times, want 3", got)
	}
	if len(slack.posts) != 1 {
		t.Fatalf("slack posted %d times, want 1", len(slack.posts))
	}
	if slack.paths[0] != "https://hooks.example/success" {
		t.Errorf("posted to %q", slack.paths[0])
	}
	// 成功通知には、errgroup の Wait 後にキャンセルされる派生 ctx ではなく
	// 有効な ctx が渡されなければならない。
	if slack.postErrs[0] != nil {
		t.Errorf("slack.Post received a cancelled context: %v", slack.postErrs[0])
	}
	// 先頭のフォールバック + 3つのランキング添付が順番通りに並ぶ。
	if got := len(slack.posts[0]); got != 4 {
		t.Fatalf("attachments = %d, want 4", got)
	}
	wantTitles := []string{"", "今日のpv数ランキング", "今月のpv数ランキング", "累計pv数ランキング"}
	for i, want := range wantTitles {
		if got := slack.posts[0][i].Title; got != want {
			t.Errorf("attachment %d title = %q, want %q", i, got, want)
		}
	}
}

func TestRunAnalyticsError(t *testing.T) {
	wantErr := errors.New("boom")
	analytics := &fakeAnalytics{err: wantErr}
	slack := &fakeSlack{}
	n := NewNotifyUsecase(analytics, slack)

	err := n.Run(context.Background())
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want wrap of %v", err, wantErr)
	}
	if len(slack.posts) != 0 {
		t.Errorf("slack should not be posted on analytics failure")
	}
}

func TestError(t *testing.T) {
	t.Setenv("FAILD_WEBHOOK_URL", "https://hooks.example/fail")
	t.Setenv("FAILD_FALLBACK", "failed")
	t.Setenv("SLACK_ID", "U123")

	slack := &fakeSlack{}
	n := NewNotifyUsecase(&fakeAnalytics{}, slack)

	// 既にキャンセル済みのコンテキストでも通知は送られなければならない。
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	n.Error(ctx, errors.New("something broke"))

	if len(slack.posts) != 1 {
		t.Fatalf("slack posted %d times, want 1", len(slack.posts))
	}
	if slack.paths[0] != "https://hooks.example/fail" {
		t.Errorf("posted to %q", slack.paths[0])
	}
	post := slack.posts[0][0]
	if post.Title != "something broke" {
		t.Errorf("title = %q", post.Title)
	}
	if !strings.Contains(post.Pretext, "<@U123>") {
		t.Errorf("pretext missing mention: %q", post.Pretext)
	}
}
