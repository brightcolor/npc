package textdiff

import (
	"strings"
	"testing"
)

func TestUnifiedShowsChangedLines(t *testing.T) {
	out := Unified("a", "one\ntwo", "b", "one\nthree")
	if !strings.Contains(out, "-two") || !strings.Contains(out, "+three") {
		t.Fatalf("diff did not include changed lines:\n%s", out)
	}
}
