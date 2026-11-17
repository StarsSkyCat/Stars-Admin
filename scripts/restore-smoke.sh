#!/usr/bin/env sh
set -eu
# 本脚本只接受由 smoke.sh 创建的可丢弃项目，不能用于真实环境恢复。
test "${1:-}" = stars-admin-smoke || { printf '%s\n' '仅允许隔离 smoke 恢复演练' >&2; exit 2; }
cd "$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd -P)"
compose() { docker compose -p stars-admin-smoke -f deployments/compose.yaml -f deployments/compose.test.yaml "$@"; }
archive_dir=$(mktemp -d "$PWD/.cache/restore-smoke.XXXXXX")
chmod 700 "$archive_dir"
trap 'rm -rf "$archive_dir"; docker volume rm stars-admin-smoke-restored >/dev/null 2>&1 || true' EXIT HUP INT TERM
compose exec -T postgres pg_dump -U stars -d stars_admin_smoke -Fc > "$archive_dir/database.dump"
compose exec -T postgres createdb -U stars stars_admin_restore
compose exec -T postgres pg_restore -U stars -d stars_admin_restore --exit-on-error --no-owner --no-acl < "$archive_dir/database.dump"
query="SELECT version::text || ':' || dirty::text FROM schema_migrations UNION ALL SELECT 'admin:' || id::text || ':' || auth_version::text FROM sys_user WHERE id=1 UNION ALL SELECT 'tasks:' || count(*)::text FROM job_task UNION ALL SELECT 'logs:' || count(*)::text FROM job_log ORDER BY 1"
original=$(compose exec -T postgres psql -U stars -d stars_admin_smoke -At -v ON_ERROR_STOP=1 -c "$query")
restored=$(compose exec -T postgres psql -U stars -d stars_admin_restore -At -v ON_ERROR_STOP=1 -c "$query")
test "$original" = "$restored"
docker run --rm --user 65532:65532 --mount type=volume,source=stars-admin-smoke-uploads,target=/uploads \
    --entrypoint sh stars-admin-dev:go1.27.4 -c 'printf "%s" "隔离恢复演练" > /uploads/restore-fixture.txt; tar -C /uploads -cf - .' > "$archive_dir/uploads.tar"
docker volume create stars-admin-smoke-restored >/dev/null
docker run --rm -i --mount type=volume,source=stars-admin-smoke-restored,target=/restored \
    --entrypoint sh stars-admin-dev:go1.27.4 -c 'tar -C /restored -xpf -' < "$archive_dir/uploads.tar"
docker run --rm --user 65532:65532 --mount type=volume,source=stars-admin-smoke-uploads,target=/original,readonly \
    --mount type=volume,source=stars-admin-smoke-restored,target=/restored,readonly \
    --entrypoint sh stars-admin-dev:go1.27.4 -c 'diff -r /original /restored; test "$(stat -c %u:%g /restored/restore-fixture.txt)" = 65532:65532; test "$(stat -c %a /original/restore-fixture.txt)" = "$(stat -c %a /restored/restore-fixture.txt)"'
printf '%s\n' '隔离备份恢复演练通过：PG 结构/数据/版本与上传卷夹具内容/UID/GID/权限一致'
