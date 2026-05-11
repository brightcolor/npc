package cmd

import (
	"fmt"

	"github.com/brightcolor/npc/internal/nginx"
	"github.com/brightcolor/npc/internal/system"
	"github.com/spf13/cobra"
)

func testCommand() *cobra.Command {
	return &cobra.Command{Use: "test", Short: "Run nginx -t", RunE: func(cmd *cobra.Command, args []string) error {
		out, err := nginx.Test()
		fmt.Println(out)
		if err != nil {
			return nginxTestError{err}
		}
		return nil
	}}
}

func reloadCommand() *cobra.Command {
	return &cobra.Command{Use: "reload", Short: "Test and reload nginx", RunE: func(cmd *cobra.Command, args []string) error {
		if err := system.RequireRoot(); err != nil {
			return permissionError{err}
		}
		out, err := nginx.Reload()
		fmt.Println(out)
		if err != nil {
			return nginxTestError{err}
		}
		return nil
	}}
}

func restartCommand() *cobra.Command {
	return &cobra.Command{Use: "restart", Short: "Test and restart nginx", RunE: func(cmd *cobra.Command, args []string) error {
		if err := system.RequireRoot(); err != nil {
			return permissionError{err}
		}
		fmt.Println("Warning: restart can interrupt existing connections more than reload.")
		out, err := nginx.Restart()
		fmt.Println(out)
		if err != nil {
			return nginxTestError{err}
		}
		return nil
	}}
}

func installNginxCommand() *cobra.Command {
	var force bool
	cmd := &cobra.Command{Use: "install-nginx", Short: "Install Nginx via apt", RunE: func(cmd *cobra.Command, args []string) error {
		if err := system.RequireRoot(); err != nil {
			return permissionError{err}
		}
		return nginx.InstallApt(force)
	}}
	cmd.Flags().BoolVar(&force, "force", false, "allow apt update and package installation")
	return cmd
}

func logsCommand() *cobra.Command {
	return &cobra.Command{Use: "logs [hostname]", Short: "Show nginx logs", RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 1 {
			site, err := loadSite(args[0])
			if err != nil {
				return err
			}
			if site.AccessLog != "" {
				fmt.Println("Access log:", site.AccessLog)
			}
			if site.ErrorLog != "" {
				fmt.Println("Error log:", site.ErrorLog)
			}
			return nil
		}
		res, err := system.Run("systemctl", "status", "nginx", "--no-pager")
		fmt.Println(res.Output)
		return err
	}}
}
