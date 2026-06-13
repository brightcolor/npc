package cmd

import (
	"path/filepath"

	"github.com/brightcolor/npc/internal/config"
	"github.com/brightcolor/npc/internal/importer"
	"github.com/brightcolor/npc/internal/paths"
)

func loadManagedConfig() (*config.Config, error) {
	cfg, err := config.Load("")
	if err != nil {
		return nil, err
	}
	files, err := filepath.Glob(filepath.Join(paths.NginxSitesAvailable, "*.conf"))
	if err != nil {
		return nil, err
	}
	return cfg, mergeManagedNginxConfigFiles(cfg, files)
}

func mergeManagedNginxConfigFiles(cfg *config.Config, files []string) error {
	for _, file := range files {
		candidate := importer.ParseFile(file)
		if !candidate.Managed || candidate.Site == nil || candidate.Error != "" {
			continue
		}
		if _, exists := cfg.Sites[candidate.Site.Hostname]; exists {
			continue
		}
		candidate.Site.Profile = "discovered"
		cfg.Sites[candidate.Site.Hostname] = candidate.Site
	}
	return nil
}
