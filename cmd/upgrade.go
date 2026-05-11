package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

func upgradeCommand() *cobra.Command {
	return &cobra.Command{Use: "upgrade", Short: "Upgrade npc from GitHub Releases", RunE: runUpgrade}
}

func runUpgrade(cmd *cobra.Command, args []string) error {
	artifact := fmt.Sprintf("npc-%s-%s", runtime.GOOS, runtime.GOARCH)
	return validationError{fmt.Errorf("self-upgrade is scaffolded for %s/%s (%s); configure repoOwner/repoName and enable release download in the next phase", app.build.RepoOwner, app.build.RepoName, artifact)}
}
