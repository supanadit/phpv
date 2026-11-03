package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// Regression: runtime failures used to dump the whole flag reference and then
// repeat the error (Cobra + main). They must print once, via main.
func TestRootCmd_SilencesUsageForRuntimeErrors(t *testing.T) {
	root := NewRootCmd()
	root.AddCommand(&cobra.Command{
		Use: "boom",
		RunE: func(*cobra.Command, []string) error {
			return errors.New("boom failed")
		},
	})

	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"boom"})

	if err := root.Execute(); err == nil {
		t.Fatal("expected Execute to return the runtime error")
	}
	if strings.Contains(buf.String(), "Usage:") {
		t.Fatalf("output = %q, must not dump usage for a runtime error", buf.String())
	}
	if buf.String() != "" {
		t.Fatalf("output = %q, Cobra must stay quiet; main prints the error once", buf.String())
	}
}
