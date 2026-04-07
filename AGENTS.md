## What this repo is

`atami-ai` is the Atami team's shared home for AI-related assets: the project-kb system, general team skills, and (eventually) the source for the `atami` CLI.

This repo is the **source of truth**. Files here get synced into individual project repos via the `atami` CLI. It is not itself a project that uses project-kb.

## Repo structure

- `project-kb/skills/` — canonical skill files for the project-kb system. Edit here; changes propagate to projects via cli app (not implemented yet).
- `project-kb/template/` — directory scaffolding copied into new projects by `atami kb init`. Includes the `AGENTS.md.snippet` that gets appended to a project's AGENTS.md.
- `skills/` — placeholder for general (non-KB) team skills.
- `cli/` — placeholder for the future `atami` CLI source.

## Rules

- When editing skill files in `project-kb/skills/`, remember they will be synced into multiple project repos. Changes should be backwards-compatible where possible, or clearly flagged as breaking.
- When editing `project-kb/template/`, remember that template changes only affect newly initialised projects — existing projects keep whatever they were initialised with.
- Do not initialise a `project-kb/` directory inside this repo. This repo contains the template, not an instance.
- The `AGENTS.md.snippet` in the template uses `<!-- BEGIN atami project-kb -->` and `<!-- END atami project-kb -->` markers. Preserve these markers in any edits — the CLI relies on them to find and update the section in target projects.
