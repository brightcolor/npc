package backup

import "testing"

func TestRestoreRejectsPathTraversalID(t *testing.T) {
	if _, err := Restore("../outside"); err == nil {
		t.Fatal("expected invalid backup id error")
	}
}

func TestRestoreTarget(t *testing.T) {
	if got := restoreTarget("config.yaml"); got == "" {
		t.Fatal("expected config.yaml restore target")
	}
	if got := restoreTarget("app.example.com.conf"); got == "" {
		t.Fatal("expected nginx conf restore target")
	}
	if got := restoreTarget("secret.env"); got != "" {
		t.Fatalf("unexpected restore target for unsupported file: %q", got)
	}
}
