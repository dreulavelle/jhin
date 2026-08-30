// Package rules scores, rejects and caps releases by conditions written as
// configuration rather than as Go.
//
// jhin scores what a release name says. Everything else an application knows
// — file size, age, grab count, what a probe measured in the file — reaches
// the score through here: the application declares its own attributes, and a
// profile's rules read them alongside jhin's own.
//
// # A rule
//
// A rule is a name, a condition, and what happens when it holds:
//
//	{Name: "Oversized", When: `sizeGB > 12 and resolution != "2160p"`, Action: rules.ActionReject}
//
// Four actions are built in. score adds points, reject removes the release,
// limit caps how many matching releases survive, and define does nothing at
// all — it exists to be named by matched(). An application registers more.
//
// Points are an expression, not a constant, so an attribute can be scored by
// a value rather than at a flat rate:
//
//	Seeders:    score min(grabs, 200) * 15              if grabs > 0
//	Freshness:  score max(0, 500 - ageDays * 5)         if true
//	Sweet spot: score 2000 - abs(sizePerEpisodeGB - 4) * 300 if sizePerEpisodeGB > 0
//
// # The registry
//
// A Registry declares what a rule may name. It is the whole extension
// surface: to support something new, register it — never change the language.
//
//	reg := rules.Core()                      // everything a release name says
//	reg.Tier("measured", "a probed file")
//	reg.Namespace("probed", "measured").Num("height").Bool("dolbyVision")
//	reg.Field("sizeGB", rules.Num, "reported")
//	reg.Func("imdbRating", nil, rules.Num, fetchRating)
//	reg.Effect("tag", rules.Str)
//
//	eng, err := rules.Compile(reg, profile.Rules, library...)
//
// Four seams cover what differs between applications: namespaces and fields
// for new fact sources, functions for anything the grammar deliberately
// cannot say, effects for actions jhin evaluates but does not interpret, and
// scopes, which are opaque strings jhin never reads.
//
// # Confidence tiers
//
// Facts differ in how far they can be trusted and in whether they are there
// at all. A field belongs to a tier, a compiled rule carries the tiers its
// condition reads, and a rule reading a tier the release carries nothing in
// is skipped and reported rather than judged against zero values.
//
// Without that, one rule — probed.height < 1080 → reject — empties every
// result list of everything except the releases something has opened, because
// a release nobody probed has a probed height of zero. The practical
// consequence is worth knowing: a probe rule can only ever reward, or remove
// releases that were probed. It cannot demote everything else by omission.
//
// # Asking about the result set
//
// count, exists, any and none take a condition and judge the whole set rather
// than one release, which is what turns an unconditional rejection into a
// conditional one:
//
//	Bad upscale: reject if upscaled and exists(resolution == "2160p" and "remux" in traits)
//
// They are lifted out at compile time, deduplicated, and computed once over
// the set before any rule fires — so a rule that rejects can never change
// what another rule counted, and rule order does not matter. Compute them
// with ComputeAggregates and pass the state to Evaluate.
//
// # Evaluating
//
//	aggs := eng.ComputeAggregates(everyReleasesFacts)
//	out := eng.Evaluate(facts, "movie", aggs)
//	// out.Points, out.Matched, out.Rejections, out.Skipped, out.Limits, out.Effects
//
// Limits cannot be resolved here: which releases are best is only known after
// every point is in and the list is sorted. Collect Outcome.Limits and call
// ApplyLimits once the set is in final order.
//
// A compiled Engine is immutable and safe for concurrent use.
package rules
