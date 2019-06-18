package adapter

type AnalyticsAdapter interface {
}

type analyticsImpl struct {
}

// NewAnalyticsImplAdapter access to analytics
func NewAnalyticsImplAdapter() AnalyticsAdapter {
	return &analyticsImpl{}
}
