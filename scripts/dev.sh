#!/usr/bin/env sh
set -eu

project_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd -P)
cd "$project_root"
export LOCAL_UID="$(id -u)" LOCAL_GID="$(id -g)"
mkdir -p .cache/home .cache/go-build .cache/go-mod .cache/go

compose() {
    docker compose -p stars-admin-dev -f deployments/compose.yaml "$@"
}

ensure_tools() {
    if ! docker image inspect stars-admin-dev:go1.27.4 >/dev/null 2>&1; then
        compose build dev
    fi
}

target=${1:-run}
shift || true
case "$target" in
    dev-env) compose up -d --wait postgres redis ;;
    dev-down) compose down ;;
    tools) compose build dev ;;
    test-integration)
        ensure_tools
        export STARS_DATABASE_PASSWORD=stars-test-only-password
        export STARS_DATABASE_DBNAME=stars_admin_test
        # 单独的测试项目不发布端口，清理仅针对测试项目。
        test_compose() {
            docker compose -p stars-admin-test -f deployments/compose.yaml -f deployments/compose.test.yaml "$@"
        }
        trap 'test_compose down -v --remove-orphans' EXIT HUP INT TERM
        STARS_DATABASE_DBNAME=stars_admin_test test_compose up -d --wait postgres redis
        test_compose run --rm --no-deps dev make test-integration
        ;;
    run) ensure_tools; compose run --rm --no-deps --service-ports dev make run "$@" ;;
    build|swag|lint|test|fmt|tidy|migrate-up|migrate-down|migrate-status)
        ensure_tools
        compose run --rm --no-deps dev make "$target" "$@"
        ;;
    verify)
        "$0" lint
        "$0" build
        "$0" test
        "$0" test-integration
        ;;
    *) printf '%s\n' "未知命令：$target" >&2; exit 2 ;;
esac
