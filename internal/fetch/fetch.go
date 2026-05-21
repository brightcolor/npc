package fetch

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/brightcolor/npc/internal/system"
)

func Bytes(url string) ([]byte, error) {
	if system.Exists("curl") {
		return run("curl", "-fsSL", "--proto", "=https", "--tlsv1.2", url)
	}
	if system.Exists("wget") {
		return run("wget", "-qO-", url)
	}
	return nil, fmt.Errorf("curl or wget is required to download %s", url)
}

func File(url, path string, mode os.FileMode) error {
	var cmd *exec.Cmd
	if system.Exists("curl") {
		cmd = exec.Command("curl", "-fsSL", "--proto", "=https", "--tlsv1.2", "-o", path, url)
	} else if system.Exists("wget") {
		cmd = exec.Command("wget", "-qO", path, url)
	} else {
		return fmt.Errorf("curl or wget is required to download %s", url)
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("download failed for %s: %s", url, string(out))
	}
	return os.Chmod(path, mode)
}

func run(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s failed: %s", name, string(out))
	}
	return out, nil
}
