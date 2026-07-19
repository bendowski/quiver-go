package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	lbstore "github.com/bendowski/quiver/store/ladybug"
)

func newQueryCmd(dbPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "query <cypher>",
		Short: "Execute a Cypher query and print results",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			q := args[0]
			return withStore(cmd, *dbPath, func(ctx context.Context, s *lbstore.Store) error {
				result, err := s.Querier().Query(ctx, q)
				if err != nil {
					return fmt.Errorf("query: %w", err)
				}

				w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
				fmt.Fprintln(w, strings.Join(result.Columns, "\t"))
				fmt.Fprintln(w, strings.Repeat("-\t", len(result.Columns)))
				for _, row := range result.Rows {
					vals := make([]string, len(result.Columns))
					for i, col := range result.Columns {
						vals[i] = fmt.Sprintf("%v", row[col])
					}
					fmt.Fprintln(w, strings.Join(vals, "\t"))
				}
				if err := w.Flush(); err != nil {
					return fmt.Errorf("flush output: %w", err)
				}
				fmt.Printf("\n%d row(s)\n", len(result.Rows))
				return nil
			})
		},
	}
}
