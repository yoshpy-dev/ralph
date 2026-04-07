# Definition of done

A task is done only when all applicable items are satisfied.

## For non-trivial code changes

- [ ] Active plan exists or was explicitly deemed unnecessary
- [ ] Acceptance criteria were addressed
- [ ] Self-review artifact exists (diff quality)
- [ ] Verification was run and recorded (spec compliance + static analysis)
- [ ] Test artifact exists (behavioral tests pass)
- [ ] Docs and contracts were updated if behavior changed
- [ ] Remaining gaps are explicit
- [ ] PR created via /pr skill (includes plan archival and hand-off)
- [ ] CI verify passes on the PR

## For risky or broad changes

Add:
- [ ] Walkthrough included in PR or `docs/reports/`
- [ ] Rollback note or recovery path
- [ ] Known follow-ups or tech debt recorded

## For docs-only changes

- [ ] Source of truth is aligned
- [ ] No commands or workflows became stale
- [ ] Any changed process still matches scripts and rules
