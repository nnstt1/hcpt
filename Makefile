.PHONY: build test lint clean

build:
	go build -ldflags "-X github.com/nnstt1/hcpt/internal/version.Version=dev" -o hcpt .

test:
	go test ./...

lint:
	golangci-lint run

clean:
	rm -f hcpt
