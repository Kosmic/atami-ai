# project-kb

The project knowledge base system.

## Contents

- **`skills/`** — the canonical skill files. These tell an LLM how to perform KB tasks (process inbox, generate outputs, compile release notes, generate the kanban board). When `atami kb init` runs in a project, these get copied into that project's `.project-kb/skills/` directory.
- **`template/`** — the directory scaffolding that gets copied into a new project when `atami kb init` runs. Contains empty starter files, an `AGENTS.md.snippet` to append to the project's AGENTS.md, a `.gitignore` for generated local artifacts, and a `kb-config.yaml` template.

## Editing skills

The skill files in `skills/` are the source of truth. Edit them here, commit, and the changes will propagate to projects when they run `atami kb skills pull`.

Projects that need a different version of a skill should copy the canonical file into their own `.project-kb/skills/overrides/` directory and edit there. Overrides are committed per-project and take precedence over the synced version.

## Editing the template

If you change `template/`, those changes only affect newly initialized projects. Existing projects keep whatever they were initialized with. If you need to retroactively update existing projects, that requires either a manual fix or a future `atami kb migrate` command.

The template-level `.gitignore` currently ignores only `outputs/kanban.html`. Other generated outputs and release notes remain visible to project Git status unless a project chooses to ignore them separately.
