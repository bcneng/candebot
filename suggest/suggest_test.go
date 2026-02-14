package suggest

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilterSuggestable(t *testing.T) {
	channels := []Channel{
		{Name: "general", ID: "C1", Category: "core", Tags: []string{"default", "read-only"}},
		{Name: "offtopic-random", ID: "C2", Category: "core", Tags: []string{"default"}},
		{Name: "golang", ID: "C3", Category: "languages", Tags: []string{}},
		{Name: "python", ID: "C4", Category: "languages", Tags: []string{}},
		{Name: "welcome", ID: "C5", Category: "core", Tags: []string{"default"}},
		{Name: "devops", ID: "C6", Category: "systems", Tags: []string{}},
	}

	result := FilterSuggestable(channels)

	assert.Len(t, result, 3)
	names := make([]string, len(result))
	for i, ch := range result {
		names[i] = ch.Name
	}
	assert.Contains(t, names, "golang")
	assert.Contains(t, names, "python")
	assert.Contains(t, names, "devops")
}

func TestFilterSuggestable_ExcludesDefaultTag(t *testing.T) {
	channels := []Channel{
		{Name: "hiring-job-board", ID: "C1", Category: "hr", Tags: []string{"default"}},
		{Name: "rust", ID: "C2", Category: "languages", Tags: []string{}},
	}

	result := FilterSuggestable(channels)

	assert.Len(t, result, 1)
	assert.Equal(t, "rust", result[0].Name)
}

func TestFilterSuggestable_EmptyInput(t *testing.T) {
	result := FilterSuggestable(nil)
	assert.Nil(t, result)
}

func TestSelectRandom(t *testing.T) {
	channels := []Channel{
		{Name: "golang", ID: "C1"},
		{Name: "python", ID: "C2"},
		{Name: "rust", ID: "C3"},
		{Name: "devops", ID: "C4"},
		{Name: "testing", ID: "C5"},
	}

	result := SelectRandom(channels, 3)

	require.Len(t, result, 3)
	// Verify all selected channels come from the original list
	nameSet := map[string]bool{"golang": true, "python": true, "rust": true, "devops": true, "testing": true}
	for _, ch := range result {
		assert.True(t, nameSet[ch.Name], "unexpected channel: %s", ch.Name)
	}
	// Verify no duplicates
	seen := map[string]bool{}
	for _, ch := range result {
		assert.False(t, seen[ch.Name], "duplicate channel: %s", ch.Name)
		seen[ch.Name] = true
	}
}

func TestSelectRandom_MoreThanAvailable(t *testing.T) {
	channels := []Channel{
		{Name: "golang", ID: "C1"},
		{Name: "python", ID: "C2"},
	}

	result := SelectRandom(channels, 5)

	assert.Len(t, result, 2)
}

func TestSelectRandom_EmptyInput(t *testing.T) {
	result := SelectRandom(nil, 3)
	assert.Nil(t, result)
}

func TestFormatMessage(t *testing.T) {
	channels := []Channel{
		{Name: "golang", ID: "C1", Description: "All things Go"},
		{Name: "python", ID: "C2", Description: "Python programming"},
	}

	msg := FormatMessage(channels)

	assert.Contains(t, msg, "Weekly Channel Suggestions")
	assert.Contains(t, msg, "<#C1>")
	assert.Contains(t, msg, "<#C2>")
	assert.Contains(t, msg, "All things Go")
	assert.Contains(t, msg, "Python programming")
}

func TestFormatMessage_EmptyInput(t *testing.T) {
	msg := FormatMessage(nil)
	assert.Empty(t, msg)
}

func TestHasTag(t *testing.T) {
	assert.True(t, hasTag([]string{"default", "read-only"}, "default"))
	assert.True(t, hasTag([]string{"default", "read-only"}, "read-only"))
	assert.False(t, hasTag([]string{"default", "read-only"}, "anonymous-enabled"))
	assert.False(t, hasTag(nil, "default"))
}
