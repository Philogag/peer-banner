package ban

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/philogag/peer-banner/internal/models"
)

// Manager handles ban state persistence and expiry
type Manager struct {
	stateFile string
	state     *models.BanState
	mu        sync.RWMutex
}

// NewManager creates a new ban manager
func NewManager(stateFile string) (*Manager, error) {
	m := &Manager{
		stateFile: stateFile,
		state:     models.NewBanState(),
	}
	if err := m.Load(); err != nil {
		// Log warning but continue with empty state
		return m, nil
	}
	return m, nil
}

// Load reads ban state from file
func (m *Manager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(m.stateFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No file yet, that's fine
		}
		return fmt.Errorf("failed to read ban state: %w", err)
	}

	var state models.BanState
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("failed to parse ban state: %w", err)
	}

	// Clean up expired bans during load
	cleaned := 0
	for ip, ban := range state.Bans {
		if ban.IsExpired() {
			delete(state.Bans, ip)
			cleaned++
		}
	}
	if cleaned > 0 {
		state.LastUpdated = time.Now()
	}

	m.state = &state
	return nil
}

// Save writes ban state to file
func (m *Manager) Save() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Ensure directory exists
	dir := filepath.Dir(m.stateFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	data, err := json.MarshalIndent(m.state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal ban state: %w", err)
	}

	if err := os.WriteFile(m.stateFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write ban state: %w", err)
	}

	return nil
}

// IsBanned checks if an IP is currently banned and not expired
func (m *Manager) IsBanned(ip string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ban, exists := m.state.Bans[ip]
	if !exists {
		return false
	}
	return !ban.IsExpired()
}

// GetBan returns the ban entry for an IP
func (m *Manager) GetBan(ip string) (*models.BannedIP, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ban, exists := m.state.Bans[ip]
	return ban, exists
}

// AddBan adds or updates a ban for an IP
func (m *Manager) AddBan(ip, reason, ruleName string, duration time.Duration, maxBanCount int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	existing, exists := m.state.Bans[ip]

	if !exists {
		// New ban
		ban := &models.BannedIP{
			IP:       ip,
			Reason:   reason,
			RuleName: ruleName,
			BannedAt: now,
			BanCount: 1,
		}

		// Check if should escalate to permanent
		if maxBanCount > 0 && 1 >= maxBanCount {
			ban.IsPermanent = true
			ban.Reason = fmt.Sprintf("Escalated to permanent ban after %d violations", maxBanCount)
		} else if duration > 0 {
			ban.ExpiresAt = now.Add(duration)
		}
		// If duration == 0, it's a permanent ban (expires_at stays zero)

		m.state.Bans[ip] = ban
	} else {
		// Update existing ban
		existing.BanCount++
		existing.RuleName = ruleName

		// Check if should escalate to permanent
		if existing.ShouldEscalate(maxBanCount) {
			existing.IsPermanent = true
			existing.ExpiresAt = time.Time{} // Clear expiry
			existing.Reason = fmt.Sprintf("Escalated to permanent ban after %d violations", maxBanCount)
		} else if duration > 0 {
			existing.ExpiresAt = now.Add(duration)
		}
		existing.Reason = reason
	}

	m.state.LastUpdated = now
}

// RemoveBan removes a ban explicitly
func (m *Manager) RemoveBan(ip string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.state.Bans, ip)
	m.state.LastUpdated = time.Now()
}

// CleanupExpired removes all expired bans
func (m *Manager) CleanupExpired() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	cleaned := 0
	for ip, ban := range m.state.Bans {
		if ban.IsExpired() {
			delete(m.state.Bans, ip)
			cleaned++
		}
	}

	if cleaned > 0 {
		m.state.LastUpdated = time.Now()
	}

	return cleaned
}

// GetActiveBans returns all non-expired bans
func (m *Manager) GetActiveBans() []*models.BannedIP {
	m.mu.RLock()
	defer m.mu.RUnlock()

	active := make([]*models.BannedIP, 0, len(m.state.Bans))
	for _, ban := range m.state.Bans {
		if !ban.IsExpired() {
			active = append(active, ban)
		}
	}
	return active
}

// GetPermanentBans returns all permanent bans
func (m *Manager) GetPermanentBans() []*models.BannedIP {
	m.mu.RLock()
	defer m.mu.RUnlock()

	permanent := make([]*models.BannedIP, 0)
	for _, ban := range m.state.Bans {
		if ban.IsPermanentBan() {
			permanent = append(permanent, ban)
		}
	}
	return permanent
}

// GetAllBans returns all bans including expired
func (m *Manager) GetAllBans() []*models.BannedIP {
	m.mu.RLock()
	defer m.mu.RUnlock()

	all := make([]*models.BannedIP, 0, len(m.state.Bans))
	for _, ban := range m.state.Bans {
		all = append(all, ban)
	}
	return all
}

// GetStats returns statistics about the ban list
func (m *Manager) GetStats() (total, active, expired, permanent int) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, ban := range m.state.Bans {
		total++
		if ban.IsExpired() {
			expired++
		} else {
			active++
		}
		if ban.IsPermanentBan() {
			permanent++
		}
	}
	return
}
