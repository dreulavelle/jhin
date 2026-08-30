package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	jhin "github.com/dreulavelle/jhin"
	"github.com/urfave/cli/v3"
)

func main() {
	cmd := &cli.Command{
		Name:    "jhin",
		Usage:   "torrent release name parser, ranker, and filter",
		Version: jhin.Version().String(),
		Commands: []*cli.Command{
			{
				Name:  "version",
				Usage: "print the jhin version",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					fmt.Println(jhin.Version().String())
					return nil
				},
			},
			rankCommand,
			{
				Name:      "parse",
				Usage:     "parse a release name (only the fields it set; --long for all)",
				ArgsUsage: "<title>",
				Flags: []cli.Flag{
					&cli.BoolWithInverseFlag{
						Name:  "normalize",
						Value: true,
					},
					&cli.BoolFlag{
						Name: "pretty",
					},
					&cli.BoolFlag{
						Name:  "long",
						Usage: "print every field, including the ones parsing did not set",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					torrentTitle := cmd.Args().First()
					r := jhin.Parse(torrentTitle)
					if err := r.Error(); err != nil {
						return err
					}
					if cmd.Bool("normalize") {
						r = r.Normalize()
					}
					pretty := cmd.Bool("pretty")
					var blob []byte
					var err error
					switch {
					case !cmd.Bool("long"):
						blob, err = marshalSet(r, pretty)
					case pretty:
						blob, err = json.MarshalIndent(&r, "", "  ")
					default:
						blob, err = json.Marshal(&r)
					}
					if err != nil {
						return err
					}
					fmt.Printf("%v\n", string(blob))
					return nil
				},
			},
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
