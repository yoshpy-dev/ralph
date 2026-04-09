#!/usr/bin/env sh
set -eu

echo "=== Harness scaffold bootstrap ==="
echo

# --- 1. Create required directories ---
mkdir -p .harness/state .harness/logs docs/plans/active docs/plans/archive docs/reports docs/evidence docs/tech-debt
echo "[ok] Required directories created."

# --- 2. Ensure all shell scripts are executable ---
for dir in scripts .claude/hooks packs/languages; do
  if [ -d "$dir" ]; then
    find "$dir" -type f -name '*.sh' ! -perm -u+x -exec chmod +x {} +
  fi
done
echo "[ok] Shell script permissions verified."

# --- 3. Validate settings.json exists (hooks config) ---
if [ -f .claude/settings.json ]; then
  echo "[ok] .claude/settings.json found (hooks and permissions active)."
else
  echo "[warn] .claude/settings.json not found — hooks will not be active."
  echo "       This file should be committed to git. Check if it was accidentally removed."
fi

# --- 4. Install commit-msg hook ---
if [ -d .git ]; then
  hook_src="scripts/commit-msg-guard.sh"
  hook_dst=".git/hooks/commit-msg"
  if [ -f "$hook_src" ]; then
    if [ ! -f "$hook_dst" ]; then
      cp "$hook_src" "$hook_dst"
      chmod +x "$hook_dst"
      echo "[ok] commit-msg hook installed."
    elif grep -q 'commit-msg-guard' "$hook_dst" 2>/dev/null; then
      cp "$hook_src" "$hook_dst"
      chmod +x "$hook_dst"
      echo "[ok] commit-msg hook updated."
    else
      echo "[skip] .git/hooks/commit-msg already exists (not ours). Skipping."
      echo "       To install manually: cp $hook_src $hook_dst"
    fi
  else
    echo "[warn] $hook_src not found. Skipping commit-msg hook install."
  fi
else
  echo "[skip] Not a git repository. Skipping commit-msg hook install."
fi

# --- 5. Run template structure check ---
if [ -x scripts/check-template.sh ]; then
  echo
  echo "--- Running template structure check ---"
  if ./scripts/check-template.sh; then
    echo "[ok] Template structure check passed."
  else
    echo "[warn] Template structure check found issues (see above)."
  fi
fi

echo
echo "Bootstrap complete."
echo
echo "Next steps:"
echo "  1. Edit AGENTS.md and CLAUDE.md for your project"
echo "  2. Customize .claude/rules/ and packs/languages/ as needed"
echo "  3. Create a plan: ./scripts/new-feature-plan.sh <slug>"
echo "  4. Personal overrides (extra permissions, etc): create .claude/settings.local.json (gitignored)"
