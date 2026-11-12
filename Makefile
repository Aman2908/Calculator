# All targets run with local Go (1.26+) and Node (24+) toolchains.
# Without them, use `make docker-build docker-run`.

IMAGE ?= calculator:latest
PORT  ?= 8080

.PHONY: help install dev dev-api dev-web test test-api test-web coverage lint gen build docker-build docker-run clean

help: ## Show available targets
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "  %-14s %s\n", $$1, $$2}'

install: ## Install frontend dependencies and download Go modules
	cd frontend && npm ci
	cd backend && go mod download

dev: ## Run API (:8080) and Vite dev server (:5173) together
	@$(MAKE) -j2 dev-api dev-web

dev-api:
	cd backend && go run ./cmd/server

dev-web:
	cd frontend && npm run dev

test: test-api test-web ## Run all unit tests

test-api:
	cd backend && go test -race ./...

test-web:
	cd frontend && npm test

coverage: ## Produce coverage reports in ./coverage
	mkdir -p coverage/backend
	cd backend && go test -race -coverprofile=../coverage/backend/coverage.out ./... && \
		go tool cover -func=../coverage/backend/coverage.out | tail -1 && \
		go tool cover -html=../coverage/backend/coverage.out -o ../coverage/backend/index.html
	cd frontend && npm run coverage

lint: ## Lint and format-check both projects
	cd backend && gofmt -l . | (! grep .) && go vet ./... && golangci-lint run
	cd frontend && npm run typecheck && npm run lint

gen: ## Regenerate TypeScript API types from api/openapi.yaml
	cd frontend && npm run gen:api

build: ## Build the SPA and the server binary locally
	cd frontend && npm run build
	cd backend && CGO_ENABLED=0 go build -o server ./cmd/server

docker-build: ## Build the single full-stack image
	docker build -t $(IMAGE) .

docker-run: ## Run the image on http://localhost:$(PORT)
	docker run --rm -p $(PORT):8080 $(IMAGE)

clean:
	rm -rf coverage frontend/dist backend/server
