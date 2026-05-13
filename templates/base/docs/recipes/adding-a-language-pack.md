# Adding a language pack

Worked example: adding a Terraform / OpenTofu pack (`packs/languages/terraform/`).

1. Scaffold the pack:

   ```sh
   ./scripts/new-language-pack.sh terraform
   ```

2. Customize the pack body (`packs/languages/terraform/`):
   - `README.md` — verification order, activation rules, customization points
   - `verify.sh` — honor `HARNESS_VERIFY_MODE` (`static` / `test` / `all`); skip when no marker files; fail when markers exist but the required CLI is missing (avoid fail-open)

3. Mirror the pack into the embedded templates (required for `ralph init` / `PackFS`):

   ```sh
   cp -r packs/languages/terraform templates/packs/terraform
   ```

   `scripts/check-sync.sh` enforces byte-identical mirroring.

4. Add a rule file scoped to the language's file globs:
   - `.claude/rules/terraform.md` — minimal conventions (state hygiene, provider pinning, `variable`/`output` discipline, etc.)
   - Mirror to `templates/base/.claude/rules/terraform.md` (same `cp` pattern) so scaffolded projects receive it.

5. Teach `scripts/detect-languages.sh` to emit the new pack name and
   `scripts/detect-changed-languages.sh` to map changed files to it. Edit both
   the root and `templates/base/` copies — this is a hand-edit, no scaffolding
   script handles it for you:

   ```sh
   # In scripts/detect-languages.sh
   if [ -f .terraform.lock.hcl ] || find . -type d -name .terraform -prune \
       -o -type f \( -name '*.tf' -o -name '*.tofu' \) -print 2>/dev/null | grep -q .; then
     emit terraform
   fi

   # In scripts/detect-changed-languages.sh
   *.tf|*.tofu|*.tftest.hcl|.terraform.lock.hcl)
     printf 'terraform\n'
     ;;
   ```

6. Run the gates:

   ```sh
   ./scripts/check-sync.sh        # byte-identical mirroring
   ./scripts/check-skill-sync.sh  # skill drift (if you touched skills)
   ./scripts/run-verify.sh        # pack actually runs end-to-end
   ```

7. Document any required environment or toolchain assumptions in the pack's `README.md`.

Keep the pack focused on:
- verification
- common contracts
- naming and structure conventions
- language-specific failure modes

## Mirror checklist

A new pack typically touches at least these locations — keep them in lock-step:

- `packs/languages/<lang>/` ↔ `templates/packs/<lang>/`
- `.claude/rules/<lang>.md` ↔ `templates/base/.claude/rules/<lang>.md`
- `scripts/detect-languages.sh` ↔ `templates/base/scripts/detect-languages.sh`
- `scripts/detect-changed-languages.sh` ↔ `templates/base/scripts/detect-changed-languages.sh`

## Gitignore block (when your pack ships state, cache, or secret-bearing files)

If your pack documents files that "must never be committed" (e.g., `terraform.tfstate`, provider credentials, build caches that may capture environment), ship the matching `.gitignore` block in the same commit. A rule that exists only in prose (`.claude/rules/<lang>.md`) but is not enforced by `.gitignore` is a recurring leak vector — a routine `git add .` in a scaffolded project will stage exactly the files the rule warns about.

Mirror the block to both root and scaffold:

- `.gitignore` ↔ `templates/base/.gitignore`

Both files are checked for byte-identity by `scripts/check-sync.sh` (they appear in its `SCAN_FILES` list), so a missing mirror fails the static gate. The Terraform pack ships the canonical example (see `.gitignore:15-28`).
