#!/usr/bin/env sh
set -eu
project_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd -P)
cd "$project_root"

# 公开的隔离测试凭据，只在本次创建的 smoke 项目中使用。
export STARS_DATABASE_DBNAME=stars_admin_smoke
export STARS_DATABASE_PASSWORD=smoke-only-database-password
export STARS_JWT_SECRET=smoke-only-jwt-secret-with-at-least-32-bytes
export STARS_SEED_ADMIN_PASSWORD=Smoke-Initial-Password-123
compose() {
    docker compose -p stars-admin-smoke -f deployments/compose.yaml -f deployments/compose.test.yaml "$@"
}
cleanup() {
    docker rm -f stars-admin-smoke-api >/dev/null 2>&1 || true
    docker volume rm stars-admin-smoke-uploads >/dev/null 2>&1 || true
    compose down -v --remove-orphans
}
trap cleanup EXIT HUP INT TERM

./scripts/dev.sh tools
docker build -f deployments/Dockerfile -t stars-admin:m5 .
compose up -d --wait postgres redis

runtime() {
    docker run --rm --network stars-admin-smoke_default \
        -e APP_ENV=prod -e STARS_DATABASE_HOST=postgres \
        -e STARS_DATABASE_DBNAME -e STARS_DATABASE_PASSWORD -e STARS_JWT_SECRET \
        -e STARS_DATABASE_SSLMODE=disable \
        -e STARS_REDIS_TIMEOUT=invalid -e STARS_JWT_ACCESS_TTL=invalid -e STARS_APP_READ_TIMEOUT=invalid -e STARS_JOB_TIMEOUT=invalid -e STARS_GEN_LOCAL_ENABLED=invalid "$@"
}
# 故意提供不可达 Redis，证明迁移入口不依赖 Redis。
runtime -e STARS_REDIS_ADDR=unreachable.invalid:6379 -e STARS_SEED_ADMIN_PASSWORD stars-admin:m5 migrate up
runtime -e STARS_REDIS_ADDR=unreachable.invalid:6379 stars-admin:m5 migrate status
runtime -e STARS_REDIS_ADDR=unreachable.invalid:6379 stars-admin:m5 migrate down
runtime -e STARS_REDIS_ADDR=unreachable.invalid:6379 stars-admin:m5 migrate up
# 仅隔离冒烟项目关闭图片验证码，正式迁移默认开启。
compose exec -T postgres psql -U stars -d stars_admin_smoke -v ON_ERROR_STOP=1 -c "UPDATE sys_config SET value='false' WHERE key='sys.account.captchaEnabled'"
compose exec -T postgres psql -U stars -d stars_admin_smoke -v ON_ERROR_STOP=1 < scripts/m4-smoke-fixture.sql

docker run -d --name stars-admin-smoke-api --network stars-admin-smoke_default \
    --mount type=volume,source=stars-admin-smoke-uploads,target=/app/uploads \
    -e APP_ENV=prod -e STARS_DATABASE_HOST=postgres \
    -e STARS_DATABASE_DBNAME -e STARS_DATABASE_PASSWORD -e STARS_JWT_SECRET \
    -e STARS_DATABASE_SSLMODE=disable -e STARS_REDIS_ADDR=redis:6379 stars-admin:m5

docker run --rm --network stars-admin-smoke_default --entrypoint sh stars-admin-dev:go1.27.4 -c '
    attempt=0
    until curl -fsS http://stars-admin-smoke-api:8080/api/health/ready >/dev/null; do
        attempt=$((attempt + 1))
        if [ "$attempt" -ge 30 ]; then exit 1; fi
        sleep 1
    done
    curl -fsS http://stars-admin-smoke-api:8080/api/ping
    test "$(curl -s -o /dev/null -w "%{http_code}" http://stars-admin-smoke-api:8080/swagger/index.html)" = 404
'
smoke_client() {
    docker run --rm --network stars-admin-smoke_default --user "$(id -u):$(id -g)" \
        -v "$project_root:/workspace" -w /workspace \
        -e GOCACHE=/workspace/.cache/go-build -e GOMODCACHE=/workspace/.cache/go-mod \
        -e GOPATH=/workspace/.cache/go -e STARS_SEED_ADMIN_PASSWORD \
        -e STARS_SMOKE_URL=http://stars-admin-smoke-api:8080 \
        stars-admin-dev:go1.27.4 go run ./cmd/smoke "$@"
}
smoke_client
runtime -e STARS_REDIS_ADDR=unreachable.invalid:6379 stars-admin:m5 migrate up
smoke_client after-migrate
test "$(docker inspect --format '{{.Config.User}}' stars-admin-smoke-api)" = "65532:65532"
docker exec stars-admin-smoke-api /app/probe
docker stop --time 15 stars-admin-smoke-api >/dev/null
test "$(docker inspect --format '{{.State.ExitCode}}' stars-admin-smoke-api)" = "0"
test "$(compose exec -T postgres psql -U stars -d stars_admin_smoke -At -c "SELECT count(*) FROM job_log WHERE name='shutdown-smoke' AND status=2 AND error_msg='任务已取消' AND params NOT LIKE '%smoke-secret-not-for-logs%'")" = "1"
docker logs stars-admin-smoke-api
./scripts/restore-smoke.sh stars-admin-smoke
printf '%s\n' '生产镜像、迁移独立性、认证与幂等初始化、Swagger 关闭、M4 ZIP 与任务取消日志排空及 SIGTERM 冒烟通过'
