GO_FILES = $$(find cmd internal pkg migrations configs -name '*.go' -type f)
COVER_DIR = .cache/coverage
COVER_FLAGS = -covermode=atomic -coverpkg=./internal/...,./pkg/...

.PHONY: run build swag lint test test-integration fmt tidy migrate-up migrate-down migrate-status dev-env dev-down seed
.PHONY: test-cover integration-cover coverage test-migrate check-swag vuln package
.PHONY: verify-local build-all

swag:
	swag init -g main.go -d cmd/server,internal --parseInternal --parseDependency --parseDepth 1 --overridesFile .swaggo --exclude .cache,.git,tmp,bin -o docs/swagger --outputTypes go,json,yaml

build: swag
	mkdir -p bin
	CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o bin/stars-admin ./cmd/server

build-all: swag
	go build ./...

# 候选包仅写本地忽略目录，不上传远程、不创建 tag 或发布。
package: swag
	mkdir -p .cache/release/linux-amd64/configs
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags='-s -w' -o .cache/release/linux-amd64/stars-admin ./cmd/server
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags='-s -w' -o .cache/release/linux-amd64/probe ./cmd/probe
	cp configs/config.yaml configs/config.prod.yaml configs/config.test.yaml .cache/release/linux-amd64/configs/
	cp README.md CHANGELOG.md LICENSE .cache/release/linux-amd64/
	tar -czf .cache/release/stars-admin-candidate-linux-amd64.tar.gz -C .cache/release/linux-amd64 stars-admin probe README.md CHANGELOG.md LICENSE configs/config.yaml configs/config.prod.yaml configs/config.test.yaml
	cd .cache/release && sha256sum stars-admin-candidate-linux-amd64.tar.gz > SHA256SUMS

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

# 本机检查不调用 Docker，不连接数据库/Redis；集成测试须另备隔离服务。
verify-local:
	$(MAKE) lint
	$(MAKE) build
	$(MAKE) test
	go build ./...

test-cover: swag
	mkdir -p $(COVER_DIR)
	go test -count=1 -race $(COVER_FLAGS) -coverprofile=$(COVER_DIR)/unit.out ./...

integration-cover: swag
	@test "$(STARS_INTEGRATION)" = "1" || { echo '覆盖率集成测试只能在隔离环境执行'; exit 1; }
	mkdir -p $(COVER_DIR)
	go test -count=1 -p 1 -race -tags=integration $(COVER_FLAGS) -coverprofile=$(COVER_DIR)/integration.out ./...

coverage:
	go run ./cmd/coverage -out $(COVER_DIR)/merged.out -report $(COVER_DIR)/service.md $(COVER_DIR)/unit.out $(COVER_DIR)/integration.out

test-migrate: swag
	@test "$(STARS_INTEGRATION)" = "1" || { echo '迁移验证只能在隔离环境执行'; exit 1; }
	go test -count=1 -race -tags=integration -run '^TestMigrationLifecycleAndConstraints$$' ./internal/core/database

check-swag: swag
	go test -count=1 ./internal/app -run '^TestSwagger'

vuln: swag
	govulncheck -show verbose ./...

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
