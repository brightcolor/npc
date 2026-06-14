package cmd

import (
	"os"
	"os/exec"
)

type uiKey int

const (
	keyOther uiKey = iota
	keyEnter
	keyUp
	keyDown
	keyCancel
)

func enableRawInput() bool {
	return exec.Command("stty", "-echo", "raw").Run() == nil
}

func restoreInput() {
	_ = exec.Command("stty", "echo", "-raw").Run()
}

func readUIKey() (uiKey, int) {
	var b [1]byte
	if _, err := os.Stdin.Read(b[:]); err != nil {
		return keyCancel, 0
	}
	switch b[0] {
	case '\r', '\n':
		return keyEnter, 0
	case 'q', 'Q', 3:
		return keyCancel, 0
	case '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return keyOther, int(b[0] - '0')
	case 27:
		var seq [2]byte
		if _, err := os.Stdin.Read(seq[:]); err != nil {
			return keyCancel, 0
		}
		if seq[0] != '[' {
			return keyCancel, 0
		}
		if seq[1] == 'A' {
			return keyUp, 0
		}
		if seq[1] == 'B' {
			return keyDown, 0
		}
	}
	return keyOther, 0
}
