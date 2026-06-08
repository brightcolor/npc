package cmd

import (
	"os"
	"sort"
	"strings"
	"time"

	"github.com/brightcolor/npc/internal/certinfo"
	"github.com/brightcolor/npc/internal/config"
)

type siteQuery struct {
	enabled, disabled, sslOnly, noSSL, archived, includeArchived bool
	profile, domain, backend, group, tag, search, sortBy         string
}

func (q siteQuery) apply(sites []*config.Site) []*config.Site {
	out := []*config.Site{}
	for _, site := range sites {
		if q.match(site) {
			out = append(out, site)
		}
	}
	sortSites(out, q.sortBy)
	return out
}

func (q siteQuery) match(site *config.Site) bool {
	enabled := siteEnabled(site)
	switch {
	case q.archived && !site.Archived:
		return false
	case !q.includeArchived && !q.archived && site.Archived:
		return false
	case q.enabled && !enabled:
		return false
	case q.disabled && enabled:
		return false
	case q.sslOnly && !site.SSL:
		return false
	case q.noSSL && site.SSL:
		return false
	case q.profile != "" && site.Profile != q.profile:
		return false
	case q.group != "" && site.Group != q.group:
		return false
	case q.domain != "" && !strings.HasSuffix(site.Hostname, q.domain):
		return false
	case q.backend != "" && !strings.Contains(site.BackendURL(), q.backend):
		return false
	case q.tag != "" && !hasTag(site, q.tag):
		return false
	case q.search != "" && !siteContains(site, q.search):
		return false
	default:
		return true
	}
}

func sortSites(sites []*config.Site, sortBy string) {
	sort.Slice(sites, func(i, j int) bool {
		a, b := sites[i], sites[j]
		switch sortBy {
		case "backend":
			return a.BackendURL() < b.BackendURL()
		case "profile":
			return a.Profile < b.Profile || a.Profile == b.Profile && a.Hostname < b.Hostname
		case "updated":
			return a.UpdatedAt.After(b.UpdatedAt)
		case "enabled":
			return siteEnabled(a) && !siteEnabled(b)
		case "cert-expiry":
			return certDays(a) < certDays(b)
		default:
			return a.Hostname < b.Hostname
		}
	})
}

func siteEnabled(site *config.Site) bool {
	_, err := os.Lstat(site.EnabledPath)
	return err == nil
}

func siteContains(site *config.Site, query string) bool {
	query = strings.ToLower(query)
	haystack := strings.Join([]string{
		site.Hostname, site.Alias, site.Group, strings.Join(site.Tags, " "),
		site.Profile, site.BackendURL(), site.ACMEMethod, site.DNSProvider,
	}, " ")
	return strings.Contains(strings.ToLower(haystack), query)
}

func hasTag(site *config.Site, tag string) bool {
	for _, item := range site.Tags {
		if item == tag {
			return true
		}
	}
	return false
}

func certDays(site *config.Site) int {
	if site.CertificatePath == "" {
		return 999999
	}
	info := certinfo.Read(site.CertificatePath, site.ACME)
	if !info.Exists || info.NotAfter.IsZero() {
		return 999999
	}
	return int(time.Until(info.NotAfter).Hours() / 24)
}
