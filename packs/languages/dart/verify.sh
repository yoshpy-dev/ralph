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

roots_file="$(mktemp "${TMPDIR:-/tmp}/ralph-dart-roots.XXXXXX")"
cleanup() {
  rm -f "$roots_file"
}
trap cleanup EXIT HUP INT TERM

find_project_roots() {
  find . \
    \( -type d \( \
      -name .git -o \
      -name .dart_tool -o \
      -name build -o \
      -name .pub-cache \
    \) -prune \) -o \
    -type f -name pubspec.yaml -print 2>/dev/null |
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
  echo "Skipping Dart verifier: no Dart project roots found."
  exit 0
fi

status=0

# Detect Flutter vs pure Dart project
select_tool() {
  is_flutter=false
  if grep -q 'flutter:' pubspec.yaml 2>/dev/null; then
    is_flutter=true
  fi

  if [ "$is_flutter" = true ]; then
    if ! command -v flutter >/dev/null 2>&1; then
      echo "flutter is required for Flutter project verification."
      return 1
    fi
    tool="flutter"
  else
    if ! command -v dart >/dev/null 2>&1; then
      echo "dart is required for Dart verification."
      return 1
    fi
    tool="dart"
  fi
}

run_static() {
  # Format check
  if command -v dart >/dev/null 2>&1; then
    dart format --output=none --set-exit-if-changed . || status=1
  elif [ "$tool" = "flutter" ]; then
    flutter format --output=none --set-exit-if-changed . || status=1
  fi

  # Static analysis
  "$tool" analyze --fatal-infos || status=1

  # Run code generation if build_runner is a dependency
  if grep -q 'build_runner:' pubspec.yaml 2>/dev/null; then
    echo "build_runner detected. Run 'dart run build_runner build' if generated files are stale."
  fi
}

run_tests() {
  if [ -d test ]; then
    "$tool" test || status=1
  else
    echo "Skipping tests: test/ directory not found."
  fi
}

verify_root() {
  project_root="$1"
  echo "==> Dart project root: $project_root"

  status=0
  if ! select_tool; then
    return 1
  fi

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
