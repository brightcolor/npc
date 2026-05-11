package system

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"strings"
)

type Result struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
	Output  string   `json:"output"`
}

func IsRoot() bool {
	return os.Geteuid() == 0
}

func RequireRoot() error {
	if !IsRoot() {
		return errors.New("this command needs root privileges; run it with sudo")
	}
	return nil
}

func Run(name string, args ...string) (Result, error) {
	cmd := exec.Command(name, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	return Result{Command: name, Args: args, Output: strings.TrimSpace(out.String())}, err
}

func Exists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
