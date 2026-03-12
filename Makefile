.PHONY: build test lint lint-fix fmt tidy install clean ci setup check-tidy version

# Root module is "." — sub-modules are in their own directories
ROOT_MODULE := .
SUB_MODULES := cmd
ALL_MODULES := $(ROOT_MODULE) $(SUB_MODULES)

define run_in_module
	@for m in $(ALL_MODULES); do \
		echo "→ $(1) $$m"; \
		if [ "$$m" = "." ]; then \
			$(2) \
		else \
			(cd $$m && $(3)); \
		fi \
	done
endef

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
	@golangci-lint run --config .golangci.yml ./...
	@for m in $(SUB_MODULES); do \
		echo "→ Linting $$m"; \
		(cd $$m && golangci-lint run --config ../.golangci.yml ./...); \
	done

lint-fix:
	@golangci-lint run --fix --config .golangci.yml ./...
	@for m in $(SUB_MODULES); do \
		echo "→ Lint-fixing $$m"; \
		(cd $$m && golangci-lint run --fix --config ../.golangci.yml ./...); \
	done

fmt:
	@gofmt -w .
	@for m in $(SUB_MODULES); do \
		echo "→ Formatting $$m"; \
		(cd $$m && gofmt -w .); \
	done

tidy:
	@go mod tidy
	@for m in $(SUB_MODULES); do \
		echo "→ Tidying $$m"; \
		(cd $$m && go mod tidy); \
	done

install:
	cd cmd && go install .

clean:
	@find . -name "coverage.out" -delete
	@find . -name "dist" -type d -exec rm -rf {} + 2>/dev/null || true

ci: fmt lint check-tidy test

setup:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	go install github.com/evilmartians/lefthook@latest
	lefthook install

check-tidy:
	@go mod tidy && git diff --exit-code go.mod go.sum
	@for m in $(SUB_MODULES); do \
		echo "→ Checking tidy $$m"; \
		(cd $$m && go mod tidy && git diff --exit-code go.mod go.sum); \
	done

version:
	@git describe --tags --abbrev=0 2>/dev/null || echo "no tags found"
