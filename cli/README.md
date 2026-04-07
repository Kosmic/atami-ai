# atami CLI

The `atami` command-line tool for managing project-kb in Atami team projects.

## Status

Phase 1 currently implements only `atami kb init`. The command structure is set up so `atami skills` and additional `atami kb` commands can be added in later phases.

## Building

From inside `cli/`:

```sh
go build -o atami .
```

This produces an `atami` binary in the current directory. To install it on your PATH:

```sh
go install .
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

### Locating the atami-ai repo

The CLI needs to know where the `atami-ai` shared repo lives on your machine. It checks these locations in order:

1. The `--template-source` flag if provided.
2. The `ATAMI_AI_PATH` environment variable.
3. `~/atami-ai/`
4. `~/code/atami-ai/`
5. `~/Code/atami-ai/`
6. `~/dev/atami-ai/`

If none of these work, set `ATAMI_AI_PATH` in your shell profile.

## Tests

```sh
go test ./...
```
