# Rules Guide

Rules let a profile say things no single pattern can: score an attribute *by*
a value instead of at a flat rate, reject one release because of what else the
search returned, cap how many near-duplicates survive, and fold in facts jhin
itself cannot know — file size, age, grab count, what a probe measured.

A rule is one line of text:

```
DV without fallback: reject if dolbyVision and not hdrFallback
```

That is a name, an action (`reject`), and a condition. Everything in this
guide is a variation on that line.

This page is the tutorial. The reference lives in the repo at
[`rules.md`](./rules.md).

---

## 1. Try it in thirty seconds

Save a file:

```
# my.rules — a line starting with # is a comment
No CAMs:            reject if "cam" in traits
Remux worship:      score 4000 if "remux" in traits
DV needs fallback:  reject if dolbyVision and not hdrFallback
Best 2 per flavour: keep 2 per resolution + " " + quality if true
```

Check it compiles, then run releases through it:

```console
$ jhin rules check my.rules
4 rules, 4 act
  reject     No CAMs
  score      Remux worship
  reject     DV needs fallback
  keep       Best 2 per flavour

$ jhin rank --rules my.rules "Dune.2021.2160p.UHD.BluRay.REMUX.DV.HDR10.HEVC-FraMeSToR" \
                             "Dune.2021.CAM.XviD-BADGRP"
```

A condition that cannot compile fails the whole file and names the rule, the
line, and the column — a broken rule is found where it was written, never in
the middle of a search.

---

## 2. The shape of a rule

```
Name [scope] [off]: action if condition
```

| Part | Meaning |
|---|---|
| `Name` | Identifies the rule in score breakdowns and to `matched()`. Lives on one line. |
| `[scope]` | Optional. Content kinds this rule applies to — `[movie]`, `[movie, series]`. The vocabulary is your application's; jhin never reads it. |
| `[off]` | Optional. Disables the rule without deleting it. A disabled rule is never compiled, so a half-written one cannot block a save. `off` and `disabled` are reserved — they cannot be scope names. |
| `action` | What happens when the condition holds. |
| `condition` | Anything from section 4. Must answer yes or no. |

The actions:

```
Atmos bonus:   score 800                        if "atmos" in traits
No screeners:  reject                           if "screener" in traits
Best three 4K: keep 3                           if resolution == "2160p"
Per flavour:   keep 2 per resolution            if true
UHD T1:        define                           if group in ["FraMeSToR", "W4NK3R"]
```

- **`score <expression>`** adds points. The amount is an expression, not a
  constant (section 5). Negative is fine. A bare `score` means zero — useful
  when the rule exists only to show up in a breakdown.
- **`reject`** removes the release, recording `rule:<Name>` in its rejections.
- **`keep <n>`** / **`keep <n> per <grouping>`** caps how many matching
  releases survive (section 8).
- **`define`** does nothing at all. The rule exists to be named by
  `matched()` (section 7).
- Anything else is an **application-defined effect** (section 10) — the rule
  reports a value and your application decides what it means.

A condition too long for one line continues on indented lines. Rule names
start at the left margin, so nothing else marks a continuation, and a blank
line ends a rule:

```
Untrusted UHD encode: reject if
    resolution == "2160p" and "bluray" in traits
    and not (matched("UHD T1") or matched("UHD T2"))
    and exists(resolution == "2160p" and "remux" in traits)
```

One rule per line is the canonical form, so a string literal cannot contain a
raw line break — write `\n` if you ever genuinely need one.

---

## 3. What you can name

### From the release name (always available)

Everything jhin's parser reads out of the title. These are always answerable —
a release always has a name.

```
title  year  releaseName

resolution  quality  codec  bitDepth  hdr  dolbyVision  hdrFallback

audio  channels  languages

seasons  episodes  volumes  episodeCode  seasonPack  complete

group  edition  container  extension  network  region  site  date
size  bitrate  country  extras

proper  repack  remastered  upscaled  threeD  dubbed  subbed  hardcoded
documentary  adult  trash  scene  retail  uncensored  unrated  convert
commentary  ppv  torrent

traits
```

Types: `title`, `resolution`, `group` and the like are text; `year` and
`bitDepth` are numbers; `hdr`, `audio`, `channels`, `languages`, `extras` are
lists of text; `seasons`, `episodes`, `volumes` are lists of numbers; the rest
are yes/no flags.

Two derived flags are worth knowing because no regex can say them:
**`dolbyVision`** is true when the name carries a DV tag however it was
spelled, and **`hdrFallback`** is true when a device that cannot decode Dolby
Vision still gets an HDR picture. `dolbyVision and not hdrFallback` is the
whole "block DV-only releases" rule.

### `traits` — everything the ranker has an opinion about

`traits` is the list of attributes jhin's own scoring detected, under the same
keys the policy map uses — one detection shared by baseline scoring and rules.
All 66:

```
Sources:   bdrip bluray brrip cam dvd dvdrip hdrip hdtv pdtv ppvrip r5 remux
           satrip screener telecine telesync tvrip uhdrip vhs vhsrip web
           webdl webdlrip webmux webrip
Codecs:    av1 avc hevc mpeg xvid
Range:     dolby_vision hdr hdr10plus sdr 10bit
Audio:     aac atmos clean_audio dolby_digital dolby_digital_plus
           dts_lossless dts_lossy flac mp3 opus pcm truehd
Channels:  mono stereo surround
Extras:    3d converted documentary dubbed edition hardcoded network ppv
           proper repack retail scene site size subbed uncensored upscaled
```

```
"cam" in traits                                 → the release is a CAM
"remux" in traits and "atmos" in traits         → an Atmos remux
```

The trait vocabulary is **closed and checked**: `"dual_audio" in traits`
fails to compile with `traits never holds "dual_audio" (did you mean
"clean_audio"?)` instead of silently never firing. (Dual audio surfaces as
the `dubbed` flag and trait.)

### What your application adds

Anything else — indexer data, probe results, availability — is registered by
the application in Go (section 10) and then reads exactly like the built-ins:

```
sizeGB > 30 and resolution != "2160p"
probed.bitDepth >= 10
grabs > 100
```

---

## 4. What you can say

| | |
|---|---|
| Comparison | `==` `!=` `<` `<=` `>` `>=` |
| Logic | `and` `or` `not` (`&&` and `||` are accepted spellings) |
| Membership | `"DV" in hdr` · `group not in ["EVO", "YIFY"]` · `"720" in releaseName` (substring when the right side is text) |
| Text | `releaseName matches "(?i)hybrid"` · `contains` · `startsWith` · `endsWith` |
| Arithmetic | `+` `-` `*` `/` `%` (also `+` joins text) |
| Choice | `condition ? a : b` |
| Grouping | `( … )` · lists are `["a", "b"]` |

Builtins: `len` `lower` `upper` `trim` `abs` `floor` `ceil` `round` `min`
`max` `string` `num`, plus the collection forms `count(hdr, # == "DV")`,
`any`, `all`, `none`, where `#` is the element under test.

Everything is **type-checked when the profile is saved**. A typo'd attribute
(`resolutoin`), a string compared to a number, a `count` over something that
is not a list — each fails compilation with the line and column, usually with
a "did you mean" suggestion. Comparisons deliberately do not chain:
`a < b < c` is a compile error, not silent nonsense.

`matches` takes a Go (RE2) regex, written out literally so it compiles once.
Backslashes in a condition string are taken as-is — paste `\bDV\b` or
`\d{4}` straight from a regex tool, no doubling. RE2 has no lookahead, and
rules are why you will not miss it: `\bDV\b(?!.*HDR10)` is just
`dolbyVision and not hdrFallback`.

---

## 5. Score is an expression

The part no table of weights can do — points computed *from* the release:

```
Seeders:    score min(grabs, 200) * 15                      if grabs > 0
Freshness:  score max(0, 500 - ageDays * 5)                 if true
Sweet spot: score 2000 - abs(sizeGB - 4) * 300              if sizeGB > 0
Tiered:     score resolution == "2160p" ? 3000 : 500        if true
```

Rule points are part of the final rank, so a profile's `MinRank` floor is
judged after they land: a rule can sink a release under the floor or lift one
over it.

---

## 6. Asking about the result set

Everything so far describes one release. `count(…)`, `exists(…)` and
`none(…)` (with `any` as another spelling of `exists`) ask about the **whole
result set**, which turns an unconditional rejection into a conditional one:

```
Bad upscale:  reject if upscaled and exists(resolution == "2160p" and "remux" in traits)
Scarce 4K:    score 500 if count(resolution == "2160p") < 3
No WEB yet:   score 200 if none(quality == "WEB-DL")
```

The first rule reads: *reject the upscale, but only when a real 4K remux
actually turned up to replace it.* Alone in the results, the upscale
survives — it is the best you can get.

Three things make these safe to use:

- The set is **every release the profile's own filters let through**, counted
  once before any rule fires. A remux the profile rejected for its language
  or as trash is not "something better".
- Because the counts are taken first, a rule that rejects can never change
  what another rule counted — **the order of your rules never matters**.
- Two rules asking the same question share one computation.

They cannot nest, and `matched()` works inside them.

---

## 7. Write a tier list once, use it everywhere

`matched("Name")` is true when the named rule's *condition* holds for this
release. Pair it with `define` — a rule that is only a named condition — and
a release-group tier list becomes one place to edit:

```
UHD T1: define if group in ["FraMeSToR", "W4NK3R"]
UHD T2: define if group in ["HiFi", "Positive"]

Trusted 4K:  score 3000 if resolution == "2160p" and matched("UHD T1")
Known 4K:    score 1500 if resolution == "2160p" and matched("UHD T2")
Untrusted 4K encode: reject if
    resolution == "2160p" and "bluray" in traits
    and not (matched("UHD T1") or matched("UHD T2"))
    and exists(resolution == "2160p" and "remux" in traits)
```

Add a group to `UHD T1` and every rule that names it follows. References are
resolved at compile time by copying the condition in, so they work anywhere a
condition does, order never matters, and a reference to a rule you switched
`[off]` is simply never true. Unknown names, ambiguous names, and reference
cycles are compile errors.

---

## 8. Caps: `keep N per …`

"At most three of these" is about the **final score order** — which three are
best is only known after every point is in and the list is sorted. So a cap
records who it applies to during evaluation and is settled by
`rank.ApplyLimits` after `rank.Sort`.

```
At most 3 in 4K:      keep 3 if resolution == "2160p"
Best 2 per flavour:   keep 2 per resolution + " " + quality if true
```

The grouped form replaces a run of near-identical rules with one: every
distinct value of the grouping is its own bucket. Survivors are the best by
final score, not the first the indexer returned; a release something else
already rejected holds no slot; and caps count independently of one another,
so the order they are declared in never changes who survives.

---

## 9. Trust, and why rules fail open

Attributes belong to **confidence tiers**. jhin's own tier — the release
name — is always present. The conventional names for what applications add:

| Tier | Where it comes from |
|---|---|
| `inferred` | The release name. Always present, occasionally lying. |
| `reported` | Claimed by an indexer — size, age, grabs. |
| `community` | An availability database. Possibly stale. |
| `measured` | What a probe found in the actual file. Ground truth, but only for files something has opened. |

**A rule that reads a tier this release carries nothing in is skipped, not
run.** Judged against zero values, one innocent rule — `probed.height < 1080:
reject` — would empty every result list of everything that was never probed.
Instead it is skipped and reported (`Torrent.RuleSkipped` says *"needs a
probed file"*), so a probe rule can reward or remove probed releases but can
never demote everything else by omission.

The same holds for set questions: on a fresh search where nothing was probed,
`none(probed.height >= 2000)` is unanswerable, so the rule skips rather than
reading "there is no good 4K".

---

## 10. Wiring it into an application

Rules live in the profile (`rules` in the profile JSON, or built from a text
file with `rules.ParseText`). Compile once, attach to the ranker, and settle
caps after sorting:

```go
profile.Rules, _ = rules.ParseText(ruleText)

eng, err := profile.CompileRules(nil)   // nil = jhin's own vocabulary
if err != nil {
    // names the rule, the line and the column — show it to the user
}

r, _ := rank.New(profile, rank.WithRules(eng))
sorted := rank.Sort(r.RankEntries(entries, rank.RankOptions{Kind: "movie"}), rank.SortOptions{})
rank.ApplyLimits(sorted)
```

Each result carries its receipts: `RuleMatches` (name, points, the score
expression), `RuleSkipped` (name and why), `Rejections`, `Effects`.

### Registering your own facts

The grammar is closed; the registry is the whole extension surface:

```go
reg := rank.CoreRegistry()   // everything above, trait vocabulary included
reg.Tier("reported", "indexer-reported data")
reg.Tier("measured", "a probed file")
reg.Field("sizeGB", rules.Num, "reported")
reg.Field("ageDays", rules.Num, "reported")
reg.Field("grabs", rules.Num, "reported")
reg.Namespace("probed", "measured").Num("height").Num("bitDepth")

// anything the grammar cannot say, Go can
reg.Func("onWatchlist", nil, rules.Bool, func(f rules.Facts, _ []rules.Value) (rules.Value, error) {
    title, _ := f.Lookup("title")
    return rules.BoolOf(watchlist[strings.ToLower(title.Str())]), nil
})

// an action jhin evaluates but does not interpret
reg.Effect("route", rules.Str)

eng, err := profile.CompileRules(reg)
```

Your releases answer for their own facts by implementing two methods —
`Lookup(path)` and `TierPresent(tier)` — and passing them per release as
`rank.Entry.Facts`. `TierPresent` returning false is what makes fail-open
work: report `measured` only for releases you actually probed.

An effect rule — `Send 4K to the big box: route "uhd-box" if resolution ==
"2160p"` — shows up as `Effect{Name: "route", Value: "uhd-box"}` in the
outcome, and what that means is entirely yours.

`Registry.Values(path, …)` closes a field's vocabulary so value typos fail at
save time the way `traits` already does — declare it only where the set truly
is closed.

---

## 11. Ten real-world recipes

1. **"Block Dolby Vision releases my TV can't play, but keep DV+HDR10 hybrids."**

```
DV without fallback: reject if dolbyVision and not hdrFallback
```

2. **"Score by seeders and freshness instead of flat rates."** (needs
   `grabs`/`ageDays` registered from your indexer)

```
Seeders:   score min(grabs, 200) * 15 if grabs > 0
Freshness: score max(0, 500 - ageDays * 5) if true
```

3. **"Reject AI upscales — but only when a real 4K remux actually showed up."**

```
Bad upscale: reject if upscaled and exists(resolution == "2160p" and "remux" in traits)
```

4. **"Keep my release-group tiers in one place and reuse them."**

```
UHD T1: define if group in ["FraMeSToR", "W4NK3R"]
Trusted 4K: score 3000 if resolution == "2160p" and matched("UHD T1")
```

5. **"No more than two of each resolution-and-quality flavour."**

```
Best 2 per flavour: keep 2 per resolution + " " + quality if true
```

6. **"Trust the probe over the name — and catch releases that lied about 10-bit."**
   (needs `probed.*` registered; skips gracefully for unprobed releases)

```
Real 10-bit:  score 400 if probed.bitDepth >= 10
Caught lying: reject if bitDepth >= 10 and probed.bitDepth < 10
```

7. **"Anime dual-audio first."** (dual audio parses as `dubbed`)

```
Dual audio [anime]: score 1200 if dubbed
```

8. **"Oversized is relative: 30 GB is fine for 4K, absurd for 1080p."**

```
Oversized: reject if sizeGB > 30 and resolution != "2160p"
```

9. **"Nothing good yet? Take what exists; something good arrived? Raise the bar."**

```
Scarce 4K: score 500 if resolution == "2160p" and count(resolution == "2160p") < 3
```

10. **"Send 4K to a different download client."** (an effect your app interprets)

```
Route UHD: route "uhd-box" if resolution == "2160p"
```

---

## 12. The CLI

```console
$ jhin rules check my.rules    # compile; list what each rule does and what it asks the set
$ jhin rules fields            # every attribute a rule can name (jhin's own vocabulary)
$ jhin rules fmt my.rules      # rewrite in canonical form (prints; comments don't survive)
$ jhin rank --rules my.rules --json "<title>" …
```

---

## 13. Worth knowing

- **Rules never compiled are inert.** A profile carrying rules that were
  never attached scores exactly as it did before they were written.
- **Dual audio is `dubbed`** — there is no `dual_audio` trait, and the
  compiler will tell you so, by name.
- **Unparseable text is zero, not an error**: `num(bitrate) > 5` on a release
  with no bitrate reads false; it does not reject anything.
- **A runtime failure skips the rule.** Division by zero, an unanswerable set
  question, a probe function that errored — the rule is reported in
  `RuleSkipped`, and an inconclusive check never removes a release.
- **Profiles are untrusted input.** Parsing, compiling and evaluating are all
  bounded — a hostile rule file gets an error, never a hang or a crash.
- **`syntax` in the profile** records the rule-language version; a profile
  from a newer jhin is refused with *"upgrade jhin"* instead of
  half-understood.
- Everything a rule did — or declined to do — is on the result:
  `RuleMatches`, `RuleSkipped`, `Rejections`, `Effects`, and `Explain()`
  folds rule points into the score breakdown.
