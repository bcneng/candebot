.PHONY: help lint clean build test run simulator

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

lint: ## Run linter
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.6.0 run

clean: ## Remove build artifacts
	@rm -f candebot

build: clean ## Build the bot
	go build -v -ldflags "-X main.Version=$$(git rev-parse --short HEAD)" .
	@echo candebot built and ready to serve and protect.

test: ## Run tests
	go test -v ./...

run: ## Run the bot locally
	go run .

simulator: ## Open browser to simulator
	open http://localhost:8080/_simulator/
