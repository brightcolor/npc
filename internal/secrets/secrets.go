package secrets

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

func ReadEnv(provider string) ([]string, error) {
	path := EnvPath(provider)
	if !SecureMode(path) {
		return nil, fmt.Errorf("%s is missing or not mode 0600", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	env := []string{}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok || strings.TrimSpace(value) == "" {
			return nil, fmt.Errorf("%s contains empty value for %s", path, strings.TrimSpace(key))
		}
		env = append(env, strings.TrimSpace(key)+"="+strings.TrimSpace(value))
	}
	if len(env) == 0 {
		return nil, fmt.Errorf("%s contains no environment variables", path)
	}
	return env, nil
}
