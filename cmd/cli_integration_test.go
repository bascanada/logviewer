package cmd

import (
	"bytes"
	"io"
	"os"
	"testing"
)

// captureOutput redirects stdout to a buffer while fn runs and returns the captured output.
func captureOutput(fn func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outC := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		outC <- buf.String()
	}()

	fn()

	_ = w.Close()
	os.Stdout = old
	return <-outC
}

func TestVersionCommand_Output(t *testing.T) {
	// run `logviewer version` and capture stdout
	rootCmd.SetArgs([]string{"version"})
	out := captureOutput(func() {
		if _, err := rootCmd.ExecuteC(); err != nil {
			t.Fatalf("version command failed: %v", err)
		}
	})

	// default sha1ver is 'develop' as set in init()
	if out != "develop\n" {
		t.Fatalf("unexpected version output: %q", out)
	}
}

func TestHelpOutput_RootAndQuery(t *testing.T) {
	// root help
	rootCmd.SetArgs([]string{"--help"})
	out1 := captureOutput(func() {
		if _, err := rootCmd.ExecuteC(); err != nil {
			t.Fatalf("root --help failed: %v", err)
		}
	})
	if len(out1) == 0 {
		t.Fatalf("expected root help output, got empty")
	}

	// query help (subcommand)
	rootCmd.SetArgs([]string{"query", "--help"})
	out2 := captureOutput(func() {
		if _, err := rootCmd.ExecuteC(); err != nil {
			t.Fatalf("query --help failed: %v", err)
		}
	})
	if len(out2) == 0 {
		t.Fatalf("expected query help output, got empty")
	}
}
