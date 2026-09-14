package alert

import (
	"context"
	"errors"
	"testing"

	"github.com/limit7412/analytics_notifications_slack/notify"
)

type fakePoster struct {
	posts [][]*notify.Message
	paths []string
}

func (f *fakePoster) Post(_ context.Context, webhookURL string, msgs []*notify.Message) error {
	f.paths = append(f.paths, webhookURL)
	f.posts = append(f.posts, msgs)
	return nil
}

func TestNotify(t *testing.T) {
	t.Setenv("FAILD_WEBHOOK_URL", "https://hooks.example/fail")
	t.Setenv("FAILD_FALLBACK", "failed")

	poster := &fakePoster{}
	u := NewUsecase(poster)

	// 既にキャンセル済みのコンテキストでも通知は送られなければならない。
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	u.Notify(ctx, errors.New("something broke"))

	if len(poster.posts) != 1 {
		t.Fatalf("poster posted %d times, want 1", len(poster.posts))
	}
	if poster.paths[0] != "https://hooks.example/fail" {
		t.Errorf("posted to %q", poster.paths[0])
	}
	msg := poster.posts[0][0]
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
