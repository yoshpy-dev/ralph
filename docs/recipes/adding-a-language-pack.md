# Adding a language pack

1. Create a pack:

   ```sh
   ./scripts/new-language-pack.sh go
   ```

2. Customize:
   - `packs/languages/go/README.md`
   - `packs/languages/go/verify.sh`
   - `.claude/rules/go.md`

3. Add the relevant project commands.
4. Test the pack by running `./scripts/run-verify.sh`.
5. Document any required environment or toolchain assumptions.

Keep the pack focused on:
- verification
- common contracts
- naming and structure conventions
- language-specific failure modes
