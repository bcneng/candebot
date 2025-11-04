package privacy

import "fmt"

// TrackingDetector checks for tracking parameters in URLs.
// It can be configured to only check specific channels via whitelist.
type TrackingDetector struct {
	channels map[string]struct{}
}

// NewTrackingDetector creates a new tracking detector with the given configuration.
// The getChannelID function resolves channel names to IDs.
// If config is empty, the detector will check all channels.
// Returns an error if any channel name cannot be resolved.
func NewTrackingDetector(config []TrackingDetectionConfig, getChannelID func(string) (string, error)) (*TrackingDetector, error) {
	td := &TrackingDetector{
		channels: make(map[string]struct{}),
	}

	if len(config) == 0 {
		return td, nil
	}

	for _, cfg := range config {
		channelID, err := getChannelID(cfg.ChannelName)
		if err != nil {
			return nil, fmt.Errorf("get channel ID for %q: %w", cfg.ChannelName, err)
		}
		td.channels[channelID] = struct{}{}
	}

	return td, nil
}

// TrackingDetectionConfig defines configuration for tracking detection.
type TrackingDetectionConfig struct {
	ChannelName string
}

// ShouldCheck returns true if tracking detection should be performed for the given channel.
// If no channels are configured, returns true for all channels.
func (td *TrackingDetector) ShouldCheck(channelID string) bool {
	if len(td.channels) == 0 {
		return true
	}

	_, exists := td.channels[channelID]
	return exists
}
