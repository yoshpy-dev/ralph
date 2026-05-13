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
  emit golang
fi

if [ -f pubspec.yaml ] || find . -type f -name '*.dart' | grep -q .; then
  emit dart
fi

if [ -f pom.xml ] || [ -f build.gradle ] || [ -f build.gradle.kts ]; then
  emit jvm
fi

if [ -f .terraform.lock.hcl ] || find . -type d -name .terraform -prune -o -type f \( -name '*.tf' -o -name '*.tofu' \) -print 2>/dev/null | grep -q .; then
  emit terraform
fi
