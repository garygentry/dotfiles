.PHONY: build test test-integration-ubuntu test-integration-arch test-all clean lint

build:
	go build -o bin/dotfiles .

test:
	go test ./...

test-integration-ubuntu:
	docker build -t dotfiles-test-ubuntu -f test/integration/Dockerfile.ubuntu .
	docker run --rm dotfiles-test-ubuntu

test-integration-arch:
	docker build -t dotfiles-test-arch -f test/integration/Dockerfile.arch .
	docker run --rm dotfiles-test-arch

test-all: test test-integration-ubuntu test-integration-arch

clean:
	rm -rf bin/

lint:
	go vet ./...
