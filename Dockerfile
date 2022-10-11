FROM golang:latest as build-image

WORKDIR /work
COPY ./ ./

RUN GOOS=linux GOARCH=amd64 go build handler/main.go