# Skill: kanban board

You are generating an interactive HTML kanban board of every item in `items/`. The output is a single self-contained HTML file at `.project-kb/outputs/kanban.html`.

The board is **locally interactive but not networked**. The user can drag cards between columns to stage status/type changes; those changes live only in the browser's localStorage. To apply changes back to the source `.md` files, the user copies a structured changes block from the board's tray and pastes it into the agent chat — the agent then edits the underlying frontmatter and regenerates the board.

The source of truth is the markdown files in `items/`. The kanban never writes to disk.

## When to use this skill

When the user asks you to:

- "Make a kanban board" / "build a kanban" / "generate the kanban"
- "Update the kanban" / "regenerate the kanban"
- "Show me the items as a board"

When the user pastes a block beginning with `Apply these kanban changes to project-kb items:`, do **not** invoke this skill. That's the apply-changes flow — see [Applying pasted changes](#applying-pasted-changes) below. After applying, regenerate the kanban as the final step.

## Output location

Always write the file to:

```
.project-kb/outputs/kanban.html
```

Overwrite any existing file at that path.

## Fast path: the template already exists

If `.project-kb/outputs/kanban.html` already exists from a previous generation:

1. Read it to find the `const ITEMS = [...]` array, the `const TODAY = new Date(...)`, and the "Generated YYYY-MM-DD" subtitle.
2. Rebuild the `ITEMS` array from the current `items/` directory (see [Building the ITEMS array](#building-the-items-array) below).
3. Update `TODAY` to today's date.
4. Update the "Generated" date in the header subtitle.
5. Update the `<span id="active-count">N</span>` initial value to the count of items whose status is not `done` or `dropped`.

Nothing else changes. The HTML structure, CSS, JS rendering logic, drag-and-drop logic, and pending-changes tray stay identical.

The user's pending changes live in **localStorage**, not in the HTML file, so they survive regeneration automatically. On the next page load the new JS reads localStorage, compares deltas against the new `ITEMS` array, auto-clears applied deltas, and flags stale ones.

If the file does **not** exist, generate from scratch using the design spec below.

If the existing file is missing any feature from the current design spec (no `Group by` toggle, no pending-changes tray, no drag handlers, no filename footer on cards, etc.), treat it as out of date and regenerate from scratch rather than taking the fast path.

## Design spec

### Single self-contained file

- One HTML file. All CSS in a `<style>` block. All JS inline in a `<script>` block.
- No CDN links. No external assets. Opens offline from `file://`.
- No build step.

### Layout

The page has, top to bottom:

1. **Header** with:
   - An eyebrow based on the project name from `kb-config.yaml` (e.g. `ATAMI KB`). If unset, use `PROJECT KB`.
   - `<h1>Kanban</h1>`.
   - Subtitle: `<N> active items · Generated YYYY-MM-DD`. When pending changes exist, append ` · <M> pending` in amber, where the pending count is a clickable `<button>` that opens the pending-changes tray.
   - A `Group by: [Type] [Status]` segmented toggle on the right of the header. Persisted in localStorage. Default `type`.
   - A small `?` help icon to the right of the toggle. Click opens an overlay explaining drag, tray, and paste-back flow. On first visit (no `kanban.helpSeen` localStorage flag), auto-open and set the flag on close.
2. **Inbox banner** — counts the number of unprocessed `.md` files at the top level of `inbox/` (excluding `inbox/archive/`).
   - 0 files → muted slate banner: `Inbox empty — 0 unprocessed files`.
   - 1 file → amber banner: `📥 1 inbox file pending: <filename>`.
   - 2+ files → amber banner: `📥 N inbox files pending processing` with filenames listed compactly.
3. **Active board** — a CSS grid whose columns depend on the current `Group by` mode:
   - **Group by Type**: columns are `task` and `discussion` (always rendered) plus `feature` and/or `bug` if any items of those types exist among active items. Order: `task`, `discussion`, `feature`, `bug`. The done/dropped pile lives in the separate **Archive** `<details>` below.
   - **Group by Status**: columns are `discussed`, `planned`, `in-progress`, `done`, `dropped` — always all five, in that order. The Archive `<details>` is **hidden** in this mode (done/dropped are first-class columns).
4. **Archive** — only rendered in Group by Type mode. A `<details>` element titled `Done & Dropped (N)` containing a sub-grid with two columns by status: **Done** and **Dropped**.
5. **Side panel** — fixed-position drawer that slides in from the right when a card is clicked, with a backdrop. Closes via the × button, clicking the backdrop, or the Esc key.
6. **Pending-changes tray** — fixed-position floating pill at bottom-right. Hidden when zero pending changes. When `M > 0`: an amber pill reading `<M> pending change(s)`. Click expands to a panel listing each delta with a per-entry revert button, plus footer buttons `Copy as instructions` and `Discard all`.

### Cards

Each card is a `<button>` (so it's keyboard-focusable for opening the side panel) styled to look like a card. A card shows:

- A primary badge (top-left).
  - In Group by Type, the primary badge is the **status** (since the column conveys the type).
  - In Group by Status, the primary badge is the **type** (since the column conveys the status).
  - In the **archive** (Group by Type only), the primary badge is the **type**.
- The relative updated date (top-right), e.g. "today", "yesterday", "4 weeks ago".
- The item title.
- A two-line teaser (CSS line-clamp).
- The exact item filename (e.g. `task-foo-bar.md`) as a small muted monospace footer at the bottom of the card.

Cards are also **draggable** via HTML5 native drag-and-drop (`draggable="true"`). See [Drag-and-drop logic](#drag-and-drop-logic).

Click (without drag) opens the side panel for that item.

#### Pending-state visual treatment

When a card has a pending delta against it (status or type change):

- **Border**: 1px dashed `var(--amber-400)` instead of the default solid border.
- **Indicator**: a small amber `●` dot in the top-right corner, just left of the relative-date.
- **Hover**: the dot is replaced by a `↺` revert button. Click reverts that card's delta and animates the card back to its original column.
- **Tooltip on the dot**: `Pending: <field> <from> → <to>`.

#### Stale-state visual treatment

When a delta's `from` value no longer matches the source state (the underlying file changed underneath the user):

- **Border**: 1px dashed `var(--rose-500)` (rose, not amber).
- **Indicator**: a small rose `⚠` icon top-right.
- **Hover**: the icon is replaced by a `✕` discard button. Click drops the stale delta.
- **Tooltip on the icon**: `Stale: this item changed underneath your pending <field> change. Click to discard.`

The stale card is rendered at the source-state position (whatever column the new source state implies), not the pending-state position.

### Sort order within columns

Within each column:

1. If the user has a reorder list for `<groupBy>:<columnKey>` in localStorage, items appear in that order. Items not in the list fall back to step 2.
2. Updated date, newest first.

The reorder lists are independent per `(groupBy, column)` pair and are **never exported** — they're purely local UI state.

### Side panel content

When opened, the panel shows:

1. A meta row with badges and small monospace pills:
   - Type badge (color-coded).
   - Status badge (color-coded). If the card has a pending delta, the affected badge shows the pending value with an amber outline; an inline note reads `Pending: <field> <from> → <to>`.
   - `Updated YYYY-MM-DD`.
   - `Created YYYY-MM-DD`.
   - `Source <filename>`.
2. The item title as a large heading.
3. The item's full markdown body, **pre-rendered to HTML at build time**.

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

Pending state uses amber accents (border, dot, tray pill, subtitle count). Stale state uses rose.

The page background is a soft gradient: `linear-gradient(135deg, var(--slate-50) 0%, #fff 50%, var(--sky-50) 100%)`.

### Typography

System font stack: `-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif`. Monospace for code and meta pills: `ui-monospace, 'SF Mono', Menlo, monospace`.

### Responsiveness

Below 800px, the board grid collapses to a single column and the side panel becomes full-width. **Drag-and-drop is disabled below 800px** — cards remain clickable to open the side panel, but are not draggable. The pending-changes tray remains usable.

## Local interactivity (pending changes)

The board lets the user stage status and type changes locally via drag-and-drop. The changes are not written back to disk by the kanban itself; the user copies a structured block from the tray and pastes it to the agent, which then edits the source `.md` files.

### localStorage schema

Single key: `kanban.<projectName>` (where `<projectName>` is the slugified project name from `kb-config.yaml`, or `default` if unset).

```js
{
  pendingChanges: {
    "<itemId>": {
      status?: { from: "<status>", to: "<status>" },
      type?:   { from: "<type>",   to: "<type>" }
    }
  },
  reorder: {
    "<groupBy>:<columnKey>": ["<itemId>", "<itemId>", ...]
  },
  groupBy: "type" | "status",
  helpSeen: true | undefined
}
```

`<itemId>` is the filename without `.md` (matches the `id` field in `ITEMS`).

### Operations supported

- **Change status**: drag a card between status columns (Group by Status mode). Or, to send to archive in Group by Type mode, this requires flipping to Group by Status and dragging to `done` or `dropped`. The skill doc note: there is no quick "send to archive" gesture in Group by Type mode — that's intentional, to keep one gesture per dimension.
- **Change type**: drag a card between type columns (Group by Type mode).
- **Reorder within a column**: drag a card to a new position in the same column. Stored in `reorder`, never exported.

That's the full set. No delete (use status `dropped`). No inline title/body editing. No card creation from the board.

### Pending-changes tray

A floating pill bottom-right of the viewport, `position: fixed`. Hidden when no pending changes.

- **Collapsed**: amber pill, e.g. `3 pending changes`. Click to expand.
- **Expanded**: a panel showing one row per delta with:
  - The card's filename (monospace).
  - The change description: `status discussed → planned` or `type discussion → task`.
  - A `Revert` button.
- **Footer buttons**:
  - `Copy as instructions` — formats the deltas as a paste-block (see [Export format](#export-format)) and writes it to the clipboard via `navigator.clipboard.writeText`. Shows a transient confirmation toast.
  - `Discard all` — clears all `pendingChanges` from localStorage, with a single confirm step.

The subtitle's `<M> pending` button opens the same expanded tray.

### Export format

```
Apply these kanban changes to project-kb items:

- task-foo-bar.md: status discussed → planned
- discussion-baz.md: type discussion → task

(Generated from kanban.html at <ISO timestamp>)
```

One line per delta. If a card has both status and type pending, that's two lines. Reorder is never included.

### In-page help

A `?` icon in the header opens a modal overlay with a brief explainer:

- Drag cards between columns to stage status/type changes. Drag within a column to reorder (local only).
- Use the `Group by` toggle to switch between dragging-changes-type and dragging-changes-status.
- Pending changes show as amber-dashed cards. The tray bottom-right shows them all.
- Click `Copy as instructions` and paste to the agent to apply changes to the underlying files.
- Changes don't save themselves — they live only in your browser until the agent applies them.

Auto-show on first visit (no `kanban.helpSeen` flag), then set the flag on close so it doesn't show again.

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

Place objects in the array in any order — the JS sorts within each column at render time.

## Markdown → HTML conversion rules

Pre-render at build time. Do not embed a markdown library and do not convert at runtime in the browser.

| Markdown | HTML |
|---|---|
| `## Heading` | `<h3>Heading</h3>` (bumped down because the panel title is `<h2>`) |
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

The `body` string is interpolated into the panel via `innerHTML`, so it must be valid HTML.

## Card rendering

Cards are built in JS from the `ITEMS` array, reading the localStorage state to compute the **effective state** of each item.

For each item:

1. Compute `effectiveStatus` and `effectiveType`:
   - If a `pendingChanges[id].status` delta exists and its `from` matches the source `status`, use `delta.to` as the effective status. Otherwise effective status is the source status.
   - Same logic for `type`.
2. Compute the card's `pendingFlag`:
   - `clean` — no delta.
   - `pending` — delta exists and `from` matches source.
   - `stale` — delta exists but `from` no longer matches source.
3. Auto-clear deltas where the source already matches the `to` value (the change has been applied externally — e.g., via paste-back-and-regenerate). Persist the cleared state.
4. The card is placed in the column corresponding to its effective grouping field. (Stale cards are placed at the source-state position.)

Active card primary badge:

- Group by Type → `<span class="badge badge-status-${effectiveStatus}">${effectiveStatus}</span>`.
- Group by Status → `<span class="badge badge-${effectiveType}">${effectiveType}</span>`.

Archive card primary badge: `<span class="badge badge-${effectiveType}">${effectiveType}</span>`.

The render function:

1. Splits items into the appropriate buckets based on `groupBy`.
2. Sorts each bucket using the [Sort order within columns](#sort-order-within-columns) rules.
3. Populates each column's `[data-cards-for="<key>"]` container.
4. Updates `[data-count-for="<key>"]` with the count.
5. Updates the `Done & Dropped (N)` summary count (Group by Type only).
6. Updates the subtitle's `<N> active items` and the pending-count button.
7. Updates the tray.

Re-renders are triggered by: dragging a card, reverting a delta, discarding all, toggling Group by, and on initial page load.

## Drag-and-drop logic

Implemented with HTML5 native drag-and-drop. Each card is `draggable="true"` and has `dragstart`, `dragend` handlers; each column has `dragover`, `drop` handlers; each column also exposes drop slots between cards for reorder targeting.

- `dragstart`: record the dragged card's `id` and source column key. Add a `.dragging` class.
- `dragover`: prevent default to allow drop. Show a drop indicator (1px solid amber line) at the closest insert position based on cursor Y.
- `drop`: compute the target column key and target index.
  - If the target column key differs from the source column key, that's a status or type change (depending on `groupBy`). Update `pendingChanges[id]` accordingly:
    - Set `delta.from` to the *current source* value of that field (not the effective value).
    - Set `delta.to` to the target column key's value.
    - If `delta.from === delta.to`, remove the delta entirely (user dragged it back).
  - If the target column matches the source column, that's a reorder. Update `reorder[<groupBy>:<columnKey>]` to put `id` at the target index.
- `dragend`: remove `.dragging` class; clear drop indicator.

After every drop, persist localStorage and re-render.

Below 800px viewport width, all `draggable` attributes are removed (and drag handlers no-op) so the board behaves as a clickable read-only view on mobile.

## Side panel logic

- Click handler on every card (when no drag occurred) calls `openPanel(id)`.
- `openPanel` populates `#panel-inner` with the meta row (showing pending state if applicable), title, and `item.body`, then adds `.open` to `#panel` and `#panel-backdrop`.
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

- 0 files → muted slate banner: `Inbox empty — 0 unprocessed files`.
- 1 file → amber banner: `📥 1 inbox file pending: <filename>`.
- 2+ files → amber banner: `📥 N inbox files pending processing` with filenames listed compactly.

## Applying pasted changes

When the user pastes a block beginning with `Apply these kanban changes to project-kb items:`, do **not** invoke the kanban-board skill (that regenerates the file). Instead:

1. **Parse each line** of the form `- <filename>: <field> <from> → <to>`. The arrow is the Unicode `→` (U+2192). The `<field>` is `status` or `type`. Lines that don't match are ignored with a warning.
2. **For each parsed change**, edit `items/<filename>`:
   - Update the YAML frontmatter `<field>` to `<to>`.
   - Update the YAML frontmatter `updated` to today's date in `YYYY-MM-DD` format.
   - Do not touch the markdown body.
3. **Verify** that the previous `<field>` value matched `<from>`. If not, surface a warning to the user (e.g., "expected status `discussed` but found `in-progress` in `task-foo-bar.md`; applying anyway"), but proceed with the update — the user may have already changed it elsewhere and the kanban will reconcile via stale-detection on next load.
4. **Regenerate the kanban** by invoking the kanban-board skill (this clears any deltas whose source now matches the `to` value via the auto-clear logic).
5. **Confirm** to the user: list the files edited, count of changes applied, and any warnings.

The pasted block contains only status and type changes. Reorder is never sent (it's local-only UI state).

## After generating

1. Tell the user the file path: `.project-kb/outputs/kanban.html`.
2. Suggest opening it: `open .project-kb/outputs/kanban.html`.
3. If this is the first generation in a project, mention that drag-and-drop changes are stored in browser localStorage and need to be pasted back to apply — point at the in-page `?` help.
4. Note that newly initialized project-kb directories gitignore this generated HTML file by default.

## What to leave out

- No filtering controls or search boxes. The board is intentionally minimal.
- No inline title or body editing. The markdown files are the source of truth for prose; editing happens via the agent.
- No card creation from the board. New items still come in via `inbox/` and the existing inbox-processing flow.
- No delete affordance. Use status `dropped` to soft-archive an item that won't be done.
- No touch / mobile drag-and-drop. Cards are read-only below 800px.
- No keyboard drag (no Space-pickup, arrow-move, etc.). Cards remain keyboard-focusable for opening the side panel only.
- No external CSS/JS, no CDN, no markdown library, no build step. One self-contained HTML file.
- No inbox archive count anywhere — only the live, unprocessed inbox.
- No automatic write-back to source files. The kanban never modifies anything outside the user's browser; changes flow through the paste-back protocol.
