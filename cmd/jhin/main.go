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
		Name:  "jhin",
		Usage: "torrent release name parser, ranker, and filter",
		Commands: []*cli.Command{
			rankCommand,
			{
				Name: "parse",
				Flags: []cli.Flag{
					&cli.BoolWithInverseFlag{
						Name:  "normalize",
						Value: true,
					},
					&cli.BoolFlag{
						Name: "pretty",
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
					var blob []byte
					var err error
					if cmd.Bool("pretty") {
						blob, err = json.MarshalIndent(&r, "", "  ")
					} else {
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
