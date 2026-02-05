package rules

import (
	"strconv"
	"strings"
	"time"

	"github.com/philogag/peer-banner/internal/config"
	"github.com/philogag/peer-banner/internal/models"
)

// Filter is the interface for matching filters
type Filter interface {
	// Match checks if a peer matches the filter
	Match(peer *models.Peer, torrent *models.Torrent) bool
}

// BaseFilter contains common filter configuration
type BaseFilter struct {
	Field    string
	Operator string
	Value    string
}

// parsedValue holds the parsed value with its type
type parsedValue struct {
	FloatValue    float64
	IntValue      int64
	BytesValue    int64
	DurationValue time.Duration
	StringValue   string
	ValueType     ValueType
}

// ValueType indicates the type of parsed value
type ValueType int

const (
	ValueTypeUnknown ValueType = iota
	ValueTypeFloat
	ValueTypePercent
	ValueTypeBytes
	ValueTypeDuration
	ValueTypeString
)

// ParseValue parses a value string and determines its type
func ParseValue(s string) parsedValue {
	s = strings.TrimSpace(s)
	if s == "" {
		return parsedValue{ValueType: ValueTypeUnknown}
	}

	// Check for percentage
	if strings.HasSuffix(s, "%") {
		val, err := strconv.ParseFloat(strings.TrimSuffix(s, "%"), 64)
		if err == nil {
			return parsedValue{FloatValue: val, ValueType: ValueTypePercent}
		}
	}

	// Check for duration (time)
	if strings.HasSuffix(s, "d") || strings.HasSuffix(s, "h") ||
		strings.HasSuffix(s, "m") || strings.HasSuffix(s, "s") {
		duration := ParseDuration(s)
		if duration > 0 {
			return parsedValue{DurationValue: duration, ValueType: ValueTypeDuration}
		}
	}

	// Check for bytes (TB, GB, MB, KB, B)
	if strings.HasSuffix(s, "TB") || strings.HasSuffix(s, "GB") ||
		strings.HasSuffix(s, "MB") || strings.HasSuffix(s, "KB") || strings.HasSuffix(s, "B") {
		bytes := ParseBytes(s)
		if bytes > 0 {
			return parsedValue{BytesValue: bytes, ValueType: ValueTypeBytes}
		}
	}

	// Try to parse as float
	if val, err := strconv.ParseFloat(s, 64); err == nil {
		return parsedValue{FloatValue: val, ValueType: ValueTypeFloat}
	}

	// Default to string
	return parsedValue{StringValue: s, ValueType: ValueTypeString}
}

// GenericFilter is a filter that can match any field with operator/value
type GenericFilter struct {
	Field    string
	Operator string
	Value    string
}

// NewGenericFilter creates a new GenericFilter from config
func NewGenericFilter(cfg config.FilterConfig) *GenericFilter {
	return &GenericFilter{
		Field:    cfg.Field,
		Operator: cfg.Operator,
		Value:    cfg.Value,
	}
}

// Match checks if the peer matches the filter
func (f *GenericFilter) Match(peer *models.Peer, torrent *models.Torrent) bool {
	parsedVal := ParseValue(f.Value)
	return matchField(peer, torrent, f.Field, f.Operator, parsedVal, f.Value)
}

// matchField matches a specific field with operator against parsed value
func matchField(peer *models.Peer, torrent *models.Torrent, field, operator string, parsedVal parsedValue, originalValue string) bool {
	switch field {
	case "progress":
		peerValue := peer.Progress * 100
		return compareFloat(peerValue, operator, parsedVal.FloatValue)
	case "uploaded":
		return matchBytes(peer.Uploaded, torrent, operator, parsedVal, true)
	case "downloaded":
		return matchBytes(peer.Downloaded, torrent, operator, parsedVal, false)
	case "relevance":
		return compareFloat(peer.Relevance, operator, parsedVal.FloatValue)
	case "active_time":
		peerDuration := time.Duration(peer.ActiveTime) * time.Second
		return compareDuration(peerDuration, operator, parsedVal.DurationValue)
	case "flag":
		return matchString(strings.ToLower(peer.Flags), operator, strings.ToLower(originalValue))
	default:
		return false
	}
}

// compareFloat compares a float value with operator
func compareFloat(peerValue float64, operator string, filterValue float64) bool {
	switch operator {
	case "<":
		return peerValue < filterValue
	case ">":
		return peerValue > filterValue
	case "<=":
		return peerValue <= filterValue
	case ">=":
		return peerValue >= filterValue
	default:
		return false
	}
}

// compareDuration compares a duration value with operator
func compareDuration(peerValue time.Duration, operator string, filterValue time.Duration) bool {
	switch operator {
	case "<":
		return peerValue < filterValue
	case ">":
		return peerValue > filterValue
	case "<=":
		return peerValue <= filterValue
	case ">=":
		return peerValue >= filterValue
	default:
		return false
	}
}

// matchBytes matches byte values, supporting percent mode
func matchBytes(peerBytes int64, torrent *models.Torrent, operator string, parsedVal parsedValue, isUploaded bool) bool {
	if parsedVal.ValueType == ValueTypePercent {
		// Calculate percent based on torrent size
		if torrent == nil || torrent.Size == 0 {
			return false
		}
		percent := float64(peerBytes) / float64(torrent.Size) * 100
		return compareFloat(percent, operator, parsedVal.FloatValue)
	}
	// Absolute bytes comparison
	return compareInt64(peerBytes, operator, parsedVal.BytesValue)
}

// compareInt64 compares an int64 value with operator
func compareInt64(peerValue int64, operator string, filterValue int64) bool {
	switch operator {
	case "<":
		return peerValue < filterValue
	case ">":
		return peerValue > filterValue
	case "<=":
		return peerValue <= filterValue
	case ">=":
		return peerValue >= filterValue
	default:
		return false
	}
}

// matchString matches string values with include/exclude operators
func matchString(peerValue string, operator string, filterValue string) bool {
	switch operator {
	case "include":
		return strings.Contains(peerValue, filterValue)
	case "exclude":
		return !strings.Contains(peerValue, filterValue)
	default:
		return false
	}
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

	val, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0
	}
	return int64(val * float64(multiplier))
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

	val, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0
	}
	return time.Duration(val * float64(multiplier))
}
