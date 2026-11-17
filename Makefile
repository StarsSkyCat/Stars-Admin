GO_FILES = $$(find cmd internal pkg migrations configs -name '*.go' -type f)

.PHONY: run build swag lint test test-integration fmt tidy migrate-up migrate-down migrate-status dev-env dev-down seed

swag:
	swag init -g main.go -d cmd/server,internal --parseInternal --parseDependency --parseDepth 1 --overridesFile .swaggo --exclude .cache,.git,tmp,bin -o docs/swagger --outputTypes go,json,yaml

build: swag
	mkdir -p bin
	CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o bin/stars-admin ./cmd/server

run: swag
	air -c .air.toml

fmt:
	gofmt -w $(GO_FILES)
	goimports -w $(GO_FILES)

lint: swag
	@test -z "$$(gofmt -l $(GO_FILES))" || { gofmt -l $(GO_FILES); exit 1; }
	@test -z "$$(goimports -l $(GO_FILES))" || { goimports -l $(GO_FILES); exit 1; }
	golangci-lint run

test: swag
	go test -race ./...

test-integration: swag
	@test "$(STARS_INTEGRATION)" = "1" || { echo '集成测试只能在隔离测试环境中执行'; exit 1; }
	go test -count=1 -p 1 -race -tags=integration ./...

tidy: swag
	go mod tidy

migrate-up: build
	./bin/stars-admin migrate up

migrate-down: build
	./bin/stars-admin migrate down

migrate-status: build
	./bin/stars-admin migrate status

seed: build
	./bin/stars-admin seed

dev-env:
	./scripts/dev.sh dev-env

dev-down:
	./scripts/dev.sh dev-down
