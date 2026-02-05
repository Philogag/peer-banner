package models

import "time"

// Peer represents a peer in a torrent
type Peer struct {
	IP          string    `json:"ip"`
	Port        int       `json:"port"`
	Progress    float64   `json:"progress"`
	Downloaded  int64     `json:"downloaded"`
	Uploaded     int64     `json:"uploaded"`
	Flags       string    `json:"flags"`
	Relevance   float64   `json:"relevance"`
	ActiveTime  int       `json:"active_time"` // in seconds
	Client      string    `json:"client,omitempty"`
	IsConnected bool      `json:"is_connected,omitempty"`
}

// Torrent represents a torrent in qBittorrent
type Torrent struct {
	Hash         string   `json:"hash"`
	Name         string   `json:"name"`
	Size         int64    `json:"size"`
	Progress     float64  `json:"progress"`
	Uploaded     int64    `json:"uploaded"`
	Downloaded   int64    `json:"downloaded"`
	Ratio        float64  `json:"ratio"`
	NumPeers     int      `json:"num_peers"`
	NumSeeds     int      `json:"num_seeds"`
	NumLeechers  int      `json:"num_leechers"`
	SavePath     string   `json:"save_path,omitempty"`
	Category     string   `json:"category,omitempty"`
	Tags         []string `json:"tags,omitempty"`
	AddedOn      time.Time `json:"added_on,omitempty"`
	CompletionOn time.Time `json:"completion_on,omitempty"`
}

// BannedIP represents a banned IP entry
type BannedIP struct {
	IP       string    `json:"ip"`
	Reason   string    `json:"reason,omitempty"`
	RuleName string    `json:"rule_name,omitempty"`
	BannedAt time.Time `json:"banned_at"`
}

// DetectionResult contains the result of a detection run
type DetectionResult struct {
	BannedIPs    map[string]*BannedIP
	TotalPeers    int
	TotalBanned   int
	ServerName    string
	Timestamp     time.Time
}

// NewDetectionResult creates a new detection result
func NewDetectionResult() *DetectionResult {
	return &DetectionResult{
		BannedIPs: make(map[string]*BannedIP),
	}
}

// AddBannedIP adds an IP to the ban list
func (r *DetectionResult) AddBannedIP(ip string, reason, ruleName string) {
	if _, exists := r.BannedIPs[ip]; !exists {
		r.BannedIPs[ip] = &BannedIP{
			IP:       ip,
			Reason:   reason,
			RuleName: ruleName,
			BannedAt: time.Now(),
		}
	}
}
