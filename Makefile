.PHONY: test build run
test:
	go test ./... -count=1
build:
	go build -o bin/gfireui-backend ./cmd/gfireui-backend
run: build
	./bin/gfireui-backend
