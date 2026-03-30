# Design principles for the scaffold

## 1. Map, not manual

Keep always-on guidance small.
Put detail where it can be loaded on demand.

## 2. Repository legibility matters

The easiest repo for a coding agent is:
- searchable
- explicit
- strongly named
- low on hidden rules
- rich in versioned contracts

## 3. Plans are first-class artifacts

Plans should survive:
- context compaction
- session boundaries
- reviewer handoff
- human interruptions

## 4. Evidence beats confidence

Any agent can sound certain.
Fewer agents can show:
- tests
- logs
- screenshots
- traces
- review findings
- coverage gaps

## 5. Prose is not a hard guarantee

Use prose for:
- conventions
- strategy
- heuristics

Use code for:
- safety gates
- invariant checks
- verification
- structure tests
- CI enforcement

## 6. Start with the cheapest loop that works

Default:
- single session
- file-backed plan
- deterministic verify step
- written review artifact

Escalate only when justified.

## 7. Preserve optionality

The scaffold should support:
- multiple languages
- multiple agents
- multiple vendors
- multiple maturity levels

## 8. Optimize for fast inner loop and strict outer loop

Inner loop:
- hooks
- local scripts
- targeted tests

Outer loop:
- CI
- broader integration checks
- remote review or security scanning

## 9. Human attention is scarce

Good harnesses remove avoidable interruptions.
Humans should spend time on:
- judgment
- approvals
- design taste
- risk trade-offs
- exception handling

## 10. The harness itself needs auditing

Over time, all harnesses accumulate:
- stale instructions
- duplicated rules
- weak triggers
- ritual without value

Audit the harness like production code.
