package usecase

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/limit7412/analytics_notifications_slack/repository"
)

type fakeAnalytics struct {
	pages []*repository.Page
	err   error
	calls int
}

func (f *fakeAnalytics) GetSessions(_ context.Context, _ string, _ string) ([]*repository.Page, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	return f.pages, nil
}

type fakeSlack struct {
	posts [][]*repository.Post
	paths []string
	err   error
}

func (f *fakeSlack) Post(_ context.Context, path string, msg []*repository.Post) error {
	f.paths = append(f.paths, path)
	f.posts = append(f.posts, msg)
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
	// Only the top 5 entries should be rendered.
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

	// One call per date range (today, month, cumulative).
	if analytics.calls != 3 {
		t.Errorf("GetSessions called %d times, want 3", analytics.calls)
	}
	if len(slack.posts) != 1 {
		t.Fatalf("slack posted %d times, want 1", len(slack.posts))
	}
	if slack.paths[0] != "https://hooks.example/success" {
		t.Errorf("posted to %q", slack.paths[0])
	}
	// Fallback header + 3 ranking attachments.
	if got := len(slack.posts[0]); got != 4 {
		t.Errorf("attachments = %d, want 4", got)
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

	n.Error(context.Background(), errors.New("something broke"))

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
