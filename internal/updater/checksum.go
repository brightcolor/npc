package updater

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
)

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
