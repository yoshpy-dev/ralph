#!/usr/bin/env sh
set -eu

if [ ! -f pubspec.yaml ]; then
  echo "Skipping Dart verifier: pubspec.yaml not found."
  exit 0
fi

# Detect Flutter vs pure Dart project
is_flutter=false
if grep -q 'flutter:' pubspec.yaml 2>/dev/null; then
  is_flutter=true
fi

if [ "$is_flutter" = true ]; then
  if ! command -v flutter >/dev/null 2>&1; then
    echo "flutter is required for Flutter project verification."
    exit 1
  fi
  tool="flutter"
else
  if ! command -v dart >/dev/null 2>&1; then
    echo "dart is required for Dart verification."
    exit 1
  fi
  tool="dart"
fi

status=0

# Format check
if command -v dart >/dev/null 2>&1; then
  dart format --set-exit-if-changed . || status=1
elif [ "$tool" = "flutter" ]; then
  flutter format --set-exit-if-changed . || status=1
fi

# Static analysis
$tool analyze --fatal-infos || status=1

# Run code generation if build_runner is a dependency
if grep -q 'build_runner:' pubspec.yaml 2>/dev/null; then
  echo "build_runner detected. Run 'dart run build_runner build' if generated files are stale."
fi

# Tests
if [ -d test ]; then
  $tool test || status=1
else
  echo "Skipping tests: test/ directory not found."
fi

exit "$status"
