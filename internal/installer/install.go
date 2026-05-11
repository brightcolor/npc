package installer

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/brightcolor/npc/internal/nginx"
	"github.com/brightcolor/npc/internal/paths"
)

func InstallCurrentBinary() error {
	src, err := os.Executable()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(paths.InstallPath), 0o755); err != nil {
		return err
	}
	if _, err := os.Stat(paths.InstallPath); err == nil {
		backup := fmt.Sprintf("%s.bak.%s", paths.InstallPath, nginx.Timestamp())
		if err := copyFile(paths.InstallPath, backup, 0o755); err != nil {
			return err
		}
	}
	tmp := paths.InstallPath + ".tmp"
	if err := copyFile(src, tmp, 0o755); err != nil {
		return err
	}
	_ = os.Chown(tmp, 0, 0)
	return os.Rename(tmp, paths.InstallPath)
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Chmod(mode)
}
