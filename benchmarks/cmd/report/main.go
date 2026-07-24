// Command report generates the accuracy scorecard as Markdown tables from
// the golden corpus. Run from the benchmarks module:
//
//	go run ./cmd/report                # summary tables
//	go run ./cmd/report -field source  # per-title disagreements for one field
//	go run ./cmd/report -lib go-ptn -field source
package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/dreulavelle/jhin/benchmarks"
)

func main() {
	corpusPath := flag.String("corpus", "../parser/testdata/golden.json", "path to golden corpus")
	field := flag.String("field", "", "dump per-title disagreements for this field")
	libFilter := flag.String("lib", "", "restrict -field dump to libraries whose name contains this")
	flag.Parse()

	corpus, err := benchmarks.LoadCorpus(*corpusPath)
	if err != nil {
		log.Fatal(err)
	}

	if *field != "" {
		dumpDisagreements(corpus, *field, *libFilter)
		return
	}

	scores := make([]*benchmarks.LibraryScore, len(benchmarks.Libraries))
	for i := range benchmarks.Libraries {
		scores[i] = benchmarks.Score(&benchmarks.Libraries[i], corpus)
	}

	fmt.Printf("Corpus: %d titles (parser/testdata/golden.json)\n\n", len(corpus))

	// headline table: per-field accuracy
	fmt.Print("| Field |")
	for _, s := range scores {
		fmt.Printf(" %s |", short(s.Library.Name))
	}
	fmt.Print("\n|---|")
	for range scores {
		fmt.Print("---|")
	}
	fmt.Println()
	for _, f := range benchmarks.ScoredFields {
		fmt.Printf("| %s (%d) |", f, scores[0].Fields[f].Present())
		for _, s := range scores {
			fmt.Printf(" %.1f%% |", s.Fields[f].Accuracy()*100)
		}
		fmt.Println()
	}
	fmt.Print("| **overall** |")
	for _, s := range scores {
		fmt.Printf(" **%.1f%%** |", s.Overall()*100)
	}
	fmt.Println()

	// spurious extractions + errors
	fmt.Print("\n| | ")
	for _, s := range scores {
		fmt.Printf("%s | ", short(s.Library.Name))
	}
	fmt.Print("\n|---|")
	for range scores {
		fmt.Print("---|")
	}
	fmt.Println()
	fmt.Print("| spurious extractions |")
	for _, s := range scores {
		total := 0
		for _, f := range benchmarks.ScoredFields {
			total += s.Fields[f].Spurious
		}
		fmt.Printf(" %d |", total)
	}
	fmt.Print("\n| parse errors/panics |")
	for _, s := range scores {
		fmt.Printf(" %d |", s.Errors)
	}
	fmt.Println()
}

func dumpDisagreements(corpus []benchmarks.GoldEntry, field, libFilter string) {
	for i := range benchmarks.Libraries {
		lib := &benchmarks.Libraries[i]
		if libFilter != "" && !strings.Contains(strings.ToLower(lib.Name), strings.ToLower(libFilter)) {
			continue
		}
		ds := benchmarks.Disagreements(lib, corpus, field)
		fmt.Printf("\n## %s — %s: %d disagreements\n\n", lib.Name, field, len(ds))
		for _, d := range ds {
			if d.GoldS != nil || d.GotS != nil {
				fmt.Printf("- %q: gold %v, got %v\n", d.Title, d.GoldS, d.GotS)
			} else {
				fmt.Printf("- %q: gold %q, got %q\n", d.Title, d.Gold, d.Got)
			}
		}
	}
}

func short(name string) string {
	if i := strings.IndexByte(name, '/'); i >= 0 {
		return name[i+1:]
	}
	return name
}
