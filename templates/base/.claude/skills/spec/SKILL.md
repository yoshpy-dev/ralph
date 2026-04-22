---
name: spec
description: Turn vague ideas or abstract prompts into detailed, actionable specifications through iterative brainstorming, codebase exploration, web research, and interactive clarification with the user. Manual trigger only.
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
| User interaction | Iterative brainstorming (divergent) + clarification (convergent) | Flow selection (standard/Ralph) + critical-fork resolution (convergent) |

`/spec` comes before `/plan`. Use `/spec` when the request is too vague for `/plan`.

## Goals

- Transform ambiguous requests into implementation-ready specifications
- Expand sparse inputs (even a one-line prompt) through iterative brainstorming
- Discover requirements through codebase analysis and best-practice research
- Resolve residual ambiguity through targeted user questions
- Produce a versioned spec that survives context loss

## Steps

1. **Understand the request** (internal, no user interaction): Read the user's input. List what is clear and what is ambiguous or missing. This is an internal triage step — do NOT call `AskUserQuestion` here. The list feeds the next step.

2. **Brainstorm to expand the idea** (壁打ち): Use `AskUserQuestion` iteratively to widen the problem space before converging. This step is mandatory when the input is sparse (e.g., a single sentence like "〇〇を実現したい"); skip or shorten when the input is already detailed.
   - Explore these axes one at a time (repeat as needed, no iteration cap):
     - 目的・背景（なぜ実現したいのか、解決したい課題）
     - 対象ユーザー・利用シーン
     - 代替アプローチ（他にどう解決できるか、それぞれの trade-off）
     - 成功条件（何が実現すれば「できた」と言えるか）
     - スコープ境界・優先順位（何はやらないか、MVP の範囲）
     - 既知の制約（技術・時間・体制）
   - Purpose is **divergent** (expand options), distinct from step 5 which is **convergent** (resolve remaining ambiguity).
   - Continue until the user signals that the idea is sufficiently shaped, or until further questions stop yielding new information.
   - Respect anti-bottleneck: before asking, check whether the repo already answers the question.

3. **Explore the codebase**: Use `Task(subagent_type="Explore")` to investigate:
   - Existing code related to the request
   - Current patterns and conventions
   - Potential impact areas and dependencies
   - Similar implementations that already exist

4. **Research best practices**: Use `WebSearch` and `WebFetch` to investigate:
   - Industry best practices for the problem domain
   - Common approaches and trade-offs
   - Reference implementations in well-known projects
   - Relevant library/framework documentation (via Context7 MCP if applicable)

5. **Clarify residual requirements**: Use `AskUserQuestion` actively to resolve any ambiguity that remains after brainstorming and research:
   - Underspecified aspects surfaced by exploration/research
   - Trade-off decisions informed by newly gathered context (e.g., simplicity vs extensibility)
   - Priority and scope boundaries not yet nailed down
   - User preferences and constraints tied to findings
   - Repeat this step as many times as needed. Do not guess when you can ask.
   - Purpose is **convergent** — narrow toward a single, implementable spec.

6. **Draft the spec** (in memory — do NOT write to file yet):
   - Compose the full spec content from [template.md](template.md) using findings from steps 2-5
   - Include trade-off analysis with rationale
   - List resolved and remaining open questions
   - Add references to research sources
   - Keep the draft in memory for user review in the next step

7. **Final review with user**: Present the draft spec to the user for approval before any file or issue creation.
   - Show a structured summary covering: 概要、機能要件（箇条書き）、受け入れ基準、スコープ内/外、未解決の課題
   - Use `AskUserQuestion` to ask:
     - Question: "上記の仕様内容で確定してよろしいですか？"
     - Options:
       1. **承認** — この内容で確定する
       2. **修正あり** — フィードバックを反映してから確定する
   - If the user selects "修正あり": apply the feedback to the in-memory draft and repeat this step
   - If the user selects "承認": proceed to step 8

8. **Output selection**: Use `AskUserQuestion` to ask:
   - Question: "仕様がまとまりました。どのように処理しますか？"
   - Options (4 choices):
     1. **GitHub issue を作成** — 仕様を issue として起票する
     2. **仕様書ファイルを保存** — `docs/specs/` にマークダウンとして保存のみ
     3. **仕様書ファイルを保存 + GitHub issue を起票** — 両方実行する
     4. **仕様書ファイルを保存 + /plan へ移行** — 保存後、そのまま実装計画フェーズに進む

9. **Execute the chosen path** (write only after user approval in step 7):

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
