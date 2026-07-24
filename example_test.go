package jhin_test

import (
	"fmt"

	"github.com/dreulavelle/jhin"
	"github.com/dreulavelle/jhin/rank"
)

func ExampleParse() {
	r := jhin.Parse("The.Walking.Dead.S05E03.720p.HDTV.x264-ASAP[ettv]")
	fmt.Println(r.Title, r.Seasons, r.Episodes, r.Resolution, r.Quality, r.Group)
	// Output: The Walking Dead [5] [3] 720p HDTV ASAP
}

func ExampleGetPartialParser() {
	parse := jhin.GetPartialParser([]string{"resolution", "year"})
	r := parse("The.Matrix.1999.1080p.BluRay.x264")
	fmt.Println(r.Resolution, r.Year)
	// Output: 1080p 1999
}

func Example_ranking() {
	ranker, _ := rank.New(rank.Default())

	torrents := ranker.RankAll([]string{
		"Movie.2020.1080p.BluRay.REMUX.AVC.TrueHD.7.1-GRP",
		"Movie.2020.1080p.WEB-DL.DDP5.1.H.264-GRP",
		"Movie.2020.HDCAM.x264-TRASH",
	})

	best := rank.Sort(torrents, rank.SortOptions{FetchableOnly: true})
	for _, t := range best {
		fmt.Println(t.Raw)
	}
	// Output:
	// Movie.2020.1080p.BluRay.REMUX.AVC.TrueHD.7.1-GRP
	// Movie.2020.1080p.WEB-DL.DDP5.1.H.264-GRP
}
