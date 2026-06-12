package main

import (
	"context"
	"fmt"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/limit7412/analytics_notifications_slack/repository"
	"github.com/limit7412/analytics_notifications_slack/usecase"
)

// リポジトリは遅延初期化し、Lambda のウォームスタート間でキャッシュ・再利用する
// ことで、呼び出しごとの接続・認証情報の再確立コストを避ける。
var (
	slackRepo     repository.SlackRepository
	analyticsRepo repository.AnalyticsRepository
)

// Response はハンドラーの実行結果を表す。
// テスト等で実行結果を判定できるよう、成功時にも返却する。
type Response struct {
	Message string `json:"message"`
}

// Handler は `lambda.Start` から呼び出される Lambda ハンドラー。
// スケジュール(EventBridge)起動のため、入力ペイロードは受け取らない。
func Handler(ctx context.Context) (Response, error) {
	if slackRepo == nil {
		slackRepo = repository.NewSlackRepository()
	}

	if analyticsRepo == nil {
		// キャッシュするサービスを単一呼び出しの(キャンセルされうる)コンテキストに
		// 紐付けないよう、background コンテキストで生成する。
		repo, err := repository.NewAnalyticsRepository(context.Background())
		if err != nil {
			err = fmt.Errorf("init analytics repository: %w", err)
			// slackRepo は生成済みなので、この経路でも失敗通知を送る。
			usecase.NewNotifyUsecase(nil, slackRepo).Error(ctx, err)
			return Response{}, err
		}
		analyticsRepo = repo
	}

	app := usecase.NewNotifyUsecase(analyticsRepo, slackRepo)
	if err := app.Run(ctx); err != nil {
		app.Error(ctx, err)
		return Response{}, err
	}

	return Response{Message: "success"}, nil
}

func main() {
	lambda.Start(Handler)
}
