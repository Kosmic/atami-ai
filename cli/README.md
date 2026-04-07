# atami CLI

This directory will contain the source for the `atami` CLI tool.

## Planned commands

```
atami kb init                  # Scaffold project-kb into the current project
atami kb process               # Process the inbox
atami kb release --week        # Generate weekly release notes
atami kb release --month       # Generate monthly release notes
atami kb release --item <slug> # Generate release note for a single item
atami kb output <slug> --for <role>  # Generate shareable output
atami kb status                # Show items index summary

atami skills pull              # Refresh synced skills (KB and team skills)
atami skills pull --kb-only    # Only refresh KB skills
atami skills list              # Show available skill domains
atami skills diff <skill>      # Diff local override against canonical version
```

## Behaviour notes

- `kb init` copies `project-kb/template/` and `project-kb/skills/` from the atami-ai repo into the target project's `.project-kb/` directory, then appends the `AGENTS.md.snippet` to the project's `AGENTS.md` (creating it if missing).
- `skills pull` refreshes files in `.project-kb/skills/` only — it never touches `.project-kb/skills/overrides/`.
- The CLI should detect hand-edits to synced files (via stored hashes) and warn before overwriting.

Currently empty. Implementation TBD.
