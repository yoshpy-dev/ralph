#!/usr/bin/env sh
set -eu

mkdir -p .harness/state .harness/logs

ran_any=0
status=0

if [ -x ./scripts/verify.local.sh ]; then
  echo "==> Running local verifier"
  ran_any=1
  if ! ./scripts/verify.local.sh; then
    status=1
  fi
fi

languages="$(./scripts/detect-languages.sh || true)"
for lang in $languages; do
  verifier="packs/languages/$lang/verify.sh"
  if [ -x "$verifier" ]; then
    echo "==> Running $lang verifier"
    ran_any=1
    if ! "$verifier"; then
      status=1
    fi
  fi
done

changed_files=""
if command -v git >/dev/null 2>&1; then
  changed_files="$( (git diff --name-only 2>/dev/null; git diff --name-only --cached 2>/dev/null) | sort -u )"
fi

docs_only=1
if [ -n "$changed_files" ]; then
  printf '%s\n' "$changed_files" | while IFS= read -r file; do
    case "$file" in
      ""|docs/*|README.md|AGENTS.md|CLAUDE.md|.claude/*)
        ;;
      *)
        echo "$file" > .harness/state/non_docs_change
        ;;
    esac
  done
  if [ -f .harness/state/non_docs_change ]; then
    docs_only=0
    rm -f .harness/state/non_docs_change
  fi
fi

if [ "$ran_any" -eq 0 ]; then
  if [ "$docs_only" -eq 1 ]; then
    echo "No language verifier ran. This appears to be docs or scaffold-level work only."
    exit 0
  fi
  echo "No verifier ran for code-like changes."
  echo "Add a real verifier in ./scripts/verify.local.sh or packs/languages/<name>/verify.sh."
  exit 2
fi

exit "$status"
