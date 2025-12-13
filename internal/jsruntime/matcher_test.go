package jsruntime

import (
	"testing"
)

func TestChannelMatcher_ExactMatch(t *testing.T) {
	matcher := NewChannelMatcher([]string{"general", "random"})

	tests := []struct {
		channel string
		want    bool
	}{
		{"general", true},
		{"random", true},
		{"other", false},
		{"general-2", false},
		{"#general", true}, // Should strip # prefix
		{"GENERAL", true},  // Case insensitive
	}

	for _, tt := range tests {
		t.Run(tt.channel, func(t *testing.T) {
			if got := matcher.Matches(tt.channel); got != tt.want {
				t.Errorf("Matches(%q) = %v, want %v", tt.channel, got, tt.want)
			}
		})
	}
}

func TestChannelMatcher_GlobPattern(t *testing.T) {
	matcher := NewChannelMatcher([]string{"offtopic-*", "team-*-dev"})

	tests := []struct {
		channel string
		want    bool
	}{
		{"offtopic-general", true},
		{"offtopic-random", true},
		{"offtopic", false},
		{"team-backend-dev", true},
		{"team-frontend-dev", true},
		{"team-dev", false},
		{"team-backend-prod", false},
	}

	for _, tt := range tests {
		t.Run(tt.channel, func(t *testing.T) {
			if got := matcher.Matches(tt.channel); got != tt.want {
				t.Errorf("Matches(%q) = %v, want %v", tt.channel, got, tt.want)
			}
		})
	}
}

func TestChannelMatcher_RegexPattern(t *testing.T) {
	matcher := NewChannelMatcher([]string{"/^hiring-/", "/.*-announcements$/"})

	tests := []struct {
		channel string
		want    bool
	}{
		{"hiring-frontend", true},
		{"hiring-backend", true},
		{"not-hiring", false},
		{"team-announcements", true},
		{"company-announcements", true},
		{"announcements-daily", false},
	}

	for _, tt := range tests {
		t.Run(tt.channel, func(t *testing.T) {
			if got := matcher.Matches(tt.channel); got != tt.want {
				t.Errorf("Matches(%q) = %v, want %v", tt.channel, got, tt.want)
			}
		})
	}
}

func TestChannelMatcher_Wildcard(t *testing.T) {
	matcher := NewChannelMatcher([]string{"*"})

	tests := []struct {
		channel string
		want    bool
	}{
		{"general", true},
		{"random", true},
		{"any-channel", true},
		{"", true},
	}

	for _, tt := range tests {
		t.Run(tt.channel, func(t *testing.T) {
			if got := matcher.Matches(tt.channel); got != tt.want {
				t.Errorf("Matches(%q) = %v, want %v", tt.channel, got, tt.want)
			}
		})
	}
}

func TestChannelMatcher_EmptyPatterns(t *testing.T) {
	matcher := NewChannelMatcher([]string{})

	if matcher.Matches("general") {
		t.Error("Empty patterns should not match anything")
	}
}

func TestChannelMatcher_MixedPatterns(t *testing.T) {
	matcher := NewChannelMatcher([]string{
		"general",           // Exact match
		"offtopic-*",        // Glob
		"/^hiring-/",        // Regex
	})

	tests := []struct {
		channel string
		want    bool
	}{
		{"general", true},
		{"offtopic-games", true},
		{"hiring-backend", true},
		{"random", false},
	}

	for _, tt := range tests {
		t.Run(tt.channel, func(t *testing.T) {
			if got := matcher.Matches(tt.channel); got != tt.want {
				t.Errorf("Matches(%q) = %v, want %v", tt.channel, got, tt.want)
			}
		})
	}
}

func TestMatchChannelToHandlers(t *testing.T) {
	handlers := []*Handler{
		{
			Metadata: HandlerMetadata{
				Name:     "handler1",
				Channels: []string{"general"},
				Enabled:  true,
			},
		},
		{
			Metadata: HandlerMetadata{
				Name:     "handler2",
				Channels: []string{"offtopic-*"},
				Enabled:  true,
			},
		},
		{
			Metadata: HandlerMetadata{
				Name:     "handler3",
				Channels: []string{"*"},
				Enabled:  false, // Disabled
			},
		},
	}

	// Test general channel
	matched := MatchChannelToHandlers("general", handlers)
	if len(matched) != 1 || matched[0].Metadata.Name != "handler1" {
		t.Errorf("Expected handler1 for 'general', got %v", matched)
	}

	// Test offtopic channel
	matched = MatchChannelToHandlers("offtopic-games", handlers)
	if len(matched) != 1 || matched[0].Metadata.Name != "handler2" {
		t.Errorf("Expected handler2 for 'offtopic-games', got %v", matched)
	}

	// Test random channel (no matches, handler3 is disabled)
	matched = MatchChannelToHandlers("random", handlers)
	if len(matched) != 0 {
		t.Errorf("Expected no matches for 'random', got %v", matched)
	}
}
