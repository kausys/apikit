.PHONY: build test test-coverage lint lint-fix fmt tidy install clean ci setup work check-tidy version

# Root module is "." — sub-modules are in their own directories
SUB_MODULES := cmd

# Keep in sync with the version in .github/workflows/lint.yml — otherwise local
# lint and CI can disagree about the same tree.
GOLANGCI_LINT_VERSION := v2.14.0

build:
	@go build ./...
	@for m in $(SUB_MODULES); do \
		echo "→ Building $$m"; \
		(cd $$m && go build ./...); \
	done

test:
	@go test -race ./...
	@for m in $(SUB_MODULES); do \
		echo "→ Testing $$m"; \
		(cd $$m && go test -race ./...); \
	done

test-coverage:
	@go test -race -coverprofile=coverage.out ./...
	@for m in $(SUB_MODULES); do \
		echo "→ Testing $$m (coverage)"; \
		(cd $$m && go test -race -coverprofile=coverage.out ./...); \
	done

lint:
	@golangci-lint run --config $(CURDIR)/.golangci.yml ./...
	@for m in $(SUB_MODULES); do \
		echo "→ Linting $$m"; \
		(cd $$m && golangci-lint run --config $(CURDIR)/.golangci.yml ./...); \
	done

lint-fix:
	@golangci-lint run --fix --config $(CURDIR)/.golangci.yml ./...
	@for m in $(SUB_MODULES); do \
		echo "→ Lint-fixing $$m"; \
		(cd $$m && golangci-lint run --fix --config $(CURDIR)/.golangci.yml ./...); \
	done

fmt:
	@gofmt -w .

# Each module is tidied as its users see it: without the workspace.
tidy:
	@GOWORK=off go mod tidy
	@for m in $(SUB_MODULES); do \
		echo "→ Tidying $$m"; \
		(cd $$m && GOWORK=off go mod tidy); \
	done

install:
	cd cmd && go install ./apikit

clean:
	@find . -name "coverage.out" -delete
	@find . -name "dist" -type d -exec rm -rf {} + 2>/dev/null || true

ci: fmt lint check-tidy test

setup: work
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	go install github.com/evilmartians/lefthook@latest
	lefthook install

# cmd requires the last release of the root module. The workspace builds and
# tests it against the root module in this tree instead; it stays local
# (.gitignore), because a go.work inside a module zip breaks its users.
work:
	@test -f go.work || go work init . $(SUB_MODULES)

check-tidy:
	@GOWORK=off go mod tidy && git diff --exit-code -- go.mod go.sum
	@for m in $(SUB_MODULES); do \
		echo "→ Checking tidy $$m"; \
		(cd $$m && GOWORK=off go mod tidy && git diff --exit-code -- go.mod go.sum); \
	done

version:
	@git describe --tags --abbrev=0 2>/dev/null || echo "no tags found"
