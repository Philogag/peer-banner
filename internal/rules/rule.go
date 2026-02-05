package rules

import (
	"time"

	"github.com/philogag/peer-banner/internal/config"
	"github.com/philogag/peer-banner/internal/models"
)

// Rule represents a leecher detection rule
type Rule struct {
	Name        string
	Enabled     bool
	Action      string
	BanDuration time.Duration
	MaxBanCount int
	Filters     []Filter
}

// ParseRule parses a rule configuration into a Rule struct
func ParseRule(cfg *config.RuleConfig) (*Rule, error) {
	if !cfg.Enabled {
		return nil, nil
	}

	banDuration, _ := cfg.GetBanDuration()

	rule := &Rule{
		Name:        cfg.Name,
		Enabled:     cfg.Enabled,
		Action:      cfg.Action,
		BanDuration: banDuration,
		MaxBanCount:  cfg.MaxBanCount,
	}

	// Parse each filter
	for _, f := range cfg.Filters {
		filter := NewGenericFilter(f)
		rule.Filters = append(rule.Filters, filter)
	}

	return rule, nil
}

// GetBanDuration returns the ban duration for this rule
func (r *Rule) GetBanDuration() time.Duration {
	return r.BanDuration
}

// GetMaxBanCount returns the max ban count before escalation
func (r *Rule) GetMaxBanCount() int {
	return r.MaxBanCount
}

// Match checks if a peer matches all filters in the rule (AND logic)
func (r *Rule) Match(peer *models.Peer, torrent *models.Torrent) bool {
	if !r.Enabled {
		return false
	}

	// All filters must match (AND logic)
	for _, f := range r.Filters {
		if !f.Match(peer, torrent) {
			return false
		}
	}

	return true
}
