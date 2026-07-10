# Setup action

## Problem

The root action installs and runs Structlint, so workflows that orchestrate it
through Gauntlet must duplicate installer shell code.

## Approach

Add an install-only `setup/action.yml` composite action. It accepts `version`
(default `v0.6.0`) and `install-dir` (default runner temp), invokes the existing
checksum-verifying installer, adds the directory to `PATH`, and verifies the
binary. Keep the root action backward-compatible.

## Requirements

- Never compile Structlint or silently fall back to `go install`.
- Preserve checksum verification from `install.sh`.
- Permit an explicit pinned tag or `latest`.
- Expose the installed version and binary path as outputs.
- Document pinned usage through Gauntlet.

## Test plan

- Existing executable installer regression test remains authoritative.
- Exercise the composite action in the repository PR workflow.
- Run the real Structlint and Dupehound gates through Gauntlet.

## Rollout

Pilot in Structlint, then apply the same contract to Dupehound and Gauntlet.
