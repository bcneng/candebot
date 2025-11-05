package privacy

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewTrackingDetector(t *testing.T) {
	t.Run("empty config allows all channels", func(t *testing.T) {
		td, err := NewTrackingDetector([]TrackingDetectionConfig{}, func(_ string) (string, error) {
			return "", nil
		})
		require.NoError(t, err)
		require.NotNil(t, td)
		require.True(t, td.ShouldCheck("C123"))
		require.True(t, td.ShouldCheck("C456"))
	})

	t.Run("with channel config", func(t *testing.T) {
		getChannelID := func(name string) (string, error) {
			if name == "general" {
				return "C123", nil
			}
			if name == "random" {
				return "C456", nil
			}
			return "", errors.New("channel not found")
		}

		config := []TrackingDetectionConfig{
			{ChannelName: "general"},
			{ChannelName: "random"},
		}

		td, err := NewTrackingDetector(config, getChannelID)
		require.NoError(t, err)
		require.NotNil(t, td)
		require.True(t, td.ShouldCheck("C123"))
		require.True(t, td.ShouldCheck("C456"))
		require.False(t, td.ShouldCheck("C789"))
	})

	t.Run("channel resolution error", func(t *testing.T) {
		getChannelID := func(_ string) (string, error) {
			return "", errors.New("channel not found")
		}

		config := []TrackingDetectionConfig{
			{ChannelName: "nonexistent"},
		}

		td, err := NewTrackingDetector(config, getChannelID)
		require.Error(t, err)
		require.Nil(t, td)
		require.Contains(t, err.Error(), "nonexistent")
	})
}

func TestTrackingDetectorShouldCheck(t *testing.T) {
	tests := []struct {
		name      string
		config    []TrackingDetectionConfig
		channelID string
		want      bool
	}{
		{
			name:      "empty config allows all",
			config:    []TrackingDetectionConfig{},
			channelID: "C123",
			want:      true,
		},
		{
			name: "channel in whitelist",
			config: []TrackingDetectionConfig{
				{ChannelName: "general"},
			},
			channelID: "C123",
			want:      true,
		},
		{
			name: "channel not in whitelist",
			config: []TrackingDetectionConfig{
				{ChannelName: "general"},
			},
			channelID: "C456",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getChannelID := func(name string) (string, error) {
				if name == "general" {
					return "C123", nil
				}
				return "", errors.New("not found")
			}

			td, err := NewTrackingDetector(tt.config, getChannelID)
			require.NoError(t, err)
			require.Equal(t, tt.want, td.ShouldCheck(tt.channelID))
		})
	}
}
