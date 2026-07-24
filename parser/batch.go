package parser

// Batch parsing and extraction helpers.

import (
	"runtime"
	"sync"
)

// ParseAll parses a batch in parallel. The result is index-aligned with the
// input: out[i] is the parse of titles[i].
func ParseAll(titles []string) []*Result {
	out := make([]*Result, len(titles))
	workers := min(runtime.GOMAXPROCS(0), max(1, len(titles)))
	if workers == 1 || len(titles) < 8 {
		for i, t := range titles {
			out[i] = Parse(t)
		}
		return out
	}
	var wg sync.WaitGroup
	ch := make(chan int, workers*2)
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range ch {
				out[i] = Parse(titles[i])
			}
		}()
	}
	for i := range titles {
		ch <- i
	}
	close(ch)
	wg.Wait()
	return out
}

// ExtractSeasons parses a title and returns just its season numbers.
func ExtractSeasons(rawTitle string) []int {
	return Parse(rawTitle).Seasons
}

// ExtractEpisodes parses a title and returns just its episode numbers.
func ExtractEpisodes(rawTitle string) []int {
	return Parse(rawTitle).Episodes
}

// EpisodesFromSeason returns the episode numbers only when the given season
// number is present in the title.
func EpisodesFromSeason(rawTitle string, season int) []int {
	r := Parse(rawTitle)
	for _, s := range r.Seasons {
		if s == season {
			return r.Episodes
		}
	}
	return []int{}
}
