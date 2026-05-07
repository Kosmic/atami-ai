# atami CLI

The `atami` command-line tool for managing project-kb in Atami team projects.

## Status

The CLI currently implements:

- `atami kb init`
- `atami kb skills pull`

The top-level `atami skills` namespace remains available for future non-KB skills.

## Building

From inside `cli/`:

```sh
go build -o atami ./atami
```

This produces an `atami` binary in the current directory. To install it on your PATH:

```sh
go install ./atami
```

From the repository root, the recommended install flow is:

```sh
./scripts/install-atami
```

Or, if your environment already uses `make`:

```sh
make install-atami
```

## Usage

### Initialise project-kb in a project

From inside the project's root directory:

```sh
atami kb init --name "My Project" --description "Brief description for LLM context"
```

This will:

1. Create a `.project-kb/` directory with the standard structure.
2. Sync the canonical skill files from the atami-ai shared repo.
3. Append a project-kb section to your `AGENTS.md` (creating it if missing).

The generated `.project-kb/` includes its own `.gitignore`. It ignores `.project-kb/outputs/kanban.html`, so project-local kanban board HTML can be generated without being committed.

If `.project-kb/` already exists, the command will refuse to run unless you pass `--force`.

### Template and skill source

By default, `atami kb init` downloads the current template and project-kb skill files from the canonical GitHub repository.

For local development and tests, you can explicitly override the source in this order:

1. The `--template-source` flag if provided.
2. The `ATAMI_AI_PATH` environment variable.

If neither override is set, the CLI fetches `Kosmic/atami-ai@main` from GitHub. You can override the remote source with `ATAMI_GITHUB_OWNER`, `ATAMI_GITHUB_REPO`, and `ATAMI_GITHUB_REF`.

### Refresh project-kb skills

From inside a project that already has `.project-kb/`:

```sh
atami kb skills pull
```

This command fetches the canonical files from `project-kb/skills/` in the shared repo, including shared assets such as HTML templates, and writes them into `.project-kb/skills/`. It does not modify `.project-kb/skills/overrides/`.

The command stores sync state in `.project-kb/.atami/kb-skills-state.json` and uses that to detect hand-edits to previously synced files. If a synced file has been edited locally, it is skipped by default and reported in the command summary. To overwrite those local edits intentionally:

```sh
atami kb skills pull --force
```

## Tests

```sh
go test ./...
```
