package adapter

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"
)

type SlackAdapter interface {
	Post(msg *Post) error
}

type slackImpl struct {
}

// NewSlackAdapter access to slack
func NewSlackAdapter() SlackAdapter {
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

func (a *slackImpl) Post(msg *Post) error {
	params, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	payload := url.Values{"payload": {string(params)}}
	res, err := http.PostForm(os.Getenv("WEB_HOOK_URL"), payload)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	return nil
}
