#!/usr/bin/env sh
set -eu

if [ "${1:-}" = "" ]; then
  echo "Usage: ./scripts/new-language-pack.sh <language-name>"
  exit 1
fi

lang="$1"
dir="packs/languages/$lang"

if [ -e "$dir" ]; then
  echo "Language pack already exists: $dir"
  exit 1
fi

mkdir -p "$dir"

cat > "$dir/README.md" <<EOF
# $lang pack

Describe the conventions and verification flow for $lang here.

Recommended contents:
- naming and layout rules
- stack-specific contracts
- verification commands
- common failure modes
EOF

cat > "$dir/verify.sh" <<EOF
#!/usr/bin/env sh
set -eu

echo "Customize packs/languages/$lang/verify.sh for your stack."
exit 2
EOF
chmod +x "$dir/verify.sh"

echo "Created $dir"
echo "Remember to add or update a matching .claude/rules/$lang.md file."
