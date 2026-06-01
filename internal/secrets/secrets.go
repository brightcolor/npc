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
	values := map[string]string{}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("%s contains invalid env line", path)
		}
		key = strings.TrimSpace(strings.TrimPrefix(key, "export "))
		value = cleanValue(value)
		if value == "" {
			continue
		}
		values[key] = value
	}
	if provider == "cloudflare" {
		return cloudflareEnv(path, values)
	}
	env := []string{}
	for key, value := range values {
		env = append(env, key+"="+value)
	}
	if len(env) == 0 {
		return nil, fmt.Errorf("%s contains no environment variables", path)
	}
	return env, nil
}

func cloudflareEnv(path string, values map[string]string) ([]string, error) {
	token := values["CF_Token"]
	zone := values["CF_Zone_ID"]
	account := values["CF_Account_ID"]
	key := values["CF_Key"]
	email := values["CF_Email"]
	switch {
	case token != "" && (zone != "" || account != ""):
		env := []string{"CF_Token=" + token}
		if account != "" {
			env = append(env, "CF_Account_ID="+account)
		}
		if zone != "" {
			env = append(env, "CF_Zone_ID="+zone)
		}
		return env, nil
	case key != "" && email != "":
		return []string{"CF_Key=" + key, "CF_Email=" + email}, nil
	default:
		return nil, fmt.Errorf("%s needs CF_Token plus CF_Zone_ID or CF_Account_ID; legacy CF_Key plus CF_Email is also supported", path)
	}
}

func cleanValue(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
			return value[1 : len(value)-1]
		}
	}
	return value
}
