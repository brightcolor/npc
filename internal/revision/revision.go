package revision

import (
	"os"
	"path/filepath"

	"github.com/brightcolor/npc/internal/config"
	"github.com/brightcolor/npc/internal/nginx"
	"github.com/brightcolor/npc/internal/paths"
	"gopkg.in/yaml.v3"
)

type Revision struct {
	Dir      string `json:"dir"`
	Metadata string `json:"metadata"`
	Config   string `json:"config,omitempty"`
}

func Save(site *config.Site, renderedConfig string) (*Revision, error) {
	dir := filepath.Join(paths.StateDir, "sites", site.Hostname, "revisions", nginx.Timestamp())
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	metaPath := filepath.Join(dir, "site.yaml")
	data, err := yaml.Marshal(site)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(metaPath, data, 0o600); err != nil {
		return nil, err
	}
	rev := &Revision{Dir: dir, Metadata: metaPath}
	if renderedConfig != "" {
		configPath := filepath.Join(dir, "nginx.conf")
		if err := os.WriteFile(configPath, []byte(renderedConfig), 0o600); err != nil {
			return nil, err
		}
		rev.Config = configPath
	}
	return rev, nil
}
