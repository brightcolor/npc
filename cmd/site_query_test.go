package cmd

import (
	"testing"

	"github.com/brightcolor/npc/internal/config"
)

func TestSiteQueryFiltersByGroupTagAndSearch(t *testing.T) {
	sites := []*config.Site{
		{Hostname: "api.example.com", Alias: "api", Group: "prod", Tags: []string{"docker", "api"}, BackendScheme: "http", BackendHost: "127.0.0.1", BackendPort: 3000},
		{Hostname: "old.example.com", Group: "archive", Archived: true, BackendScheme: "http", BackendHost: "127.0.0.1", BackendPort: 4000},
	}
	matched := siteQuery{group: "prod", tag: "docker", search: "api"}.apply(sites)
	if len(matched) != 1 || matched[0].Hostname != "api.example.com" {
		t.Fatalf("unexpected matches: %#v", matched)
	}
	if archived := (siteQuery{}).apply(sites); len(archived) != 1 {
		t.Fatalf("archived site should be hidden by default: %#v", archived)
	}
}
