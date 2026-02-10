package discovery

import (
	"strings"
)

type HostnameConflict struct {
	Hostname    string
	Sources     []string
	Conflicts   map[string][]string
	Recommended string
}

type HostnameCorrelator struct {
}

func NewHostnameCorrelator() *HostnameCorrelator {
	return &HostnameCorrelator{}
}

func (c *HostnameCorrelator) Correlate(sources []HostnameSource) *HostnameConflict {
	if len(sources) == 0 {
		return nil
	}

	if len(sources) == 1 {
		return &HostnameConflict{
			Hostname:    sources[0].Hostname,
			Sources:     []string{sources[0].Source},
			Conflicts:   make(map[string][]string),
			Recommended: sources[0].Hostname,
		}
	}

	uniqueHostnames := make(map[string][]HostnameSource)
	for _, src := range sources {
		normalized := c.normalizeHostname(src.Hostname)
		uniqueHostnames[normalized] = append(uniqueHostnames[normalized], src)
	}

	recommended := c.selectBestHostname(uniqueHostnames)

	var allSources []string
	conflicts := make(map[string][]string)

	for norm, srcs := range uniqueHostnames {
		var srcNames []string
		for _, s := range srcs {
			srcNames = append(srcNames, s.Source)
			allSources = append(allSources, s.Source)
		}
		if len(srcs) > 1 {
			conflicts[norm] = srcNames
		}
	}

	bestHostname := ""
	for _, src := range sources {
		if src.Hostname == recommended {
			bestHostname = src.Hostname
			break
		}
	}

	return &HostnameConflict{
		Hostname:    bestHostname,
		Sources:     allSources,
		Conflicts:   conflicts,
		Recommended: recommended,
	}
}

func (c *HostnameCorrelator) normalizeHostname(hostname string) string {
	hostname = strings.ToLower(strings.TrimSpace(hostname))
	hostname = strings.TrimSuffix(hostname, ".")
	hostname = strings.TrimSuffix(hostname, ".local")
	return hostname
}

func (c *HostnameCorrelator) selectBestHostname(hostnames map[string][]HostnameSource) string {
	var best string
	var bestConfidence int

	for _, srcs := range hostnames {
		for _, src := range srcs {
			if src.Confidence > bestConfidence {
				bestConfidence = src.Confidence
				best = src.Hostname
			}
		}
	}

	if best == "" && len(hostnames) > 0 {
		for hostname := range hostnames {
			return hostname
		}
	}

	return best
}

func (c *HostnameCorrelator) HasConflicts(conflict *HostnameConflict) bool {
	return conflict != nil && len(conflict.Conflicts) > 0
}

func (c *HostnameCorrelator) GetPreferredSource(sources []HostnameSource) string {
	if len(sources) == 0 {
		return ""
	}

	best := sources[0]
	for _, src := range sources {
		if c.compareSources(src, best) > 0 {
			best = src
		}
	}

	return best.Source
}

func (c *HostnameCorrelator) compareSources(a, b HostnameSource) int {
	if a.Confidence > b.Confidence {
		return 1
	}
	if a.Confidence < b.Confidence {
		return -1
	}

	priority := map[string]int{
		"ssh":     5,
		"snmp":    4,
		"netbios": 3,
		"mdns":    2,
		"dns":     1,
	}

	prioA, okA := priority[a.Source]
	prioB, okB := priority[b.Source]
	if !okA {
		prioA = 0
	}
	if !okB {
		prioB = 0
	}

	if prioA > prioB {
		return 1
	}
	if prioA < prioB {
		return -1
	}

	return 0
}

func (c *HostnameCorrelator) MatchHostnames(a, b string) bool {
	return c.normalizeHostname(a) == c.normalizeHostname(b)
}
