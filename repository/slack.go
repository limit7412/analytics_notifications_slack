package repository

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type SlackRepository interface {
	Post(path string, msg []*Post) error
}

type slackImpl struct {
}

// NewSlackRepository access to slack
func NewSlackRepository() SlackRepository {
	return &slackImpl{}
}

type Post struct {
	Fallback string `json:"fallback"`
	Pretext  string `json:"pretext"`
	Title    string `json:"title"`
	Text     string `json:"text"`
	Color    string `json:"color"`
	Footer   string `json:"footer"`
}

type payload struct {
	Attachments []*Post `json:"attachments"`
}

func (a *slackImpl) Post(path string, msg []*Post) error {
	params, err := json.Marshal(payload{
		Attachments: msg,
	})
	if err != nil {
		return err
	}
	payload := url.Values{"payload": {string(params)}}
	fmt.Print(payload)
	res, err := http.PostForm(path, payload)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	return nil
}
