package bot

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func requireHasError(t *testing.T, errorMessage string) {
	require.NotEmpty(t, errorMessage)

	if len(errorMessage) > 150 {
		t.Errorf("error message is too long: %s", errorMessage)
	}

}

func TestJobSubmission(t *testing.T) {

	t.Run("fields for the salary are properly parsed", func(t *testing.T) {
		max, min, _ := validateSubmission("http://foo.com", "12", "10")
		require.Equal(t, 12, max)
		require.Equal(t, 10, min)
	})

	t.Run("the job link must be a valid URI", func(t *testing.T) {
		_, _, validaterrors := validateSubmission("http://foo.com", "12", "10")
		require.Empty(t, validaterrors["job_link"])
	})

	t.Run("min salary is optional (and function returns -1 when left blank)", func(t *testing.T) {
		_, minSalary, _ := validateSubmission("hamburger", "12", "")
		require.Equal(t, -1, minSalary)
	})

	// FIXME: The following tests may be rewritten as a table-driven test
	// https://github.com/bcneng/candebot/pull/72#pullrequestreview-773870224
	t.Run("must reject non-URI links", func(t *testing.T) {
		_, _, validaterrors := validateSubmission("hamburger", "12", "10")
		requireHasError(t, validaterrors["job_link"])
	})

	t.Run("must include a max salary", func(t *testing.T) {
		_, _, validaterrors := validateSubmission("hamburger", "", "10")
		requireHasError(t, validaterrors["max_salary"])
	})

	t.Run("max salary must be a positive number", func(t *testing.T) {
		_, _, validaterrors := validateSubmission("hamburger", "-1", "10")
		requireHasError(t, validaterrors["max_salary"])
	})

	t.Run("max salary must be a number", func(t *testing.T) {
		_, _, validaterrors := validateSubmission("hamburger", "-asdf", "10")
		requireHasError(t, validaterrors["max_salary"])
	})

	t.Run("min salary must be a number", func(t *testing.T) {
		_, _, validaterrors := validateSubmission("hamburger", "12", "asdf")
		requireHasError(t, validaterrors["min_salary"])
	})

	t.Run("min salary must be a positive number", func(t *testing.T) {
		_, _, validaterrors := validateSubmission("hamburger", "12", "-5")
		requireHasError(t, validaterrors["min_salary"])
	})

	t.Run("max salary must be greater than the min salary", func(t *testing.T) {
		_, _, validaterrors := validateSubmission("hamburger", "8", "10")
		requireHasError(t, validaterrors["min_salary"])
	})

	t.Run("max salary must not be greater than 2.5x the min salary", func(t *testing.T) {
		_, _, validaterrors := validateSubmission("hamburger", "13", "5")
		requireHasError(t, validaterrors["max_salary"])
	})

}

func TestMessageSanitizing(t *testing.T) {
	t.Run("urls cannot contain double scaped backlashes", func(t *testing.T) {
		url := "https:\\/\\/bcneng.slack.com\\/archives"
		expected := "https://bcneng.slack.com/archives"
		actual := sanitizeReportState(url)
		require.Equal(t, expected, actual)
	})
}
