# Violation codes

Every violation structlint emits carries a stable machine-readable `code` (visible in `--format json` and `--format sarif`). This document is the canonical reference.

**The codes below are frozen and append-only.** Existing codes are never renamed or removed; new rules add new codes. If you're building tooling on top of structlint output — an editor plugin, a CI parser, an AI agent — key on `code`, not on `message` text.

The 13 codes below correspond one-to-one to the `CodeDescriptions` registry in `internal/validator/codes.go`. Both directions are enforced by a test: adding a code without documenting it (or documenting one that doesn't exist) breaks the build.

## Codes

### `disallowed_directory`
Emitted by: `dir_structure.disallowedPaths` matching a directory.  
Meaning: a directory that is explicitly forbidden was found.  
Typical fix: remove or move the directory. The prohibition is deliberate — never loosen the rule to accommodate it.

### `unallowed_directory`
Emitted by: `dir_structure.allowedPaths` not matching a directory.  
Meaning: a directory exists that isn't allow-listed.  
Typical fix: **config** if the directory is intentional (add a generalized glob to `allowedPaths`, or use `structlint suggest` to get the exact addition); **tree** if it's stray leftover.

### `disallowed_file_pattern`
Emitted by: `file_naming_pattern.disallowed` matching a file.  
Meaning: a file matches an explicitly forbidden name pattern (e.g. `.env*`, `*.log`, `.DS_Store`).  
Typical fix: **tree.** Remove the file or move it out of scope. Never loosen the rule — these prohibitions typically guard secrets, junk, or files that shouldn't be committed.

### `unallowed_file_pattern`
Emitted by: `file_naming_pattern.allowed` not matching a file.  
Meaning: a file exists whose name isn't allow-listed.  
Typical fix: **config** — add `*.ext` (or the exact name for extensionless files like `Makefile`) to `file_naming_pattern.allowed`.

### `missing_required_directory`
Emitted by: `dir_structure.requiredPaths` referencing a missing directory.  
Meaning: a directory the config declares required does not exist.  
Typical fix: create the directory.

### `missing_required_file`
Emitted by: `file_naming_pattern.required` with no file matching the pattern.  
Meaning: no file matching the required pattern exists anywhere in the tree.  
Typical fix: create the file (or the first pattern-matching file).

### `placement_violation`
Emitted by: `placement[]` rule matching a file that isn't under the required root.  
Meaning: a file that matches the rule's `files` pattern lives outside every `mustBeUnder` root.  
Typical fix: `git mv` the file to a directory covered by `mustBeUnder`. `structlint suggest` produces the exact command.

### `missing_required_group`
Emitted by: `requiredGroups[].oneOf` with none of the listed files present.  
Meaning: none of the "one of" candidates exist (e.g. no `Makefile` OR `Taskfile.yml` OR `justfile`).  
Typical fix: create at least one of the listed files.

### `missing_required_group_match`
Emitted by: `requiredGroups[].eachDirMatching` with `requireMatch: true` and no directories matching.  
Meaning: the pattern matched zero directories, so the group's per-directory checks never ran.  
Typical fix: **config or tree** — either the pattern is wrong, or the expected directories are missing entirely.

### `missing_group_file`
Emitted by: `requiredGroups[].mustContain` / `mustContainOneOf` inside a matching directory.  
Meaning: a directory selected by `eachDirMatching` is missing a file it should contain (e.g. `cmd/*` matches `cmd/app/` but `cmd/app/main.go` is absent).  
Typical fix: add the required file to that directory.

### `boundary_violation`
Emitted by: `boundaries[]` rule matching a source file that imports a forbidden path.  
Meaning: a Go / TypeScript / JavaScript / Python file imports something the rule forbids.  
Typical fix: refactor the import. No mechanical suggestion — architectural intent lives in the boundary rules, and the fix depends on why the dependency exists.

### `parse_error`
Emitted by: boundary rule failing to parse a source file's imports.  
Meaning: the file is unparseable (syntactically broken source, unsupported language variant).  
Typical fix: operational — fix the source file, or exclude it via `ignore`.

### `walk_error`
Emitted by: any rule whose filesystem walk fails.  
Meaning: an I/O error reached structlint (permission denied, missing directory, disk error).  
Typical fix: operational — resolve the underlying filesystem issue.
