#!/usr/bin/env sh
set -eu

OUTPUT_PATH="${1:-}"

emit() {
  printf '%s\n' "$1"
}

run_optional() {
  emit ""
  emit "## $1"
  shift
  if "$@" >/tmp/loong64-b1-go-stage7.out 2>/tmp/loong64-b1-go-stage7.err; then
    cat /tmp/loong64-b1-go-stage7.out
  else
    cat /tmp/loong64-b1-go-stage7.out 2>/dev/null || true
    cat /tmp/loong64-b1-go-stage7.err 2>/dev/null || true
  fi
  rm -f /tmp/loong64-b1-go-stage7.out /tmp/loong64-b1-go-stage7.err
}

generate_report() {
  emit "# Stage 7 Deployment Verification"
  emit ""
  emit "date: $(date -Iseconds)"
  emit "host: $(hostname 2>/dev/null || echo unknown)"
  emit "arch: $(uname -m)"

  run_optional "uname -a" uname -a
  run_optional "/etc/os-release" sh -c "cat /etc/os-release"
  run_optional "glibc" sh -c "ldd --version | head -n 1"
  run_optional "go version" sh -c "go version"
  run_optional "psql version" sh -c "psql --version"
  run_optional "systemd version" sh -c "systemctl --version | head -n 1"
  run_optional "font match" sh -c "fc-match 'Noto Sans CJK SC' || fc-match 'Source Han Sans SC' || true"
  run_optional "service status" sh -c "systemctl status loong64-b1-go.service --no-pager || true"
}

if [ -n "$OUTPUT_PATH" ]; then
  generate_report >"$OUTPUT_PATH"
  echo "Wrote deployment environment report to $OUTPUT_PATH"
else
  generate_report
fi
