.PHONY: build test run
build:
	go build -trimpath -o bin/iptv-spider .
test:
	go test -race ./...
run:
	go run .
