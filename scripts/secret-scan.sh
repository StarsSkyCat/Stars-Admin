#!/usr/bin/env sh
set -eu
cd "$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd -P)"
# 只输出文件名与提交号，禁止将匹配的凭据原文打印到 CI。
pattern='-----BEGIN (RSA |EC |OPENSSH |DSA )?PRIVATE KEY-----|AKIA[0-9A-Z]{16}|gh[pousr]_[A-Za-z0-9]{36,}|github_pat_[A-Za-z0-9_]{60,}'
failed=0
result=0
git grep --untracked --exclude-standard -I -l -E -e "$pattern" -- . || result=$?
if [ "$result" = 0 ]; then failed=1; elif [ "$result" != 1 ]; then exit "$result"; fi
for revision in $(git rev-list --all); do
    result=0
    git grep -I -l -E -e "$pattern" "$revision" -- . || result=$?
    if [ "$result" = 0 ]; then failed=1; elif [ "$result" != 1 ]; then exit "$result"; fi
done
if [ "$failed" != 0 ]; then
    printf '%s\n' '敏感信息规则扫描失败，请在受控环境核实列出的文件，勿打印匹配内容' >&2
    exit 1
fi
printf '%s\n' '工作区与可见 Git 历史的敏感信息规则扫描通过'
