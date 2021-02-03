#!/bin/bash

stg=${1}
[ "$stg" = "" ] && stg="dev"

region="ap-northeast-1"
container="analytics_notifications_slack_$stg"

docker build -t $container .
docker run --name $container -d $container /bin/sh
docker cp $container:/work/bootstrap .
docker rm $container

sls deploy --stage $stg