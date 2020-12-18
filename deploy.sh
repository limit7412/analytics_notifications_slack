#!/bin/bash

account=${1}
stg=${2}
[ "$stg" = "" ] && stg="dev"

region="ap-northeast-1"

aws ecr get-login-password --region $region                                         |
docker login --username AWS --password-stdin $account.dkr.ecr.$region.amazonaws.com

container="analytics_notifications_slack_$stg"
target="$account.dkr.ecr.ap-northeast-1.amazonaws.com/$container"

docker build -t $container .
docker tag $container:latest $target:latest
docker push $target:latest

digest=$(aws ecr list-images --repository-name $container | jq '.imageIds[] | select(.imageTag=="latest") | .imageDigest' | tr -d '"')

sls deploy --account $account --stage $stg --digest $digest