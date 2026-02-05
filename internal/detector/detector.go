package detector

import (
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/philogag/peer-banner/internal/api"
	"github.com/philogag/peer-banner/internal/config"
	"github.com/philogag/peer-banner/internal/models"
	"github.com/philogag/peer-banner/internal/rules"
)

// Detector is the leecher detection engine
type Detector struct {
	client    *api.Client
	rules     []*rules.Rule
	whitelist Whitelist
}

// Whitelist represents an IP whitelist
type Whitelist struct {
	IPs []string
}

// NewDetector creates a new detection engine
func NewDetector(client *api.Client, ruleConfigs []config.RuleConfig, whitelistCfg config.WhitelistConfig) (*Detector, error) {
	// Parse rules
	var parsedRules []*rules.Rule
	for _, rc := range ruleConfigs {
		rule, err := rules.ParseRule(&rc)
		if err != nil {
			return nil, fmt.Errorf("failed to parse rule %s: %w", rc.Name, err)
		}
		if rule != nil {
			parsedRules = append(parsedRules, rule)
		}
	}

	return &Detector{
		client:    client,
		rules:     parsedRules,
		whitelist: parseWhitelist(whitelistCfg.IPs),
	}, nil
}

// parseWhitelist parses whitelist CIDR and IP ranges
func parseWhitelist(ips []string) Whitelist {
	var whitelist Whitelist
	for _, ip := range ips {
		ip = strings.TrimSpace(ip)
		if ip != "" {
			whitelist.IPs = append(whitelist.IPs, ip)
		}
	}
	return whitelist
}

// IsWhitelisted checks if an IP is in the whitelist
func (w *Whitelist) IsWhitelisted(ip string) bool {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	for _, cidr := range w.IPs {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			// Try as a plain IP
			if ip == cidr {
				return true
			}
			continue
		}
		if network.Contains(parsedIP) {
			return true
		}
	}

	return false
}

// Detect performs the leecher detection
func (d *Detector) Detect() (*models.DetectionResult, error) {
	result := models.NewDetectionResult()
	result.ServerName = d.client.Name()
	result.Timestamp = time.Now()

	// Get all peers from all torrents
	torrents, err := d.client.GetTorrents()
	if err != nil {
		return nil, fmt.Errorf("failed to get torrents: %w", err)
	}

	log.Printf("[%s] Checking %d torrents...", d.client.Name(), len(torrents))

	// Track seen IPs to avoid duplicates
	seenIPs := make(map[string]bool)

	var mu sync.Mutex
	var wg sync.WaitGroup

	// Process each torrent concurrently
	for _, torrent := range torrents {
		wg.Add(1)
		go func(t models.Torrent) {
			defer wg.Done()

			peers, err := d.client.GetTorrentPeers(t.Hash)
			if err != nil {
				log.Printf("[%s] Failed to get peers for torrent %s: %v", d.client.Name(), t.Name, err)
				return
			}

			for _, peer := range peers {
				mu.Lock()
				result.TotalPeers++
				ip := peer.IP
				mu.Unlock()

				// Skip if already processed
				mu.Lock()
				if seenIPs[ip] {
					mu.Unlock()
					continue
				}
				seenIPs[ip] = true
				mu.Unlock()

				// Check whitelist
				if d.whitelist.IsWhitelisted(ip) {
					continue
				}

				// Check against all rules
				for _, rule := range d.rules {
					if rule.Match(&peer, &t) {
						mu.Lock()
						result.AddBannedIP(ip, "Matched rule: "+rule.Name, rule.Name)
						result.TotalBanned++
						log.Printf("[%s] Banned %s (rule: %s, progress: %.1f%%, uploaded: %d)",
							d.client.Name(), ip, rule.Name, peer.Progress*100, peer.Uploaded)
						mu.Unlock()
						break // Only ban once per IP
					}
				}
			}
		}(torrent)
	}

	wg.Wait()

	return result, nil
}

// GetRuleCount returns the number of enabled rules
func (d *Detector) GetRuleCount() int {
	return len(d.rules)
}
