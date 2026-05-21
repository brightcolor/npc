package revision

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/brightcolor/npc/internal/config"
	"github.com/brightcolor/npc/internal/nginx"
	"github.com/brightcolor/npc/internal/paths"
	"gopkg.in/yaml.v3"
)

type Revision struct {
	Dir      string `json:"dir"`
	ID       string `json:"id,omitempty"`
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
	rev := &Revision{Dir: dir, ID: filepath.Base(dir), Metadata: metaPath}
	if renderedConfig != "" {
		configPath := filepath.Join(dir, "nginx.conf")
		if err := os.WriteFile(configPath, []byte(renderedConfig), 0o600); err != nil {
			return nil, err
		}
		rev.Config = configPath
	}
	return rev, nil
}

func List(hostname string) ([]Revision, error) {
	base := filepath.Join(paths.StateDir, "sites", hostname, "revisions")
	entries, err := os.ReadDir(base)
	if os.IsNotExist(err) {
		return []Revision{}, nil
	}
	if err != nil {
		return nil, err
	}
	revisions := make([]Revision, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dir := filepath.Join(base, entry.Name())
		rev := Revision{
			Dir:      dir,
			ID:       entry.Name(),
			Metadata: filepath.Join(dir, "site.yaml"),
			Config:   filepath.Join(dir, "nginx.conf"),
		}
		revisions = append(revisions, rev)
	}
	sort.Slice(revisions, func(i, j int) bool { return revisions[i].ID < revisions[j].ID })
	return revisions, nil
}

func Latest(hostname string) (*Revision, error) {
	revisions, err := List(hostname)
	if err != nil || len(revisions) == 0 {
		return nil, err
	}
	return &revisions[len(revisions)-1], nil
}

func Find(hostname, id string) (*Revision, error) {
	revisions, err := List(hostname)
	if err != nil {
		return nil, err
	}
	for _, rev := range revisions {
		if rev.ID == id || strings.HasSuffix(rev.ID, id) {
			return &rev, nil
		}
	}
	return nil, os.ErrNotExist
}

func LoadSite(rev Revision) (*config.Site, error) {
	data, err := os.ReadFile(rev.Metadata)
	if err != nil {
		return nil, err
	}
	var site config.Site
	if err := yaml.Unmarshal(data, &site); err != nil {
		return nil, err
	}
	return &site, nil
}
