package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootCommandVersion(t *testing.T) {
	cmd := newRootCommand()
	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetArgs([]string{"--version"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute version command: %v", err)
	}

	if got, want := strings.TrimSpace(out.String()), "ggo "+currentVersion(); got != want {
		t.Fatalf("version output = %q, want %q", got, want)
	}
}
