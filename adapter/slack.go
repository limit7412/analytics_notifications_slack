package adapter

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type SlackAdapter interface {
	Post(msg []Post) error
}

type slackImpl struct {
	url string
}

// NewSlackAdapter access to slack
func NewSlackAdapter(url string) SlackAdapter {
	return &slackImpl{url: url}
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
	Attachments []Post `json:"attachments"`
}

func (a *slackImpl) Post(msg []Post) error {
	params, err := json.Marshal(payload{
		Attachments: msg,
	})
	if err != nil {
		return err
	}
	payload := url.Values{"payload": {string(params)}}
	fmt.Print(payload)
	res, err := http.PostForm(a.url, payload)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	return nil
}
