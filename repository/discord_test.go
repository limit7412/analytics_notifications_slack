package repository

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newDiscordTestServer は受信した JSON ペイロードを記録するテストサーバーを返す。
func newDiscordTestServer(t *testing.T, status int, payloads *[]discordPayload) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		var p discordPayload
		if err := json.Unmarshal(body, &p); err != nil {
			t.Fatalf("unmarshal payload: %v", err)
		}
		*payloads = append(*payloads, p)
		w.WriteHeader(status)
	}))
}

func TestDiscordPost(t *testing.T) {
	t.Setenv("MENTION_ID", "D123")

	payloads := []discordPayload{}
	server := newDiscordTestServer(t, http.StatusNoContent, &payloads)
	defer server.Close()

	repo := NewDiscordRepository()
	msgs := []*Message{
		{Fallback: "ok", Pretext: "ok"},
		{Title: "ランキング", Text: "[1] [a](https://h/a): 3pv", Color: "#4286f4"},
		{Mention: true, Pretext: "failed", Title: "boom", Color: "#EB4646", Footer: "footer"},
	}
	if err := repo.Post(context.Background(), server.URL, msgs); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(payloads) != 1 {
		t.Fatalf("posted %d times, want 1", len(payloads))
	}
	p := payloads[0]

	// pretext とメンションは content にまとめられる。
	if want := "ok <@D123> failed"; p.Content != want {
		t.Errorf("content = %q, want %q", p.Content, want)
	}
	// pretext のみのメッセージは embed にならない。
	if len(p.Embeds) != 2 {
		t.Fatalf("embeds = %d, want 2", len(p.Embeds))
	}
	// markdown リンクはそのまま description に載る。
	if p.Embeds[0].Description != "[1] [a](https://h/a): 3pv" {
		t.Errorf("description = %q", p.Embeds[0].Description)
	}
	if p.Embeds[0].Color != 0x4286f4 {
		t.Errorf("color = %d, want %d", p.Embeds[0].Color, 0x4286f4)
	}
	if p.Embeds[1].Title != "boom" {
		t.Errorf("title = %q", p.Embeds[1].Title)
	}
	if p.Embeds[1].Footer == nil || p.Embeds[1].Footer.Text != "footer" {
		t.Errorf("footer = %+v", p.Embeds[1].Footer)
	}
}

func TestDiscordPostMentionFallsBackToSlackID(t *testing.T) {
	t.Setenv("MENTION_ID", "")
	t.Setenv("SLACK_ID", "U123")

	payloads := []discordPayload{}
	server := newDiscordTestServer(t, http.StatusNoContent, &payloads)
	defer server.Close()

	repo := NewDiscordRepository()
	if err := repo.Post(context.Background(), server.URL, []*Message{{Mention: true, Title: "t"}}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if payloads[0].Content != "<@U123>" {
		t.Errorf("content = %q, want %q", payloads[0].Content, "<@U123>")
	}
}

func TestDiscordPostChunksEmbeds(t *testing.T) {
	payloads := []discordPayload{}
	server := newDiscordTestServer(t, http.StatusNoContent, &payloads)
	defer server.Close()

	msgs := make([]*Message, 0, 11)
	msgs = append(msgs, &Message{Pretext: "head"})
	for i := 0; i < 11; i++ {
		msgs = append(msgs, &Message{Title: "t"})
	}

	repo := NewDiscordRepository()
	if err := repo.Post(context.Background(), server.URL, msgs); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// embeds は最大10件のため 10 + 1 に分割される。
	if len(payloads) != 2 {
		t.Fatalf("posted %d times, want 2", len(payloads))
	}
	if len(payloads[0].Embeds) != 10 || len(payloads[1].Embeds) != 1 {
		t.Errorf("embeds = %d, %d, want 10, 1", len(payloads[0].Embeds), len(payloads[1].Embeds))
	}
	// content は先頭のメッセージにのみ載る。
	if payloads[0].Content != "head" || payloads[1].Content != "" {
		t.Errorf("content = %q, %q", payloads[0].Content, payloads[1].Content)
	}
}

func TestDiscordPostTruncates(t *testing.T) {
	payloads := []discordPayload{}
	server := newDiscordTestServer(t, http.StatusNoContent, &payloads)
	defer server.Close()

	msgs := []*Message{
		{Pretext: strings.Repeat("あ", discordMaxContentLen+1)},
		{Text: strings.Repeat("い", discordMaxDescriptionLen+1)},
	}

	repo := NewDiscordRepository()
	if err := repo.Post(context.Background(), server.URL, msgs); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := len([]rune(payloads[0].Content)); got != discordMaxContentLen {
		t.Errorf("content length = %d, want %d", got, discordMaxContentLen)
	}
	if got := len([]rune(payloads[0].Embeds[0].Description)); got != discordMaxDescriptionLen {
		t.Errorf("description length = %d, want %d", got, discordMaxDescriptionLen)
	}
}

func TestDiscordPostEmptyMessages(t *testing.T) {
	payloads := []discordPayload{}
	server := newDiscordTestServer(t, http.StatusNoContent, &payloads)
	defer server.Close()

	repo := NewDiscordRepository()
	if err := repo.Post(context.Background(), server.URL, []*Message{{}}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 空ペイロードは Discord がエラーを返すため送信しない。
	if len(payloads) != 0 {
		t.Errorf("posted %d times, want 0", len(payloads))
	}
}

func TestDiscordPostErrorStatus(t *testing.T) {
	payloads := []discordPayload{}
	server := newDiscordTestServer(t, http.StatusTooManyRequests, &payloads)
	defer server.Close()

	repo := NewDiscordRepository()
	err := repo.Post(context.Background(), server.URL, []*Message{{Title: "t"}})
	if err == nil || !strings.Contains(err.Error(), "429") {
		t.Fatalf("error = %v, want status 429 error", err)
	}
}

func TestHexToColor(t *testing.T) {
	cases := []struct {
		hex  string
		want int
	}{
		{"#4286f4", 0x4286f4},
		{"#EB4646", 0xEB4646},
		{"4286f4", 0x4286f4},
		{"", 0},
		{"#zzzzzz", 0},
		{"#1234567", 0},
	}
	for _, c := range cases {
		if got := hexToColor(c.hex); got != c.want {
			t.Errorf("hexToColor(%q) = %d, want %d", c.hex, got, c.want)
		}
	}
}
