package main

import (
	"context"
	"fmt"
	"strings"
	"text/tabwriter"
	"os"

	"github.com/spf13/cobra"

	"github.com/bendowski/quiver/query/cypher"
	lbstore "github.com/bendowski/quiver/store/ladybug"
)

var queryCmd = &cobra.Command{
	Use:   "query <cypher>",
	Short: "Execute a Cypher query and print results",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		q := args[0]
		ctx := context.Background()

		s, err := lbstore.Open(dbPath)
		if err != nil {
			return fmt.Errorf("open db: %w", err)
		}
		defer func() { _ = s.Close() }()

		querier := cypher.New(s.Connection())
		result, err := querier.Query(ctx, q)
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
		w.Flush()
		fmt.Printf("\n%d row(s)\n", len(result.Rows))
		return nil
	},
}
