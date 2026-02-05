package models

import (
	"time"
)

const (
	// BanStateVersion is the current version of the ban state file
	BanStateVersion = 2
)

// BanState represents the persisted ban state
type BanState struct {
	Version     int                `json:"version"`
	LastUpdated time.Time          `json:"last_updated"`
	Bans        map[string]*BannedIP `json:"bans"`
}

// NewBanState creates a new ban state
func NewBanState() *BanState {
	return &BanState{
		Version:     BanStateVersion,
		LastUpdated: time.Now(),
		Bans:        make(map[string]*BannedIP),
	}
}

// BannedIP represents a banned IP entry
type BannedIP struct {
	IP           string    `json:"ip"`
	Reason       string    `json:"reason,omitempty"`
	RuleName     string    `json:"rule_name,omitempty"`
	BannedAt     time.Time `json:"banned_at"`
	ExpiresAt    time.Time `json:"expires_at"`    // Zero value = never expires
	BanCount     int       `json:"ban_count"`     // Number of times this IP has been banned
	IsPermanent  bool      `json:"is_permanent"` // True if escalated to permanent ban
}

// IsExpired checks if the ban has expired
func (b *BannedIP) IsExpired() bool {
	if b.IsPermanent {
		return false
	}
	return !b.ExpiresAt.IsZero() && time.Now().After(b.ExpiresAt)
}

// IsPermanentBan returns true if this is a permanent ban
func (b *BannedIP) IsPermanentBan() bool {
	return b.IsPermanent || (b.ExpiresAt.IsZero() && b.BanCount > 0)
}

// ShouldEscalate checks if this ban should be escalated to permanent
func (b *BannedIP) ShouldEscalate(maxBanCount int) bool {
	if maxBanCount <= 0 {
		return false // Feature disabled
	}
	return b.BanCount >= maxBanCount
}
