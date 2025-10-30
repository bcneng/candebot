package bot

import (
	"fmt"
	"sync"
	"time"
)

// RateLimiter enforces per-user, per-channel message rate limits using
// a sliding window algorithm. It is safe for concurrent use.
type RateLimiter struct {
	mu      sync.RWMutex
	limits  map[string]*ChannelLimit
	storage map[string]*UserRateState
}

// ChannelLimit defines the rate limiting configuration for a specific channel.
type ChannelLimit struct {
	RateLimitSeconds int
	MaxMessages      int
	ApplyToStaff     bool
}

// UserRateState tracks a user's message history in a specific channel.
type UserRateState struct {
	Messages  []time.Time
	NextReset time.Time
}

// NewRateLimiter creates a new rate limiter with the given configuration.
// The getChannelID function resolves channel names to IDs.
// Returns an error if any channel name cannot be resolved.
func NewRateLimiter(config []RateLimitConfig, getChannelID func(string) (string, error)) (*RateLimiter, error) {
	rl := &RateLimiter{
		limits:  make(map[string]*ChannelLimit),
		storage: make(map[string]*UserRateState),
	}

	for _, cfg := range config {
		channelID, err := getChannelID(cfg.ChannelName)
		if err != nil {
			return nil, fmt.Errorf("get channel ID for %q: %w", cfg.ChannelName, err)
		}
		rl.limits[channelID] = &ChannelLimit{
			RateLimitSeconds: cfg.RateLimitSeconds,
			MaxMessages:      cfg.MaxMessages,
			ApplyToStaff:     cfg.ApplyToStaff,
		}
	}

	return rl, nil
}

// ShouldCheckStaff returns true if rate limits should apply to staff members
// in the given channel.
func (rl *RateLimiter) ShouldCheckStaff(channelID string) bool {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	limit, exists := rl.limits[channelID]
	if !exists {
		return false
	}

	return limit.ApplyToStaff
}

// CheckLimit checks if a user is allowed to post a message in a channel.
// Returns (true, zero time) if allowed, or (false, nextAllowedTime) if rate limited.
func (rl *RateLimiter) CheckLimit(channelID, userID string) (allowed bool, nextAllowedTime time.Time) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limit, exists := rl.limits[channelID]
	if !exists {
		return true, time.Time{}
	}

	key := channelID + ":" + userID
	state := rl.storage[key]
	now := time.Now()

	if state == nil {
		state = &UserRateState{
			Messages:  []time.Time{now},
			NextReset: now.Add(time.Duration(limit.RateLimitSeconds) * time.Second),
		}
		rl.storage[key] = state
		return true, time.Time{}
	}

	if now.After(state.NextReset) {
		state.Messages = []time.Time{now}
		state.NextReset = now.Add(time.Duration(limit.RateLimitSeconds) * time.Second)
		return true, time.Time{}
	}

	cutoff := now.Add(-time.Duration(limit.RateLimitSeconds) * time.Second)
	validMessages := make([]time.Time, 0, len(state.Messages))
	for _, msgTime := range state.Messages {
		if msgTime.After(cutoff) {
			validMessages = append(validMessages, msgTime)
		}
	}

	if len(validMessages) >= limit.MaxMessages {
		oldestMessage := validMessages[0]
		nextAllowed := oldestMessage.Add(time.Duration(limit.RateLimitSeconds) * time.Second)
		return false, nextAllowed
	}

	validMessages = append(validMessages, now)
	state.Messages = validMessages

	return true, time.Time{}
}
