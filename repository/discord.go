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
	"unicode/utf8"
)

// Discord webhook の制約値。
// https://discord.com/developers/docs/resources/webhook
const (
	// 1メッセージあたりの embeds の最大数
	discordMaxEmbeds = 10
	// content の最大文字数
	discordMaxContentLen = 2000
	// embed title の最大文字数
	discordMaxTitleLen = 256
	// embed description の最大文字数
	discordMaxDescriptionLen = 4096
	// embed footer text の最大文字数
	discordMaxFooterLen = 2048
	// 1メッセージ内の全 embed の文字数合計の上限
	discordMaxTotalLen = 6000
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
		if msg.Mention && !mentioned {
			contentParts = append(contentParts, "@everyone")
			mentioned = true
		}
		if msg.Pretext != "" {
			contentParts = append(contentParts, msg.Pretext)
		}
		if msg.Title == "" && msg.Text == "" && msg.Footer == "" {
			continue
		}

		title := truncateRunes(msg.Title, discordMaxTitleLen)
		var footer *discordFooter
		footerText := ""
		if msg.Footer != "" {
			footerText = truncateRunes(msg.Footer, discordMaxFooterLen)
			footer = &discordFooter{Text: footerText}
		}
		// 単体の embed でもメッセージ全体の文字数合計上限に収まるよう、
		// title / footer を差し引いた分まで description を切り詰める。
		descLimit := min(discordMaxDescriptionLen, discordMaxTotalLen-runeCount(title)-runeCount(footerText))
		embeds = append(embeds, &discordEmbed{
			Title:       title,
			Description: truncateRunes(msg.Text, descLimit),
			Color:       hexToColor(msg.Color),
			Footer:      footer,
		})
	}

	content := truncateRunes(strings.Join(contentParts, " "), discordMaxContentLen)
	if content == "" && len(embeds) == 0 {
		return nil
	}

	// embeds は1メッセージ最大10件・文字数合計最大6000字のため、
	// 超える場合はチャンクして複数回送信する。content は先頭のメッセージにのみ載せる。
	chunks := [][]*discordEmbed{}
	chunk := []*discordEmbed{}
	chunkLen := 0
	for _, embed := range embeds {
		size := embedRuneCount(embed)
		if len(chunk) > 0 && (len(chunk) >= discordMaxEmbeds || chunkLen+size > discordMaxTotalLen) {
			chunks = append(chunks, chunk)
			chunk, chunkLen = nil, 0
		}
		chunk = append(chunk, embed)
		chunkLen += size
	}
	if len(chunk) > 0 {
		chunks = append(chunks, chunk)
	}

	for first := true; first || len(chunks) > 0; first = false {
		payload := &discordPayload{}
		if len(chunks) > 0 {
			payload.Embeds = chunks[0]
			chunks = chunks[1:]
		}
		if first {
			payload.Content = content
		}
		if err := a.post(ctx, webhookURL, payload); err != nil {
			return err
		}
	}

	return nil
}

// embedRuneCount は embed が文字数合計上限に対して占める文字数を返す。
func embedRuneCount(e *discordEmbed) int {
	count := runeCount(e.Title) + runeCount(e.Description)
	if e.Footer != nil {
		count += runeCount(e.Footer.Text)
	}
	return count
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

// runeCount は文字数(rune 数)を返す。
func runeCount(s string) int {
	return utf8.RuneCountInString(s)
}
