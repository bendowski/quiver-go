package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/bendowski/quiver/goparse"
	lbstore "github.com/bendowski/quiver/store/ladybug"
)

func newLoadCmd(dbPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "load <dir>",
		Short: "Parse Go files in dir and store the graph in the database",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := args[0]
			return withStore(cmd, *dbPath, func(ctx context.Context, s *lbstore.Store) error {
				p := goparse.New(afero.NewOsFs())
				res, err := p.ParseDir(ctx, dir)
				if err != nil {
					return fmt.Errorf("parse: %w", err)
				}

				var nodeErrs, edgeErrs int
				for _, n := range res.Nodes {
					if err := s.AddNode(ctx, n); err != nil {
						fmt.Fprintf(os.Stderr, "AddNode %s: %v\n", n.ID, err)
						nodeErrs++
					}
				}
				for _, e := range res.Edges {
					if err := s.AddEdge(ctx, e); err != nil {
						fmt.Fprintf(os.Stderr, "AddEdge %s→%s: %v\n", e.SourceID, e.TargetID, err)
						edgeErrs++
					}
				}

				fmt.Printf("Loaded %d nodes (%d errors), %d edges (%d errors)\n",
					len(res.Nodes), nodeErrs, len(res.Edges), edgeErrs)
				if nodeErrs+edgeErrs > 0 {
					return fmt.Errorf("load completed with %d insert errors", nodeErrs+edgeErrs)
				}
				return nil
			})
		},
	}
}
