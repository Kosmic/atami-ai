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

## Build flow

The skill ships a fully built template at `.project-kb/skills/assets/kanban-board.template.html`. All HTML, CSS, JS, drag-and-drop, side panel, pending-changes tray, help modal, and the inline markdown renderer live there — you do not regenerate any of that.

For migration compatibility, older projects may still have the template at `.project-kb/skills/kanban-board.template.html`. Prefer the `assets/` path when it exists, but use the old top-level path as a fallback rather than regenerating the template by hand.

**Two slots in the template are designed to be substituted at build time**, each bounded by stable marker comments. Use the `Edit` tool to swap each slot in place; do **not** rewrite the whole file.

### Slot 1: `// === KANBAN_VARS_START === … // === KANBAN_VARS_END ===`

The default block looks like:

```js
// === KANBAN_VARS_START === do not remove (build-time substitution slot)
const PROJECT = 'default';
const TODAY = new Date();
const EYEBROW = 'PROJECT KB';
const INBOX = [];
// === KANBAN_VARS_END ===
```

Replace with values for this build:

- `PROJECT` — slugified project name from `kb-config.yaml` (lowercase, dashes), or `'default'` if unset. Used as the localStorage key suffix (`kanban.<project>`), so each project has independent pending-change state.
- `TODAY` — `new Date('YYYY-MM-DDT00:00:00Z')` set to today's date in UTC. Drives all relative-date strings on cards and the "Generated YYYY-MM-DD" subtitle.
- `EYEBROW` — short uppercase label shown above the H1, derived from the project name (e.g. `'ATAMI KB'`). Falls back to `'PROJECT KB'`.
- `INBOX` — array of unprocessed inbox filenames, e.g. `['raw-feedback-2026-05-06.md', 'idea-search.md']`. Count files matching `inbox/*.md` (top level only, not `inbox/archive/**`). The banner is rendered by JS from this array — do not template the banner HTML yourself.

### Slot 2: `// === KANBAN_ITEMS_START === … // === KANBAN_ITEMS_END ===`

The default block looks like:

```js
// === KANBAN_ITEMS_START === do not remove (build-time substitution slot)
const ITEMS = [];
// === KANBAN_ITEMS_END ===
```

Replace with the array of item objects (see [Item shape](#item-shape) below).

### Steps

1. If `.project-kb/outputs/kanban.html` does not exist, or exists but does not contain both `KANBAN_VARS_START` and `KANBAN_ITEMS_START` markers, copy the template over it. Prefer the current asset path:
   ```
   cp .project-kb/skills/assets/kanban-board.template.html .project-kb/outputs/kanban.html
   ```
   If that file is missing but `.project-kb/skills/kanban-board.template.html` exists, copy the old top-level template instead.
2. Compute the new values for slot 1 and the new `ITEMS` array for slot 2.
3. `Edit` the output file twice — once for each slot, matching the marker-bounded block exactly.
4. Done. Tell the user the path and suggest `open .project-kb/outputs/kanban.html`.

The user's pending changes live in **localStorage**, not in the HTML file, so they survive regeneration automatically. On the next page load the JS reads localStorage, auto-clears deltas whose `to` value matches the new source state, and flags stale ones whose `from` no longer matches.

If both `.project-kb/skills/assets/kanban-board.template.html` and `.project-kb/skills/kanban-board.template.html` are missing (older project that hasn't run `atami kb skills pull` recently), see [Recovering when the template is missing](#recovering-when-the-template-is-missing) at the bottom of this file.

## Item shape

For each `.md` file in `items/` (excluding `index.md`), build:

```js
{
  id: "task-foo-bar",            // filename without .md
  title: "Foo bar",              // the H1 of the markdown body
  type: "task",                  // task | feature | bug | discussion (from frontmatter)
  status: "planned",             // discussed | planned | in-progress | done | dropped (from frontmatter)
  created: "2026-04-06",         // YYYY-MM-DD (from frontmatter)
  updated: "2026-04-06",         // YYYY-MM-DD (from frontmatter)
  source: "raw-inbox-filename.md", // from frontmatter
  body: "## Summary\n\n…full markdown body, raw, after the H1…"
}
```

`body` is the **raw markdown** (everything after the H1, kept verbatim). Do not pre-render it to HTML — the template's `md2html()` function converts at panel-open time. Do not include a `teaser` field; the template derives it from `body`.

Order of objects in the array doesn't matter — JS sorts within each column at render time.

Use a JS array literal in the slot, with `body` as a backtick-delimited template literal so multi-line markdown stays readable. Backticks inside the body must be escaped (`` \` ``); `${` sequences inside the body must be escaped (`\${`).

## Applying pasted changes

When the user pastes a block beginning with `Apply these kanban changes to project-kb items:`, do **not** invoke this skill (that regenerates the file). Instead:

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

## What the template provides (for reference)

You don't need to re-derive any of this when building — it's all already in `assets/kanban-board.template.html`. Listed so you know what features the user is getting:

- Header with eyebrow, "Kanban" H1, subtitle showing `<active count> active items · Generated YYYY-MM-DD`, plus pending-changes link when any deltas are staged.
- Browser tab title set to `<project> Kanban` when a project name is available.
- `Group by: [Type] [Status]` segmented toggle (default `type`, persisted in localStorage), and a `?` help icon that auto-opens on first visit.
- Inbox banner with three states: empty (slate), 1 file (amber, lists name), 2+ files (amber, lists names).
- Active board grid: in **Group by Type** mode, columns are `task` and `discussion` (always rendered) plus `feature`/`bug` if any active items have those types. In **Group by Status**, columns are `discussed`/`planned`/`in-progress`/`done`/`dropped` always.
- Archive `<details>` (Group by Type only) with sub-columns Done and Dropped.
- Cards: primary badge (status when grouped by type, type when grouped by status; always type in archive), relative updated date ("today", "yesterday", "N days ago", etc.), title, two-line teaser derived from `## Summary` (or first paragraph if absent), copyable filename footer.
- Pending-state amber-dashed border and ● indicator with a hover ↺ revert button. Stale-state rose-dashed border and ⚠ indicator with hover ✕ discard button.
- Drag-and-drop between columns to stage status/type changes; drag within a column to reorder (local-only). Disabled below 800px viewport width.
- Side panel slides in from the right when a card is clicked. Shows type/status/updated/created badges, a clickable `Source <filename>` button that copies the filename to clipboard, the rendered markdown body, and any pending-change notes.
- Pending-changes tray bottom-right: amber pill expanding to a panel listing each delta with per-row `Revert`, plus footer `Discard all` and `Copy as instructions` (which writes the paste-back block to the clipboard).

## Recovering when the template is missing

If neither `.project-kb/skills/assets/kanban-board.template.html` nor the legacy `.project-kb/skills/kanban-board.template.html` exists (e.g., the project was initialized before the template existed and hasn't synced skills since):

1. Tell the user. Suggest they run `atami kb skills pull` to sync the latest skills, including the template.
2. If they want to proceed anyway without the template, you can copy the canonical template content from the canonical source — but do not attempt to regenerate it from a spec by hand. The template is ~700 lines of CSS/JS/HTML and re-deriving it is slow and error-prone. Prefer asking them to sync.
