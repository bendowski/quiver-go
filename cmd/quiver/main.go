// Command quiver parses Go source code into a property graph database and
// provides query and dump operations over that graph.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var dbPath string

var rootCmd = &cobra.Command{
	Use:   "quiver",
	Short: "Go source-code property graph tool",
	Long: `quiver stores Go source code as a property graph in LadyBug and lets
you query or reconstruct it via Cypher.`,
}

func init() {
	rootCmd.PersistentFlags().StringVar(&dbPath, "db", "", "path to LadyBug database directory")
	_ = rootCmd.MarkPersistentFlagRequired("db")
	rootCmd.AddCommand(loadCmd, queryCmd, dumpCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
