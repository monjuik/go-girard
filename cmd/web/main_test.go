package main

import (
	"bytes"
	"testing"
)

func TestRunVersion(t *testing.T) {
	var stdout bytes.Buffer

	if err := run([]string{"-version"}, &stdout); err != nil {
		t.Fatalf("run -version: %v", err)
	}

	if got, want := stdout.String(), "go-girard dev\n"; got != want {
		t.Fatalf("version output = %q, want %q", got, want)
	}
}
