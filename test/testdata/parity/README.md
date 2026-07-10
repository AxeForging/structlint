# Parity fixtures for the rule-engine refactor

Each subdirectory here is a self-contained project fixture used by
`test/engine_parity_test.go`. Golden files (`.golden.txt`, `.golden.json`,
`.golden.exit`) were captured from the pre-refactor binary; the refactor
in spec 005 must produce byte-identical output on every fixture.

To regenerate goldens (only when the fixture itself changes or a
deliberate output change is being merged), set `UPDATE_PARITY=1` before
running `go test`.
