package usecase

type NotifyUsecase interface {
}

type notifyImpl struct {
}

// NewNotifyImplUsecase notification analytics to slack
func NewNotifyImplUsecase() NotifyUsecase {
	return &notifyImpl{}
}
