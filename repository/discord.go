package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Discord webhook の制約値。
// https://discord.com/developers/docs/resources/webhook
const (
	// 1メッセージあたりの embeds の最大数
	discordMaxEmbeds = 10
	// content の最大文字数
	discordMaxContentLen = 2000
	// embed description の最大文字数
	discordMaxDescriptionLen = 4096
)

type discordImpl struct {
	client *http.Client
}

// NewDiscordRepository は Discord へ投稿するリポジトリを生成する
func NewDiscordRepository() NotifyRepository {
	return &discordImpl{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

type discordFooter struct {
	Text string `json:"text"`
}

type discordEmbed struct {
	Title       string         `json:"title,omitempty"`
	Description string         `json:"description,omitempty"`
	Color       int            `json:"color,omitempty"`
	Footer      *discordFooter `json:"footer,omitempty"`
}

type discordPayload struct {
	Content string          `json:"content,omitempty"`
	Embeds  []*discordEmbed `json:"embeds,omitempty"`
}

func (a *discordImpl) Post(ctx context.Context, webhookURL string, msgs []*Message) error {
	// メンションは embed 内では機能しないため content に出力する。
	// pretext 相当のテキストも embed には対応する場所がないので content にまとめる。
	contentParts := []string{}
	mentioned := false
	embeds := []*discordEmbed{}
	for _, msg := range msgs {
		if msg == nil {
			continue
		}
		// ID 未設定時は壊れたメンション `<@>` を出力しない。
		if id := mentionID(); msg.Mention && !mentioned && id != "" {
			contentParts = append(contentParts, "<@"+id+">")
			mentioned = true
		}
		if msg.Pretext != "" {
			contentParts = append(contentParts, msg.Pretext)
		}
		if msg.Title == "" && msg.Text == "" && msg.Footer == "" {
			continue
		}

		var footer *discordFooter
		if msg.Footer != "" {
			footer = &discordFooter{Text: msg.Footer}
		}
		embeds = append(embeds, &discordEmbed{
			Title:       msg.Title,
			Description: truncateRunes(msg.Text, discordMaxDescriptionLen),
			Color:       hexToColor(msg.Color),
			Footer:      footer,
		})
	}

	content := truncateRunes(strings.Join(contentParts, " "), discordMaxContentLen)
	if content == "" && len(embeds) == 0 {
		return nil
	}

	// embeds は1メッセージ最大10件のため、超える場合はチャンクして複数回送信する。
	// content は先頭のメッセージにのみ載せる。
	for first := true; first || len(embeds) > 0; first = false {
		chunk := embeds[:min(len(embeds), discordMaxEmbeds)]
		embeds = embeds[len(chunk):]

		payload := &discordPayload{Embeds: chunk}
		if first {
			payload.Content = content
		}
		if err := a.post(ctx, webhookURL, payload); err != nil {
			return err
		}
	}

	return nil
}

func (a *discordImpl) post(ctx context.Context, webhookURL string, payload *discordPayload) error {
	params, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal discord payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(params))
	if err != nil {
		return fmt.Errorf("create discord request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("post to discord: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		resBody, _ := io.ReadAll(io.LimitReader(res.Body, 1024))
		return fmt.Errorf("discord returned status %d: %s", res.StatusCode, strings.TrimSpace(string(resBody)))
	}
	_, _ = io.Copy(io.Discard, res.Body)

	return nil
}

// hexToColor は `#4286f4` 形式の hex 文字列を Discord の整数指定へ変換する。
// 変換できない場合は 0(色指定なし)を返す。
func hexToColor(hex string) int {
	c, err := strconv.ParseUint(strings.TrimPrefix(hex, "#"), 16, 32)
	if err != nil || c > 0xFFFFFF {
		return 0
	}
	return int(c)
}

// truncateRunes は文字数(rune 数)が limit を超える場合に切り詰める。
// []rune への変換によるアロケーションを避けるため、byte offset で走査する。
func truncateRunes(s string, limit int) string {
	count := 0
	for i := range s {
		if count == limit {
			return s[:i]
		}
		count++
	}
	return s
}
