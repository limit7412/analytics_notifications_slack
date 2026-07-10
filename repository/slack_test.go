package repository

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSlackPost(t *testing.T) {
	t.Setenv("MENTION_ID", "")
	t.Setenv("SLACK_ID", "U123")

	payloads := []slackPayload{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ct := r.Header.Get("Content-Type"); ct != "application/x-www-form-urlencoded" {
			t.Errorf("Content-Type = %q, want application/x-www-form-urlencoded", ct)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		var p slackPayload
		if err := json.Unmarshal([]byte(r.PostFormValue("payload")), &p); err != nil {
			t.Fatalf("unmarshal payload: %v", err)
		}
		payloads = append(payloads, p)
	}))
	defer server.Close()

	repo := NewSlackRepository()
	msgs := []*Message{
		{Fallback: "ok", Pretext: "ok"},
		{Title: "ランキング", Text: "[1] [a](https://h/a): 3pv\n[2] [b](https://h/b): 1pv", Color: "#4286f4"},
		{Mention: true, Pretext: "failed", Title: "boom", Color: "#EB4646", Footer: "footer"},
	}
	if err := repo.Post(context.Background(), server.URL, msgs); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(payloads) != 1 {
		t.Fatalf("posted %d times, want 1", len(payloads))
	}
	attachments := payloads[0].Attachments
	if len(attachments) != 3 {
		t.Fatalf("attachments = %d, want 3", len(attachments))
	}

	if attachments[0].Fallback != "ok" || attachments[0].Pretext != "ok" {
		t.Errorf("unexpected fallback attachment: %+v", attachments[0])
	}
	// 中立表現の markdown リンクは Slack mrkdwn 形式へ変換される。
	wantText := "[1] <https://h/a|a>: 3pv\n[2] <https://h/b|b>: 1pv"
	if attachments[1].Text != wantText {
		t.Errorf("text = %q, want %q", attachments[1].Text, wantText)
	}
	if attachments[1].Color != "#4286f4" {
		t.Errorf("color = %q", attachments[1].Color)
	}
	// Mention フラグは pretext のメンションへ変換される(MENTION_ID 未設定時は SLACK_ID)。
	if attachments[2].Pretext != "<@U123> failed" {
		t.Errorf("pretext = %q, want %q", attachments[2].Pretext, "<@U123> failed")
	}
	if attachments[2].Footer != "footer" {
		t.Errorf("footer = %q", attachments[2].Footer)
	}
}

func TestSlackPostErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "invalid_payload", http.StatusBadRequest)
	}))
	defer server.Close()

	repo := NewSlackRepository()
	err := repo.Post(context.Background(), server.URL, []*Message{{Title: "t"}})
	if err == nil || !strings.Contains(err.Error(), "400") {
		t.Fatalf("error = %v, want status 400 error", err)
	}
}

func TestToMrkdwnLinks(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"[a](https://h/a)", "<https://h/a|a>"},
		{"[1] [a](https://h/a): 3pv", "[1] <https://h/a|a>: 3pv"},
		{"no link", "no link"},
		{"", ""},
	}
	for _, c := range cases {
		if got := toMrkdwnLinks(c.in); got != c.want {
			t.Errorf("toMrkdwnLinks(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
