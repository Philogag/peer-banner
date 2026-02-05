package rules

import (
	"strings"
	"time"

	"github.com/philogag/peer-banner/internal/config"
	"github.com/philogag/peer-banner/internal/models"
)

// FlagCriteria checks for specific peer flags
type FlagCriteria struct {
	Flags []string
}

// NewFlagCriteria creates a new FlagCriteria
func NewFlagCriteria(flags []string) *FlagCriteria {
	return &FlagCriteria{
		Flags: flags,
	}
}

// Match checks if the peer has any of the specified flags
func (c *FlagCriteria) Match(peer *models.Peer, torrent *models.Torrent) bool {
	peerFlags := strings.ToLower(peer.Flags)
	for _, flag := range c.Flags {
		flag = strings.ToLower(flag)
		if strings.Contains(peerFlags, flag) {
			return true
		}
	}
	return false
}

// ProgressCriteria checks peer download progress
type ProgressCriteria struct {
	Min float64
	Max float64
}

// NewProgressCriteria creates a new ProgressCriteria
func NewProgressCriteria(criteria *config.ProgressCriteria) *ProgressCriteria {
	return &ProgressCriteria{
		Min: criteria.Min,
		Max: criteria.Max,
	}
}

// Match checks if the peer's progress is within range
func (c *ProgressCriteria) Match(peer *models.Peer, torrent *models.Torrent) bool {
	progress := peer.Progress * 100 // Convert to percentage

	if c.Min > 0 && progress < c.Min {
		return false
	}
	if c.Max > 0 && progress > c.Max {
		return false
	}
	return true
}

// BytesCriteria checks uploaded/downloaded bytes
type BytesCriteria struct {
	Mode       string
	Min        int64
	Max        int64
	MinPercent float64
	MaxPercent float64
}

// NewBytesCriteria creates a new BytesCriteria
func NewBytesCriteria(criteria *config.BytesCriteria) *BytesCriteria {
	c := &BytesCriteria{
		Mode:       criteria.Mode,
		MinPercent: criteria.MinPercent,
		MaxPercent: criteria.MaxPercent,
	}

	if criteria.Min != "" {
		c.Min = ParseBytes(criteria.Min)
	}
	if criteria.Max != "" {
		c.Max = ParseBytes(criteria.Max)
	}

	return c
}

// Match checks if the peer's bytes are within range
func (c *BytesCriteria) Match(peer *models.Peer, torrent *models.Torrent) bool {
	var peerBytes int64
	var torrentSize int64

	if c.Mode == "percent" {
		// Use downloaded as the reference for percent calculation
		peerBytes = peer.Downloaded
		if torrent != nil && torrent.Size > 0 {
			torrentSize = torrent.Size
		} else {
			// Can't calculate percent without torrent size
			return false
		}

		percent := float64(peerBytes) / float64(torrentSize) * 100

		if c.MinPercent > 0 && percent < c.MinPercent {
			return false
		}
		if c.MaxPercent > 0 && percent > c.MaxPercent {
			return false
		}
	} else {
		// Absolute mode
		if c.Min > 0 && peerBytes < c.Min {
			return false
		}
		if c.Max > 0 && peerBytes > c.Max {
			return false
		}
	}

	return true
}

// RangeCriteria checks a value range (like relevance)
type RangeCriteria struct {
	Min float64
	Max float64
}

// NewRangeCriteria creates a new RangeCriteria
func NewRangeCriteria(criteria *config.RangeCriteria) *RangeCriteria {
	return &RangeCriteria{
		Min: criteria.Min,
		Max: criteria.Max,
	}
}

// Match checks if the value is within range
func (c *RangeCriteria) Match(peer *models.Peer, torrent *models.Torrent) bool {
	value := peer.Relevance

	if c.Min > 0 && value < c.Min {
		return false
	}
	if c.Max > 0 && value > c.Max {
		return false
	}
	return true
}

// TimeCriteria checks active time duration
type TimeCriteria struct {
	Min time.Duration
	Max time.Duration
}

// NewTimeCriteria creates a new TimeCriteria
func NewTimeCriteria(criteria *config.TimeCriteria) *TimeCriteria {
	c := &TimeCriteria{}

	if criteria.Min != "" {
		c.Min = ParseDuration(criteria.Min)
	}
	if criteria.Max != "" {
		c.Max = ParseDuration(criteria.Max)
	}

	return c
}

// Match checks if the active time is within range
func (c *TimeCriteria) Match(peer *models.Peer, torrent *models.Torrent) bool {
	activeTime := time.Duration(peer.ActiveTime) * time.Second

	if c.Min > 0 && activeTime < c.Min {
		return false
	}
	if c.Max > 0 && activeTime > c.Max {
		return false
	}
	return true
}
