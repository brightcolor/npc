package secrets

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestSecureMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX file mode semantics are not reliable on Windows")
	}
	path := filepath.Join(t.TempDir(), "secret.env")
	if err := os.WriteFile(path, []byte("TOKEN=redacted\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if !SecureMode(path) {
		t.Fatal("expected 0600 mode to be secure")
	}
}
