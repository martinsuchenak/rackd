package extension_test

import (
	"os/exec"
	"strings"
	"testing"
)

// Extensions in other modules can't import internal/, so neither may the
// packages they build on.
func TestPublicPackagesDoNotImportInternal(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go tool not available")
	}
	for _, pkg := range []string{"github.com/martinsuchenak/rackd/pkg/extension", "github.com/martinsuchenak/rackd/pkg/model"} {
		out, err := exec.Command("go", "list", "-deps", pkg).CombinedOutput()
		if err != nil {
			t.Fatalf("go list %s: %v\n%s", pkg, err, out)
		}
		for _, dep := range strings.Fields(string(out)) {
			if strings.HasPrefix(dep, "github.com/martinsuchenak/rackd/") && strings.Contains(dep, "/internal/") {
				t.Errorf("%s imports %s", pkg, dep)
			}
		}
	}
}
