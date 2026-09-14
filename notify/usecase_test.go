package notify

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/limit7412/analytics_notifications_slack/analytics"
)

type fakeAnalytics struct {
	pages []*analytics.Page
	err   error
	calls atomic.Int64
}

func (f *fakeAnalytics) GetSessions(_ context.Context, _ string, _ string) ([]*analytics.Page, error) {
	f.calls.Add(1)
	if f.err != nil {
		return nil, f.err
	}
	return f.pages, nil
}

type fakeNotify struct {
	posts    [][]*Message
	paths    []string
	postErrs []error // Post 呼び出し時点の ctx.Err()
	err      error
}

func (f *fakeNotify) Post(ctx context.Context, webhookURL string, msgs []*Message) error {
	f.paths = append(f.paths, webhookURL)
	f.posts = append(f.posts, msgs)
	f.postErrs = append(f.postErrs, ctx.Err())
	return f.err
}

func TestCreateRankingData(t *testing.T) {
	n := &usecaseImpl{}
	pages := make([]*analytics.Page, 0, 7)
	for i := 0; i < 7; i++ {
		pages = append(pages, &analytics.Page{Title: "t", Path: "h/p", PV: i})
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

func TestCreateRankingDataSanitizesLinkParts(t *testing.T) {
	n := &usecaseImpl{}
	pages := []*analytics.Page{
		{Title: "[Go] <generics|tips>", Path: "h/foo(bar) baz", PV: 1},
	}

	msg := n.createRankingData("t", "#000", pages)

	// markdown リンクを壊す文字はタイトルでは全角へ、URL ではパーセント
	// エンコードへ置き換えられる。期待値の全角文字は ASCII への正規化で
	// テストが素通りしないよう Unicode エスケープで明示する。
	want := "[1] [［Go］ ＜generics｜tips＞](https://h/foo%28bar%29%20baz): 1pv"
	if msg.Text != want {
		t.Errorf("text = %q, want %q", msg.Text, want)
	}
	// 期待値の全角文字が ASCII に化けてテストが素通りしないよう、
	// 元の ASCII 文字が消えていることも直接検証する。
	for _, ng := range []string{"[Go]", "<generics", "|tips>", "(bar)", " baz"} {
		if strings.Contains(msg.Text, ng) {
			t.Errorf("text still contains unsanitized %q: %q", ng, msg.Text)
		}
	}
}

func TestCreateRankingDataFewerThanFive(t *testing.T) {
	n := &usecaseImpl{}
	pages := []*analytics.Page{{Title: "a", Path: "h/a", PV: 3}}

	msg := n.createRankingData("t", "#000", pages)

	if msg.Text != "[1] [a](https://h/a): 3pv" {
		t.Errorf("unexpected text: %q", msg.Text)
	}
}

func TestRunSuccess(t *testing.T) {
	t.Setenv("SUCCESS_WEBHOOK_URL", "https://hooks.example/success")
	t.Setenv("SUCCESS_FALLBACK", "ok")

	ga := &fakeAnalytics{pages: []*analytics.Page{{Title: "a", Path: "h/a", PV: 1}}}
	poster := &fakeNotify{}
	n := NewUsecase(ga, poster)

	if err := n.Run(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 期間(今日・今月・累計)ごとに1回ずつ、並列に呼び出される。
	if got := ga.calls.Load(); got != 3 {
		t.Errorf("GetSessions called %d times, want 3", got)
	}
	if len(poster.posts) != 1 {
		t.Fatalf("poster posted %d times, want 1", len(poster.posts))
	}
	if poster.paths[0] != "https://hooks.example/success" {
		t.Errorf("posted to %q", poster.paths[0])
	}
	// 成功通知には、errgroup の Wait 後にキャンセルされる派生 ctx ではなく
	// 有効な ctx が渡されなければならない。
	if poster.postErrs[0] != nil {
		t.Errorf("poster.Post received a cancelled context: %v", poster.postErrs[0])
	}
	// 先頭のフォールバック + 3つのランキングが順番通りに並ぶ。
	if got := len(poster.posts[0]); got != 4 {
		t.Fatalf("messages = %d, want 4", got)
	}
	wantTitles := []string{"", "今日のpv数ランキング", "今月のpv数ランキング", "累計pv数ランキング"}
	for i, want := range wantTitles {
		if got := poster.posts[0][i].Title; got != want {
			t.Errorf("message %d title = %q, want %q", i, got, want)
		}
	}
}

func TestRunAnalyticsError(t *testing.T) {
	wantErr := errors.New("boom")
	ga := &fakeAnalytics{err: wantErr}
	poster := &fakeNotify{}
	n := NewUsecase(ga, poster)

	err := n.Run(context.Background())
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want wrap of %v", err, wantErr)
	}
	if len(poster.posts) != 0 {
		t.Errorf("poster should not be posted on analytics failure")
	}
}
