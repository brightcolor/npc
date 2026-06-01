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

func TestCloudflareEnvAcceptsTokenAndZone(t *testing.T) {
	env, err := cloudflareEnv("test.env", map[string]string{
		"CF_Token":   "token",
		"CF_Zone_ID": "zone",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !containsEnv(env, "CF_Token=token") || !containsEnv(env, "CF_Zone_ID=zone") {
		t.Fatalf("unexpected env: %#v", env)
	}
}

func containsEnv(env []string, want string) bool {
	for _, item := range env {
		if item == want {
			return true
		}
	}
	return false
}
