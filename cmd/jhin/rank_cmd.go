package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/dreulavelle/jhin/rank"
	"github.com/dreulavelle/jhin/rules"
	"github.com/urfave/cli/v3"
)

// rankCommand evaluates release names from arguments or stdin (one per
// line) against a profile and prints them ranked.
var rankCommand = &cli.Command{
	Name:      "rank",
	Usage:     "rank and filter release names (from args or stdin, one per line)",
	ArgsUsage: "[titles...]",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "profile",
			Usage: "path to a profile JSON (default: built-in default profile)",
		},
		&cli.StringFlag{
			Name:  "rules",
			Usage: "path to a rules file in the text form (one rule per line)",
		},
		&cli.StringFlag{
			Name:  "target",
			Usage: "target title to match against (rejects mismatches)",
		},
		&cli.BoolFlag{
			Name:  "json",
			Usage: "output full JSON instead of a summary table",
		},
		&cli.BoolFlag{
			Name:  "all",
			Usage: "include releases that failed filtering",
		},
		&cli.IntFlag{
			Name:  "limit",
			Usage: "max results per resolution bucket (0 = unlimited)",
		},
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		profile := rank.Default()
		if path := cmd.String("profile"); path != "" {
			var err error
			if profile, err = rank.Load(path); err != nil {
				return fmt.Errorf("loading profile: %w", err)
			}
		}
		if path := cmd.String("rules"); path != "" {
			blob, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("loading rules: %w", err)
			}
			parsed, err := rules.ParseText(string(blob))
			if err != nil {
				return fmt.Errorf("loading rules: %w", err)
			}
			profile.Rules = parsed
		}
		eng, err := profile.CompileRules(nil)
		if err != nil {
			return err
		}
		ranker, err := rank.New(profile, rank.WithRules(eng))
		if err != nil {
			return err
		}

		titles := cmd.Args().Slice()
		if len(titles) == 0 {
			scanner := bufio.NewScanner(os.Stdin)
			scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)
			for scanner.Scan() {
				if line := scanner.Text(); line != "" {
					titles = append(titles, line)
				}
			}
			if err := scanner.Err(); err != nil {
				return err
			}
		}
		if len(titles) == 0 {
			return fmt.Errorf("no titles given (pass as arguments or on stdin)")
		}

		opts := rank.RankOptions{TargetTitle: cmd.String("target")}
		torrents := ranker.RankAll(titles, opts)
		sorted := rank.Sort(torrents, rank.SortOptions{
			FetchableOnly: !cmd.Bool("all"),
			BucketLimit:   cmd.Int("limit"),
		})
		// Caps are counted once the list is in final score order, which is the
		// only point at which "the best three of these" means anything.
		rank.ApplyLimits(sorted)

		if cmd.Bool("json") {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(sorted)
		}
		for _, t := range sorted {
			verdict := "OK"
			if !t.Fetch {
				verdict = "REJECT " + fmt.Sprint(t.Rejections)
			}
			fmt.Printf("%8d  %-7s %-8s %s\n", t.Rank, t.Resolution(), verdict, t.Raw)
		}
		return nil
	},
}
