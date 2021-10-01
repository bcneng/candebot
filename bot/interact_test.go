package bot

import (
	"testing"

	"github.com/stretchr/testify/require"
)

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

	t.Run("must reject non-URI links", func(t *testing.T) {
		_, _, validaterrors := validateSubmission("hamburger", "12", "10")
		require.NotEmpty(t, validaterrors["job_link"])
	})

	t.Run("must include a max salary", func(t *testing.T) {
		_, _, validaterrors := validateSubmission("hamburger", "", "10")
		require.NotEmpty(t, validaterrors["max_salary"])
	})

	t.Run("max salary must be a positive number", func(t *testing.T) {
		_, _, validaterrors := validateSubmission("hamburger", "-1", "10")
		require.NotEmpty(t, validaterrors["max_salary"])
	})

	t.Run("max salary must be a number", func(t *testing.T) {
		_, _, validaterrors := validateSubmission("hamburger", "-asdf", "10")
		require.NotEmpty(t, validaterrors["max_salary"])
	})

	t.Run("min salary is optional (and function returns -1 when left blank)", func(t *testing.T) {
		_, minSalary, _ := validateSubmission("hamburger", "12", "")
		require.Equal(t, -1, minSalary)
	})

	t.Run("min salary must be a number", func(t *testing.T) {
		_, _, validaterrors := validateSubmission("hamburger", "12", "asdf")
		require.NotEmpty(t, validaterrors["min_salary"])
	})

	t.Run("min salary must be a positive number", func(t *testing.T) {
		_, _, validaterrors := validateSubmission("hamburger", "12", "-5")
		require.NotEmpty(t, validaterrors["min_salary"])
	})

	t.Run("max salary must be greater than the min salary", func(t *testing.T) {
		_, _, validaterrors := validateSubmission("hamburger", "8", "10")
		require.Equal(t, "The Salary Min field should contain a lower value than the specified in Salary Max field.", validaterrors["min_salary"])
	})

	t.Run("max salary must not be greater than 2.5x the min salary", func(t *testing.T) {
		_, _, validaterrors := validateSubmission("hamburger", "13", "5")
		require.Equal(t, "The range MinSalary-MaxSalary is too large. Salary is a relevant field, keep it meaningful to increase offer impact.", validaterrors["max_salary"])
	})

	t.Run("error messages should be shorter than 150 chars", func(t *testing.T) {
		_, _, validaterrors := validateSubmission("hamburger", "13", "5")
		if len([]rune(validaterrors["max_salary"])) > 150 {
			t.Errorf("error message is too long: %s", validaterrors["max_salary"])
		}

	})

}
