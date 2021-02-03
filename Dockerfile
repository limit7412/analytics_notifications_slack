FROM golang:latest as build-image

WORKDIR /work
COPY ./ ./

RUN go build handler/main.go