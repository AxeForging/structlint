# Spec 001 — `--staged` validation mode

## Problem

`structlint validate --changed-only` diffs against `HEAD`, which is wrong for a pre-commit hook: it sees files modified in the working tree that the user has NOT staged, and misses files that are ONLY staged (added with `git add -p`, staged renames, etc.). This makes the tool unusable as a real pre-commit gate — either it complains about work-in-progress that isn't part of the commit, or lets forbidden files slip through if staged from a clean tree.

Related: `--changed-only` ignores its own contract for directory-level rules. `ValidateDirStructure` walks the whole tree unfiltered, so a pre-existing `unallowed_directory` violation somewhere else in the repo blocks every commit even when the touched files are fine. This is a latent bug worth fixing at the same time.

## Approach

1. Add a `--staged` bool flag to `validate` (env `STRUCTLINT_STAGED`). When set, run `git diff --cached --name-only --diff-filter=ACMRT` instead of `HEAD`. `--staged` implies changed-only. If both `--staged` and `--changed-only` are given, `--staged` wins (staged is a more specific mode of the same feature).

2. Apply the changed-set to `ValidateDirStructure` as well: a directory is in-scope if it equals a changed path OR is an ancestor of one. Rationale: if a commit doesn't touch `tmp/` there's nothing new to say about `tmp/` — that's a separate cleanup, not a hook blocker.

3. Keep `ValidateRequiredPaths`, `ValidateRequiredFiles`, and `ValidateRequiredGroups` global — they assert existence. Filtering them by changed-set would let a commit that accidentally deletes `README.md` pass silently because "README.md wasn't in the diff". Document this asymmetry.

## Non-goals

- Not changing `--changed-only` semantics against HEAD; only adding staged as an alternative source of the changed set + tightening dir filtering.
- Not implementing `hook install` (that is spec 002).

## Backward compatibility

- Exit codes unchanged (0/1/2/3).
- New flag is additive; env var is additive.
- Full validation runs (no `--changed-only`, no `--staged`) are byte-identical.
- Changed-only runs will report FEWER dir-level violations than before. This is a behavior change positioned as a bugfix — mention prominently in the changelog. Callers who relied on `--changed-only` reporting global dir drift should switch to full-tree validation for that use case (which is what they actually wanted).

## Design details

### Flag wiring (`internal/cli/validate.go`)

```go
&cli.BoolFlag{
    Name:    "staged",
    Usage:   "only validate staged files (git diff --cached); implies --changed-only",
    Sources: cli.EnvVars("STRUCTLINT_STAGED"),
},
```

Action:

```go
staged := cmd.Bool("staged")
changedOnly := cmd.Bool("changed-only")
if staged || changedOnly {
    v.LoadChangedPathsMode(path, staged)
}
```

### Validator (`internal/validator/validator.go`)

- New unexported `loadChangedPaths(path string, staged bool)` doing the work.
- New exported `LoadChangedPathsMode(path string, staged bool)` — the primary API going forward.
- Keep exported `LoadChangedPaths(path string)` as a wrapper calling `loadChangedPaths(path, false)` so the root `validator_test.go` and any external callers keep compiling.
- Staged git command: `git diff --cached --name-only --diff-filter=ACMRT`. Otherwise unchanged from today.
- New `shouldSkipChangedDir(relPath string) bool`: `false` when `ChangedOnly` is off; otherwise `true` unless the dir equals or is an ancestor of some changed path (prefix check with `/` boundary; treat `"."` as always in-scope so the root is walked).
- `ValidateDirStructure` calls `shouldSkipChangedDir(relPath)` on directories; returns `filepath.SkipDir` when skipping so we don't descend into unrelated subtrees at all (wall-clock win + prevents deep unrelated walks).

### Ancestor check

`isAncestorOfChanged(relPath)` — for each changed file `f`, `strings.HasPrefix(f, relPath+"/")` (or `relPath == "."`) is a hit. Small (# of changed files × # of dirs visited); no need for a trie.

## Tests

`test/staged_mode_test.go` (binary-based, per team convention):

- `TestStagedMode_CatchesStagedViolation` — init git repo in tmpdir, structlint config forbidding `*.env*`, `git add .env.local` (no commit), run `validate --staged`, expect exit 1 + `.env.local` in output.
- `TestStagedMode_IgnoresUnstagedViolation` — same setup but file is NOT staged (only written to working tree), expect exit 0.
- `TestStagedMode_IgnoresPreExistingDirViolation` — commit a `tmp/` dir with a stub file, then in a separate commit stage a change to an allowed file; run `validate --staged`, expect exit 0 (the pre-existing `tmp/` is out of scope).
- `TestStagedMode_NoGitRepoIsGraceful` — tmpdir with no `.git`, expect the existing fallback behavior (git command fails → empty changed set → all file checks skipped, no crash).
- `TestChangedOnly_DirFilterAlsoApplies` — full-tree HEAD-diff variant of the pre-existing-dir case to lock in the dir filtering fix.

Helper `initGitRepo(t, dir)` in the test file: `git init`, `git config user.email/user.name` (local), initial commit of `README.md` so `HEAD` exists.

## Verification

- `go test -race ./...` including the new file.
- `make build && ./bin/structlint validate` (self-dogfood).
- Manual: in a scratch git repo, `git add .env.local`, run `bin/structlint validate --staged` from that repo — should fail. `git reset HEAD .env.local`, rerun — should pass.
