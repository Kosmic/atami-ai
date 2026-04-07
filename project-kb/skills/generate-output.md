# Skill: generate output

You are generating a shareable artifact from an item in `items/`, targeted at a specific audience. The output goes in `outputs/`.

## When to use this skill

When the user asks you to create something shareable — an explanation, a visual, a summary — for a specific item, usually to send to another team member who may or may not be technical.

## Before you start

1. Read the item file the user references.
2. Ask or infer who the audience is. If the user says "explain this to the product lead" or "make this understandable to a non-technical person," that tells you the audience. If unclear, ask.
3. Read `kb-config.yaml` if it exists — it may list team members and their roles, which helps you calibrate the output.

## Audience calibration

Adapt the output based on who will read it. The same item should produce very different outputs for different people.

### Non-technical audience (product leads, stakeholders, designers)

- Never use code snippets, database column names, API endpoints, JSON, or technical jargon.
- Explain the problem in terms of what users experience or what the product does.
- If there are options to choose between, frame them as trade-offs with real consequences: what does the user see? what breaks? what takes longer to build?
- Use analogies where they genuinely help. Don't force them.
- If the item involves a visual flow or a before/after, create a diagram or simple HTML page.

### Technical audience (other developers)

- Include code references, schemas, file paths, and technical specifics.
- Focus on implementation detail: what needs to change, where, and what the risks are.
- Include relevant snippets from the item's detail section.
- If there are architectural options, compare them with concrete trade-offs (performance, complexity, migration cost).

### Mixed audience

- Lead with the non-technical explanation.
- Add a clearly labelled "Technical detail" section at the end for developers who want specifics.
- The non-technical section should stand alone — someone should be able to read just that part and fully understand the situation.

## Output format

Choose the format that best communicates the content:

### Markdown summary

Use when the item is straightforward and text is sufficient. Good for status updates, simple decisions, and brief explanations.

### HTML page

Use when the item benefits from visual structure — flow diagrams, before/after comparisons, option cards, or interactive elements. The HTML should be self-contained (inline CSS, no external dependencies) so it can be opened in any browser.

When creating HTML outputs:

- Keep the design clean and simple. No frameworks needed.
- Use a readable font stack and sensible spacing.
- If showing options or trade-offs, use cards or a side-by-side layout.
- If showing a flow or sequence, use a simple diagram (CSS boxes and arrows are fine).
- Make it look good enough to share externally without embarrassment.

### Slide deck (Marp markdown)

Use when the user specifically asks for slides, or when the item needs to be presented in a meeting.

## File naming

Name outputs descriptively with the item slug and audience:

- `outputs/mobile-login-slow-product-lead.html`
- `outputs/api-versioning-options-team.md`
- `outputs/dark-mode-scope-summary.md`

## What to include

Every output, regardless of format, should cover:

1. **What this is about** — a clear, jargon-appropriate summary of the item.
2. **Why it matters** — what's the impact on users, the product, or the team?
3. **What the options are** — if there's a decision to make, lay out the choices clearly.
4. **What's needed** — what action, decision, or feedback is being requested?

## What to leave out

- Internal log entries and processing metadata.
- The raw inbox source text (the output is a polished version, not a copy).
- Technical detail that the specific audience doesn't need (see audience calibration above).
- Speculation — only include information that's in the item file. If something is uncertain, label it as an open question.

## After generating

Tell the user where the output file is and suggest how to share it (e.g. "you can send this HTML file via Slack" or "this markdown can be pasted into an email").
