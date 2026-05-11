package nginx

import (
	"errors"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/brightcolor/npc/internal/paths"
	"github.com/brightcolor/npc/internal/system"
)

func Version() string {
	res, err := system.Run("nginx", "-v")
	if err != nil && res.Output == "" {
		return "not installed"
	}
	return strings.TrimSpace(res.Output)
}

func Test() (string, error) {
	res, err := system.Run("nginx", "-t")
	if err != nil {
		return res.Output, err
	}
	return res.Output, nil
}

func Reload() (string, error) {
	if out, err := Test(); err != nil {
		return out, err
	}
	res, err := system.Run("systemctl", "reload", "nginx")
	if err != nil {
		return res.Output, err
	}
	return "nginx configuration test passed; nginx reloaded", nil
}

func Restart() (string, error) {
	if out, err := Test(); err != nil {
		return out, err
	}
	res, err := system.Run("systemctl", "restart", "nginx")
	if err != nil {
		return res.Output, err
	}
	return "nginx configuration test passed; nginx restarted", nil
}

func ServiceActive() bool {
	res, err := system.Run("systemctl", "is-active", "nginx")
	return err == nil && strings.TrimSpace(res.Output) == "active"
}

func WriteSite(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func Enable(configPath, enabledPath string) error {
	if _, err := os.Lstat(enabledPath); err == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(enabledPath), 0o755); err != nil {
		return err
	}
	return os.Symlink(configPath, enabledPath)
}

func Disable(enabledPath string) error {
	if _, err := os.Lstat(enabledPath); errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return os.Remove(enabledPath)
}

func Managed(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), "# Managed by npc")
}

func SitePaths(hostname string) (string, string) {
	name := hostname + ".conf"
	return path.Join(paths.NginxSitesAvailable, name), path.Join(paths.NginxSitesEnabled, name)
}

func InstallApt(assumeYes bool) error {
	if !system.Exists("apt-get") {
		return errors.New("apt-get was not found; automatic nginx installation currently supports apt-based systems")
	}
	if !assumeYes {
		return errors.New("apt update/install requires explicit confirmation; rerun with --force")
	}
	if _, err := system.Run("apt-get", "update"); err != nil {
		return err
	}
	if _, err := system.Run("apt-get", "install", "-y", "nginx"); err != nil {
		return err
	}
	_, _ = system.Run("systemctl", "enable", "nginx")
	_, err := system.Run("systemctl", "start", "nginx")
	return err
}

func Timestamp() string {
	return time.Now().UTC().Format("20060102T150405Z")
}
