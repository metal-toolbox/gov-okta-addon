all: lint test
PHONY: test coverage lint golint clean vendor docker-up docker-down unit-test
GOOS=linux

# OAuth client generated secret
SECRET := $(shell bash -c 'echo $$RANDOM|md5')

test: | unit-test

unit-test: | lint
	@echo Running unit tests...
	@go test -cover -short -tags testtools ./...

coverage:
	@echo Generating coverage report...
	@go test ./... -race -coverprofile=coverage.out -covermode=atomic -tags testtools -p 1
	@go tool cover -func=coverage.out
	@go tool cover -html=coverage.out

lint: golint

golint: | vendor
	@echo Linting Go files...
	@golangci-lint run --build-tags "-tags testtools"

build:
	@go mod download
	@CGO_ENABLED=0 GOOS=linux go build -mod=readonly -v -o gov-okta-addon

clean: docker-clean
	@echo Cleaning...
	@rm -rf ./dist/
	@rm -rf coverage.out
	@go clean -testcache

vendor:
	@go mod download
	@go mod tidy

docker-up: | build
	@docker-compose -f docker-compose.yml up -d gov-okta-addon

docker-down:
	@docker-compose -f docker-compose.yml down

docker-clean:
	@docker-compose -f docker-compose.yml down --volumes
