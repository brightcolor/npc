package backup

import (
	"io"
	"os"
	"path/filepath"

	"github.com/brightcolor/npc/internal/nginx"
	"github.com/brightcolor/npc/internal/paths"
)

type Set struct {
	Dir   string   `json:"dir"`
	Files []string `json:"files"`
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
