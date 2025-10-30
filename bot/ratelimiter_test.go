package bot

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRateLimiter_CheckLimit(t *testing.T) {
	config := []RateLimitConfig{
		{
			ChannelName:      "test-channel",
			RateLimitSeconds: 60,
			MaxMessages:      3,
		},
	}

	getChannelID := func(_ string) (string, error) {
		return "C123456", nil
	}

	rl, err := NewRateLimiter(config, getChannelID)
	require.NoError(t, err, "failed to create rate limiter")

	channelID := "C123456"
	userID := "U123456"

	t.Run("allows messages under limit", func(t *testing.T) {
		for i := 0; i < 3; i++ {
			allowed, _ := rl.CheckLimit(channelID, userID)
			require.True(t, allowed, "message %d should be allowed", i+1)
		}
	})

	t.Run("blocks messages over limit", func(t *testing.T) {
		allowed, nextAllowed := rl.CheckLimit(channelID, userID)
		require.False(t, allowed, "message should be blocked")
		require.False(t, nextAllowed.IsZero(), "nextAllowed time should be set")
	})

	t.Run("allows messages after time window", func(t *testing.T) {
		config2 := []RateLimitConfig{
			{
				ChannelName:      "test-channel2",
				RateLimitSeconds: 1,
				MaxMessages:      2,
			},
		}

		getChannelID2 := func(_ string) (string, error) {
			return "C789012", nil
		}

		rl2, err := NewRateLimiter(config2, getChannelID2)
		require.NoError(t, err, "failed to create rate limiter")

		channelID2 := "C789012"
		userID2 := "U789012"

		allowed, _ := rl2.CheckLimit(channelID2, userID2)
		require.True(t, allowed, "first message should be allowed")

		allowed, _ = rl2.CheckLimit(channelID2, userID2)
		require.True(t, allowed, "second message should be allowed")

		allowed, _ = rl2.CheckLimit(channelID2, userID2)
		require.False(t, allowed, "third message should be blocked")

		time.Sleep(1100 * time.Millisecond)

		allowed, _ = rl2.CheckLimit(channelID2, userID2)
		require.True(t, allowed, "message should be allowed after window expires")
	})

	t.Run("allows messages in non-rate-limited channels", func(t *testing.T) {
		nonLimitedChannel := "C999999"
		allowed, nextAllowed := rl.CheckLimit(nonLimitedChannel, userID)
		require.True(t, allowed, "message in non-rate-limited channel should be allowed")
		require.True(t, nextAllowed.IsZero(), "nextAllowed should not be set for non-rate-limited channels")
	})

	t.Run("tracks users independently", func(t *testing.T) {
		config3 := []RateLimitConfig{
			{
				ChannelName:      "test-channel3",
				RateLimitSeconds: 60,
				MaxMessages:      2,
			},
		}

		getChannelID3 := func(_ string) (string, error) {
			return "C111222", nil
		}

		rl3, err := NewRateLimiter(config3, getChannelID3)
		require.NoError(t, err, "failed to create rate limiter")

		channelID3 := "C111222"
		user1 := "U111111"
		user2 := "U222222"

		allowed, _ := rl3.CheckLimit(channelID3, user1)
		require.True(t, allowed, "user1 first message should be allowed")

		allowed, _ = rl3.CheckLimit(channelID3, user1)
		require.True(t, allowed, "user1 second message should be allowed")

		allowed, _ = rl3.CheckLimit(channelID3, user2)
		require.True(t, allowed, "user2 first message should be allowed")

		allowed, _ = rl3.CheckLimit(channelID3, user1)
		require.False(t, allowed, "user1 third message should be blocked")

		allowed, _ = rl3.CheckLimit(channelID3, user2)
		require.True(t, allowed, "user2 second message should be allowed")
	})

	t.Run("apply_to_staff flag controls staff exemption", func(t *testing.T) {
		config4 := []RateLimitConfig{
			{
				ChannelName:      "test-channel4",
				RateLimitSeconds: 60,
				MaxMessages:      1,
				ApplyToStaff:     true,
			},
		}

		getChannelID4 := func(_ string) (string, error) {
			return "C444444", nil
		}

		rl4, err := NewRateLimiter(config4, getChannelID4)
		require.NoError(t, err, "failed to create rate limiter")

		channelID4 := "C444444"

		shouldCheck := rl4.ShouldCheckStaff(channelID4)
		require.True(t, shouldCheck, "should check staff when apply_to_staff is true")

		allowed, _ := rl4.CheckLimit(channelID4, "U999999")
		require.True(t, allowed, "first message should be allowed")

		allowed, _ = rl4.CheckLimit(channelID4, "U999999")
		require.False(t, allowed, "second message should be blocked even for staff")
	})

	t.Run("apply_to_staff false allows staff exemption", func(t *testing.T) {
		config5 := []RateLimitConfig{
			{
				ChannelName:      "test-channel5",
				RateLimitSeconds: 60,
				MaxMessages:      1,
				ApplyToStaff:     false,
			},
		}

		getChannelID5 := func(_ string) (string, error) {
			return "C555555", nil
		}

		rl5, err := NewRateLimiter(config5, getChannelID5)
		require.NoError(t, err, "failed to create rate limiter")

		channelID5 := "C555555"

		shouldCheck := rl5.ShouldCheckStaff(channelID5)
		require.False(t, shouldCheck, "should not check staff when apply_to_staff is false")
	})
}
