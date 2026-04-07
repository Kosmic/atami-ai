# atami-ai

Shared home for the Atami team's AI-related assets.

This repository contains:

- **`project-kb/`** — the project knowledge base system: a folder structure and set of LLM skill files for capturing meeting notes, tracking items, and generating release notes.
- **`skills/`** — general team skills not tied to project-kb (code review, testing conventions, etc.). Currently a placeholder.
- **`cli/`** — source for the `atami` CLI tool that initializes project-kb in repos and syncs skills. Currently a placeholder.

## What is project-kb?

Project-kb is a lightweight, LLM-powered knowledge base that lives inside each of our project repos. It's designed to:

1. Turn unstructured meeting notes into organised, actionable items.
2. Generate shareable outputs (HTML pages, summaries) that explain technical problems to non-developers.
3. Maintain release notes automatically as work gets completed.

Each project gets its own `.project-kb/` directory containing inputs, compiled items, and project-specific overrides. The skill files that drive the LLM behaviour are synced from this shared repo.

See `project-kb/README.md` for the full system description.

## The atami CLI

The `atami` CLI (not yet built) will provide commands like:

```
atami kb init                  # Scaffold project-kb into the current project
atami kb process               # Process the inbox
atami kb release --week        # Generate weekly release notes
atami skills pull              # Refresh synced skills from this shared repo
```

For now, this repo just contains the source-of-truth files. Until the CLI exists, manual `cp` from `project-kb/template/` and `project-kb/skills/` into a target project achieves the same result.
