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
  PROFILE_ID: <対象にしたいgoogleアナリティクスのプロファイルID>
  NOTIFY_MODE: <投稿先。slack または discord。未設定時は slack>
  SUCCESS_WEBHOOK_URL: <集計結果を投稿するwebhook>
  SUCCESS_FALLBACK: <投稿時に通知に表示するテキスト>
  FAILD_WEBHOOK_URL: <エラー時に通知をするwebhook>
  FAILD_FALLBACK: <エラーを投稿すつ際に通知に表示するテキスト>
  MENTION_ID: <エラー時に通知(メンション)をするユーザーid。未設定時は SLACK_ID にフォールバック>
  SLACK_ID: <エラー時に通知をするslackのユーザーid(後方互換用。MENTION_ID を推奨)>
  TITLE_SPLIT: <ここに指定した文字列以降を無視する>
```

discordモードの場合は `SUCCESS_WEBHOOK_URL` / `FAILD_WEBHOOK_URL` にdiscordのwebhook URLを、`MENTION_ID` にdiscordのユーザーidを設定する
