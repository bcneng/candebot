package privacy

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDetectTracking(t *testing.T) {
	tests := []struct {
		name         string
		url          string
		wantParam    string
		wantPlatform string
		wantFound    bool
	}{
		{
			name:         "Instagram igsh parameter",
			url:          "https://www.instagram.com/p/abc123/?igsh=xyz789",
			wantParam:    "igsh",
			wantPlatform: "Instagram",
			wantFound:    true,
		},
		{
			name:         "Instagram igshid parameter",
			url:          "https://www.instagram.com/reel/xyz/?igshid=abc123",
			wantParam:    "igshid",
			wantPlatform: "Instagram",
			wantFound:    true,
		},
		{
			name:         "Facebook fbclid parameter",
			url:          "https://www.facebook.com/post/123?fbclid=IwAR123456",
			wantParam:    "fbclid",
			wantPlatform: "Facebook",
			wantFound:    true,
		},
		{
			name:         "X (Twitter) twclid parameter",
			url:          "https://twitter.com/user/status/123?twclid=abc123",
			wantParam:    "twclid",
			wantPlatform: "X (Twitter)",
			wantFound:    true,
		},
		{
			name:         "TikTok ttclid parameter",
			url:          "https://www.tiktok.com/@user/video/123?ttclid=xyz789",
			wantParam:    "ttclid",
			wantPlatform: "TikTok",
			wantFound:    true,
		},
		{
			name:         "LinkedIn li_fat_id parameter",
			url:          "https://www.linkedin.com/posts/activity-123?li_fat_id=abc-def-123",
			wantParam:    "li_fat_id",
			wantPlatform: "LinkedIn",
			wantFound:    true,
		},
		{
			name:         "YouTube si parameter",
			url:          "https://www.youtube.com/watch?v=abc123&si=xyz789",
			wantParam:    "si",
			wantPlatform: "YouTube/Spotify",
			wantFound:    true,
		},
		{
			name:         "Spotify si parameter",
			url:          "https://open.spotify.com/track/abc?si=xyz123",
			wantParam:    "si",
			wantPlatform: "YouTube/Spotify",
			wantFound:    true,
		},
		{
			name:      "clean URL without tracking",
			url:       "https://example.com/article?page=1",
			wantFound: false,
		},
		{
			name:      "URL with no query params",
			url:       "https://example.com/page",
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			param, platform, found := detectTracking(tt.url)
			require.Equal(t, tt.wantFound, found, "found mismatch")
			if found {
				require.Equal(t, tt.wantParam, param, "param mismatch")
				require.Equal(t, tt.wantPlatform, platform, "platform mismatch")
			}
		})
	}
}

func TestStripTracking(t *testing.T) {
	tests := []struct {
		name         string
		url          string
		wantCleaned  string
		wantStripped bool
		wantErr      bool
	}{
		{
			name:         "strip Instagram igsh",
			url:          "https://www.instagram.com/p/abc/?igsh=xyz&other=value",
			wantCleaned:  "https://www.instagram.com/p/abc/?other=value",
			wantStripped: true,
		},
		{
			name:         "strip Facebook fbclid",
			url:          "https://example.com/article?fbclid=abc123&ref=home",
			wantCleaned:  "https://example.com/article?ref=home",
			wantStripped: true,
		},
		{
			name:         "strip YouTube si",
			url:          "https://www.youtube.com/watch?v=abc123&si=xyz789",
			wantCleaned:  "https://www.youtube.com/watch?v=abc123",
			wantStripped: true,
		},
		{
			name:         "strip only tracking param from multiple params",
			url:          "https://example.com/page?a=1&fbclid=xyz&b=2",
			wantCleaned:  "https://example.com/page?a=1&b=2",
			wantStripped: true,
		},
		{
			name:         "preserve anchor",
			url:          "https://example.com/page?igsh=abc#section",
			wantCleaned:  "https://example.com/page#section",
			wantStripped: true,
		},
		{
			name:         "URL without tracking params unchanged",
			url:          "https://example.com/page?normal=param",
			wantCleaned:  "https://example.com/page?normal=param",
			wantStripped: false,
		},
		{
			name:         "multiple tracking params stripped",
			url:          "https://example.com/page?igsh=a&fbclid=b&normal=c",
			wantCleaned:  "https://example.com/page?normal=c",
			wantStripped: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleaned, stripped, err := stripTracking(tt.url)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantCleaned, cleaned, "cleaned URL mismatch")
			require.Equal(t, tt.wantStripped, stripped, "stripped flag mismatch")
		})
	}
}

func TestExtractURLs(t *testing.T) {
	tests := []struct {
		name string
		text string
		want []string
	}{
		{
			name: "single URL",
			text: "Check this out https://example.com/article",
			want: []string{"https://example.com/article"},
		},
		{
			name: "multiple URLs",
			text: "Visit https://site1.com and http://site2.com for more",
			want: []string{"https://site1.com", "http://site2.com"},
		},
		{
			name: "URL with query params",
			text: "Link: https://example.com/page?foo=bar&baz=qux",
			want: []string{"https://example.com/page?foo=bar&baz=qux"},
		},
		{
			name: "no URLs",
			text: "Just plain text without links",
			want: nil,
		},
		{
			name: "URL at start",
			text: "https://example.com is a site",
			want: []string{"https://example.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractURLs(tt.text)
			require.Equal(t, tt.want, got, "extracted URLs mismatch")
		})
	}
}

func TestFindTrackedURLs(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		wantCount int
	}{
		{
			name:      "single tracked URL",
			text:      "Check this https://instagram.com/p/abc?igsh=xyz",
			wantCount: 1,
		},
		{
			name:      "multiple tracked URLs",
			text:      "See https://facebook.com?fbclid=a and https://youtube.com/watch?v=b&si=c",
			wantCount: 2,
		},
		{
			name:      "mixed tracked and clean URLs",
			text:      "Visit https://example.com and https://facebook.com?fbclid=abc",
			wantCount: 1,
		},
		{
			name:      "no tracked URLs",
			text:      "Clean links: https://example.com and https://google.com",
			wantCount: 0,
		},
		{
			name:      "no URLs at all",
			text:      "Just text without any links",
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FindTrackedURLs(tt.text)
			require.Len(t, got, tt.wantCount, "tracked URL count mismatch")
			for _, s := range got {
				require.NotEmpty(t, s.Original, "Original URL should not be empty")
				require.NotEmpty(t, s.Cleaned, "Cleaned URL should not be empty")
				require.NotEmpty(t, s.Param, "Param should not be empty")
				require.NotEmpty(t, s.Platform, "Platform should not be empty")
			}
		})
	}
}

func TestFormatWarningMessage(t *testing.T) {
	tests := []struct {
		name    string
		tracked []SanitizedURL
		want    string
	}{
		{
			name:    "empty slice",
			tracked: []SanitizedURL{},
			want:    "",
		},
		{
			name: "single tracked URL",
			tracked: []SanitizedURL{
				{
					Original: "https://instagram.com/p/abc?igsh=xyz",
					Cleaned:  "https://instagram.com/p/abc",
					Param:    "igsh",
					Platform: "Instagram",
				},
			},
			want: "⚠️ *Privacy Notice:* Your message contains tracking parameters that may expose who shared the link.\n\n*Link detected:*\n• Platform: Instagram\n• Parameter: `igsh`\n• Cleaned link: https://instagram.com/p/abc\n\nConsider reposting with the cleaned link(s) above to protect your privacy.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatWarningMessage(tt.tracked)
			require.Equal(t, tt.want, got, "warning message mismatch")
		})
	}
}
