.PHONY: build test test-integration test-integration-ubuntu test-integration-arch test-all clean lint lint-shell lint-all

build:
	go build -o bin/dotfiles .

test:
	go test ./...

test-integration-ubuntu:
	DOCKER_BUILDKIT=1 docker build -t dotfiles-test-ubuntu -f test/integration/Dockerfile.ubuntu .
	docker run --rm dotfiles-test-ubuntu

test-integration-arch:
	DOCKER_BUILDKIT=1 docker build -t dotfiles-test-arch -f test/integration/Dockerfile.arch .
	docker run --rm dotfiles-test-arch

test-integration: test-integration-ubuntu test-integration-arch

test-all: test test-integration

clean:
	rm -rf bin/
	-docker rmi dotfiles-test-ubuntu dotfiles-test-arch 2>/dev/null

lint:
	go vet ./...

lint-shell:
	@echo "Linting shell scripts with shellcheck..."
	@find modules lib -name "*.sh" -type f -print0 | xargs -0 shellcheck --severity=warning || true
	@echo "Checking for errors (will fail on errors)..."
	@find modules lib -name "*.sh" -type f -print0 | xargs -0 shellcheck --severity=error

lint-all: lint lint-shell
