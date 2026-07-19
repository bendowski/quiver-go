package main

import (
	"context"
	"fmt"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/bendowski/quiver/dump"
	lbstore "github.com/bendowski/quiver/store/ladybug"
)

func newDumpCmd(dbPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "dump <outdir>",
		Short: "Reconstruct Go source files from the graph and write them to outdir",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			outDir := args[0]
			return withStore(cmd, *dbPath, func(ctx context.Context, s *lbstore.Store) error {
				d := dump.New(s, afero.NewOsFs())
				if err := d.DumpAll(ctx, outDir); err != nil {
					return fmt.Errorf("dump: %w", err)
				}
				fmt.Printf("Dumped all files to %s\n", outDir)
				return nil
			})
		},
	}
}
