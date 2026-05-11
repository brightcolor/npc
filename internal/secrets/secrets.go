package secrets

import (
	"os"
	"path/filepath"

	"github.com/brightcolor/npc/internal/paths"
)

func EnvPath(provider string) string {
	return filepath.Join(paths.SecretsDir, provider+".env")
}

func WriteProviderEnv(provider string, content []byte) error {
	if err := os.MkdirAll(paths.SecretsDir, 0o700); err != nil {
		return err
	}
	path := EnvPath(provider)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		return err
	}
	_ = os.Chown(path, 0, 0)
	return nil
}

func SecureMode(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.Mode().Perm() == 0o600
}
