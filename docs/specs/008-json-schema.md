# Spec 008 — JSON Schema for the config file

## Problem

Writing `.structlint.yaml` is trial-and-error: no completion, no hover docs, and the strict parser's `field placment not found in type config.Config` errors only appear at run time. Editors (VS Code YAML extension, JetBrains, Neovim + yamlls) can provide all of that from a JSON Schema — structlint just doesn't ship one.

## Approach

Hand-write `schema/structlint.schema.json` (JSON Schema **draft-07** — the dialect yaml-language-server supports best) describing the full config surface. Set `additionalProperties: false` on every object, mirroring the strict parse (`yaml.UnmarshalStrict` / `DisallowUnknownFields`) so the editor squiggle matches the runtime error. Document the `# yaml-language-server:` modeline so users get validation with zero editor config. A test keeps the schema honest by asserting every yaml tag reachable from `config.Config` appears in the schema.

Hand-written, not generated: the struct tags don't carry descriptions, enums (`severity`), or string-or-list shapes (`extends`), and a generator dependency isn't worth four small object definitions. The sync test is the drift guard.

## Non-goals

- No SchemaStore submission in this repo — that is an out-of-repo follow-up (PR to `SchemaStore/schemastore` mapping `.structlint.{yaml,yml,json}`), tracked after the schema has shipped in a tagged release so the raw URL is stable.
- No runtime schema validation in the binary — the strict parser + `Config.Validate()` already do that job.
- No `$ref` to remote schemas; the file is self-contained.

## Backward compatibility

- Purely additive: a new static file plus docs. No binary behavior changes.
- The self-config gains one `allowedPaths` entry (cross-cutting trap: any new top-level dir must land in `.structlint.yaml` in the same PR, or the lefthook self-validation blocks the commit).

## Design

### `schema/structlint.schema.json` (new, hand-written)

- `$schema: "http://json-schema.org/draft-07/schema#"`, `$id` pointing at the raw GitHub URL on `main`, top-level `title`/`description`.
- `type: object`, `additionalProperties: false`, `properties`:
  - `extends` — `oneOf: [string, array of string]` with the preset names listed in the description (only if spec 007 has landed; the sync test enforces inclusion the moment the field exists).
  - `dir_structure` — object, `additionalProperties: false`; `allowedPaths` / `disallowedPaths` / `requiredPaths` as `array of string` with glob-hint descriptions.
  - `file_naming_pattern` — object; `allowed` / `disallowed` / `required` arrays.
  - `ignore` — array of string.
  - `placement` — array of object `{id (required), files (required), mustBeUnder (required), severity}`.
  - `requiredGroups` — array of object `{id (required), oneOf, eachDirMatching, mustContain, mustContainOneOf, requireMatch, severity}`.
  - `boundaries` — array of object `{id (required), from (required), cannotImport (required), severity}`.
  - `severity` everywhere as `enum: ["error", "warning"]` via a shared `definitions.severity`.
- `required` arrays in the schema mirror `Config.Validate()` (e.g. placement needs `id`, `files`, `mustBeUnder`), so editors flag what the binary would reject.

### Docs modeline snippet

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/AxeForging/structlint/main/schema/structlint.schema.json
```

Goes in `README.md` and the config docs; note that VS Code needs the Red Hat YAML extension and that JSON configs can use a `"$schema"` key instead. (The schema must therefore also allow a top-level `$schema` property for the JSON case — one carve-out from strict mirroring, called out in a comment-adjacent description.)

### Self-config

`.structlint.yaml` `dir_structure.allowedPaths` += `"schema/**"`. `*.json` is already in `file_naming_pattern.allowed`.

### Sync test (`test/schema_test.go`, new)

Parses the schema as generic JSON, then reflects over `config.Config` — recursively walking struct fields, collecting `yaml` tag names — and asserts each tag appears in the corresponding `properties` map at the right nesting level. Fails with the missing tag name, so adding a config field without touching the schema breaks CI with an actionable message. (This is a consistency test over repo artifacts, not binary behavior, so plain `reflect` in-process is the right tool; the team's binary-based convention still governs the behavioral tests.)

## Implementation steps

1. Write `schema/structlint.schema.json` covering every current `config.Config` field, `additionalProperties: false` on all objects, shared `severity` definition, `$schema` top-level carve-out.
2. Add `"schema/**"` to `.structlint.yaml` `allowedPaths` (same commit — self-validation lockstep).
3. Add `test/schema_test.go`: JSON-validity test + reflection sync test.
4. Docs: modeline snippet + editor setup notes in `README.md` and the config docs; note the SchemaStore follow-up in the tracking issue, not the repo.

## Checklist

- [ ] `schema/structlint.schema.json` — draft-07, `additionalProperties: false` everywhere, severity enum, required keys mirroring `Config.Validate()`
- [ ] `.structlint.yaml` gains `schema/**` in `allowedPaths` (self-validation stays green)
- [ ] `test/schema_test.go` — valid JSON + every yaml tag in `config.Config` present in schema properties
- [ ] Docs: yaml-language-server modeline + JSON `$schema` key variant
- [ ] SchemaStore submission noted as out-of-repo follow-up
- [ ] `make build && ./bin/structlint validate` passes with the new `schema/` dir

## Tests

`test/schema_test.go`:

- `TestSchema_IsValidJSON` — `schema/structlint.schema.json` unmarshals; asserts `$schema` is draft-07 and top-level `additionalProperties == false`.
- `TestSchema_CoversAllConfigFields` — reflection walk over `config.Config` (nested structs and rule slices included); every `yaml` tag exists under the matching `properties` path; failure message names the missing tag.
- `TestSchema_RejectsUnknownProperties` — every object node in the schema (walked recursively) declares `additionalProperties: false`, so the schema can't silently drift into permissiveness.
- `TestSchema_SelfConfigValidates` (binary-based) — run the built binary's `validate` against this repo root; the `schema/` dir and its `.json` file produce no violations (locks in the `.structlint.yaml` update).

## Verification

- `go test -race ./...`; `make build && ./bin/structlint validate` (self-dogfood with `schema/` present).
- Manual: open `.structlint.yaml` in VS Code with the YAML extension, add the modeline pointing at the local file (`$schema=./schema/structlint.schema.json`), confirm completion works and a typo key (`placment:`) gets flagged — the same typo the binary rejects.
- `python3 -m json.tool schema/structlint.schema.json` (or `jq .`) exits 0.
