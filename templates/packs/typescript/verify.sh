#!/usr/bin/env sh
set -eu

mode="${HARNESS_VERIFY_MODE:-all}"
case "$mode" in
  static|test|all) ;;
  *)
    echo "Unknown HARNESS_VERIFY_MODE: $mode" >&2
    exit 2
    ;;
esac

roots_file="$(mktemp "${TMPDIR:-/tmp}/ralph-typescript-roots.XXXXXX")"
cleanup() {
  rm -f "$roots_file"
}
trap cleanup EXIT HUP INT TERM

find_project_roots() {
  find . \
    \( -type d \( \
      -name .git -o \
      -name node_modules -o \
      -name .pnpm-store -o \
      -name .yarn -o \
      -name .cache -o \
      -name .turbo -o \
      -name .next -o \
      -name coverage -o \
      -name dist -o \
      -name build \
    \) -prune \) -o \
    -type f \( \
      -name package.json -o \
      -name package-lock.json -o \
      -name pnpm-lock.yaml -o \
      -name yarn.lock -o \
      -name tsconfig.json -o \
      -name 'tsconfig.*.json' \
    \) -print 2>/dev/null |
    while IFS= read -r marker; do
      dirname "$marker"
    done |
    sort -u
}

root_selected() {
  [ -z "${RALPH_VERIFY_PROJECT_ROOTS:-}" ] && return 0
  root="${1#./}"
  [ -n "$root" ] || root="."

  for selected in $RALPH_VERIFY_PROJECT_ROOTS; do
    selected="${selected#./}"
    [ -n "$selected" ] || selected="."
    [ "$root" = "$selected" ] && return 0
  done
  return 1
}

find_project_roots |
  while IFS= read -r project_root; do
    if root_selected "$project_root"; then
      printf '%s\n' "$project_root"
    fi
  done > "$roots_file"

if [ ! -s "$roots_file" ]; then
  echo "Skipping TypeScript verifier: no TypeScript project roots found."
  exit 0
fi

has_script() {
  grep -q "\"$1\"[[:space:]]*:" package.json 2>/dev/null
}

run_script() {
  script="$1"
  case "$pm" in
    npm)
      npm run "$script" --if-present
      ;;
    pnpm)
      if has_script "$script"; then
        pnpm run "$script"
      else
        echo "Skipping $script: script not defined."
      fi
      ;;
    yarn)
      if has_script "$script"; then
        yarn "$script"
      else
        echo "Skipping $script: script not defined."
      fi
      ;;
  esac
}

run_static() {
  run_script lint || status=1
  run_script typecheck || status=1
}

run_tests() {
  run_script test || status=1
}

verify_root() {
  project_root="$1"
  echo "==> TypeScript project root: $project_root"

  if [ ! -f package.json ]; then
    echo "Skipping TypeScript project root $project_root: package.json not found."
    return 0
  fi

  pm="npm"
  if [ -f pnpm-lock.yaml ] && command -v pnpm >/dev/null 2>&1; then
    pm="pnpm"
  elif [ -f yarn.lock ] && command -v yarn >/dev/null 2>&1; then
    pm="yarn"
  elif ! command -v npm >/dev/null 2>&1; then
    echo "No supported package manager found for TypeScript verification."
    return 1
  fi

  status=0
  case "$mode" in
    static) run_static ;;
    test)   run_tests ;;
    all)    run_static; run_tests ;;
  esac

  return "$status"
}

overall_status=0
while IFS= read -r project_root; do
  [ -n "$project_root" ] || continue
  if ! (cd "$project_root" && verify_root "$project_root"); then
    overall_status=1
  fi
done < "$roots_file"

exit "$overall_status"
