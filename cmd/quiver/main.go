// Command quiver parses Go source code into a property graph database and
// provides query and dump operations over that graph.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	lbstore "github.com/bendowski/quiver/store/ladybug"
)

// newRootCmd builds the quiver command tree. The --db flag value is owned
// here and shared with the subcommand constructors by pointer; it is only
// dereferenced inside RunE, after cobra has parsed the flags.
func newRootCmd() *cobra.Command {
	var dbPath string

	root := &cobra.Command{
		Use:   "quiver",
		Short: "Go source-code property graph tool",
		Long: `quiver stores Go source code as a property graph in LadyBug and lets
you query or reconstruct it via Cypher.`,
		// main prints the error once; without these cobra would print it a
		// second time and dump usage text on runtime failures.
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	root.PersistentFlags().StringVar(&dbPath, "db", "", "path to LadyBug database directory")
	_ = root.MarkPersistentFlagRequired("db")
	root.AddCommand(newLoadCmd(&dbPath), newQueryCmd(&dbPath), newDumpCmd(&dbPath))
	return root
}

// withStore opens the store at dbPath, runs fn with the command's context,
// and closes the store afterwards.
func withStore(cmd *cobra.Command, dbPath string, fn func(context.Context, *lbstore.Store) error) error {
	s, err := lbstore.Open(dbPath)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer func() { _ = s.Close() }()
	return fn(cmd.Context(), s)
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := newRootCmd().ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "quiver:", err)
		os.Exit(1)
	}
}
