package main

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/limit7412/analytics_notifications_slack/alert"
	"github.com/limit7412/analytics_notifications_slack/analytics"
	"github.com/limit7412/analytics_notifications_slack/discord"
	"github.com/limit7412/analytics_notifications_slack/notify"
	"github.com/limit7412/analytics_notifications_slack/slack"
)

// 依存は遅延初期化し、Lambda のウォームスタート間でキャッシュ・再利用する
// ことで、呼び出しごとの接続・認証情報の再確立コストを避ける。
var (
	poster        notify.Poster
	analyticsRepo analytics.Repository
)

// Response はハンドラーの実行結果を表す。
// テスト等で実行結果を判定できるよう、成功時にも返却する。
type Response struct {
	Message string `json:"message"`
}

// newPoster は NOTIFY_MODE(slack | discord、未設定時は slack)に
// 応じた通知先(Poster)を生成する。
func newPoster() notify.Poster {
	if os.Getenv("NOTIFY_MODE") == "discord" {
		return discord.NewRepository()
	}
	return slack.NewRepository()
}

// Handler は `lambda.Start` から呼び出される Lambda ハンドラー。
// スケジュール(EventBridge)起動のため、入力ペイロードは受け取らない。
func Handler(ctx context.Context) (Response, error) {
	if poster == nil {
		poster = newPoster()
	}
	// 失敗通知は集計に依存しないため、analytics の初期化より先に用意する。
	alerter := alert.NewUsecase(poster)

	if analyticsRepo == nil {
		// キャッシュするサービスを単一呼び出しの(キャンセルされうる)コンテキストに
		// 紐付けないよう、background コンテキストで生成する。
		repo, err := analytics.NewRepository(context.Background())
		if err != nil {
			err = fmt.Errorf("init analytics repository: %w", err)
			alerter.Notify(ctx, err)
			return Response{}, err
		}
		analyticsRepo = repo
	}

	app := notify.NewUsecase(analyticsRepo, poster)
	if err := app.Run(ctx); err != nil {
		alerter.Notify(ctx, err)
		return Response{}, err
	}

	return Response{Message: "success"}, nil
}

func main() {
	lambda.Start(Handler)
}
