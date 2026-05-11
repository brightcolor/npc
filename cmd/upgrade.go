package cmd

import (
	"fmt"

	"github.com/brightcolor/npc/internal/updater"
	"github.com/spf13/cobra"
)

func upgradeCommand() *cobra.Command {
	var version string
	cmd := &cobra.Command{Use: "upgrade", Short: "Upgrade npc from GitHub Releases", RunE: func(cmd *cobra.Command, args []string) error {
		return runUpgradeVersion(version)
	}}
	cmd.Flags().StringVar(&version, "version", "", "release version to install, for example v0.1.3; defaults to latest")
	return cmd
}

func runUpgrade(cmd *cobra.Command, args []string) error {
	return runUpgradeVersion("")
}

func runUpgradeVersion(version string) error {
	result, err := updater.Upgrade(updater.Options{
		RepoOwner:      app.build.RepoOwner,
		RepoName:       app.build.RepoName,
		Version:        version,
		CurrentVersion: app.build.Version,
	})
	if err != nil {
		return fmt.Errorf("upgrade failed: %w", err)
	}
	if app.jsonOut {
		return writeJSON(result)
	}
	if !result.Changed {
		fmt.Printf("npc is already up to date (%s)\n", result.ToVersion)
		return nil
	}
	fmt.Printf("Upgraded npc from %s to %s\nArtifact: %s\nTarget: %s\nBackup: %s\n", result.FromVersion, result.ToVersion, result.Artifact, result.Target, result.Backup)
	return nil
}
