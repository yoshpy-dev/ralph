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
| Per-slice pipelines do NOT stop on CRITICAL self-review findings | Differs from standard `/work` flow behavior | Autonomous pipelines benefit from letting verify/test confirm true positives before halting | If false-negative CRITICAL findings slip through to merge | `.claude/rules/post-implementation-pipeline.md` |
