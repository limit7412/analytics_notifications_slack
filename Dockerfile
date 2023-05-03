FROM golang:latest as build-image

WORKDIR /work
COPY ./ ./

RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build handler/main.go