# analytics_notifications_slack
![go-sls](https://github.com/limit7412/analytics_notifications_slack/workflows/go-sls/badge.svg)

googleアナリティクスのpvを集計してランキングを作成して投稿するslackbot

## deploy
  - 事前にserverlessからawsに接続を確立する
  - 以下の2つのファイルを用意
    - ./secret.json
      - googleアナリティクスapiへのアクセス用
    - ./env.yml
      - 環境変数を定義しserverless.ymlに渡すためのyml
  - ./deploy.sh <account_id> <環境名>

### env.yml
```
  GOOGLE_APPLICATION_CREDENTIALS: secret.json
  PROFILE_ID: <対象にしたいgoogleアナリティクスのプロファイルID>
  SUCCESS_WEBHOOK_URL: <集計結果を投稿するwebhook>
  SUCCESS_FALLBACK: <投稿時に通知に表示するテキスト>
  FAILD_WEBHOOK_URL: <エラー時に通知をするwebhook>
  FAILD_FALLBACK: <エラーを投稿すつ際に通知に表示するテキスト>
  SLACK_ID: <エラー時に通知をするslackのユーザーid>
```
