# analytics_notifications_slack
[![serverless-dev](https://github.com/limit7412/analytics_notifications_slack/actions/workflows/serverless-dev.yml/badge.svg?branch=develop)](https://github.com/limit7412/analytics_notifications_slack/actions/workflows/serverless-dev.yml)

googleアナリティクスのpvを集計してランキングを作成して投稿するslackbot

`NOTIFY_MODE: discord` を設定するとエラー通知(Alert)も含めてdiscordへ投稿できる

## deploy
  - 事前にserverlessからawsに接続を確立する
  - serverless-plugin-scripts をインストール
    - `npm ci`
  - 以下の2つのファイルを用意
    - ./secret.json
      - googleアナリティクスapiへのアクセス用
    - ./env.yml
      - 環境変数を定義しserverless.ymlに渡すためのyml
  - `sls deploy --stage <環境名>`
    - serverless-plugin-scripts により、パッケージング時にローカルで `bootstrap` がビルドされる

### env.yml
```
  GOOGLE_APPLICATION_CREDENTIALS: secret.json
  PROPERTY_ID: <対象にしたいGA4のプロパティID。カンマ区切りで複数指定可>
  NOTIFY_MODE: <投稿先。slack または discord。未設定時は slack>
  SUCCESS_WEBHOOK_URL: <集計結果を投稿するwebhook>
  SUCCESS_FALLBACK: <投稿時に通知に表示するテキスト>
  FAILD_WEBHOOK_URL: <エラー時に通知をするwebhook>
  FAILD_FALLBACK: <エラーを投稿すつ際に通知に表示するテキスト>
  TITLE_SPLIT: <ここに指定した文字列以降を無視する>
```

エラー時は全体通知(slack: `@channel`、discord: `@everyone`)でメンションされる

discordモードの場合は `SUCCESS_WEBHOOK_URL` / `FAILD_WEBHOOK_URL` にdiscordのwebhook URLを設定する

## ディレクトリ構成

ディレクトリはレイヤーではなくドメインで分けている。通知先を追加するときは、ディレクトリを一つ足して `cmd/bootstrap/main.go` の分岐を増やす。

```
cmd/bootstrap/  Lambda のエントリポイント。NOTIFY_MODE に応じて通知先を選ぶ
analytics/      Google アナリティクス(GA4 Data API)から PV を取得して集計する
notify/         通知先に依存しない Message と、その送信インターフェース Poster、
                ランキングを組み立てて投稿するユースケース
alert/          処理失敗を通知するユースケース
slack/          Poster の Slack 実装
discord/        Poster の Discord 実装
```

依存は次の向きに限り、循環しない。

- `notify` → `analytics`
- `alert`、`slack`、`discord` → `notify`
- `analytics` は他のパッケージに依存しない
