package main

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/dreulavelle/jhin/rules"
	"github.com/urfave/cli/v3"
)

// rulesCommand checks and inspects a rule file without running a search, so a
// broken condition is found where it was written rather than where it fails.
var rulesCommand = &cli.Command{
	Name:  "rules",
	Usage: "check and inspect rule files",
	Commands: []*cli.Command{
		{
			Name:      "check",
			Usage:     "compile a rule file and report what it does",
			ArgsUsage: "<file>",
			Action: func(ctx context.Context, cmd *cli.Command) error {
				path := cmd.Args().First()
				if path == "" {
					return fmt.Errorf("no rule file given")
				}
				blob, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				parsed, err := rules.ParseText(string(blob))
				if err != nil {
					return err
				}
				eng, err := rules.Compile(rules.Core(), parsed)
				if err != nil {
					return err
				}

				fmt.Printf("%d rules, %d act\n", len(parsed), eng.Len())
				for _, r := range parsed {
					state := ""
					if !r.IsEnabled() {
						state = "  (off)"
					}
					fmt.Printf("  %-10s %s%s\n", r.EffectiveAction(), r.Name, state)
				}
				if q := eng.AggregateSources(); len(q) > 0 {
					fmt.Printf("\nasks about the result set:\n")
					for _, src := range q {
						fmt.Printf("  %s\n", src)
					}
				}
				return nil
			},
		},
		{
			Name:  "fields",
			Usage: "list the attributes a rule can name",
			Action: func(ctx context.Context, cmd *cli.Command) error {
				fields := rules.Core().Fields()
				sort.Strings(fields)
				for _, f := range fields {
					fmt.Println(f)
				}
				fmt.Fprintf(os.Stderr, "\n%d attributes. An application adds its own; see the rules package.\n", len(fields))
				return nil
			},
		},
		{
			Name:      "fmt",
			Usage:     "rewrite a rule file in canonical form",
			ArgsUsage: "<file>",
			Action: func(ctx context.Context, cmd *cli.Command) error {
				path := cmd.Args().First()
				if path == "" {
					return fmt.Errorf("no rule file given")
				}
				blob, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				parsed, err := rules.ParseText(string(blob))
				if err != nil {
					return err
				}
				// Comments and blank lines are not part of a rule, so they do
				// not survive; print rather than overwrite so nothing is lost
				// without the author seeing it first.
				fmt.Print(strings.TrimLeft(rules.FormatText(parsed), "\n"))
				return nil
			},
		},
	},
}
