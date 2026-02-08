.PHONY: build test lint clean

build:
	go build -o hcpt .

test:
	go test ./...

lint:
	golangci-lint run

clean:
	rm -f hcpt
