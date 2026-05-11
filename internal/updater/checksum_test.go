package updater

import (
	"os"
	"path/filepath"
	"testing"
)

func TestVerifyChecksum(t *testing.T) {
	path := filepath.Join(t.TempDir(), "npc-linux-amd64")
	if err := os.WriteFile(path, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	sums := "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824  npc-linux-amd64\n"
	if err := VerifyChecksum(path, sums, "npc-linux-amd64"); err != nil {
		t.Fatal(err)
	}
}
