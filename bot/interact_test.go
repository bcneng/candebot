package bot

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJobSubmission(t *testing.T) {

	t.Run("The submission fields for the salary are properly parsed", func(t *testing.T) {
		max, min,_ := ValidateSubmission("http://foo.com", "12", "10")
		require.Equal(t, 12, max)
		require.Equal(t, 10, min)
	})

	t.Run("The job's link must be a valid URI", func(t *testing.T) {
		_, _, validaterrors := ValidateSubmission("http://foo.com", "12", "10")
		require.Empty(t, validaterrors["job_link"])
	})

	t.Run("The submission must reject Non-URI links", func(t *testing.T) {
		_, _, validaterrors := ValidateSubmission("hamburger", "12", "10")
		require.NotEmpty(t, validaterrors["job_link"])
	})

	t.Run("The submission must include a max salary", func(t *testing.T) {
		_, _, validaterrors := ValidateSubmission("hamburger", "", "10")
		require.NotEmpty(t, validaterrors["max_salary"])
	})

	t.Run("The submission max salary must be a positive number", func(t *testing.T) {
		_, _, validaterrors := ValidateSubmission("hamburger", "-1", "10")
		require.NotEmpty(t, validaterrors["max_salary"])
	})

	t.Run("The submission max salary must be a number", func(t *testing.T) {
		_, _, validaterrors := ValidateSubmission("hamburger", "-asdf", "10")
		require.NotEmpty(t, validaterrors["max_salary"])
	})

	t.Run("The submission min salary is optional (function returns -1)", func(t *testing.T) {
		_, minSalary, _ := ValidateSubmission("hamburger", "12", "")
		require.Equal(t,-1, minSalary)
	})

	t.Run("The submission min salary must be a positive number", func(t *testing.T) {
		_, _, validaterrors := ValidateSubmission("hamburger", "12", "-5")
		require.NotEmpty(t, validaterrors["min_salary"])
	})

	t.Run("The submission min salary must be a number", func(t *testing.T) {
		_, _, validaterrors := ValidateSubmission("hamburger", "12", "asdf")
		require.NotEmpty(t, validaterrors["min_salary"])
	})

	t.Run("The submission max salary must be greater than the min salary", func(t *testing.T) {
		_, _, validaterrors := ValidateSubmission("hamburger", "8", "10")
		require.Equal(t, "The Salary Min field should contain a lower value than the specified in Salary Max field.", validaterrors["min_salary"])
	})

	t.Run("The submission max salary must not be greater than 2.5x the min salary", func(t *testing.T) {
		_, _, validaterrors := ValidateSubmission("hamburger", "13", "5")
		require.Equal(t, "The gap between MinSalary and MaxSalary is rather large. Maybe you should post two different job offers with different responsibilities and required qualifications. Salary is a relevant field, we recommend you try to keep it meaningful to increase the chances of taking the position seriously by potential candidates.", validaterrors["max_salary"])
	})



}
