package updater

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/brightcolor/npc/internal/fetch"
	"github.com/brightcolor/npc/internal/nginx"
	"github.com/brightcolor/npc/internal/paths"
	"github.com/brightcolor/npc/internal/system"
)

type ReleaseInfo struct {
	CurrentVersion  string `json:"current_version"`
	LatestVersion   string `json:"latest_version"`
	UpdateAvailable bool   `json:"update_available"`
	URL             string `json:"url,omitempty"`
	Changelog       string `json:"changelog,omitempty"`
}

type Options struct {
	RepoOwner      string
	RepoName       string
	Version        string
	CurrentVersion string
}

type Result struct {
	FromVersion string `json:"from_version"`
	ToVersion   string `json:"to_version"`
	Artifact    string `json:"artifact"`
	Target      string `json:"target"`
	Backup      string `json:"backup,omitempty"`
	Changed     bool   `json:"changed"`
}

func Check(owner, repo, current string) (*ReleaseInfo, error) {
	if owner == "" || repo == "" {
		return nil, fmt.Errorf("repo owner and name are required")
	}
	release, err := latestRelease(owner, repo)
	if err != nil {
		return nil, err
	}
	return &ReleaseInfo{
		CurrentVersion:  current,
		LatestVersion:   release.TagName,
		UpdateAvailable: !sameVersion(current, release.TagName),
		URL:             release.HTMLURL,
		Changelog:       release.Body,
	}, nil
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
	toVersion := version
	if version == "latest" {
		latest, err := latestReleaseTag(opts.RepoOwner, opts.RepoName)
		if err != nil {
			return nil, err
		}
		toVersion = latest
	}
	if sameVersion(opts.CurrentVersion, toVersion) {
		return &Result{FromVersion: opts.CurrentVersion, ToVersion: toVersion, Artifact: artifact, Target: target}, nil
	}
	base := releaseDownloadBase(opts.RepoOwner, opts.RepoName, version)
	dir, err := os.MkdirTemp("", "npc-upgrade-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	binaryPath := filepath.Join(dir, artifact)
	sumsPath := filepath.Join(dir, "SHA256SUMS")
	if err := fetch.File(base+"/"+artifact, binaryPath, 0o755); err != nil {
		return nil, err
	}
	if err := fetch.File(base+"/SHA256SUMS", sumsPath, 0o644); err != nil {
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
	return &Result{FromVersion: opts.CurrentVersion, ToVersion: toVersion, Artifact: artifact, Target: target, Backup: backup, Changed: true}, nil
}

func releaseDownloadBase(owner, repo, version string) string {
	if version == "" || version == "latest" {
		return fmt.Sprintf("https://github.com/%s/%s/releases/latest/download", owner, repo)
	}
	return fmt.Sprintf("https://github.com/%s/%s/releases/download/%s", owner, repo, version)
}

func sameVersion(current, target string) bool {
	current = strings.TrimSpace(current)
	target = strings.TrimSpace(target)
	if current == "" || target == "" || current == "unknown" || current == "0.1.0-dev" {
		return false
	}
	return strings.TrimPrefix(current, "v") == strings.TrimPrefix(target, "v")
}

func latestReleaseTag(owner, repo string) (string, error) {
	release, err := latestRelease(owner, repo)
	if err != nil {
		return "", err
	}
	return release.TagName, nil
}

type githubRelease struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
	Body    string `json:"body"`
}

func latestRelease(owner, repo string) (*githubRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)
	data, err := fetch.Bytes(url)
	if err != nil {
		return nil, err
	}
	var payload githubRelease
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	if payload.TagName == "" {
		return nil, fmt.Errorf("latest release response did not include tag_name")
	}
	return &payload, nil
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
