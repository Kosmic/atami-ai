# Skill: kanban board

You are generating an interactive read-only HTML kanban board of every item in `items/`. The output is a single self-contained HTML file at `.project-kb/outputs/kanban.html`.

## When to use this skill

When the user asks you to:
- "Make a kanban board" / "build a kanban" / "generate the kanban"
- "Update the kanban" / "regenerate the kanban"
- "Show me the items as a board"

The board is read-only. It is a viewing/triage surface; nothing in it writes back to the markdown files. To change item state, the user edits or asks you to edit the underlying item file, then regenerates the board.

## Output location

Always write the file to:

```
.project-kb/outputs/kanban.html
```

Overwrite any existing file at that path.

## Fast path: the template already exists

If `.project-kb/outputs/kanban.html` already exists from a previous generation:

1. Read it to find the `const ITEMS = [...]` array, the `const TODAY = new Date(...)`, and the "Generated YYYY-MM-DD" subtitle.
2. Rebuild the `ITEMS` array from the current `items/` directory (see "Building the ITEMS array" below).
3. Update `TODAY` to today's date.
4. Update the "Generated" date in the header subtitle.
5. Update the `<span id="active-count">N</span>` initial value to the count of items whose status is not `done` or `dropped`.

Nothing else needs to change. The HTML structure, CSS, and JS rendering logic stay identical.

If the file does **not** exist, generate from scratch using the design spec below.

## Design spec

Use this spec when generating from scratch or when verifying that an existing file matches the agreed design.

### Single self-contained file

- One HTML file. All CSS in a `<style>` block. All JS inline in a `<script>` block.
- No CDN links. No external assets. Opens offline from `file://`.
- No build step.

### Layout

The page has, top to bottom:

1. **Header** with an eyebrow based on the project name from `kb-config.yaml` (for example, `ATAMI KB`), an `<h1>Kanban</h1>`, a subtitle `<N> active items · Generated YYYY-MM-DD`, and an inbox banner. If the project name is not configured, use `PROJECT KB`.
2. **Inbox banner** — counts the number of unprocessed `.md` files at the top level of `inbox/` (excluding `inbox/archive/`). If 0: muted slate styling, text "Inbox empty — 0 unprocessed files". If >0: amber styling, text e.g. "📥 1 inbox file pending: <filename>" or "📥 3 inbox files pending processing" with the filenames listed.
3. **Active board** — a CSS grid with two columns: **Tasks** and **Discussions**. Columns are by `type`, not by `status`. If items of `type: feature` or `type: bug` exist, render extra columns for them in this order: `task`, `discussion`, `feature`, `bug`. Skip columns with zero items? No — always render `task` and `discussion` (they are the canonical types). Only render `feature`/`bug` columns when items of those types exist among active items.
4. **Archive** — a `<details>` element titled `Done & Dropped (N)` containing a board with two columns by `status`: **Done** and **Dropped**. Cards inside use the type as the primary badge.
5. **Side panel** — fixed-position drawer that slides in from the right when a card is clicked, with a backdrop. Closes via the × button, clicking the backdrop, or the Esc key.

### Cards

Each card is a `<button>` (so it's keyboard-focusable) styled to look like a card. A card shows:

- A primary badge (top-left).
  - In the **active board**, the primary badge is the **status** (`discussed`, `planned`, or `in-progress`), since the column already conveys the type.
  - In the **archive**, the primary badge is the **type**, since the column already conveys the status.
- The relative updated date (top-right), e.g. "today", "yesterday", "4 weeks ago".
- The item title.
- A two-line teaser (CSS line-clamp).

Click → opens the side panel for that item.

### Sort order within columns

Updated date, newest first. Same rule everywhere (active and archive).

### Side panel content

When opened, the panel shows:

1. A meta row with badges and small monospace pills:
   - Type badge (color-coded by type).
   - Status badge (color-coded by status).
   - `Updated YYYY-MM-DD`
   - `Created YYYY-MM-DD`
   - `Source <filename>`
2. The item title as a large heading.
3. The item's full markdown body, **pre-rendered to HTML at build time** — not converted at runtime. The agent generating the file is responsible for converting markdown → HTML.

### Color palette

CSS custom properties at the top of `:root` provide slate, amber, sky, emerald, rose, and violet ramps. Map types and statuses as follows:

- `task` → sky (blue)
- `discussion` → violet
- `feature` → emerald
- `bug` → rose
- `status: discussed` → amber
- `status: planned` → emerald
- `status: in-progress` → sky
- `status: done` → slate (muted)
- `status: dropped` → slate (muted)

The page background is a soft gradient: `linear-gradient(135deg, var(--slate-50) 0%, #fff 50%, var(--sky-50) 100%)`.

### Typography

System font stack: `-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif`. Monospace for code and meta pills: `ui-monospace, 'SF Mono', Menlo, monospace`.

### Responsiveness

Below 800px, the board grid collapses to a single column and the side panel becomes full-width.

## Building the ITEMS array

For each `.md` file in `items/` (excluding `index.md`):

1. Parse the YAML frontmatter to get `status`, `type`, `created`, `updated`, `source`.
2. Take the H1 (`# Title`) of the markdown body as the title.
3. Generate a `teaser`: the first 1–2 sentences of the `## Summary` section, plain text, no markdown syntax.
4. Pre-render the rest of the markdown body (everything after the H1) to HTML and store in `body`.
5. Use the filename without the `.md` extension as the `id`.

Resulting object shape:

```js
{
  id: "task-foo-bar",
  title: "Foo bar",
  type: "task",                  // task | feature | bug | discussion
  status: "planned",             // discussed | planned | in-progress | done | dropped
  created: "2026-04-06",
  updated: "2026-04-06",
  source: "raw-inbox-filename.md",
  teaser: "First sentences of the summary, plain text.",
  body: `<h3>Summary</h3><p>...</p>...`
}
```

Place objects in the array in any order — the JS sorts by `updated` desc within each column.

## Markdown → HTML conversion rules

Pre-render at build time. Do not embed a markdown library and do not convert at runtime in the browser. The conversion rules used by the existing items:

| Markdown | HTML |
|---|---|
| `## Heading` | `<h3>Heading</h3>` (note: bumped down because the panel title is `<h2>`) |
| `### Heading` | `<h4>Heading</h4>` |
| `# Heading` | Skip — it's the item title, shown separately above the body |
| Paragraph | `<p>...</p>` |
| `- item` | `<ul><li>item</li></ul>` |
| `1. item` | `<ol><li>item</li></ol>` |
| `**bold**` | `<strong>bold</strong>` |
| `` `code` `` | `<code>code</code>` |
| ` ```lang ... ``` ` | `<pre><code>...</code></pre>` |
| `[text](url)` | `<a href="url" target="_blank" rel="noopener">text</a>` for external; drop `target` for repo-relative paths |

Inside `<pre><code>` blocks, HTML-escape `<`, `>`, `&`. Outside code, escape only what's necessary to keep the HTML valid.

The `body` string is interpolated into the panel via `innerHTML`, so it must be valid HTML — no unescaped `<`, `>`, or stray markdown markers.

## Card rendering

Cards are built in JS from the `ITEMS` array. The render function:

1. Splits items into `byType` (active) and `archiveByStatus` (terminal states).
   - If `status === 'done' || status === 'dropped'`, goes to archive bucketed by status.
   - Otherwise, goes to active bucketed by type.
2. Sorts each bucket by `updated` descending.
3. Populates each column's `[data-cards-for="<key>"]` container.
4. Updates `[data-count-for="<key>"]` with the count.
5. Updates the `Done & Dropped (N)` summary count.

Active card primary badge: `<span class="badge badge-status-${status}">${status}</span>`.
Archive card primary badge: `<span class="badge badge-${type}">${type}</span>`.

## Side panel logic

- Click handler on every card calls `openPanel(id)`.
- `openPanel` populates `#panel-inner` with the meta row, title, and `item.body`, then adds `.open` to `#panel` and `#panel-backdrop`.
- `closePanel` removes `.open` from both. Wired to the × button, the backdrop, and the Esc key.

Set `aria-hidden="true"` initially and `"false"` when open.

## Relative date formatting

A small JS function converts an ISO date to a human-friendly relative string against `TODAY`:

- 0 days → "today"
- 1 day → "yesterday"
- < 7 days → "N days ago"
- < 30 days → "N week(s) ago"
- < 365 days → "N month(s) ago"
- otherwise → "N year(s) ago"

`TODAY` is a `Date` object set to today's UTC midnight when the file is generated.

## Inbox banner content

Before generating, count files matching `inbox/*.md` (top level only, not `inbox/archive/**`).

- 0 files → muted slate banner: `Inbox empty — 0 unprocessed files`
- 1 file → amber banner: `📥 1 inbox file pending: <filename>`
- 2+ files → amber banner: `📥 N inbox files pending processing` with filenames listed compactly

## After generating

1. Tell the user the file path: `.project-kb/outputs/kanban.html`.
2. Suggest opening it: `open .project-kb/outputs/kanban.html`.
3. Note that newly initialized project-kb directories gitignore this generated HTML file by default.
4. Note that to reflect future item changes, regenerate by asking again.

## What to leave out

- Do not add filtering controls, search boxes, or sort toggles. The board is intentionally minimal — 8 items today, and even at 30+ items the simple list-by-column scales fine.
- Do not add edit affordances, drag-and-drop, or any interaction that implies write capability. The board is read-only by design; the source of truth is the markdown files in `items/`.
- Do not embed a markdown library, fetch any external CSS/JS, or split the output across multiple files.
- Do not include the inbox archive count anywhere — only the live, unprocessed inbox.
