package rules

import (
	"github.com/philogag/peer-banner/internal/config"
	"github.com/philogag/peer-banner/internal/models"
)

// Rule represents a leecher detection rule
type Rule struct {
	Name     string
	Enabled  bool
	Action   string
	Filters  []Filter
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

	// Parse each filter
	for _, f := range cfg.Filters {
		filter := NewGenericFilter(f)
		rule.Filters = append(rule.Filters, filter)
	}

	return rule, nil
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
