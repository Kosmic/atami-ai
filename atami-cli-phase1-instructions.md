# atami CLI — Phase 1 implementation plan

You are implementing Phase 1 of the `atami` CLI, written in Go. This phase focuses on the core scaffolding and the `atami kb init` command, which sets up the `.project-kb` directory structure inside a target project.

This work happens inside the existing `atami-ai` repository, in the `cli/` directory. The `cli/README.md` file currently in that directory is a placeholder spec — you can replace it with a proper README as part of this work.

## Scope of Phase 1

**In scope:**

- Go module setup inside `atami-ai/cli/`
- CLI argument parsing with `cobra`
- The `atami kb init` command, fully functional
- Basic command structure ready to be extended (`atami`, `atami kb`, with a clear path to add `atami skills` in Phase 2)
- Tests for the `kb init` behaviour
- A minimal `cli/README.md` documenting how to build and use what exists so far

**Out of scope (Phase 2):**

- `atami skills pull` and related sync commands
- `atami skills diff`
- `atami skills list`
- The `atami kb process`, `atami kb release`, `atami kb output`, `atami kb status` runtime commands
- Cross-platform release binaries and distribution
- Tampering detection via stored hashes

Do not implement anything in the "out of scope" list. Leave clean extension points (e.g. command groups in cobra) so Phase 2 can slot in cleanly.

## Command surface for Phase 1

By the end of Phase 1, the following commands must work:

```
atami                    # Print top-level help
atami --version          # Print version (use a hardcoded "0.1.0-dev" string for now)
atami kb                 # Print help for the kb command group
atami kb init            # Initialise .project-kb in the current directory
atami kb init --name "Project Name" --description "Brief description"
atami kb init --force    # Overwrite existing .project-kb if present
atami kb init --help     # Print help for init
```

The structure should make it obvious how to add more commands later — `atami skills pull`, `atami kb process`, etc.

## Repository structure to create

Inside `atami-ai/cli/`, create the following Go project structure:

```
cli/
├── README.md
├── go.mod
├── go.sum
├── main.go
├── cmd/
│   ├── root.go              # Defines the top-level `atami` command
│   ├── kb.go                # Defines the `atami kb` command group
│   ├── kb_init.go           # Implements `atami kb init`
│   └── root_test.go         # CLI surface tests (help/version/exit formatting)
├── internal/
│   ├── kbinit/
│   │   ├── kbinit.go        # Core init logic (copy template, sync skills, update config, append snippet)
│   │   └── kbinit_test.go   # Tests for the init logic
│   ├── templatefs/
│   │   ├── templatefs.go    # Locates the atami-ai repo paths needed by the CLI
│   │   └── templatefs_test.go
│   └── agentsmd/
│       ├── agentsmd.go      # Functions for reading/appending AGENTS.md
│       └── agentsmd_test.go
└── testdata/
    └── (used by tests, see test plan below)
```

The `internal/` package layout matters: it keeps the command-handling layer (`cmd/`) thin and puts the actual work in testable packages. Cobra commands should call functions in `internal/` packages and mostly just handle argument parsing, error formatting, and exit codes.

## Dependencies

Use the following Go modules. Pin to recent stable versions at the time of writing.

- `github.com/spf13/cobra` — CLI framework. This is the de facto standard for Go CLIs and is what you should use for all command parsing and help text.
- `gopkg.in/yaml.v3` — YAML parsing. Needed for validating the final `kb-config.yaml` after placeholder substitution (full schema validation is Phase 2).

Do not pull in any other dependencies in Phase 1. Specifically:

- No colour/styling libraries — use plain stdout/stderr for now. Phase 2 can add `fatih/color` if desired.
- No interactive prompt libraries — `kb init` should be fully non-interactive, driven by flags.
- No filesystem abstraction libraries — use the standard `os` and `io/fs` packages.

Use Go 1.22 or later. Specify this in `go.mod`.

## Detailed behaviour of `atami kb init`

This is the most important command in Phase 1. Implement it carefully and exactly as specified.

### Inputs

- The current working directory (this is where `.project-kb` gets created).
- Optional `--name` flag: a string to use as the project name in `kb-config.yaml`.
- Optional `--description` flag: a string to use as the project description.
- Optional `--force` flag: a boolean. When set, overwrite an existing `.project-kb/` directory.
- Optional `--template-source` flag: an explicit path to the atami-ai template directory. Used primarily for testing. If not provided, the CLI must locate the template using the strategy described below.

### Locating the template

The CLI needs to find the local `atami-ai` repo so it can read both the template and the canonical skills. Use this resolution strategy in order:

1. If `--template-source` is provided, treat it as the explicit template directory and validate it directly.
2. If the environment variable `ATAMI_AI_PATH` is set, treat it as the repo root and resolve paths from there.
3. Check `~/atami-ai/`.
4. Check `~/code/atami-ai/`.
5. Check `~/Code/atami-ai/`.
6. Check `~/dev/atami-ai/`.
7. If none of these exist, return an error explaining that the `atami-ai` repo could not be found and suggesting the user either set `ATAMI_AI_PATH` or clone the repo to one of the standard locations.

After resolution, validate the full contract the init flow needs:

- Template directory: `<repo-root>/project-kb/template/` (or the explicit template directory)
- Template content: `<template-dir>/project-kb/` and `<template-dir>/AGENTS.md.snippet`
- Canonical skills directory: `<repo-root>/project-kb/skills/`
- Canonical skills content: at least one top-level `.md` file

If any of these checks fail, return an error that says which required path is missing or invalid.

The same resolution strategy will be reused in Phase 2 for `atami skills pull`, so put it in `internal/templatefs/templatefs.go` as a reusable function that returns all required paths, for example:

```go
type ResolvedPaths struct {
    RepoRoot    string
    TemplateDir string
    SkillsDir   string
}

func ResolvePaths(explicitTemplate string) (ResolvedPaths, error)
```

Keep the search list injectable in tests so `templatefs_test.go` does not depend on the real home directory.

### The init flow

When `atami kb init` runs, perform these steps in order:

**Step 1: Resolve the required repo paths.**

Call the path resolution function. If it fails, exit with an error message and a non-zero exit code. Do not proceed.

**Step 2: Check the target directory.**

The target directory is the current working directory. Check whether `.project-kb/` already exists inside it.

- If it does not exist, proceed.
- If it exists and `--force` is not set, exit with an error: ".project-kb/ already exists in this directory. Use --force to overwrite."
- If it exists and `--force` is set, remove the existing `.project-kb/` directory entirely and proceed.

**Step 3: Copy the template directory.**

Recursively copy the contents of `<resolved.TemplateDir>/project-kb/` into `<target>/.project-kb/`. This includes all subdirectories and files, including the `.gitkeep` files. Preserve file modes.

Implement this as a reusable function in `internal/kbinit/`. Walk the source tree using `filepath.WalkDir` and replicate each file and directory at the destination. Do not use `os/exec` to shell out to `cp` — use pure Go file operations.

**Step 4: Copy the canonical skill files.**

The canonical skills live at `<atami-ai-repo>/project-kb/skills/` (note: `project-kb/skills/`, NOT `project-kb/template/project-kb/skills/`). The template's `project-kb/skills/` directory only contains the `overrides/` subdirectory; the actual skill files live in the sibling `skills/` directory under the repo root.

Copy every `.md` file from `<resolved.SkillsDir>/*.md` into `<target>/.project-kb/skills/`. Do not recurse — only top-level `.md` files. The `overrides/` subdirectory is already present from Step 3 and must not be touched.

Do not derive this path ad hoc inside `kbinit`; use the validated path returned from `ResolvePaths`. If the directory is missing or contains no top-level `.md` files, fail with a clear error.

**Step 5: Apply name and description to kb-config.yaml.**

Read `<target>/.project-kb/kb-config.yaml`. If `--name` was provided, replace the full quoted placeholder `"TODO: replace with project name"` with a YAML-safe quoted string containing the supplied value. If `--description` was provided, replace the full quoted placeholder `"TODO: brief description for LLM context"` with a YAML-safe quoted string containing the supplied value.

Do this with targeted string replacement, not full YAML rewriting, so comments and formatting are preserved. Do not insert raw user input directly into the file. Instead, generate the replacement values using `strconv.Quote(...)` so embedded quotes, newlines, and other special characters remain valid inside a YAML double-quoted scalar.

After applying any replacements, parse the final file content with `yaml.v3` before writing success output. If the resulting file is not valid YAML, return an error rather than writing a broken config.

If a placeholder is not found in the file (unlikely but possible if the template format changes), log a warning to stderr but continue.

If neither `--name` nor `--description` is provided, leave the file untouched and let the user fill in the placeholders manually.

**Step 6: Append the AGENTS.md snippet.**

Read `<resolved.TemplateDir>/AGENTS.md.snippet`. The snippet content includes the begin and end markers (`<!-- BEGIN atami project-kb -->` and `<!-- END atami project-kb -->`).

Then, in the target directory:

- If `<target>/AGENTS.md` does not exist, create it. Initial content should be a single H1 header `# AGENTS.md` followed by a blank line, then the snippet.
- If `<target>/AGENTS.md` exists, check whether it already contains the begin marker `<!-- BEGIN atami project-kb -->`. If it does, do nothing — the snippet is already present from a previous init. If not, append a blank line and then the snippet to the end of the file.

Implement the AGENTS.md handling as a function in `internal/agentsmd/`, something like `func AppendSnippet(targetPath, snippet string) error`. This will be reused in Phase 2 if we need to update the snippet.

**Step 7: Print a success summary.**

When everything completes successfully, print a summary to stdout:

```
✓ Initialised .project-kb in /path/to/target

Created:
  .project-kb/inbox/archive/
  .project-kb/items/index.md
  .project-kb/releases/
  .project-kb/outputs/
  .project-kb/skills/
  .project-kb/skills/overrides/
  .project-kb/kb-config.yaml

Synced 3 skill files into .project-kb/skills/

Updated AGENTS.md (added project-kb section)

Next steps:
  1. Edit .project-kb/kb-config.yaml to set the project name and team members.
  2. Drop your first meeting notes into .project-kb/inbox/.
  3. Ask your coding agent to process the inbox.
```

Use the literal `✓` character (U+2713). The "Created" list should reflect what was actually created (so if a directory already existed, etc., the message should be accurate). Use the relative paths shown above.

### Error handling

All errors should be wrapped with context using `fmt.Errorf("doing thing: %w", err)`. The `cmd/kb_init.go` layer should print errors to stderr in the format `Error: <message>` and exit with code 1. The `internal/kbinit/` functions should return errors, never call `os.Exit` directly.

For user errors (like ".project-kb already exists, use --force"), the message should be friendly and not include a stack trace. For unexpected errors (like a permission denied while writing a file), the error should be wrapped with enough context that the user can see what went wrong.

Have the core init function return a typed result object that records what happened, for example created paths, number of synced skills, whether `AGENTS.md` was created or updated, and whether warnings were emitted. The Cobra layer should format the success summary from this result instead of reconstructing state from side effects.

## Test plan

Tests must cover both the core behaviour of `kb init` and the public CLI surface. Use Go's standard `testing` package — no external test framework.

### Test fixtures

Create a `testdata/` directory inside `cli/`. Inside it, create a fake atami-ai template that mirrors the real structure but with minimal content. Specifically:

```
cli/testdata/fake-atami-ai/
└── project-kb/
    ├── skills/
    │   ├── process-inbox.md       # Content: "# Test process-inbox skill\n"
    │   ├── generate-output.md     # Content: "# Test generate-output skill\n"
    │   └── release-notes.md       # Content: "# Test release-notes skill\n"
    └── template/
        ├── AGENTS.md.snippet      # The actual snippet content with markers
        └── project-kb/
            ├── inbox/
            │   └── archive/
            │       └── .gitkeep   # empty
            ├── items/
            │   └── index.md       # Minimal items index content
            ├── releases/
            │   └── .gitkeep
            ├── outputs/
            │   └── .gitkeep
            ├── skills/
            │   └── overrides/
            │       └── .gitkeep
            └── kb-config.yaml     # The full kb-config.yaml template content
```

The `AGENTS.md.snippet` in the fixture should be a realistic version with the begin/end markers, but the body content can be minimal (just enough to test that it gets appended correctly).

Tests should always pass `--template-source <testdata path>` so they don't depend on the user's actual atami-ai location.

### Required tests

Implement at least these tests in `internal/kbinit/kbinit_test.go`:

1. **TestInit_FreshDirectory**: Run init in an empty temp directory with `--name` and `--description` set. Verify that all expected files exist, that `kb-config.yaml` contains the supplied values (not the placeholders), that `AGENTS.md` exists and contains the snippet markers, and that all three skill files are present in `.project-kb/skills/` with the correct content.

2. **TestInit_ExistingProjectKbWithoutForce**: Pre-create a `.project-kb/` directory in the temp dir, then run init without `--force`. Verify that init returns an error and that the existing directory is unchanged.

3. **TestInit_ExistingProjectKbWithForce**: Pre-create a `.project-kb/` directory with some sentinel file inside it, then run init with `--force`. Verify that the sentinel file is gone and the new structure is in place.

4. **TestInit_ExistingAgentsMdWithoutSnippet**: Pre-create an `AGENTS.md` with some unrelated content (e.g. "# Existing project rules\n\nDo not commit secrets.\n"). Run init. Verify that the existing content is preserved and the snippet is appended after a blank line.

5. **TestInit_ExistingAgentsMdWithSnippet**: Pre-create an `AGENTS.md` that already contains the begin marker. Run init. Verify that the file is unchanged (no duplicate snippet).

6. **TestInit_NoNameOrDescription**: Run init without supplying `--name` or `--description`. Verify that `kb-config.yaml` still contains the placeholders unchanged.

7. **TestInit_OverridesDirectoryNotTouched**: Run init twice (the second time with `--force`). Before the second run, drop a file into `.project-kb/skills/overrides/`. Wait — actually, with `--force` the entire `.project-kb/` directory is removed and recreated, so this test doesn't apply to init. Instead, write a comment in the test file noting that override preservation across re-init is intentionally NOT supported (re-init means starting over) and that override preservation will be tested in Phase 2 for `atami skills pull`.

8. **TestInit_MissingTemplateFails**: Pass `--template-source` pointing to a non-existent directory. Verify that init returns a clear error.

9. **TestInit_MalformedTemplateFails**: Pass `--template-source` pointing to a directory that exists but doesn't contain the expected structure (e.g. missing the `AGENTS.md.snippet` file). Verify that init returns a clear error.
10. **TestInit_SpecialCharactersInNameAndDescription**: Run init with values containing quotes, `#`, and newlines. Verify that the resulting `kb-config.yaml` is still valid YAML and that parsing it produces the exact supplied values.

For each test, use `t.TempDir()` to get an isolated working directory and clean up automatically.

### Tests for `internal/templatefs/`

In `templatefs_test.go`, test the resolution function:

1. **TestResolve_ExplicitPath**: Pass an explicit path that exists and is valid. Verify it's returned.
2. **TestResolve_ExplicitPathDoesNotExist**: Pass an explicit path that doesn't exist. Verify an error is returned.
3. **TestResolve_EnvVar**: Set the `ATAMI_AI_PATH` environment variable to a valid fixture path. Verify it's resolved correctly. Use `t.Setenv` to set and restore the environment variable.
4. **TestResolve_NoneFound**: Unset the env var, pass no explicit path, and run in an environment where none of the standard locations exist. Verify an error is returned. (Use `t.Setenv` to clear the env var, and keep the standard search paths configurable via a parameter so the test does not depend on the real home directory.)
5. **TestResolve_MissingSkillsDirFails**: Use a fixture with a valid template but no canonical skills directory. Verify resolution fails before `kb init` starts copying files.
6. **TestResolve_UppercaseCodePath**: Provide a valid repo rooted under `~/Code/atami-ai/` and verify it resolves correctly.

### Tests for `internal/agentsmd/`

In `agentsmd_test.go`, test the AGENTS.md handling:

1. **TestAppendSnippet_NewFile**: Call AppendSnippet on a path that doesn't exist. Verify the file is created with the expected initial content plus the snippet.
2. **TestAppendSnippet_ExistingFileWithoutSnippet**: Pre-create a file with arbitrary markdown content. Verify the snippet is appended with a blank line separator and original content is preserved.
3. **TestAppendSnippet_ExistingFileWithSnippet**: Pre-create a file that already contains the begin marker. Verify the file is unchanged.

### CLI surface tests

Add command-level tests in `cmd/root_test.go` that exercise the real Cobra commands with stdout/stderr captured:

1. **TestRoot_VersionFlag**: Verify `atami --version` prints `0.1.0-dev` and exits successfully.
2. **TestRoot_Help**: Verify `atami` prints top-level help and includes the `kb` command group.
3. **TestKb_Help**: Verify `atami kb` prints help for the `kb` group and includes `init`.
4. **TestKbInit_Help**: Verify `atami kb init --help` prints the expected flags.
5. **TestKbInit_UserErrorFormatting**: Run `atami kb init` in a directory where `.project-kb/` already exists without `--force`. Verify stderr starts with `Error: ` and the command exits with a non-zero status.
6. **TestKbInit_EndToEndHappyPath**: Run the command through the Cobra layer with `--template-source`, `--name`, and `--description`. Verify the command succeeds and the stdout summary matches the returned init result closely enough to catch regressions in user-facing output.

## README content

Replace `cli/README.md` with content along these lines:

```markdown
# atami CLI

The `atami` command-line tool for managing project-kb in Atami team projects.

## Status

Phase 1: only `atami kb init` is implemented. See the comments in `cmd/` for the planned commands that aren't yet built.

## Building

From inside `cli/`:

​```
go build -o atami .
​```

This produces an `atami` binary in the current directory. To install it on your PATH:

​```
go install .
​```

## Usage

### Initialise project-kb in a project

From inside the project's root directory:

​```
atami kb init --name "My Project" --description "Brief description for LLM context"
​```

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

​```
go test ./...
​```
```

(Note: in the actual README, the inner code fences should be regular triple backticks.)

## Conventions to follow

- Use `fmt.Errorf("doing thing: %w", err)` for error wrapping. Never use `errors.New` for an error that's wrapping another error.
- Use `filepath.Join` for path construction, never string concatenation with `/`.
- Use `os.ReadFile`, `os.WriteFile`, `os.MkdirAll` from the standard library — not the deprecated `ioutil` package.
- All exported functions in `internal/` packages should have a doc comment.
- Run `gofmt` on all generated code.
- Run `go vet` and address any warnings before considering the work done.

## When you're done

1. Run `go test ./...` from inside `cli/` and confirm all tests pass.
2. Run `go vet ./...` and confirm no warnings.
3. Run `go build .` and confirm the binary builds.
4. Manually test the happy path: in a fresh temp directory, run the binary with `--template-source` pointing at the real `project-kb/template/` in the atami-ai repo, with `--name` and `--description` set. Verify the output matches the expected summary and the created files match what's expected.
5. Commit all the new files with a message like "Phase 1: implement atami kb init".

## What you should NOT do

- Do not implement any command other than `atami kb init` (and the help/version output).
- Do not implement hash-based tampering detection — that's Phase 2.
- Do not add any dependencies beyond `cobra` and `yaml.v3`.
- Do not modify any files outside the `atami-ai/cli/` directory.
- Do not push to git remotes — leave commits local for the user to review and push.
