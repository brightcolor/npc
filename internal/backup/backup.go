package backup

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/brightcolor/npc/internal/nginx"
	"github.com/brightcolor/npc/internal/paths"
)

type Set struct {
	Dir   string   `json:"dir"`
	Files []string `json:"files"`
}

func List() ([]string, error) {
	entries, err := os.ReadDir(paths.BackupsDir)
	if os.IsNotExist(err) {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}
	var backups []string
	for _, entry := range entries {
		if entry.IsDir() {
			backups = append(backups, filepath.Join(paths.BackupsDir, entry.Name()))
		}
	}
	sort.Strings(backups)
	return backups, nil
}

func Restore(id string) ([]string, error) {
	if !filepath.IsAbs(id) && (filepath.Clean(id) != id || filepath.Base(id) != id) {
		return nil, fmt.Errorf("invalid backup id %q", id)
	}
	dir := filepath.Join(paths.BackupsDir, id)
	if filepath.IsAbs(id) {
		dir = id
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var restored []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		src := filepath.Join(dir, entry.Name())
		dst := restoreTarget(entry.Name())
		if dst == "" {
			continue
		}
		mode := os.FileMode(0o600)
		if filepath.Ext(dst) == ".conf" {
			mode = 0o644
		}
		if err := copyFile(src, dst, mode); err != nil {
			return restored, err
		}
		restored = append(restored, dst)
	}
	return restored, nil
}

func restoreTarget(name string) string {
	if name == "config.yaml" {
		return paths.ConfigFile
	}
	if filepath.Ext(name) == ".conf" {
		return filepath.Join(paths.NginxSitesAvailable, name)
	}
	return ""
}

func Create(files ...string) (*Set, error) {
	dir := filepath.Join(paths.BackupsDir, nginx.Timestamp())
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	set := &Set{Dir: dir}
	for _, file := range files {
		if file == "" {
			continue
		}
		if _, err := os.Stat(file); err != nil {
			continue
		}
		dst := filepath.Join(dir, filepath.Base(file))
		if err := copyFile(file, dst, 0o600); err != nil {
			return nil, err
		}
		set.Files = append(set.Files, dst)
	}
	return set, nil
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
	_, err = io.Copy(out, in)
	return err
}
