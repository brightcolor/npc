package updater

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/brightcolor/npc/internal/nginx"
	"github.com/brightcolor/npc/internal/paths"
	"github.com/brightcolor/npc/internal/system"
)

type Options struct {
	RepoOwner string
	RepoName  string
	Version   string
}

type Result struct {
	Version  string `json:"version"`
	Artifact string `json:"artifact"`
	Target   string `json:"target"`
	Backup   string `json:"backup,omitempty"`
}

func SHA256File(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func VerifyChecksum(path, sumsText, artifact string) error {
	got, err := SHA256File(path)
	if err != nil {
		return err
	}
	for _, line := range strings.Split(sumsText, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		name := strings.TrimPrefix(fields[1], "*")
		if name == artifact && strings.EqualFold(fields[0], got) {
			return nil
		}
	}
	return fmt.Errorf("checksum verification failed for %s", artifact)
}

func ArtifactName(goos, goarch string) (string, error) {
	if goos != "linux" {
		return "", fmt.Errorf("self-upgrade supports Linux binaries only, got %s/%s", goos, goarch)
	}
	switch goarch {
	case "amd64", "arm64":
		return fmt.Sprintf("npc-%s-%s", goos, goarch), nil
	default:
		return "", fmt.Errorf("unsupported architecture %s", goarch)
	}
}

func Upgrade(opts Options) (*Result, error) {
	if opts.RepoOwner == "" || opts.RepoName == "" {
		return nil, fmt.Errorf("repo owner and name are required")
	}
	artifact, err := ArtifactName(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return nil, err
	}
	target := paths.InstallPath
	if _, err := os.Stat(target); os.IsNotExist(err) {
		target, err = os.Executable()
		if err != nil {
			return nil, err
		}
	}
	if filepath.Clean(target) == filepath.Clean(paths.InstallPath) {
		if err := system.RequireRoot(); err != nil {
			return nil, err
		}
	}
	version := opts.Version
	if version == "" {
		version = "latest"
	}
	base := releaseDownloadBase(opts.RepoOwner, opts.RepoName, version)
	dir, err := os.MkdirTemp("", "npc-upgrade-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	binaryPath := filepath.Join(dir, artifact)
	sumsPath := filepath.Join(dir, "SHA256SUMS")
	if err := downloadFile(base+"/"+artifact, binaryPath, 0o755); err != nil {
		return nil, err
	}
	if err := downloadFile(base+"/SHA256SUMS", sumsPath, 0o644); err != nil {
		return nil, err
	}
	sums, err := os.ReadFile(sumsPath)
	if err != nil {
		return nil, err
	}
	if err := VerifyChecksum(binaryPath, string(sums), artifact); err != nil {
		return nil, err
	}
	backup := fmt.Sprintf("%s.bak.%s", target, nginx.Timestamp())
	if err := copyFile(target, backup, 0o755); err != nil {
		return nil, err
	}
	tmp := target + ".new"
	if err := copyFile(binaryPath, tmp, 0o755); err != nil {
		return nil, err
	}
	if err := os.Rename(tmp, target); err != nil {
		_ = os.Rename(backup, target)
		return nil, err
	}
	return &Result{Version: version, Artifact: artifact, Target: target, Backup: backup}, nil
}

func releaseDownloadBase(owner, repo, version string) string {
	if version == "" || version == "latest" {
		return fmt.Sprintf("https://github.com/%s/%s/releases/latest/download", owner, repo)
	}
	return fmt.Sprintf("https://github.com/%s/%s/releases/download/%s", owner, repo, version)
}

func downloadFile(url, path string, mode os.FileMode) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("download failed for %s: HTTP %d", url, resp.StatusCode)
	}
	out, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, resp.Body); err != nil {
		return err
	}
	return out.Chmod(mode)
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
