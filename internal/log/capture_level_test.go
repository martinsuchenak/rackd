package log

import (
	"io"
	"strings"
	"testing"
)

// The in-app log shows what the configured level shows: debug entries carry
// request paths, queries and usernames.
func TestCaptureFollowsTheConfiguredLevel(t *testing.T) {
	Init("json", "info", io.Discard)
	Debug("hidden", "path", "/x")
	Trace("hidden too")
	Info("shown")
	entries := ListRecentEntries(nil)
	if len(entries) != 1 || entries[0].Message != "shown" {
		t.Errorf("entries = %+v, want only the info one", entries)
	}

	Init("json", "debug", io.Discard)
	Debug("now shown")
	if entries := ListRecentEntries(nil); len(entries) != 1 {
		t.Errorf("debug level: %d entries, want 1", len(entries))
	}
}

func TestQueryFieldsAreRedacted(t *testing.T) {
	Init("json", "info", io.Discard)
	Info("request", "query", "code=abc123&state=s&client_secret=shh&page=2")
	q := ListRecentEntries(nil)[0].Fields["query"]
	for _, secret := range []string{"abc123", "shh"} {
		if strings.Contains(q, secret) {
			t.Errorf("query field %q still has %q", q, secret)
		}
	}
	if !strings.Contains(q, "page=2") {
		t.Errorf("query field %q lost harmless parameters", q)
	}
}
