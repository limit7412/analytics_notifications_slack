FROM golang:latest as build-image

WORKDIR /go/work
COPY ./ ./

RUN go build handler/main.go

FROM public.ecr.aws/lambda/go:1

COPY --from=build-image /go/work/ /var/task/

CMD ["main"]
