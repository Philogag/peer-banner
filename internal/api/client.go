package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/philogag/peer-banner/internal/config"
	"github.com/philogag/peer-banner/internal/models"
)

// Client wraps qBittorrent Web API client
type Client struct {
	baseURL  string
	username string
	password string
	client   *http.Client
	cookies  []*http.Cookie
}

// NewClient creates a new qBittorrent API client
func NewClient(cfg *config.ServerConfig) *Client {
	return &Client{
		baseURL:  cfg.URL,
		username: cfg.Username,
		password: cfg.Password,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Login authenticates with the qBittorrent Web API
func (c *Client) Login() error {
	// Build form data
	form := url.Values{}
	form.Set("username", c.username)
	form.Set("password", c.password)

	// Create request
	req, err := http.NewRequest("POST", c.baseURL+"/api/v2/auth/login", bytes.NewBufferString(form.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Execute request
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to login: %w", err)
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("login failed (status %d): %s", resp.StatusCode, string(body))
	}

	// Save cookies
	c.cookies = resp.Cookies()

	return nil
}

// EnsureAuthenticated checks if we have valid cookies and re-logins if needed
func (c *Client) EnsureAuthenticated() error {
	if len(c.cookies) == 0 {
		return c.Login()
	}
	return nil
}

// doRequest performs an authenticated API request
func (c *Client) doRequest(method, path string, body io.Reader) (*http.Response, error) {
	url := c.baseURL + path
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if len(c.cookies) > 0 {
		for _, cookie := range c.cookies {
			req.AddCookie(cookie)
		}
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	// If unauthorized, try to re-login
	if resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()
		if err := c.Login(); err != nil {
			return nil, err
		}
		return c.doRequest(method, path, body)
	}

	return resp, nil
}

// GetTorrents retrieves the list of torrents
func (c *Client) GetTorrents() ([]models.Torrent, error) {
	resp, err := c.doRequest("GET", "/api/v2/torrents/info?json=1", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get torrents: status %d", resp.StatusCode)
	}

	var torrents []models.Torrent
	if err := json.NewDecoder(resp.Body).Decode(&torrents); err != nil {
		return nil, fmt.Errorf("failed to decode torrents: %w", err)
	}

	return torrents, nil
}

// GetTorrentPeers retrieves the list of peers for a specific torrent
func (c *Client) GetTorrentPeers(hash string) ([]models.Peer, error) {
	resp, err := c.doRequest("GET", "/api/v2/sync/torrentPeers?hash="+hash+"&json=1", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get torrent peers: status %d", resp.StatusCode)
	}

	// Parse the sync response
	var syncResp struct {
		Peers map[string]struct {
			IP          string  `json:"ip"`
			Port        int     `json:"port"`
			Progress    float64 `json:"progress"`
			Downloaded  int64   `json:"downloaded"`
			Uploaded    int64   `json:"uploaded"`
			Flags       string  `json:"flags"`
			Relevance   float64 `json:"relevance"`
			DownloadedT int64   `json:"downloadedt"`
			UploadedT   int64   `json:"uploadedt"`
			Client      string  `json:"client,omitempty"`
		} `json:"peers"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&syncResp); err != nil {
		return nil, fmt.Errorf("failed to decode peers: %w", err)
	}

	// Convert to Peer slice
	peers := make([]models.Peer, 0, len(syncResp.Peers))
	for _, p := range syncResp.Peers {
		peers = append(peers, models.Peer{
			IP:         p.IP,
			Port:       p.Port,
			Progress:   p.Progress,
			Downloaded: p.Downloaded,
			Uploaded:   p.Uploaded,
			Flags:      p.Flags,
			Relevance:  p.Relevance,
			Client:     p.Client,
		})
	}

	return peers, nil
}

// GetAllPeers retrieves all peers from all torrents
func (c *Client) GetAllPeers() (map[string][]models.Peer, error) {
	torrents, err := c.GetTorrents()
	if err != nil {
		return nil, err
	}

	allPeers := make(map[string][]models.Peer)
	for _, t := range torrents {
		peers, err := c.GetTorrentPeers(t.Hash)
		if err != nil {
			continue // Skip this torrent on error
		}
		allPeers[t.Hash] = peers
	}

	return allPeers, nil
}

// Name returns the server name (for logging)
func (c *Client) Name() string {
	return c.username + "@" + c.baseURL
}
