# jhin — agent guide

Go library: parse → rank → filter → sort torrent release names. Zero runtime
dependencies. Accuracy is contractual, speed is a feature, slop is a bug.

## Hard invariants — violating any of these is a broken change

- **The golden corpus is law.** `parser/testdata/golden.json` pins byte-exact
  output for 1,156 real titles. Never edit expected outputs to make a test
  pass — fix the code. New behavior lands as new corpus titles with verified
  expectations.
- **Handler order is priority.** The table in `parser/table.go` resolves
  conflicts by position; earlier handlers win and may consume text later ones
  would see. Do not reorder, merge, or "clean up" handlers.
- **The prefilter may only skip work, never change results.** Gates must stay
  necessary conditions. After touching `parser/prefilter.go`, `table.go`, or
  `handlers*.go`, run:
  `go test -run 'TestGoldenCorpus|TestPrefilterEquivalence' ./parser` and a
  fuzz smoke `go test -run '^$' -fuzz FuzzPrefilterEquivalence -fuzztime 30s ./parser`.
- **Zero runtime dependencies**, library and CLI alike. Anything needing a
  third-party package belongs in a separate module, not this one. The rule
  language in `rules/` is hand-written for this reason: an expression engine
  is a dependency, and the registry is what makes one unnecessary.
- **The rule grammar is closed; the registry is open.** Every "can it support
  X?" is answered by registering a namespace, a function or an effect — never
  by adding syntax. A closed grammar is what keeps `rules/` fuzzable, cheap
  per release, and explainable. After touching `rules/`, run
  `go test -race ./rules` and a fuzz smoke
  `go test -run '^$' -fuzz FuzzCompile -fuzztime 30s ./rules`.
- **A rule that cannot be answered is skipped, not failed.** A condition
  reading a tier the release carries nothing in never runs. Judging it against
  zero values would let one rule empty a result list, so any change that makes
  an unanswerable rule act is a bug however reasonable it looks.
- **Libraries don't log.** Failures come back as errors or data
  (`Result.Error()`, `Torrent.Rejections`, `Explain()`). Logging belongs to
  `cmd/jhin` only.

## Style

- Comments state constraints the code can't express — never narration,
  provenance, or a restatement of the next line. Match the file's density.
- No filler in docs: no marketing adjectives, no "simply", no emoji.
- Benchmark numbers live in `docs/benchmark.md` and come from a recorded run
  on a stated machine. Never hand-edit a number; re-measure and say when.

## Workflow

- `go test -race ./...` green, `gofmt -l .` empty, golangci-lint clean.
- Conventional commits drive release-please: `feat` = minor, `fix`/`perf` =
  patch, `docs`/`ci`/`test`/`chore` = no release. Don't invent types.
- Releases are immutable: never delete or re-tag a published version. The
  rollback path is a `retract` directive plus a patch release. `version.go`
  and `CHANGELOG.md` are release-please-managed — don't edit by hand.
- `parser/gatestats_test.go` guards prefilter coverage (< 70 regex
  executions per title). If it fails, the change weakened gating — tighten
  it, don't raise the limit.
