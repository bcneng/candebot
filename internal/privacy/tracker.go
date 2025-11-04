package privacy

import (
	"net/url"
	"regexp"
	"strings"
)

// trackedParam represents a known tracking parameter.
type trackedParam struct {
	name     string
	platform string
}

// knownTrackedParams lists all tracked parameters to detect.
var knownTrackedParams = []trackedParam{
	{name: "igsh", platform: "Instagram"},
	{name: "igshid", platform: "Instagram"},
	{name: "fbclid", platform: "Facebook"},
	{name: "twclid", platform: "X (Twitter)"},
	{name: "ttclid", platform: "TikTok"},
	{name: "li_fat_id", platform: "LinkedIn"},
	{name: "si", platform: "YouTube/Spotify"},
}

var urlRegex = regexp.MustCompile(`https?://[^\s<>]+`)

// extractURLs extracts all HTTP/HTTPS URLs from text.
func extractURLs(text string) []string {
	return urlRegex.FindAllString(text, -1)
}

// detectTracking checks if URL contains any known tracking parameters.
// Returns the tracking parameter found and the platform name.
func detectTracking(rawURL string) (param string, platform string, found bool) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", "", false
	}

	query := u.Query()
	for _, tp := range knownTrackedParams {
		if query.Has(tp.name) {
			return tp.name, tp.platform, true
		}
	}

	return "", "", false
}

// stripTracking removes known tracking parameters from URL.
// Returns the cleaned URL and whether any parameters were removed.
func stripTracking(rawURL string) (cleaned string, stripped bool, err error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL, false, err
	}

	query := u.Query()
	modified := false

	for _, tp := range knownTrackedParams {
		if query.Has(tp.name) {
			query.Del(tp.name)
			modified = true
		}
	}

	if !modified {
		return rawURL, false, nil
	}

	u.RawQuery = query.Encode()
	return u.String(), true, nil
}

// SanitizedURL processes all URLs in text and returns cleaned versions with tracking removed.
type SanitizedURL struct {
	Original string
	Cleaned  string
	Param    string
	Platform string
}

// FindTrackedURLs finds all URLs with tracking parameters.
func FindTrackedURLs(text string) []SanitizedURL {
	urls := extractURLs(text)
	results := make([]SanitizedURL, 0, len(urls))

	for _, rawURL := range urls {
		param, platform, found := detectTracking(rawURL)
		if !found {
			continue
		}

		cleaned, _, err := stripTracking(rawURL)
		if err != nil {
			continue
		}

		results = append(results, SanitizedURL{
			Original: rawURL,
			Cleaned:  cleaned,
			Param:    param,
			Platform: platform,
		})
	}

	return results
}

// FormatWarningMessage creates the warning message for tracked URLs.
func FormatWarningMessage(tracked []SanitizedURL) string {
	if len(tracked) == 0 {
		return ""
	}

	var sb strings.Builder
	_, _ = sb.WriteString("⚠️ *Privacy Notice:* Your message contains tracking parameters that may expose who shared the link.\n\n")

	for i, t := range tracked {
		_, _ = sb.WriteString("*Link ")
		if len(tracked) > 1 {
			_, _ = sb.WriteString(string(rune('1' + i)))
			_, _ = sb.WriteString(":*\n")
		} else {
			_, _ = sb.WriteString("detected:*\n")
		}
		_, _ = sb.WriteString("• Platform: ")
		_, _ = sb.WriteString(t.Platform)
		_, _ = sb.WriteString("\n• Parameter: `")
		_, _ = sb.WriteString(t.Param)
		_, _ = sb.WriteString("`\n• Cleaned link: ")
		_, _ = sb.WriteString(t.Cleaned)
		_, _ = sb.WriteString("\n\n")
	}

	_, _ = sb.WriteString("Consider reposting with the cleaned link(s) above to protect your privacy.")

	return sb.String()
}
