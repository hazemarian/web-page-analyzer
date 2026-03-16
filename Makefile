.PHONY: run build test test-e2e down logs lint deps

## env: create .env from .env.example if it does not exist
.env:
	cp .env.example .env

## run: build image and start all services (app + redis)
run: .env
	docker compose up --build

## build: build the Docker image only
build:
	docker compose build

## test: run unit and integration tests (no external dependencies needed)
test:
	go test $(shell go list ./... | grep -v /e2e) -v -race -count=1

## test-e2e: run end-to-end tests (requires running stack via make run)
test-e2e:
	go test ./e2e/ -v -race -count=1

## down: stop and remove all containers
down:
	docker compose down

## logs: tail application logs
logs:
	docker compose logs -f app

## lint: run golangci-lint
lint:
	golangci-lint run ./...

## deps: download and tidy dependencies (run once after fresh clone)
deps:
	go mod tidy
