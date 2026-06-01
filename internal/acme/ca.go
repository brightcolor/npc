package acme

import (
	"fmt"
	"strings"

	"github.com/brightcolor/npc/internal/system"
)

const DefaultCA = "letsencrypt"

func NormalizeCA(ca string) string {
	ca = strings.ToLower(strings.TrimSpace(ca))
	switch ca {
	case "", "letsencrypt", "le":
		return "letsencrypt"
	case "buypass":
		return "buypass"
	default:
		return ca
	}
}

func ValidateCA(ca string) error {
	switch NormalizeCA(ca) {
	case "letsencrypt", "buypass":
		return nil
	default:
		return fmt.Errorf("unsupported ACME CA %q; use letsencrypt or buypass", ca)
	}
}

func SetDefaultCA(ca string) error {
	ca = NormalizeCA(ca)
	if err := ValidateCA(ca); err != nil {
		return err
	}
	cmd := CommandPath()
	if cmd == "" {
		return fmt.Errorf("acme.sh was not found")
	}
	res, err := system.Run(cmd, "--set-default-ca", "--server", ca)
	if err != nil {
		return fmt.Errorf("acme.sh default CA setup failed: %s%s", res.Output, DiagnoseOutput(res.Output))
	}
	return nil
}
