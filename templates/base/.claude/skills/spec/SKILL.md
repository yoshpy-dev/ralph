---
name: spec
description: Turn vague ideas or abstract prompts into detailed, actionable specifications through codebase exploration, web research, and interactive clarification with the user. Manual trigger only.
disable-model-invocation: true
allowed-tools: Task, Read, Grep, Glob, Write, Edit, Bash, AskUserQuestion, WebSearch, WebFetch, Skill
---
Turn an abstract idea into a detailed specification.

## Role separation: /spec vs /plan

| Aspect | /spec | /plan |
|--------|-------|-------|
| Input | Vague idea, abstract prompt | Clear spec, defined task |
| Focus | **What** to build (requirements, constraints) | **How** to build (implementation strategy, files) |
| Output | Spec doc (`docs/specs/*.md`) / GitHub issue | Implementation plan (`docs/plans/active/*.md`) |
| Research | Codebase exploration, web search, best practices | Affected files, risk analysis |
| User interaction | Active clarification (requirements) | Flow selection (standard/Ralph) |

`/spec` comes before `/plan`. Use `/spec` when the request is too vague for `/plan`.

## Goals

- Transform ambiguous requests into implementation-ready specifications
- Discover requirements through codebase analysis and best-practice research
- Resolve ambiguity through targeted user questions
- Produce a versioned spec that survives context loss

## Steps

1. **Understand the request**: Read the user's input. Identify what is clear and what is ambiguous.

2. **Explore the codebase**: Use `Task(subagent_type="Explore")` to investigate:
   - Existing code related to the request
   - Current patterns and conventions
   - Potential impact areas and dependencies
   - Similar implementations that already exist

3. **Research best practices**: Use `WebSearch` and `WebFetch` to investigate:
   - Industry best practices for the problem domain
   - Common approaches and trade-offs
   - Reference implementations in well-known projects
   - Relevant library/framework documentation (via Context7 MCP if applicable)

4. **Clarify requirements**: Use `AskUserQuestion` actively to resolve:
   - Ambiguous or underspecified aspects
   - Trade-off decisions (e.g., simplicity vs extensibility)
   - Priority and scope boundaries
   - User preferences and constraints
   - Repeat this step as many times as needed. Do not guess when you can ask.

5. **Draft the spec** (in memory — do NOT write to file yet):
   - Compose the full spec content from [template.md](template.md) using findings from steps 2-4
   - Include trade-off analysis with rationale
   - List resolved and remaining open questions
   - Add references to research sources
   - Keep the draft in memory for user review in the next step

6. **Final review with user**: Present the draft spec to the user for approval before any file or issue creation.
   - Show a structured summary covering: 概要、機能要件（箇条書き）、受け入れ基準、スコープ内/外、未解決の課題
   - Use `AskUserQuestion` to ask:
     - Question: "上記の仕様内容で確定してよろしいですか？"
     - Options:
       1. **承認** — この内容で確定する
       2. **修正あり** — フィードバックを反映してから確定する
   - If the user selects "修正あり": apply the feedback to the in-memory draft and repeat this step
   - If the user selects "承認": proceed to step 7

7. **Output selection**: Use `AskUserQuestion` to ask:
   - Question: "仕様がまとまりました。どのように処理しますか？"
   - Options (4 choices):
     1. **GitHub issue を作成** — 仕様を issue として起票する
     2. **仕様書ファイルを保存** — `docs/specs/` にマークダウンとして保存のみ
     3. **仕様書ファイルを保存 + GitHub issue を起票** — 両方実行する
     4. **仕様書ファイルを保存 + /plan へ移行** — 保存後、そのまま実装計画フェーズに進む

8. **Execute the chosen path** (write only after user approval in step 6):

   **File save** (options 2, 3, 4):
   - Ensure `docs/specs/` directory exists
   - Write the approved spec to `docs/specs/<date>-<slug>.md`

   **GitHub issue creation** (options 1, 3):
   - Save the spec file first (always, as fallback), then run: `gh issue create --title "<spec title>" --body-file docs/specs/<date>-<slug>.md`
   - If `gh` fails (auth error, network issue):
     a. Warn the user: "GitHub issue の作成に失敗しました。仕様書は docs/specs/<date>-<slug>.md に保存済みです。"

   **Plan transition** (option 4):
   - After saving the spec file, invoke: `Skill(skill="plan")`
   - The spec file path should be referenced in the plan's "Related request" field

## Anti-bottleneck

Before asking the user a question, first check whether you can answer it by:
- Inspecting the codebase (existing patterns, conventions, tests)
- Reading existing docs or plans
- Running scripts or checks
- Choosing a reasonable default and documenting it

Only use `AskUserQuestion` for genuine ambiguity that cannot be resolved from repo context. See the `anti-bottleneck` skill for the full checklist.

## Output

- Spec file at `docs/specs/<date>-<slug>.md`
- Optionally: GitHub issue with spec content
- Optionally: `/plan` skill invocation for immediate implementation planning
