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

if [ -f tsconfig.json ] || find . -type f \( -name '*.ts' -o -name '*.tsx' \) | grep -q .; then
  emit typescript
fi

if [ -f pyproject.toml ] || [ -f requirements.txt ] || [ -f setup.py ] || find . -type f -name '*.py' | grep -q .; then
  emit python
fi

if [ -f Cargo.toml ] || find . -type f -name '*.rs' | grep -q .; then
  emit rust
fi

if [ -f go.mod ]; then
  emit go
fi

if [ -f pom.xml ] || [ -f build.gradle ] || [ -f build.gradle.kts ]; then
  emit jvm
fi
