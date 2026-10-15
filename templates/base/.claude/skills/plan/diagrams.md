# Plan diagrams

How to build the visual review page that `/plan` shows before asking for
approval. Start from [visual-template.html](visual-template.html).

## Purpose

- The page lets a human see what the agent will build before approving the
  plan.
- The approver reads only the top of the page: the easy figures and the
  "Before you approve" box. The closed fold below keeps the figures drawn
  with paths and names, for checking the mechanism and for `/pr`'s match of
  the figure to the diff.
- Optimize for legibility and quick understanding, not for record-keeping.
  The plan text stays the record; the page does not need to repeat it.
- The page lives at `.harness/state/plan-visual/<slug>.html` (gitignored).
  Never commit the page or its PNGs.

## When to draw, when to skip

Draw a page unless the plan meets **both** conditions:

1. It is one of:
   - a docs-only change
   - a change to a single file plus its generated mirrors
   - a mechanical rename or value change
2. It changes none of:
   - interactions between components (who calls whom, in what order)
   - states or transitions
   - data formats, schemas, or file layouts
   - module boundaries or dependency direction

When skipping, write `None (<reason>)` in the plan's `## Visual review`
section, e.g. `None (docs-only change)`.

## Page structure

Copy [visual-template.html](visual-template.html) to
`.harness/state/plan-visual/<slug>.html` and fill it in. Keep the page
self-contained: inline CSS and inline SVG only — no external scripts, fonts,
images, or CDNs.

Escape `&`, `<`, and `>` in all page text, including SVG `<text>`: write
`&amp;`, `&lt;`, and `&gt;` (e.g. `plan-visual/&lt;slug&gt;.html`). A raw
`<slug>` is parsed as a tag, and the rest of the label silently disappears.

Order:

1. Header: `<h1>` is a sentence saying what changes for the reader, then the
   plan path · branch, then the lead (`<p class="lead">`), two sentences: how
   it works today, then how it works after this plan
2. Easy figures, 1 to 3: `<figure class="easy" id="overview">` first, then
   `<figure class="easy" id="easy-<topic>">`
3. The "Before you approve" box: `<section class="before chrome">`
4. The fold: `<details class="deep">` with a `<summary>` line, closed. Inside
   it: the summary ("What changes" / "What stays") and the legend, then
   `<figure id="detail-overview">`, then the detail figures
   (`<figure id="detail-<topic>">`), then "Not in this page" (the plan's
   non-goals)

The header, the "Before you approve" box, the summary and legend inside the
fold, and "Not in this page" carry `class="chrome"`. Opening the page as
`<page>#<figure-id>` hides them, the fold's summary line, and every other
figure. A target inside the closed fold opens it (Chrome), so
`<page>#detail-overview` shows that figure alone. Attribute values may use
either quote style. Write page text in the user's language.

## Easy figures

One to three per page, each answering one question the approver has.

1. The `<h2>` heading is the conclusion, as a sentence with a subject and a
   predicate (e.g. 「止めるときは、合言葉が合う部屋だけを閉じる」), not a
   noun phrase such as 「全体図」.
2. The actors are people and roles (あなた, leader, 見張り, ...); draw the
   human with the person icon. No file paths, function names, event names,
   config keys, or monospace text. A command the human types themselves may
   appear (`/director`, `ralph org stop --all`).
3. A term the reader cannot avoid gets a one-line meaning in
   `<p class="words">` under the figure, the first time it appears.
4. At most 5 boxes (`n-*` classes): `n-new` is what is added for the reader,
   `n-old` is what stays, `n-fail` (dashed) is what happens on failure. An
   `area` panel behind related boxes is not a box. One arrow kind (`line`);
   `line off` with a ✕ (`t-no`) for what does not happen, and a ✓ (`t-ok`)
   on the line it is set against. No legend.
5. Default form: before → after in two rows (今 / これから). Other forms: one
   flow with a failure branch; one action that reaches the right target and
   not the wrong one.
6. Name the slices under the figure in `<span class="made">`
   (「S1・S2 で作る」), not as badges on boxes.
7. Grounding: every box states a behavior the plan names. Say nothing the
   plan does not.

| Item | Budget |
|------|--------|
| SVG `viewBox` width | 1080; choose the height to fit |
| Box label | 18px. Text width = full-width characters × 18 + half-width characters × 0.55 × 18 |
| Sub line (`t-sub`) | 14px |
| Box width | ≥ text width + 24px |
| Gap between boxes | ≥ 30px |

Give each figure its own `<defs>`, with marker and icon ids prefixed by the
figure number (`e1-arrow`, `e1-person`).

Figures 2 and 3 use the same classes as figure 1, including `no` for the
round number in the `<h2>`, `t-row` for a row label (`t-row now` for the row
after this plan), and `rule` for the line between rows.

## Before you approve

The box (承認の前に) has three columns, in plain words:

- **Approving starts**: the slices, one line each.
- **What to watch**: what the approver should know from Design decisions and
  Risks, such as a permission default or an action that does not stop
  something.
- **Not in this plan**: the non-goals.

## Detail figures (in the fold)

### Granularity

- **One detail overview** (`<figure id="detail-overview">`) shows every
  feature or slice on a single canvas, at the level of files. Label each part
  with its slice id (`S1`, `S2`, ...) or AC id, as a badge, so the reader can
  map figure ↔ plan text.
- **Detail figures**: one per question about the mechanism. Pick the type
  from the table below.
- Every detail figure carries a one-line `What to check:` caption
  (`<p class="q">`) that states that question.
- More than 4 detail figures after the detail overview → do not add figures;
  propose splitting the plan instead.

### What changes → figure type

| What changes | Figure type |
|--------------|-------------|
| Interactions or call order | Sequence diagram |
| States or transitions | State diagram |
| Data format, schema, or file layout | Structure diagram (boxes with fields, or ER-style) |
| Module boundary or dependency direction | Before/after side by side (or one figure with the diff overlaid) |
| User-visible or workflow flow | Before/after flow |
| Many files, mirrors, or registries | File map grouped by area |

### Visual vocabulary

Use color for meaning, not decoration. Always include the legend in the fold.

| Meaning | Look | Template class |
|---------|------|----------------|
| Added | blue fill, blue border | `node add` |
| Changed | orange fill, orange border | `node mod` |
| Existing / unchanged | white fill, gray border | `node keep` |
| Removed | gray dashed border, label struck through or marked ✕ | `node del` |
| Copy, generated mirror | gray dashed line | `edge copy` |
| "Uses this part" | blue dashed line | `edge uses` |
| Call, data flow, order | solid dark line with an arrowhead | `edge` |
| Slice badge | filled blue pill with white `S1` | `badge` + `badge-text` |
| Area / group | light beige panel behind related nodes | `group` |

- Set colors through these classes. A class rule overrides a `fill` or
  `stroke` attribute on the same element.
- Element ids are page-wide: give each SVG its own `<defs>` with markers
  prefixed by the figure id (`do-arrow`), so figures never depend on one
  another.

### Layout budgets

| Item | Budget |
|------|--------|
| SVG `viewBox` width | 1080 (the page scales it to the column); choose the height to fit |
| Text size | ≥ 12px for labels and captions; badge text and small symbols (such as ✕) may be 11px; node labels 13–14px |
| Node width | ≥ text width + 24px. Text width = full-width characters × font size + half-width characters × 0.55 × font size (0.6 in the mono font) |
| Nodes per figure | ≤ 10 (groups and badges do not count) |
| Gap between nodes | ≥ 16px |

- Never let a line run through a label. Put a background rect
  (`class="label-bg"`) behind a label that sits on a line.
- Avoid crossing lines; reorder nodes before adding bends.
- Put badges where no arrow lands (e.g. a node's top-right corner when arrows
  enter from the left).

### Grounding

- Every node is either an existing path or component you have read, or a new
  path the plan names.
- Label nodes with the path (`scripts/foo.sh`) or with the name the plan uses
  for it.
- Remove anything you cannot verify from the repo. A part the plan does not
  name yet gets named in the plan first, or stays out of the figure.

## Self-check before showing

1. Lint the page and fix it until the command exits 0:
   ```sh
   scripts/plan-visual.sh lint .harness/state/plan-visual/<slug>.html
   ```
   A worktree made before `lint` existed has an older helper: its `--help`
   has no `scripts/plan-visual.sh lint <html>` line. Then run the main
   checkout's helper on the page's absolute path:
   ```sh
   "$(git rev-parse --path-format=absolute --git-common-dir)/../scripts/plan-visual.sh" lint <absolute page path>
   ```
   If that helper has no `lint` either, skip the lint and say why in the
   plan's `## Visual review` section; approval can still proceed.
2. Shoot the full page, the overview, and every other figure one by one:
   ```sh
   scripts/plan-visual.sh shot .harness/state/plan-visual/<slug>.html .harness/state/plan-visual/<slug>-full.png
   scripts/plan-visual.sh shot .harness/state/plan-visual/<slug>.html .harness/state/plan-visual/<slug>-overview.png --fragment overview --width 1150 --height <fit> --scale 2
   scripts/plan-visual.sh shot .harness/state/plan-visual/<slug>.html .harness/state/plan-visual/<slug>-<id>.png --fragment <id> --width 1150 --height <fit>
   ```
   `<fit>`: about the figure's `viewBox` height + 200. Adjust it after reading
   the PNG so the figure is not clipped and little empty space is left below.
   Shoot each remaining easy figure and each fold figure with its own id. The
   full page shows the fold closed, so the fold figures appear only in their
   own PNGs. If a fold figure's PNG comes out blank, add `open` to the
   `<details>` while shooting it, then remove it.
3. Read every PNG. Fix overlapping text, clipped or cramped labels, lines
   through labels, and arrows that miss their target.
4. Repeat until every PNG is clean.

What `lint` checks, from the HTML text alone:

- L1: exactly one `id="overview"`, on a `<figure class="easy">`
- L2: 1 to 3 easy figures, none inside the fold
- L3: in an easy figure, no `mono` class, `<code>` element, path-like word
  (`internal/org`, also `S1/S2`), file name with an extension such as `.md`
  or `.sh`, or snake_case word; slash commands (`/director`) and flags
  (`--all`) pass. It reads the figure's text (the words between tags), its
  `class` names, and its `<code>` tags, not other attribute values such as
  `aria-label`. Write the `aria-label` in the same plain words as the figure
- L4: in each easy figure, at most 5 `n-*` boxes, an `<h2>`, and an `<svg>`
  with a non-empty `aria-label`
- L5: one `<section class="before">`, outside the fold
- L6: one fold (`<details class="deep">`) holding `id="detail-overview"`
  and every figure that is not easy

What `lint` does not check: whether a heading is a sentence, CamelCase
function names, whether boxes say only what the plan says, readability, and
overlaps. Reading the PNGs covers those.

Skip the shots and the PNG reading, and say so in the plan's
`## Visual review` section, when:

- `shot` exits 2 (no Chrome / Chromium found)
- the agent cannot read images

## Showing the page

- Run `scripts/plan-visual.sh open .harness/state/plan-visual/<slug>.html`.
- It always prints the absolute path on stdout. If no browser opened, give
  that path to the user.

## Credit

The idea of choosing a figure type from what must be shown, and of marking
before/after changes by color, comes from
[nntto/skills — explanatory-diagrams](https://github.com/nntto/skills/tree/main/skills/explanatory-diagrams).
No text or templates are copied from it (it has no license file).

The easy figures follow the "big pictures, few words" idea of the eli5 skill
in [anthropics/claude-plugins-community](https://github.com/anthropics/claude-plugins-community/tree/main/eli5).
No text or templates are copied from it.
