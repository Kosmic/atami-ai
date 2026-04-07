# Skill: process inbox

You are processing raw, unstructured input from the `inbox/` directory and producing structured item files in `items/`.

## When to use this skill

When the user asks you to process the inbox, compile notes, or when new files appear in `inbox/`.

## How to process

1. Read all `.md` files in `inbox/` (ignore the `inbox/archive/` subdirectory — those have already been processed).
2. For each file, extract discrete items. A single file (e.g. meeting notes) will often contain multiple items. Each item is one of:
   - **Task** (`task-`): a small task, not big enough to be a feature.
   - **Feature** (`feat-`): a new capability or enhancement to build.
   - **Bug** (`bug-`): something that's broken or not working as expected.
   - **Discussion** (`disc-`): an open question, decision to be made, or topic that needs further input.
3. For each extracted item, check whether a matching item already exists in `items/`. Match on substance, not exact wording — if someone describes the same bug differently, it's the same item. If a match exists, update that item with the new information rather than creating a duplicate.
4. Create or update the item file using the format below.
5. After processing all items, update `items/index.md`.
6. After processing a file, move it from `inbox/` to `inbox/archive/`. This keeps the inbox clean — it should only ever contain unprocessed input. The archive serves as a safety net if you ever need to revisit the original notes. Never delete files from the archive.

## Item file format

Every item file uses this structure. Use YAML frontmatter for machine-readable metadata and markdown body for human-readable content.

```markdown
---
status: discussed | planned | in-progress | done | dropped
type: task | feature | bug | discussion
created: YYYY-MM-DD
updated: YYYY-MM-DD
source: filename(s) in inbox/archive/ that this item originated from
---

# [Short, descriptive title]

## Summary

[2-3 sentences describing what this item is about. Write clearly enough that someone with no context can understand the core point.]

## Detail

[Everything relevant from the source material. Preserve technical specifics, names of people who raised or own the item, and any constraints or requirements mentioned. If the source contained code snippets, error messages, or specific data, include them here.]

## Open questions

- [List anything unresolved. Who needs to weigh in? What decisions are pending? What information is missing?]

## Log

- YYYY-MM-DD: Created from [source filename]. Status: discussed.
```

## Item file naming

Use the prefix and a short kebab-case slug:

- `task-cleanup-migrations.md`
- `feat-dark-mode.md`
- `bug-mobile-login-timeout.md`
- `disc-api-versioning-strategy.md`

Keep slugs short but specific enough to identify the item at a glance.

## Index file format

`items/index.md` is a master list of all items. Keep it updated every time you create or modify an item.

```markdown
# Items

## Active

| Item | Type | Status | Updated |
|------|------|--------|---------|
| [Dark mode](feat-dark-mode.md) | feature | discussed | 2026-04-06 |
| [Mobile login slow](bug-mobile-login-slow.md) | bug | in-progress | 2026-04-08 |

## Done

| Item | Type | Completed |
|------|------|-----------|
| [Fix dashboard timeout](bug-dashboard-timeout.md) | bug | 2026-03-28 |

## Dropped

| Item | Type | Reason |
|------|------|--------|
| [Custom themes](feat-custom-themes.md) | feature | Deprioritised in favour of dark mode |
```

## Updating existing items

When new information arrives about an existing item (e.g. someone adds feedback, a decision is made, work begins):

- Update the `status` and `updated` fields in frontmatter.
- Add new information to the relevant section (detail, open questions).
- Append an entry to the log with the date and what changed.
- Update the index to reflect the new status.

## What to preserve from raw input

Err on the side of keeping too much rather than too little. Specific details matter:

- Names of people ("Sarah thinks it might be the token refresh").
- Exact error messages or code references.
- Constraints ("need to decide before the next release").
- Priority signals ("low priority but everyone agrees").
- Dissenting views or alternative approaches mentioned.

Restructure and clarify, but do not discard specifics from the source material.
