package rules

import (
	"strings"
	"time"

	"github.com/philogag/peer-banner/internal/config"
	"github.com/philogag/peer-banner/internal/models"
)

// Criteria is the interface for matching criteria
type Criteria interface {
	// Match checks if a peer matches the criteria
	Match(peer *models.Peer, torrent *models.Torrent) bool
}

// Rule represents a leecher detection rule
type Rule struct {
	Name     string
	Enabled  bool
	Action   string
	Criteria []Criteria
}

// ParseRule parses a rule configuration into a Rule struct
func ParseRule(cfg *config.RuleConfig) (*Rule, error) {
	if !cfg.Enabled {
		return nil, nil
	}

	rule := &Rule{
		Name:    cfg.Name,
		Enabled: cfg.Enabled,
		Action:  cfg.Action,
	}

	// Parse each criteria
	for _, c := range parseCriteria(&cfg.Criteria) {
		if c != nil {
			rule.Criteria = append(rule.Criteria, c)
		}
	}

	return rule, nil
}

// Match checks if a peer matches all criteria in the rule (AND logic)
func (r *Rule) Match(peer *models.Peer, torrent *models.Torrent) bool {
	if !r.Enabled {
		return false
	}

	// All criteria must match (AND logic)
	for _, c := range r.Criteria {
		if !c.Match(peer, torrent) {
			return false
		}
	}

	return true
}

// parseCriteria parses all criteria from a CriteriaConfig
func parseCriteria(cfg *config.CriteriaConfig) []Criteria {
	var result []Criteria

	if len(cfg.Flag) > 0 {
		result = append(result, NewFlagCriteria(cfg.Flag))
	}

	if cfg.Progress != nil {
		result = append(result, NewProgressCriteria(cfg.Progress))
	}

	if cfg.Uploaded != nil {
		result = append(result, NewBytesCriteria(cfg.Uploaded))
	}

	if cfg.Downloaded != nil {
		result = append(result, NewBytesCriteria(cfg.Downloaded))
	}

	if cfg.Relevance != nil {
		result = append(result, NewRangeCriteria(cfg.Relevance))
	}

	if cfg.ActiveTime != nil {
		result = append(result, NewTimeCriteria(cfg.ActiveTime))
	}

	return result
}

// ParseBytes parses a byte string like "1GB" to bytes
func ParseBytes(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}

	var multiplier int64 = 1
	var numStr string

	switch {
	case strings.HasSuffix(s, "TB"):
		multiplier = 1024 * 1024 * 1024 * 1024
		numStr = s[:len(s)-2]
	case strings.HasSuffix(s, "GB"):
		multiplier = 1024 * 1024 * 1024
		numStr = s[:len(s)-2]
	case strings.HasSuffix(s, "MB"):
		multiplier = 1024 * 1024
		numStr = s[:len(s)-2]
	case strings.HasSuffix(s, "KB"):
		multiplier = 1024
		numStr = s[:len(s)-2]
	case strings.HasSuffix(s, "B"):
		multiplier = 1
		numStr = s[:len(s)-1]
	default:
		numStr = s
	}

	var value int64
	for _, c := range numStr {
		if c >= '0' && c <= '9' {
			value = value*10 + int64(c-'0')
		}
	}

	return value * multiplier
}

// ParseDuration parses a duration string like "24h" to time.Duration
func ParseDuration(s string) time.Duration {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}

	var multiplier time.Duration
	var numStr string

	switch {
	case strings.HasSuffix(s, "d"):
		multiplier = 24 * time.Hour
		numStr = s[:len(s)-1]
	case strings.HasSuffix(s, "h"):
		multiplier = time.Hour
		numStr = s[:len(s)-1]
	case strings.HasSuffix(s, "m"):
		multiplier = time.Minute
		numStr = s[:len(s)-1]
	case strings.HasSuffix(s, "s"):
		multiplier = time.Second
		numStr = s[:len(s)-1]
	default:
		numStr = s
	}

	var value int64
	for _, c := range numStr {
		if c >= '0' && c <= '9' {
			value = value*10 + int64(c-'0')
		}
	}

	return time.Duration(value) * multiplier
}
