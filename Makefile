.PHONY: build clean deploy

build:
	env GOOS=linux CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/handler handler/main.go

clean:
	rm -rf ./bin

deploy: clean build
	sls deploy --verbose
