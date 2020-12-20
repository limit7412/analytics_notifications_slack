FROM golang:latest as build-image

WORKDIR /work
COPY ./ ./

RUN go build handler/main.go

FROM public.ecr.aws/lambda/go:1

COPY --from=build-image /work/ /var/task/

CMD ["main"]
