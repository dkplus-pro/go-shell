#!/usr/bin/env bash
# Fast local verification that mirrors CI checks without forcing dependency install.

set -Eeuo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/lib/ci-common.sh
source "$SCRIPT_DIR/lib/ci-common.sh"

ROOT="$(repo_root)"
cd "$ROOT"

PM="$(detect_package_manager)"
ci_notice "Detected package manager: $PM"

for script in lint typecheck test; do
  ci_group "Run $script"
  run_package_script_if_present "$PM" "$script"
  ci_endgroup
done

if [[ -f "$ROOT/go.work" ]]; then
  if ! has_command go; then
    ci_error "go.work exists but the 'go' toolchain was not found; Go >= 1.24 is required."
    exit 1
  fi
  # The repo root itself is not a Go module (go.work only references sub-module
  # directories), so run the checks inside every module listed by go.work.
  ci_group "Run Go checks (vet/build/test)"
  go_modules_found=0
  while IFS= read -r mod_dir; do
    [[ -n "$mod_dir" ]] || continue
    go_modules_found=1
    (cd "$mod_dir" && go vet ./... && go build ./... && go test ./...)
  done < <(go_work_use_dirs "$ROOT/go.work")
  if [[ "$go_modules_found" -ne 1 ]]; then
    ci_error "go.work lists no modules; cannot run Go checks."
    exit 1
  fi
  ci_endgroup
fi
