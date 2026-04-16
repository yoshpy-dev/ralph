# Tech debt

Record debt that should not disappear into chat history.

Recommended fields:
- debt item
- impact
- why it was deferred
- trigger for paying it down
- related plan or report

## Entries

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| Signal test does not exercise `_INTERRUPTED` gating logic | The core fix (flag-based status update in `cleanup_on_exit`) has no dedicated test | The dry-run + SIGINT test is timing-dependent and may be flaky; improving it requires careful process lifecycle management | If the `_INTERRUPTED` logic regresses in a future change | `docs/reports/self-review-2026-04-15-ralph-pipeline-hardening-v2.md` |
| Per-slice pipelines do NOT stop on CRITICAL self-review findings | Differs from standard `/work` flow behavior | Autonomous pipelines benefit from letting verify/test confirm true positives before halting | If false-negative CRITICAL findings slip through to merge | `.claude/rules/post-implementation-pipeline.md` |
