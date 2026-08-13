.PHONY: build dev-install dev-list test test-integration test-integration-ubuntu test-integration-arch test-all clean lint lint-shell lint-all

build:
	go build -o bin/dotfiles .

test:
	go test ./...

# Run the working tree against ITSELF rather than the installed clone.
#
# DOTFILES_DIR decides which repo's modules, profiles and state are used — the binary's
# own location is irrelevant. Unset, it defaults to ~/.dotfiles, so `go run .` from a
# development checkout silently operates on the installed clone: you edit a module here,
# run it here, and watch the old definition execute. These targets remove the choice.
#
# For testing a *fresh* install, use test-integration-* instead: those run in a container
# and cannot damage your real home directory.
dev-install: build
	DOTFILES_DIR=$(CURDIR) ./bin/dotfiles install $(ARGS)

dev-list: build
	DOTFILES_DIR=$(CURDIR) ./bin/dotfiles list

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
