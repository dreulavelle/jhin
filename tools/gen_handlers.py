#!/usr/bin/env python3
"""Generate parser/handlers_gen.go from the vendored PTT sources.

Reads tools/upstream/ptt/{handlers.py,anime.py} via the Python AST, translates
each parser.add_handler(...) call into a Go handler struct literal, and emits
the full ordered handler table.

Translation rules:
  - Python `regex` lookarounds are not supported by Go's RE2 engine:
      * a leading lookbehind  (?<=X) / (?<!X)  -> ValidateMatch: validate_lookbehind
      * a trailing lookahead  (?=X) / (?!X)    -> ValidateMatch: validate_lookahead
      * a leading (?!^)                        -> validate_not_at_start()
      * anything else (mid-pattern lookaround) -> OVERRIDES table below
  - Transformers map to the Go helpers in parser/handlers.go (to_boolean,
    to_value_set, to_int_range, ...).
  - Custom Python function handlers map to hand-written Go entries in
    parser/handlers_custom.go via the CUSTOM table.

Usage: python3 tools/gen_handlers.py   (from the repo root)
"""

from __future__ import annotations

import ast
import re
import sys
from dataclasses import dataclass, field as dc_field
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
UPSTREAM = ROOT / "tools" / "upstream" / "ptt"
OUT = ROOT / "parser" / "handlers_gen.go"

VERSION = (UPSTREAM / "VERSION").read_text().strip()

FIELD_MAP = {
    "bit_depth": "bitDepth",
    "episode_code": "episodeCode",
    "3d": "threeD",
}

# Fields accumulated via uniq_concat -> represented as value_set in the engine.
# Derived during generation; emitted as value_set_field_map.


@dataclass
class Handler:
    field: str
    pattern: str | None  # python pattern source (raw string content)
    ignorecase: bool
    transformer: ast.expr | None
    options: dict
    line: int
    source: str  # 'handlers' | 'anime'
    go: str | None = None  # final Go entry text
    notes: list = dc_field(default_factory=list)


def extract_calls(path: Path, source: str) -> list[Handler]:
    tree = ast.parse(path.read_text())
    out: list[Handler] = []
    for node in ast.walk(tree):
        if not (
            isinstance(node, ast.Call)
            and isinstance(node.func, ast.Attribute)
            and node.func.attr == "add_handler"
        ):
            continue
        args = node.args
        if not args or not isinstance(args[0], ast.Constant):
            continue  # add_handler(func) form is handled via CUSTOM by callee name
        fld = args[0].value
        pattern = None
        ignorecase = False
        transformer = args[2] if len(args) > 2 else None
        options: dict = {}
        if len(args) > 1:
            a1 = args[1]
            if isinstance(a1, ast.Call) and ast.unparse(a1.func).endswith("compile"):
                pat_node = a1.args[0]
                if isinstance(pat_node, ast.Constant):
                    pattern = pat_node.value
                else:
                    pattern = ast.unparse(pat_node)  # f-string etc: flag for override
                flags = [ast.unparse(a) for a in a1.args[1:]] + [
                    ast.unparse(k.value) for k in a1.keywords
                ]
                ignorecase = any("IGNORECASE" in f for f in flags)
            elif isinstance(a1, ast.Name):
                # custom function handler: add_handler("episodes", handle_episodes, {...})
                out.append(
                    Handler(
                        fld,
                        None,
                        False,
                        None,
                        _opts(args[2] if len(args) > 2 else None),
                        node.lineno,
                        source,
                    )
                )
                out[-1].notes.append(f"custom:{a1.id}")
                continue
            elif isinstance(a1, ast.Call):
                # pattern built by a factory, e.g. create_adult_pattern()
                out.append(Handler(fld, None, False, None, {}, node.lineno, source))
                out[-1].notes.append(f"custom:{ast.unparse(a1.func)}")
                continue
        options_node = args[3] if len(args) > 3 else None
        for kw in node.keywords:
            if kw.arg == "options":
                options_node = kw.value
            elif kw.arg == "transformer":
                transformer = kw.value
        options = _opts(options_node)
        out.append(
            Handler(fld, pattern, ignorecase, transformer, options, node.lineno, source)
        )
    out.sort(key=lambda h: h.line)
    return out


def _opts(node) -> dict:
    if node is None:
        return {}
    try:
        return ast.literal_eval(node)
    except Exception:
        return {"<non-literal>": ast.unparse(node)}


# ---------------------------------------------------------------------------
# Pattern translation
# ---------------------------------------------------------------------------

LOOKBEHIND_START = re.compile(r"^\(\?<(=|!)((?:[^()\\]|\\.|\([^()]*\))*)\)")
LOOKAHEAD_END = re.compile(r"\(\?(=|!)((?:[^()\\]|\\.|\([^()]*\))*)\)$")
ANY_LOOKAROUND = re.compile(r"\(\?<?[=!]")


def go_quote_pattern(pat: str, ignorecase: bool) -> str:
    # Python regex \uXXXX escapes -> RE2 \x{XXXX}
    pat = re.sub(r"\\u([0-9a-fA-F]{4})", r"\\x{\1}", pat)
    prefix = "(?i)" if ignorecase else ""
    if "`" not in pat:
        return f"regexp.MustCompile(`{prefix}{pat}`)"
    esc = pat.replace("\\", "\\\\").replace('"', '\\"')
    return f'regexp.MustCompile("{prefix}{esc}")'


def go_quote_str(s: str) -> str:
    if "`" in s:
        return '"' + s.replace("\\", "\\\\").replace('"', '\\"') + '"'
    return "`" + s + "`"


def translate_pattern(pat: str, ignorecase: bool) -> tuple[str, list[str]] | None:
    """Return (go MustCompile expr, validator exprs) or None if untranslatable."""
    validators: list[str] = []
    flags = "i" if ignorecase else ""

    if pat.startswith("(?!^)"):
        pat = pat[len("(?!^)") :]
        validators.append("validate_not_at_start()")

    m = LOOKBEHIND_START.match(pat)
    if m:
        polarity = "true" if m.group(1) == "=" else "false"
        validators.append(
            f"validate_lookbehind({go_quote_str(m.group(2))}, {go_quote_str(flags)}, {polarity})"
        )
        pat = pat[m.end() :]

    m = LOOKAHEAD_END.search(pat)
    if m and not ANY_LOOKAROUND.search(pat[: m.start()]):
        polarity = "true" if m.group(1) == "=" else "false"
        validators.append(
            f"validate_lookahead({go_quote_str(m.group(2))}, {go_quote_str(flags)}, {polarity})"
        )
        pat = pat[: m.start()]

    if ANY_LOOKAROUND.search(pat):
        return None  # needs manual override

    return go_quote_pattern(pat, ignorecase), validators


# ---------------------------------------------------------------------------
# Transformer translation
# ---------------------------------------------------------------------------


def translate_transformer(h: Handler, uniq_fields: set) -> list[str] | None:
    """Return list of Go struct field lines, or None if needs override."""
    t = h.transformer
    lines: list[str] = []
    if t is None or (isinstance(t, ast.Name) and t.id == "none"):
        return lines
    src = ast.unparse(t)

    if isinstance(t, ast.Name):
        match t.id:
            case "boolean":
                lines.append("Transform: to_boolean(),")
            case "lowercase":
                lines.append("Transform: to_lowercase(),")
            case "uppercase":
                lines.append("Transform: to_uppercase(),")
            case "integer":
                # int-list fields get arrays; string fields (year) keep the digits
                if h.field in ("episodes", "seasons", "volumes"):
                    lines.append("Transform: to_int_array(),")
                else:
                    lines.append("Transform: to_int_string(),")
            case "first_integer":
                if h.field in ("episodes", "seasons", "volumes"):
                    lines.append("Transform: to_first_int_array(),")
                else:
                    lines.append("Transform: to_first_int_string(),")
            case "range_func":
                lines.append("Transform: to_int_range(),")
            case "year_range":
                lines.append("Transform: to_year(),")
            case "transform_resolution":
                lines.append("Transform: to_transformed_resolution(),")
            case _:
                return None
        return lines

    if isinstance(t, ast.Call):
        fn = ast.unparse(t.func)
        if fn == "value" and len(t.args) == 1 and isinstance(t.args[0], ast.Constant):
            v = t.args[0].value
            if isinstance(v, str):
                if "$1" in v:
                    lines.append(f"Transform: to_value_sub({go_quote_str(v)}),")
                else:
                    lines.append(f"Transform: to_value({go_quote_str(v)}),")
                return lines
            if isinstance(v, int):
                lines.append(f"Transform: to_int_value({v}),")
                return lines
            return None
        if fn == "array":
            if not t.args:
                lines.append("Transform: to_string_array(),")
                return lines
            inner = ast.unparse(t.args[0])
            if inner == "integer":
                lines.append("Transform: to_int_array(),")
                return lines
            return None
        if fn == "uniq_concat":
            uniq_fields.add(FIELD_MAP.get(h.field, h.field))
            if not t.args:
                return None
            inner = t.args[0]
            inner_src = ast.unparse(inner)
            if (
                isinstance(inner, ast.Call)
                and ast.unparse(inner.func) == "value"
                and isinstance(inner.args[0], ast.Constant)
            ):
                v = inner.args[0].value
                if isinstance(v, str) and "$1" not in v:
                    lines.append(f"Transform: to_value_set({go_quote_str(v)}),")
                    return lines
                return None
            if inner_src == "lowercase":
                lines.append(
                    "Transform: to_value_set_with_transform(func(v string) any { return strings.ToLower(v) }),"
                )
                return lines
            if inner_src == "uppercase":
                lines.append(
                    "Transform: to_value_set_with_transform(func(v string) any { return strings.ToUpper(v) }),"
                )
                return lines
            if inner_src == "none":
                lines.append(
                    "Transform: to_value_set_with_transform(func(v string) any { return v }),"
                )
                return lines
            return None
        if fn == "date":
            return None  # date handlers come from OVERRIDES (Go time layouts)
        return None

    if isinstance(t, ast.Dict):
        # PTT quirk: two calls pass the options dict in the transformer slot
        h.options.update(_opts(t))
        return lines

    h.notes.append(f"transformer:{src}")
    return None


def options_lines(options: dict) -> list[str]:
    lines = []
    if options.get("remove"):
        lines.append("Remove: true,")
    if options.get("skipIfAlreadyFound") is False:
        lines.append("KeepMatching: true,")
    if options.get("skipFromTitle"):
        lines.append("SkipFromTitle: true,")
    if options.get("skipIfFirst"):
        lines.append("SkipIfFirst: true,")
    return lines


# ---------------------------------------------------------------------------
# Overrides & customs (hand-maintained Go entries, keyed by python source)
# ---------------------------------------------------------------------------

sys.path.insert(0, str(ROOT / "tools"))
from gen_overrides import CUSTOM, OVERRIDES  # noqa: E402


def emit(handlers: list[Handler], uniq_fields: set) -> str:
    out = []
    out.append(
        "// Code generated by tools/gen_handlers.py from PTT %s. DO NOT EDIT.\n"
        % VERSION
    )
    out.append("// Source of truth: tools/upstream/ptt/{handlers.py,anime.py}\n")
    out.append("\npackage parser\n")
    out.append('\nimport (\n\t"regexp"\n\t"strings"\n)\n')
    out.append("\nvar value_set_field_map = map[string]struct{}{\n")
    for f in sorted(uniq_fields):
        out.append(f'\t"{f}": {{}},\n')
    out.append("}\n")
    out.append("\nvar handlers = []handler{\n")
    unresolved = []
    for h in handlers:
        key = (h.field, h.pattern if h.pattern is not None else ";".join(h.notes))
        comment = f"\t// {h.source}.py:{h.line} {h.field}: {h.pattern if h.pattern else h.notes}"

        custom_note = next((n for n in h.notes if n.startswith("custom:")), None)
        if custom_note:
            name = custom_note.split(":", 1)[1]
            if name in CUSTOM:
                out.append(comment[:200] + "\n")
                out.append(CUSTOM[name].strip("\n") + "\n")
            else:
                unresolved.append(
                    (h, f"custom function {name} missing in gen_overrides.CUSTOM")
                )
            continue

        if key in OVERRIDES:
            out.append(comment[:200] + "\n")
            out.append(OVERRIDES[key].strip("\n") + "\n")
            continue

        if h.pattern is None or not isinstance(h.pattern, str):
            unresolved.append((h, "no literal pattern"))
            continue

        translated = translate_pattern(h.pattern, h.ignorecase)
        tlines = translate_transformer(h, uniq_fields)
        if translated is None or tlines is None:
            why = []
            if translated is None:
                why.append("mid-pattern lookaround")
            if tlines is None:
                why.append(
                    f"transformer {ast.unparse(h.transformer) if h.transformer else '?'}"
                )
            unresolved.append((h, ", ".join(why)))
            continue

        pat_expr, validators = translated
        gofield = FIELD_MAP.get(h.field, h.field)
        entry = ["\t{\n", f'\t\tField:   "{gofield}",\n', f"\t\tPattern: {pat_expr},\n"]
        if validators:
            if len(validators) == 1:
                entry.append(f"\t\tValidateMatch: {validators[0]},\n")
            else:
                entry.append(
                    f"\t\tValidateMatch: validate_and({', '.join(validators)}),\n"
                )
        for line in tlines:
            entry.append(f"\t\t{line}\n")
        for line in options_lines(h.options):
            entry.append(f"\t\t{line}\n")
        entry.append("\t},\n")
        out.append(comment[:200] + "\n")
        out.extend(entry)
    out.append("}\n")

    if unresolved:
        sys.stderr.write(f"\nUNRESOLVED: {len(unresolved)} handlers need overrides:\n")
        for h, why in unresolved:
            pat = (h.pattern or "")[:90]
            sys.stderr.write(
                f"  {h.source}.py:{h.line} {h.field}: {why}\n    pattern: {pat}\n"
            )
    return "".join(out), len(unresolved)


def main():
    uniq_fields: set = set()
    handlers = extract_calls(UPSTREAM / "handlers.py", "handlers")
    # anime handlers are appended after the main table, matching PTT's
    # add_defaults() which registers them at the end via anime_handler().
    all_handlers = handlers

    # first pass to collect uniq fields (transform translation mutates the set)
    for h in all_handlers:
        if h.transformer is not None and isinstance(h.transformer, ast.Call):
            if ast.unparse(h.transformer.func) == "uniq_concat":
                uniq_fields.add(FIELD_MAP.get(h.field, h.field))

    src, unresolved = emit(all_handlers, uniq_fields)
    OUT.write_text(src)
    print(f"wrote {OUT} ({len(all_handlers)} handlers, {unresolved} unresolved)")
    return 1 if unresolved else 0


if __name__ == "__main__":
    sys.exit(main())
