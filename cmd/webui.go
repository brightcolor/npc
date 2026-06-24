package cmd

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/brightcolor/npc/internal/paths"
	"github.com/brightcolor/npc/internal/system"
	"github.com/spf13/cobra"
)

const webuiServicePath = "/etc/systemd/system/npc-webui.service"

type webuiOptions struct {
	listen string
}

func webuiCommand() *cobra.Command {
	o := webuiOptions{listen: "127.0.0.1:8088"}
	cmd := &cobra.Command{Use: "webui", Short: "Start the npc web interface", RunE: func(cmd *cobra.Command, args []string) error {
		return runWebUI(o)
	}}
	bindWebUIFlags(cmd, &o)
	cmd.AddCommand(webuiUnitCommand(&o), webuiInstallServiceCommand(&o), webuiUninstallServiceCommand())
	return cmd
}

func bindWebUIFlags(cmd *cobra.Command, o *webuiOptions) {
	cmd.Flags().StringVar(&o.listen, "listen", o.listen, "listen address, for example 127.0.0.1:8088 or 0.0.0.0:8088")
}

func runWebUI(o webuiOptions) error {
	if err := validateListenAddress(o.listen); err != nil {
		return validationError{err}
	}
	server := &http.Server{Addr: o.listen, Handler: webUIHandler()}
	fmt.Println("npc web UI listening on http://" + o.listen)
	fmt.Println("Put this web UI behind your reverse proxy authentication before exposing it.")
	return server.ListenAndServe()
}

func webuiUnitCommand(o *webuiOptions) *cobra.Command {
	cmd := &cobra.Command{Use: "unit", Short: "Print a systemd unit for npc webui", RunE: func(cmd *cobra.Command, args []string) error {
		if err := validateListenAddress(o.listen); err != nil {
			return validationError{err}
		}
		fmt.Print(renderWebUIUnit(o.listen))
		return nil
	}}
	bindWebUIFlags(cmd, o)
	return cmd
}

func webuiInstallServiceCommand(o *webuiOptions) *cobra.Command {
	cmd := &cobra.Command{Use: "install-service", Short: "Install and start npc webui as a systemd service", RunE: func(cmd *cobra.Command, args []string) error {
		if err := system.RequireRoot(); err != nil {
			return permissionError{err}
		}
		if err := validateListenAddress(o.listen); err != nil {
			return validationError{err}
		}
		unit := []byte(renderWebUIUnit(o.listen))
		if err := os.WriteFile(webuiServicePath, unit, 0o644); err != nil {
			return err
		}
		if res, err := system.Run("systemctl", "daemon-reload"); err != nil {
			return fmt.Errorf("systemctl daemon-reload failed: %s", res.Output)
		}
		if res, err := system.Run("systemctl", "enable", "--now", "npc-webui.service"); err != nil {
			return fmt.Errorf("systemctl enable --now failed: %s", res.Output)
		}
		fmt.Println("Installed and started npc-webui.service on http://" + o.listen)
		return nil
	}}
	bindWebUIFlags(cmd, o)
	return cmd
}

func webuiUninstallServiceCommand() *cobra.Command {
	return &cobra.Command{Use: "uninstall-service", Short: "Stop and remove the npc webui systemd service", RunE: func(cmd *cobra.Command, args []string) error {
		if err := system.RequireRoot(); err != nil {
			return permissionError{err}
		}
		_, _ = system.Run("systemctl", "disable", "--now", "npc-webui.service")
		if err := os.Remove(webuiServicePath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if res, err := system.Run("systemctl", "daemon-reload"); err != nil {
			return fmt.Errorf("systemctl daemon-reload failed: %s", res.Output)
		}
		fmt.Println("Removed npc-webui.service")
		return nil
	}}
}

func validateListenAddress(value string) error {
	host, port, err := net.SplitHostPort(value)
	if err != nil || host == "" || port == "" {
		return fmt.Errorf("listen must be host:port, for example 127.0.0.1:8088")
	}
	if strings.ContainsAny(value, " \t\r\n'\"") {
		return fmt.Errorf("listen address contains invalid characters")
	}
	if net.ParseIP(host) == nil && !validListenHostname(host) {
		return fmt.Errorf("listen host must be an IP address, localhost, or a plain hostname")
	}
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 {
		return fmt.Errorf("listen port must be between 1 and 65535")
	}
	return nil
}

func validListenHostname(host string) bool {
	if host != "localhost" && !strings.Contains(host, ".") {
		return host == "0.0.0.0"
	}
	for _, r := range host {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '.' {
			continue
		}
		return false
	}
	return true
}

func renderWebUIUnit(listen string) string {
	return `[Unit]
Description=npc web interface
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=` + paths.InstallPath + ` webui --listen ` + listen + ` --no-upgrade
Restart=on-failure
RestartSec=3
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=full
ProtectHome=read-only

[Install]
WantedBy=multi-user.target
`
}
