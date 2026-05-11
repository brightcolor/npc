package main

import "github.com/brightcolor/npc/cmd"

var (
	version   = "0.1.0-dev"
	commit    = "none"
	date      = "unknown"
	repoOwner = "example"
	repoName  = "npc"
)

func main() {
	cmd.Execute(cmd.BuildInfo{
		Version:   version,
		Commit:    commit,
		Date:      date,
		RepoOwner: repoOwner,
		RepoName:  repoName,
	})
}
