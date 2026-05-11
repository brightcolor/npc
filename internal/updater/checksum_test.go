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

func TestArtifactName(t *testing.T) {
	name, err := ArtifactName("linux", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	if name != "npc-linux-amd64" {
		t.Fatalf("unexpected artifact name: %s", name)
	}
	if _, err := ArtifactName("windows", "amd64"); err == nil {
		t.Fatal("expected non-linux artifact to fail")
	}
}

func TestReleaseDownloadBase(t *testing.T) {
	got := releaseDownloadBase("brightcolor", "npc", "latest")
	if got != "https://github.com/brightcolor/npc/releases/latest/download" {
		t.Fatalf("unexpected latest URL: %s", got)
	}
	got = releaseDownloadBase("brightcolor", "npc", "v0.1.3")
	if got != "https://github.com/brightcolor/npc/releases/download/v0.1.3" {
		t.Fatalf("unexpected version URL: %s", got)
	}
}

func TestSameVersion(t *testing.T) {
	if !sameVersion("v0.1.7", "0.1.7") {
		t.Fatal("expected v-prefixed versions to match")
	}
	if sameVersion("v0.1.6", "v0.1.7") {
		t.Fatal("expected different versions not to match")
	}
	if sameVersion("0.1.0-dev", "v0.1.7") {
		t.Fatal("expected dev version not to match release")
	}
}
