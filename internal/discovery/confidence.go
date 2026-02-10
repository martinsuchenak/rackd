package discovery

const (
	ConfidenceHigh   = 3
	ConfidenceMedium = 2
	ConfidenceLow    = 1
)

type HostnameSource struct {
	Hostname   string
	Source     string
	Confidence int
}

type ConfidenceScorer struct {
	sources []HostnameSource
}

func NewConfidenceScorer() *ConfidenceScorer {
	return &ConfidenceScorer{
		sources: []HostnameSource{},
	}
}

func (s *ConfidenceScorer) Add(hostname, source string, confidence int) {
	if hostname == "" {
		return
	}
	s.sources = append(s.sources, HostnameSource{
		Hostname:   hostname,
		Source:     source,
		Confidence: confidence,
	})
}

func (s *ConfidenceScorer) GetBest() (string, int) {
	if len(s.sources) == 0 {
		return "", 0
	}

	best := s.sources[0]
	for _, source := range s.sources {
		if source.Confidence > best.Confidence {
			best = source
		}
	}

	return best.Hostname, best.Confidence
}

func (s *ConfidenceScorer) GetAll() []HostnameSource {
	return s.sources
}

func GetHostnameSourceConfidence(source string) int {
	switch source {
	case "ssh":
		return ConfidenceHigh
	case "snmp":
		return ConfidenceHigh
	case "netbios":
		return ConfidenceMedium
	case "mdns":
		return ConfidenceMedium
	case "dns":
		return ConfidenceLow
	default:
		return ConfidenceLow
	}
}

type OSSource struct {
	OS         string
	Source     string
	Confidence int
}

type OSConfidenceScorer struct {
	sources []OSSource
}

func NewOSConfidenceScorer() *OSConfidenceScorer {
	return &OSConfidenceScorer{
		sources: []OSSource{},
	}
}

func (s *OSConfidenceScorer) Add(os, source string, confidence int) {
	if os == "" {
		return
	}
	s.sources = append(s.sources, OSSource{
		OS:         os,
		Source:     source,
		Confidence: confidence,
	})
}

func (s *OSConfidenceScorer) GetBest() (string, int) {
	if len(s.sources) == 0 {
		return "", 0
	}

	best := s.sources[0]
	for _, source := range s.sources {
		if source.Confidence > best.Confidence {
			best = source
		}
	}

	return best.OS, best.Confidence
}

func GetOSSourceConfidence(source string) int {
	switch source {
	case "fingerprinting":
		return ConfidenceHigh
	case "ssh":
		return ConfidenceMedium
	case "snmp":
		return ConfidenceLow
	default:
		return ConfidenceLow
	}
}
