# Plan diagrams

How to build the visual review page that `/plan` shows before asking for
approval. Start from [visual-template.html](visual-template.html).

## Purpose

- The page lets a human see what the agent will build before approving the
  plan.
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

## Granularity

- **One overview figure** (`<figure id="overview">`) shows every feature or
  slice on a single canvas. Label each part with its slice id (`S1`, `S2`,
  ...) or AC id so the reader can map figure ↔ plan text.
- **Detail figures**: one per question the approver needs answered. Pick the
  type from the table below.
- Every figure carries a one-line `What to check:` caption
  (`<p class="q">`) that states that question.
- More than 4 detail figures → do not add figures; propose splitting the plan
  instead.

## What changes → figure type

| What changes | Figure type |
|--------------|-------------|
| Interactions or call order | Sequence diagram |
| States or transitions | State diagram |
| Data format, schema, or file layout | Structure diagram (boxes with fields, or ER-style) |
| Module boundary or dependency direction | Before/after side by side (or one figure with the diff overlaid) |
| User-visible or workflow flow | Before/after flow |
| Many files, mirrors, or registries | File map grouped by area |

## Visual vocabulary

Use color for meaning, not decoration. Always include the legend.

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
  prefixed by the figure id (`ov-arrow`), so figures never depend on one
  another.

## Layout budgets

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

## Grounding

- Every node is either an existing path or component you have read, or a new
  path the plan names.
- Label nodes with the path (`scripts/foo.sh`) or with the name the plan uses
  for it.
- Remove anything you cannot verify from the repo. A part the plan does not
  name yet gets named in the plan first, or stays out of the figure.

## Page structure

Copy [visual-template.html](visual-template.html) to
`.harness/state/plan-visual/<slug>.html` and fill it in. Keep the page
self-contained: inline CSS and inline SVG only — no external scripts, fonts,
images, or CDNs.

Escape `&`, `<`, and `>` in all page text, including SVG `<text>`: write
`&amp;`, `&lt;`, and `&gt;` (e.g. `plan-visual/&lt;slug&gt;.html`). A raw
`<slug>` is parsed as a tag, and the rest of the label silently disappears.

Order:

1. Header: plan title, plan path, branch
2. Summary: "What changes" / "What stays"
3. Legend
4. `<figure id="overview">` — always first
5. Detail figures, each with its own `id`
6. "Not in this page": the plan's non-goals

Header, summary, legend, and "Not in this page" carry `class="chrome"`.
Opening the page as `<page>#overview` hides them and every other figure; the
overview PNG is taken in that mode. Write page text in the user's language.

## Self-check before showing

1. Shoot the full page and the overview:
   ```sh
   scripts/plan-visual.sh shot .harness/state/plan-visual/<slug>.html .harness/state/plan-visual/<slug>-full.png
   scripts/plan-visual.sh shot .harness/state/plan-visual/<slug>.html .harness/state/plan-visual/<slug>-overview.png --fragment overview --width 1150 --height <fit> --scale 2
   ```
   `<fit>`: about the overview `viewBox` height + 160. Adjust it after reading
   the PNG so the figure is not clipped and little empty space is left below.
2. Read both PNGs. Fix overlapping text, clipped or cramped labels, lines
   through labels, and arrows that miss their target.
3. Repeat until both PNGs are clean.

Skip the self-check, and say so in the plan's `## Visual review` section,
when:

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
