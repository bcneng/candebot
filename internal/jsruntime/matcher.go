package jsruntime

import (
	"path/filepath"
	"regexp"
	"strings"
)

// ChannelMatcher matches channel names against patterns.
type ChannelMatcher struct {
	patterns []channelPattern
}

type channelPattern struct {
	original string
	matcher  func(channel string) bool
}

// NewChannelMatcher creates a new channel matcher from a list of patterns.
// Patterns can be:
//   - Exact match: "general"
//   - Glob pattern: "offtopic-*", "team-*-announcements"
//   - Regex (enclosed in /): "/^hiring-/"
//   - Wildcard: "*" matches all channels
func NewChannelMatcher(patterns []string) *ChannelMatcher {
	cm := &ChannelMatcher{
		patterns: make([]channelPattern, 0, len(patterns)),
	}

	for _, p := range patterns {
		cm.patterns = append(cm.patterns, parsePattern(p))
	}

	return cm
}

// Matches returns true if the channel matches any of the patterns.
func (cm *ChannelMatcher) Matches(channel string) bool {
	if len(cm.patterns) == 0 {
		return false
	}

	// Normalize channel name (remove # prefix if present)
	channel = strings.TrimPrefix(channel, "#")
	channel = strings.ToLower(channel)

	for _, p := range cm.patterns {
		if p.matcher(channel) {
			return true
		}
	}

	return false
}

// Patterns returns the original pattern strings.
func (cm *ChannelMatcher) Patterns() []string {
	result := make([]string, len(cm.patterns))
	for i, p := range cm.patterns {
		result[i] = p.original
	}
	return result
}

func parsePattern(pattern string) channelPattern {
	cp := channelPattern{original: pattern}
	pattern = strings.TrimSpace(pattern)

	// Wildcard pattern
	if pattern == "*" {
		cp.matcher = func(channel string) bool {
			return true
		}
		return cp
	}

	// Regex pattern (enclosed in /)
	if strings.HasPrefix(pattern, "/") && strings.HasSuffix(pattern, "/") && len(pattern) > 2 {
		regexStr := pattern[1 : len(pattern)-1]
		re, err := regexp.Compile("(?i)" + regexStr) // Case insensitive
		if err != nil {
			// Invalid regex, fall back to exact match
			cp.matcher = createExactMatcher(pattern)
		} else {
			cp.matcher = func(channel string) bool {
				return re.MatchString(channel)
			}
		}
		return cp
	}

	// Glob pattern (contains * or ?)
	if strings.ContainsAny(pattern, "*?") {
		cp.matcher = createGlobMatcher(pattern)
		return cp
	}

	// Exact match
	cp.matcher = createExactMatcher(pattern)
	return cp
}

func createExactMatcher(pattern string) func(string) bool {
	pattern = strings.ToLower(strings.TrimPrefix(pattern, "#"))
	return func(channel string) bool {
		return channel == pattern
	}
}

func createGlobMatcher(pattern string) func(string) bool {
	pattern = strings.ToLower(strings.TrimPrefix(pattern, "#"))
	return func(channel string) bool {
		matched, err := filepath.Match(pattern, channel)
		if err != nil {
			return false
		}
		return matched
	}
}

// MatchChannelToHandlers finds all handlers that match a given channel.
func MatchChannelToHandlers(channel string, handlers []*Handler) []*Handler {
	var matched []*Handler

	for _, h := range handlers {
		if !h.Metadata.Enabled {
			continue
		}

		matcher := NewChannelMatcher(h.Metadata.Channels)
		if matcher.Matches(channel) {
			matched = append(matched, h)
		}
	}

	return matched
}
