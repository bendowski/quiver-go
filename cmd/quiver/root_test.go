package main

import (
	"io"
	"testing"
)

// These tests exercise command wiring only; none of them reach RunE, so no
// database is opened. The package still needs the native LadyBug library to
// compile (it imports the store), so they run under -tags system_ladybug.

func TestRootHasSubcommands(t *testing.T) {
	found := map[string]bool{}
	for _, c := range newRootCmd().Commands() {
		found[c.Name()] = true
	}
	for _, name := range []string{"load", "query", "dump"} {
		if !found[name] {
			t.Errorf("subcommand %q not registered", name)
		}
	}
}

func TestRootRequiresDBFlag(t *testing.T) {
	root := newRootCmd()
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	root.SetArgs([]string{"query", "MATCH (n) RETURN n"})
	if err := root.Execute(); err == nil {
		t.Fatal("expected an error when --db is missing")
	}
}

func TestUnknownSubcommand(t *testing.T) {
	root := newRootCmd()
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	root.SetArgs([]string{"--db", "/tmp/x", "frobnicate"})
	if err := root.Execute(); err == nil {
		t.Fatal("expected an error for an unknown subcommand")
	}
}
