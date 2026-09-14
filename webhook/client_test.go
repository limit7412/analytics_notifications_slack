package webhook

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPost(t *testing.T) {
	var gotContentType, gotBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")
		b, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		gotBody = string(b)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	c := New("test")
	if err := c.Post(context.Background(), server.URL, "application/json", []byte(`{"a":1}`)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotContentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", gotContentType)
	}
	if gotBody != `{"a":1}` {
		t.Errorf("body = %q", gotBody)
	}
}

func TestPostErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("invalid_payload\n"))
	}))
	defer server.Close()

	err := New("slack").Post(context.Background(), server.URL, "text/plain", []byte("x"))
	if err == nil {
		t.Fatal("expected error for 400 response")
	}
	// 送信先の名前、ステータス、応答本文(前後の空白は除去)がエラーに含まれる。
	if got, want := err.Error(), "slack returned status 400: invalid_payload"; got != want {
		t.Errorf("error = %q, want %q", got, want)
	}
}

func TestPostErrorBodyTruncated(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(strings.Repeat("x", 5000)))
	}))
	defer server.Close()

	err := New("discord").Post(context.Background(), server.URL, "text/plain", nil)
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
	// 応答本文は先頭 1024 バイトまでしかエラーに含めない。
	if got := strings.Count(err.Error(), "x"); got != maxErrorBodyLen {
		t.Errorf("error body has %d bytes of response, want %d", got, maxErrorBodyLen)
	}
}

func TestPostCancelledContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := New("test").Post(ctx, server.URL, "text/plain", nil)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
	if !strings.HasPrefix(err.Error(), "post to test: ") {
		t.Errorf("error = %q, want prefix %q", err.Error(), "post to test: ")
	}
}
