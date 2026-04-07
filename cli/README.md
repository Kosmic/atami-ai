# atami CLI

The `atami` command-line tool for managing project-kb in Atami team projects.

## Status

Phase 1 currently implements `atami kb init`. The command structure now also reserves the `atami kb skills` namespace for project-kb skill sync commands in a later phase, while top-level `atami skills` remains available for non-KB skills in the future.

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

If `.project-kb/` already exists, the command will refuse to run unless you pass `--force`.

### Template and skill source

By default, `atami kb init` downloads the current template and project-kb skill files from the canonical GitHub repository.

For local development and tests, you can explicitly override the source in this order:

1. The `--template-source` flag if provided.
2. The `ATAMI_AI_PATH` environment variable.

If neither override is set, the CLI fetches `atami-ai/atami-ai@main` from GitHub. You can override the remote source with `ATAMI_GITHUB_OWNER`, `ATAMI_GITHUB_REPO`, and `ATAMI_GITHUB_REF`.

## Tests

```sh
go test ./...
```
