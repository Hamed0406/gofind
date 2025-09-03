.PHONY: build test clean

build:
	mkdir -p bin
	go build -o bin/gofind ./cmd/gofind

test:
	go test ./... -v

clean:
	rm -rf bin
