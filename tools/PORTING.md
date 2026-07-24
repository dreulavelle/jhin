# PTT → Jhin parser sync

The Go handler table is **generated** from the vendored PTT sources, so Jhin
stays in lockstep with [dreulavelle/PTT](https://github.com/dreulavelle/PTT).

## Layout

- `tools/upstream/ptt/` — pinned PTT source snapshot (see `VERSION`)
- `tools/gen_handlers.py` — AST-based generator → `parser/handlers_gen.go`
- `tools/gen_overrides.py` — hand-maintained Go entries for what the
  generator cannot translate (keyed by exact upstream pattern, so upstream
  changes force re-review)
- `parser/handlers_custom.go` — Go ports of PTT's custom function handlers
  and transformer helpers the generated table references

## Re-syncing to a new PTT version

1. Update `tools/upstream/ptt/` with the new source files + `VERSION`
2. `python3 tools/gen_handlers.py`
3. If it reports unresolved handlers: update `tools/gen_overrides.py`
   (a changed upstream pattern invalidates its override key on purpose)
4. `go test ./...` — the ported PTT test corpus is the accuracy contract

## Translation rules

| Python (PTT) | Go (jhin/parser) |
|---|---|
| leading `(?<=X)` / `(?<!X)` | `ValidateMatch: validate_lookbehind(X, flags, ±)` |
| trailing `(?=X)` / `(?!X)` | `ValidateMatch: validate_lookahead(X, flags, ±)` |
| leading `(?!^)` | `validate_not_at_start()` |
| mid-pattern lookaround | manual entry in `gen_overrides.py` |
| `boolean` | `to_boolean()` |
| `value("x")` | `to_value("x")` / `to_value_sub` for `$1` templates |
| `uniq_concat(value("x"))` | `to_value_set("x")` + field in `value_set_field_map` |
| `range_func` | `to_int_range()` |
| `year_range` | `to_year()` |
| `integer` | `to_int_array()` |
| `first_integer` | `to_first_int_array()` |
| `array(integer)` | `to_int_array()` |
| `date(fmt)` | `to_date(goLayout)` via overrides (arrow→Go layout) |
| `transform_resolution` | `to_transformed_resolution()` |
| custom function handler | `CUSTOM[name]` → `parser/handlers_custom.go` |
| `skipIfAlreadyFound: False` | `KeepMatching: true` |
| `remove` / `skipFromTitle` / `skipIfFirst` | `Remove` / `SkipFromTitle` / `SkipIfFirst` |

## Engine semantics ported from parse.py (v1.6.16)

- `end_of_title` only moves when `match_index > 1` (not merely non-zero)
- an empty capture group falls back to the raw match
- `matched[name]` keeps the FIRST match index per field (skipIfFirst uses it)
- `episodes`/`seasons`/`languages` default to empty arrays in the result
- adult detection: one big `\b(kw1|kw2|...)\b` alternation built from
  `keywords/combined-keywords.txt` (embedded via `go:embed`), RE2 handles the
  1000+ alternation efficiently
- `translate_languages` option maps ISO codes → English names
- `clean_title`: expanded non-English ranges (Arabic, Thai, Kannada,
  Malayalam, CJK compat), empty-bracket stripping, trailing `mp3` strip,
  special-char spacing cleanup

## Notes

- PTT's `anime.py` handler set is NOT registered by `add_defaults()` in
  v1.6.16 — it is intentionally excluded here too (future opt-in).
- The old hand-ported table (pre-generator) seeded the override entries; the
  ported PTT test suite is the arbiter of semantic equivalence.

## Known emulation subtleties (all golden-verified)

- Go `\w`/`\W` are ASCII-only vs Python's unicode-aware classes; clean_title
  regexes are widened with `\p{L}\p{N}` and the episode fallback patterns use
  explicit unicode classes. Generated handler patterns keep plain `\W` — the
  golden corpus found no divergence there.
- `endOfTitle` is tracked in runes (Python slices characters, Go bytes).
- Python strips whitespace from every string value post-transform; the engine
  replicates this.
- Handler side effects (remove/end-of-title/transform) apply only on an
  actual per-iteration match (`matchedNow`), mirroring PTT's match-dict flow.
