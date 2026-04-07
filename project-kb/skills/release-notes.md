# Skill: release notes

You are generating release notes from completed items in `items/`. The output goes in `releases/`.

## When to use this skill

When the user asks for release notes, a changelog, or a summary of what's been completed. They will specify a scope — a time range, a single item, or a custom selection.

## How to gather completed items

1. Read `items/index.md` to find items with status `done`.
2. For each done item, read the item file and check:
   - The `updated` date in frontmatter (this is when it was marked done).
   - The log section for the completion entry.
3. Filter to only items completed within the requested scope.
4. If the user asks for notes on a specific item, use only that item regardless of date.

## Scaling detail to scope

The core rule: **more items = less detail per item.** The user should get a complete picture at every granularity, just at different zoom levels.

### Single item

Full detail. This is a mini-article about one change.

- What was the problem or motivation (2-3 sentences).
- What changed (specific and concrete).
- What the user-facing impact is (what they'll notice).
- Any caveats, known limitations, or follow-ups.
- Roughly 100-200 words.

Example:

```markdown
# Release note: Mobile login performance fix

## What changed

The login flow on mobile devices was taking 3-5 seconds longer than expected.
Investigation revealed that the token refresh was making a redundant network call
on every login attempt — the client was refreshing a token that was still valid.

The fix skips the refresh when the current token hasn't expired, reducing mobile
login time by approximately 3 seconds on average.

## Impact

Users on mobile devices will see noticeably faster login. No changes to the
login UI or flow — this is a backend optimisation that's transparent to users.

## Notes

Desktop login was not affected by this issue. The same redundant call existed
but had negligible impact due to faster network conditions.
```

### One week (roughly 1-5 items)

Concise list. Each item gets a short paragraph — enough to understand what changed and why it matters, but no deep background.

- Group by type: features, bug fixes, improvements.
- 2-3 sentences per item.
- Include the user-facing impact in each entry.

Example:

```markdown
# Release notes — week of 2026-04-06

## Bug fixes

- **Mobile login performance**: Fixed a redundant token refresh that was adding
  3-5 seconds to mobile login times. Mobile users should see significantly faster
  logins.

- **Dashboard timeout**: Resolved an intermittent timeout when loading the main
  dashboard with large datasets. The query now uses pagination instead of loading
  all records at once.

## Improvements

- **API response times**: List endpoints now return paginated results by default,
  reducing average response times from 800ms to 200ms for large collections.
```

### One month (roughly 5-20 items)

High-level summary. Each item gets a single line. The focus is on the shape of progress — what areas were worked on, what the highlights are.

- Group by type.
- One line per item, no more than one sentence.
- Open with a 1-2 sentence overview of the month if there's a theme.

Example:

```markdown
# Release notes — April 2026

This month focused on performance improvements and laying the groundwork
for dark mode support.

## Features

- Dark mode support across all screens.
- API v2 with URL-based versioning.

## Bug fixes

- Fixed slow mobile login (redundant token refresh).
- Resolved intermittent dashboard timeout on large datasets.
- Corrected currency formatting in billing exports.

## Improvements

- Reduced API response times for list endpoints.
- Improved error messages for form validation failures.
```

### Custom range or large scope (20+ items)

At this scale, group by area of the product rather than (or in addition to) type. Each item gets a single line. Consider adding a "highlights" section at the top with the 3-5 most significant changes.

### Adaptive rule

If the user asks for "last week" but only one item was completed, produce single-item detail — don't pad a sparse week into a list format. If they ask for "last month" but it was an unusually busy month with 30+ items, compress to one-liners or group into sub-categories. Always match the level of detail to the actual volume of content.

## Tone and audience

Release notes are typically read by a mix of people — developers, product leads, sometimes end users. Write in plain English by default:

- Lead with what changed, not how.
- Use "users will see..." or "this fixes..." rather than "we refactored the token service to..."
- Avoid internal jargon unless the user specifies the notes are for an internal developer audience.
- If the user asks for technical release notes, include implementation detail and code references.

## File naming

Name the output with the scope:

- `releases/2026-04-06-weekly.md`
- `releases/2026-04-monthly.md`
- `releases/feat-dark-mode-release-note.md`
- `releases/2026-q1-quarterly.md`

## After generating

Tell the user where the file is and what it covers. If any done items had sparse information (e.g. marked done but with no detail about what was actually implemented), flag them so the user can fill in the gaps.
