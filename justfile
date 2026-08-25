# List available recipes
[group('help')]
default:
    @just --list

# Run the basic HTTP server example
[group('run')]
examples:
    go run ./examples/basic

# Build the library and executable examples
[group('quality')]
build:
    go build ./...

# Run all unit and integration tests
[group('tests')]
test:
    go test ./...

# Run all tests with Go's race detector
[group('tests')]
test-race:
    go test -race ./...

# Report statement coverage for the library package
[group('tests')]
coverage:
    go test -coverprofile=coverage.out .
    go tool cover -func=coverage.out

# Check that go.mod and go.sum are tidy without changing them
[group('quality')]
mod-tidy-check:
    go mod tidy -diff

# Run all repository quality checks
[group('quality')]
check: fmt-check mod-tidy-check vet test build

# Format Go code
[group('quality')]
fmt:
    go fmt ./...

# Check that Go code is formatted
[group('quality')]
fmt-check:
    @files="$(gofmt -l $(rg --files -g '*.go'))"; if [ -n "$files" ]; then printf '%s\n' "$files"; exit 1; fi

# Run go vet
[group('quality')]
vet:
    go vet ./...
