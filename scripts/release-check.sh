#!/usr/bin/env bash
# ============================================================
# landc-go 发布前冻结检查脚本（对应清单 9.1）
#
# 用法：
#   GOLANGCI=/path/to/golangci-lint GOVULNCHECK=/path/to/govulncheck \
#     bash scripts/release-check.sh
#
# 工具版本要求（与 CI 一致，见 docs/dependencies.md）：
#   - golangci-lint v2.12.2（v1.x 无法解析 go 1.26 模块）
#   - govulncheck v1.6.0+
# 退出码：0 = 全部通过；非 0 = 存在阻断项
# ============================================================
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MODULES=(log tools api frame workflow saas)
GO_CMD="${GO_CMD:-go}"
GOLANGCI="${GOLANGCI:-golangci-lint}"
GOVULNCHECK="${GOVULNCHECK:-govulncheck}"
REQUIRED_GO="1.26"

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; NC='\033[0m'
fail=0
check() { # $1=描述 $2=结果(0/1)
  if [ "$2" -eq 0 ]; then printf "${GREEN}✓${NC} %s\n" "$1"; else printf "${RED}✗${NC} %s\n" "$1"; fail=1; fi
}

echo "== 1. git status 干净 =="
if [ -z "$(cd "$ROOT" && git status --porcelain)" ]; then check "git working tree clean" 0; else check "git working tree clean" 1; fi

echo "== 2. make build / vet / test =="
(cd "$ROOT" && make build >/dev/null 2>&1); check "make build" $?
(cd "$ROOT" && make vet >/dev/null 2>&1); check "make vet" $?
(cd "$ROOT" && make test >/dev/null 2>&1); check "make test (-race)" $?

echo "== 3. golangci-lint 0 error =="
lint_ok=0
for m in "${MODULES[@]}"; do
  if (cd "$ROOT/$m" && "$GOLANGCI" run ./... >/dev/null 2>&1); then :; else lint_ok=1; fi
done
check "golangci-lint all modules" $((lint_ok == 0 ? 0 : 1))

echo "== 4. govulncheck 无 Critical/High =="
vuln_ok=0
for m in "${MODULES[@]}"; do
  if (cd "$ROOT/$m" && "$GOVULNCHECK" ./... >/dev/null 2>&1); then :; else vuln_ok=1; fi
done
check "govulncheck all modules" $((vuln_ok == 0 ? 0 : 1))

echo "== 5. go mod verify =="
verify_ok=0
for m in "${MODULES[@]}"; do
  if (cd "$ROOT/$m" && "$GO_CMD" mod verify >/dev/null 2>&1); then :; else verify_ok=1; fi
done
check "go mod verify all modules" $((verify_ok == 0 ? 0 : 1))

echo "== 6. go 指令统一 =="
go_directive_ok=0
for m in "${MODULES[@]}"; do
  got="$(sed -n 's/^go[[:space:]]*//p' "$ROOT/$m/go.mod" | head -1)"
  case "$got" in
    "$REQUIRED_GO"|"$REQUIRED_GO."*) : ;;
    *) printf "${RED}    %s go directive = %s (want %s)${NC}\n" "$m" "${got:-<none>}" "$REQUIRED_GO"; go_directive_ok=1 ;;
  esac
done
if [ -f "$ROOT/go.work" ]; then
  wg="$(sed -n 's/^go[[:space:]]*//p' "$ROOT/go.work" | head -1)"
  case "$wg" in
    "$REQUIRED_GO"|"$REQUIRED_GO."*) : ;;
    *) printf "${RED}    go.work go directive = %s (want %s)${NC}\n" "$wg" "$REQUIRED_GO"; go_directive_ok=1 ;;
  esac
fi
check "go directives = $REQUIRED_GO" $((go_directive_ok == 0 ? 0 : 1))

echo "== 7. 无未关闭 release-blocker =="
# 提示性检查：列出 GH issue 需人工确认（此处仅提示）
printf "${YELLOW}ℹ${NC} 请人工确认无未关闭 release-blocker issue（GitHub）\n"

echo
if [ "$fail" -eq 0 ]; then
  printf "${GREEN}全部检查通过，可以发布。${NC}\n"
else
  printf "${RED}存在阻断项（$fail 类），请先修复。${NC}\n"
fi
exit "$fail"
