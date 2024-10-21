GO_MOD_DIRS := $(shell find . -not -path "./vendor/*" -type f -name 'go.mod' -exec dirname {} \; | sort)

GOTEST_MIN = go test -v -timeout 90s
GOTEST = $(GOTEST_MIN) -race
GOTEST_WITH_COVERAGE = $(GOTEST) -coverprofile=coverage.txt -covermode=atomic

.PHONY: up
up: ## Run containers and print logs in stdout
	$(info Make: Starting containers...)
	@TAG=$(TAG) docker compose up --build
	@make -s logs

.PHONY: down
down: ## Stop containers
	$(info Make: Stopping containers...)
	@TAG=$(TAG) docker compose down

.PHONY: logs
logs: ## Print logs in stdout
	@TAG=$(TAG) docker compose logs

.PHONY: test
test: ## Run tests
	set -e; for dir in $(GO_MOD_DIRS); do \
	  (cd "$${dir}" && \
	    $(GOTEST) ./...); \
	done

.PHONY: test-with-coverage
test-with-coverage: ## Run tests with coverage
	set -e; for dir in $(GO_MOD_DIRS); do \
	  (cd "$${dir}" && \
	    $(GOTEST_WITH_COVERAGE) ./... && \
	    go tool cover -html=coverage.txt -o coverage.html); \
	done

.PHONY: tidy
tidy: ## Run tidy
	set -e; for dir in $(ALL_GO_MOD_DIRS); do \
	  (cd "$${dir}" && \
	    go mod tidy); \
	done

.PHONY: vendor
vendor: ## Run vendor
	set -e; for dir in $(ALL_GO_MOD_DIRS); do \
	  (cd "$${dir}" && \
	    go mod vendor); \
	done