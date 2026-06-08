package cmd

import (
	"fmt"
	"strings"

	"github.com/brightcolor/npc/internal/config"
	"github.com/brightcolor/npc/internal/system"
	"github.com/spf13/cobra"
)

func setSiteCommand() *cobra.Command {
	var alias, group, tags string
	cmd := &cobra.Command{Use: "set <hostname-or-alias>", Args: cobra.ExactArgs(1), Short: "Set site alias, group, or tags", RunE: func(cmd *cobra.Command, args []string) error {
		if err := system.RequireRoot(); err != nil {
			return permissionError{err}
		}
		cfg, err := config.Load("")
		if err != nil {
			return err
		}
		site, ok := cfg.FindSite(args[0])
		if !ok {
			return validationError{fmt.Errorf("site %s is not managed by npc", args[0])}
		}
		if alias != "" {
			if other, ok := cfg.FindSite(alias); ok && other.Hostname != site.Hostname {
				return validationError{fmt.Errorf("alias %s is already used by %s", alias, other.Hostname)}
			}
			site.Alias = alias
		}
		if group != "" {
			site.Group = group
		}
		if tags != "" {
			site.Tags = splitTags(tags)
		}
		if err := config.Save("", cfg); err != nil {
			return err
		}
		fmt.Println("Updated", site.Hostname)
		return nil
	}}
	cmd.Flags().StringVar(&alias, "alias", "", "short site alias")
	cmd.Flags().StringVar(&group, "group", "", "site group")
	cmd.Flags().StringVar(&tags, "tags", "", "comma-separated tags")
	return cmd
}

func archiveCommand() *cobra.Command {
	return archiveSetCommand("archive", true)
}

func unarchiveCommand() *cobra.Command {
	return archiveSetCommand("unarchive", false)
}

func archiveSetCommand(name string, archived bool) *cobra.Command {
	return &cobra.Command{Use: name + " <hostname-or-alias>", Args: cobra.ExactArgs(1), Short: name + " a managed site", RunE: func(cmd *cobra.Command, args []string) error {
		if err := system.RequireRoot(); err != nil {
			return permissionError{err}
		}
		cfg, err := config.Load("")
		if err != nil {
			return err
		}
		site, ok := cfg.FindSite(args[0])
		if !ok {
			return validationError{fmt.Errorf("site %s is not managed by npc", args[0])}
		}
		site.Archived = archived
		if err := config.Save("", cfg); err != nil {
			return err
		}
		fmt.Println(name+"d", site.Hostname)
		return nil
	}}
}

func splitTags(value string) []string {
	var tags []string
	for _, item := range strings.Split(value, ",") {
		item = strings.TrimSpace(item)
		if item != "" {
			tags = append(tags, item)
		}
	}
	return tags
}
