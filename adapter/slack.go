package adapter

type SlackAdapter interface {
}

type slackImpl struct {
}

// NewSlackImplAdapter access to slack
func NewSlackImplAdapter() SlackAdapter {
	return &slackImpl{}
}
