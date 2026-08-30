# Rules

A rule is a named condition attached to a profile, plus what happens when it
holds. Rules are the general form of the weighted pattern lists: a regex sees
one string, while a rule sees everything known about a release — including
attributes the application registers itself.

They exist because some things cannot be said with one pattern:

- *Dolby Vision, but only when there is no HDR fallback.*
- *Over 30 GB, unless it is 4K.*
- *Measured 10-bit, not just claimed 10-bit in the name.*
- *The shaky 4K encode, but only when a remux stands ready to replace it.*

```
DV without fallback: reject if dolbyVision and not hdrFallback
Oversized:           reject if sizeGB > 30 and resolution != "2160p"
Real 10-bit:         score 400 if probed.bitDepth >= 10
Bad upscale:         reject if upscaled and exists(resolution == "2160p" and "remux" in traits)
```

## Anatomy

| Field | What it does |
|---|---|
| `name` | Identifies the rule in a score breakdown and to `matched()`. |
| `when` | The condition. Must answer yes or no. |
| `action` | `score` · `reject` · `limit` · `define`, or one the application registered. Empty means `score`. |
| `score` | What a matching release earns — **an expression**, so points can be computed from the release. Negative allowed. |
| `count` | For `limit`: how many matching releases survive. |
| `group_by` | For `limit`: what the cap is counted per. |
| `scope` | Content kinds this applies to. Empty applies to all. The vocabulary is the application's. |
| `enabled` | Turn a rule off without deleting it. A disabled rule is never compiled, so a half-written one cannot block a save. |

A condition that will not compile fails the whole set and names the rule.

## Writing them as text

```
Atmos: score -800 if "atmos" in traits
DV without HDR fallback: reject if dolbyVision and not hdrFallback
At most 3 in 4K [movie]: keep 3 if resolution == "2160p"
Best 3 of each flavour: keep 3 per resolution + " " + quality if true
UHD BluRay T1: define if group in ["FraMeSToR", "W4NK3R"]
Old experiment [off]: score 100 if "remux" in traits
```

A line is `Name: action if condition`. The action is `score <expression>`,
`reject`, `keep <n>`, `keep <n> per <grouping>`, `define`, or a registered
effect and its value. Brackets before the colon carry the scope and `off`, in
either order and both optional.

This grammar wraps conditions; it never parses them. Everything after `if`
goes to the expression language exactly as written, so a condition containing
a colon, a bracket, or the word "if" inside a string survives.

`jhin rules check <file>` compiles a file and reports what it does;
`jhin rules fmt <file>` rewrites it in canonical form.

## Score is an expression

This is what lets an attribute be scored *by* something rather than at a flat
rate. A bare number is still a valid expression, so `score -800` means what it
always did.

```
Seeders:    score min(grabs, 200) * 15                    if grabs > 0
Freshness:  score max(0, 500 - ageDays * 5)               if true
Sweet spot: score 2000 - abs(sizePerEpisodeGB - 4) * 300  if sizePerEpisodeGB > 0
Tiered:     score resolution == "2160p" ? 3000 : 500      if true
```

## Limits

"At most three of these" is about the **final score order** — which three are
best is only known after every point is in and the list is sorted, later than
any condition runs. So `limit` is an action, not a function in a condition.

The survivors are the best matching releases by final score, not the first the
indexer happened to return. A release something else already rejected is gone,
so it does not hold a slot.

A cap counted **per group** turns a run of near-identical rules into one:

```
Best 3 of each flavour: keep 3 per resolution + " " + quality if true
```

Two things to know about a grouping: it is judged with the condition, so
grouping by `probed.height` makes the whole rule measured-only; and every
distinct value is its own group, including the empty one.

## What a rule can read, and how much to trust it

The attribute namespace is organised by how far a value can be trusted rather
than by which subsystem produced it. **jhin itself supplies only the first
tier** — everything a release name says. The rest is registered by the
application, and these names are conventions rather than guarantees.

| Tier | Where it comes from |
|---|---|
| inferred | Read out of the release name. Always present, never stale, wrong often enough that a careful rule says so. |
| reported | Claimed by the indexer — size, age, grabs. |
| community | An availability database. Per source, possibly months old. |
| measured | What a probe found in the file. Ground truth, but only for releases something has opened. |

### From the release name

```
adult audio bitDepth bitrate channels codec commentary complete container
convert country date documentary dolbyVision dubbed edition episodeCode
episodes extension extras group hardcoded hdr hdrFallback languages network
ppv proper quality region releaseName remastered repack resolution retail
scene seasonPack seasons site size subbed threeD title torrent traits trash
uncensored unrated upscaled volumes year
```

`hdr` lists the dynamic-range tags in the parser's vocabulary — `"DV"` for
Dolby Vision — so `"HDR10" in hdr` is rarely what you want. Prefer
`dolbyVision` and `hdrFallback`, which mean the same thing whichever way the
tags were written. `hdrFallback` is true when a device that cannot decode
Dolby Vision still gets an HDR picture; it is false for SDR and for DV-only
releases alike, which is the distinction "block DV without a fallback" needs.

`traits` is every attribute the ranker scores, by the same keys: `"remux"`,
`"webdl"`, `"cam"`, `"hevc"`, `"10bit"`, `"atmos"` and so on. It is what lets
a rule reach anything the baseline has an opinion about:

```
"cam" in traits                                → reject
"remux" in traits                              → +4000
"webrip" in traits and resolution == "2160p"   → reject
```

## Fail-open

**A rule that reads a tier the release carries nothing in does not run.** It
is skipped and reported, not failed.

Without this, one rule — `probed.height < 1080 → reject` — would empty every
result list of everything except the releases something has opened, because a
release nobody probed has a probed height of zero.

The practical consequence is worth knowing: a probe rule can only ever
*reward*, or remove releases that were probed. It cannot demote everything
else by omission.

Tiers travel. Grouping a cap by `probed.height` makes the whole rule
measured-only, and referring to a probe rule through `matched()` makes the
referring rule probe-dependent.

## Asking about the result set

`count(condition)` · `exists(condition)` · `none(condition)`, with `any` as
another name for `exists`.

Everything else describes the release being judged. These ask about the whole
set, which is what turns an unconditional rejection into a conditional one:

```
upscaled and exists(resolution == "2160p" and "remux" in traits)   → reject
count(resolution == "2160p") < 3                                   → +500
none(quality == "WEB-DL")                                          → +200
```

The set is fixed before any rule fires and includes the release being judged,
so a 4K remux always has `count(resolution == "2160p") >= 1`. Because the
counts are taken first, a rule that rejects can never change what another rule
counted — **so the order of your rules does not matter**. They cannot nest.

Fail-open extends to the set: a release missing a tier the inner condition
reads is not counted, and when *no* release carries that tier the question is
unanswerable, so the rule is skipped rather than fed a zero.

## Referring to another rule

`matched("Rule name")` holds when the named rule's *condition* holds for this
release. The case it exists for is a tier list:

```
UHD BluRay T1: define if group matches "(?i)^(FraMeSToR|W4NK3R)$"
UHD BluRay T2: define if group matches "(?i)^(HiFi|Positive)$"

Trusted 4K: score 3000 if resolution == "2160p" and matched("UHD BluRay T1")
Untrusted UHD encode: reject if
    resolution == "2160p" and "bluray" in traits
    and not (matched("UHD BluRay T1") or matched("UHD BluRay T2"))
    and exists(resolution == "2160p" and "remux" in traits)
```

Change a tier list afterwards and the rejection follows it. A reference is
resolved when the set is compiled, by copying the other rule's condition into
this one, so:

- **It works wherever a condition does** — on its own, inside a result-set
  question, inside a limit's grouping. Nothing is evaluated twice.
- **Order does not matter, and neither does the other rule's action.** A rule
  may name one written below it.
- **The referenced rule's scope and tier come with it.**
- **A rule that is switched off classifies nothing**, so a reference to it is
  never true — and its condition is never looked at, which keeps a broken rule
  you have turned off from blocking a save.

A `define` rule is a named condition and nothing more: it never pays out,
never rejects, and never appears in a breakdown, so it cannot leak points the
way a zero-point score rule still leaks its name.

A reference to a name no rule has, to a name two rules share, or one that
closes a circle is refused when the set is compiled.

## Syntax

| | |
|---|---|
| Comparison | `==` `!=` `<` `<=` `>` `>=` |
| Logic | `and` `or` `not` |
| Membership | `"DV" in hdr`, `group not in [...]` |
| Text | `releaseName matches "(?i)regex"`, `contains`, `startsWith`, `endsWith` |
| Arithmetic | `+` `-` `*` `/` `%` |
| Grouping | `( … )` |
| Conditional | `cond ? a : b` |
| Lists | `["a", "b"]` |

Comparisons do not chain: `a < b < c` is a compile error rather than the
silent nonsense it is elsewhere.

Builtins: `len` `lower` `upper` `trim` `abs` `floor` `ceil` `round` `min`
`max` `string` `num`, plus the collection forms `count(list, # == "x")`,
`any`, `all`, `none`, where `#` stands for the element under test.

`matches` takes a Go (RE2) regular expression, which must be written out so it
can be compiled once. RE2 has no lookahead or lookbehind, and rules are why it
does not need one: `\bDV\b(?!.*HDR10)` becomes
`dolbyVision and not hdrFallback`.

Write the regex as-is. Backslashes in a condition string are taken literally,
so `\+`, `\d` and `\b` mean what they mean in a regex — no doubling needed,
though a defensively written `\\+` means the same thing.

## Extending it

The language is closed; the registry is open. Anything an application needs
beyond the above is registered in Go, and is then type-checked like everything
else:

```go
reg := rules.Core()
reg.Tier("measured", "a probed file")
reg.Namespace("probed", "measured").Num("height").Num("bitDepth")
reg.Field("sizeGB", rules.Num, "reported")
reg.Func("imdbRating", nil, rules.Num, fetchRating)
reg.Effect("tag", rules.Str)
```

- **Namespaces and fields** for a new fact source — an indexer, a probe, a
  metadata provider.
- **Functions** for anything the grammar deliberately cannot say. If Go can
  compute it, a rule can use it.
- **Effects** for actions jhin evaluates but does not interpret: the outcome
  reports `Effect{Name, Value}` and the application decides what it means.
- **Scopes** are opaque strings jhin never reads.

See the [`rules`](../rules/) package documentation for the API.
