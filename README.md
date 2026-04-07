# atami-ai

Shared home for the Atami team's AI-related assets.

This repository contains:

- **`project-kb/`** — the project knowledge base system: a folder structure and set of LLM skill files for capturing meeting notes, tracking items, and generating release notes.
- **`skills/`** — general team skills not tied to project-kb (code review, testing conventions, etc.). Currently a placeholder.
- **`cli/`** — source for the `atami` CLI tool that initializes project-kb in repos and will sync shared skills.

## What is project-kb?

Project-kb is a lightweight, LLM-powered knowledge base that lives inside each of our project repos. It's designed to:

1. Turn unstructured meeting notes into organised, actionable items.
2. Generate shareable outputs (HTML pages, summaries) that explain technical problems to non-developers.
3. Maintain release notes automatically as work gets completed.

Each project gets its own `.project-kb/` directory containing inputs, compiled items, and project-specific overrides. The skill files that drive the LLM behaviour are synced from this shared repo.

See `project-kb/README.md` for the full system description.

## The atami CLI

Install the CLI from the repo root with:

```sh
./scripts/install-atami
```

If your team already uses `make`, the same flow is available as:

```sh
make install-atami
```

The `atami` CLI provides commands like:

```
atami kb init                  # Scaffold project-kb into the current project
atami kb process               # Process the inbox
atami kb release --week        # Generate weekly release notes
atami kb skills pull           # Refresh synced project-kb skills from this shared repo
```

`atami kb init` fetches canonical project-kb content from the shared repo by default. Local checkouts are only used when explicitly requested for development.
