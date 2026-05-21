#!/usr/bin/env sh
set -eu

seen=""

emit() {
  name="$1"
  case " $seen " in
    *" $name "*) ;;
    *)
      seen="$seen $name"
      printf '%s\n' "$name"
      ;;
  esac
}

has_file() {
  marker="$(find . \
    \( -type d \( \
      -name .git -o \
      -name node_modules -o \
      -name .pnpm-store -o \
      -name .yarn -o \
      -name .venv -o \
      -name venv -o \
      -name env -o \
      -name __pycache__ -o \
      -name .mypy_cache -o \
      -name .pytest_cache -o \
      -name .ruff_cache -o \
      -name target -o \
      -name .dart_tool -o \
      -name .terraform -o \
      -name .terragrunt-cache -o \
      -name .harness -o \
      -name coverage -o \
      -name dist -o \
      -name build \
    \) -prune \) -o \
    -type f \( "$@" \) -print -quit 2>/dev/null)"
  [ -n "$marker" ]
}

if has_file -name package.json -o -name tsconfig.json -o -name 'tsconfig.*.json' -o -name '*.ts' -o -name '*.tsx'; then
  emit typescript
fi

if has_file -name pyproject.toml -o -name requirements.txt -o -name 'requirements-*.txt' -o -name setup.py -o -name tox.ini -o -name '*.py'; then
  emit python
fi

if has_file -name Cargo.toml -o -name '*.rs'; then
  emit rust
fi

if has_file -name go.mod; then
  emit golang
fi

if has_file -name pubspec.yaml -o -name '*.dart'; then
  emit dart
fi

if has_file -name '.terraform.lock.hcl' -o -name '*.tf' -o -name '*.tofu'; then
  emit terraform
fi
