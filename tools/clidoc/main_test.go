package main

import (
	"os"
	"testing"
)

// docs/cli-reference.md must match the commands. Regenerate it with
// `go run ./tools/clidoc > docs/cli-reference.md`.
func TestReferenceIsCurrent(t *testing.T) {
	want, err := reference()
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile("../../docs/cli-reference.md")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Error("docs/cli-reference.md is out of date; run: go run ./tools/clidoc > docs/cli-reference.md")
	}
}
