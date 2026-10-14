package app

import (
	"strings"
	"testing"

	"github.com/martinsuchenak/rackd/pkg/extension"
	"github.com/paularlott/cli"
)

type withCommands struct {
	name     string
	commands []string
}

func (f withCommands) Manifest() extension.Manifest { return extension.Manifest{Name: f.name} }

func (f withCommands) Commands() []*cli.Command {
	var out []*cli.Command
	for _, c := range f.commands {
		out = append(out, &cli.Command{Name: c})
	}
	return out
}

func TestCommandAddsExtensionCommands(t *testing.T) {
	cmd, err := Command(Options{Name: "custom", Version: "1.2.3", Features: []extension.Feature{withCommands{"one", []string{"widgets"}}}})
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Name != "custom" || cmd.Version != "1.2.3" {
		t.Errorf("name/version = %s %s", cmd.Name, cmd.Version)
	}
	var names []string
	for _, c := range cmd.Commands {
		names = append(names, c.Name)
	}
	if !strings.Contains(strings.Join(names, " "), "widgets") || !strings.Contains(strings.Join(names, " "), "server") {
		t.Errorf("commands = %v", names)
	}
}

func TestCommandRejectsDuplicateCommands(t *testing.T) {
	for _, features := range [][]extension.Feature{
		{withCommands{"one", []string{"device"}}},
		{withCommands{"one", []string{"widgets"}}, withCommands{"two", []string{"widgets"}}},
	} {
		if _, err := Command(Options{Features: features}); err == nil || !strings.Contains(err.Error(), "already provided by") {
			t.Errorf("Command(%v) = %v", features, err)
		}
	}
}
