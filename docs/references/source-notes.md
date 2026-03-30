# Source notes

The scaffold was informed by the following sources and ideas.

## Official sources

- OpenAI: Harness engineering for Codex
  - https://openai.com/ja-JP/index/harness-engineering/
  - Key ideas: short map file, repo-as-system-of-record, first-class plans, observability, agent legibility

- OpenAI: Unlocking the Codex harness
  - https://openai.com/ja-JP/index/unlocking-the-codex-harness/
  - Key ideas: reusable runtime, thread persistence, tool and sandbox integration

- Anthropic: Harness design for long-running application development
  - https://www.anthropic.com/engineering/harness-design-long-running-apps
  - Key ideas: context resets vs compaction, planner-generator-evaluator, explicit grading, simplify when possible

- Claude Code docs
  - https://code.claude.com/docs/ja/features-overview
  - https://code.claude.com/docs/ja/memory
  - https://code.claude.com/docs/ja/skills
  - https://code.claude.com/docs/ja/sub-agents
  - https://code.claude.com/docs/en/hooks
  - https://code.claude.com/docs/ja/agent-teams
  - Key ideas: layered control plane using CLAUDE.md, rules, skills, subagents, hooks, teams

## Practitioner sources

- Mitchell Hashimoto: My AI Adoption Journey
  - https://mitchellh.com/writing/my-ai-adoption-journey
  - Key ideas: separate planning from execution, give the agent verification

- Geoffrey Huntley: Ralph Wiggum as a software engineer
  - https://ghuntley.com/ralph/
  - Key ideas: continuous loop, persistent task files, greenfield bias, heavy tuning

- nyosegawa articles
  - https://nyosegawa.com/posts/skill-creator-and-orchestration-skill/
  - https://nyosegawa.com/posts/claude-code-verify-command/
  - https://nyosegawa.com/posts/skill-auditor/
  - https://nyosegawa.com/posts/coding-agent-workflow-2026/
  - Key ideas: orchestration skill design, anti-human-bottleneck triggers, description tuning, skill portfolio audits

- ignission article
  - https://zenn.dev/ignission/articles/f1c15646c990f1
  - Key ideas: four-layer harness, hooks plus local plus CI layering, promote hard rules out of prose

- Chachamaru127/claude-code-harness
  - https://github.com/Chachamaru127/claude-code-harness
  - Key ideas: stage-gated workflow, runtime guardrails, evidence pack, runtime enforcement over prompt-only skill packs
