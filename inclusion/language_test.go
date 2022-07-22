package inclusion

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFilter(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		filtered bool
	}{
		{name: "LGTB should be LGTB+", input: "I do really support LGTB groups", filtered: true},
		{name: "abcdlgtbe is ok", input: "abcdlgtbe", filtered: false},
		{name: "LGTB+ should be ok", input: "I do really support LGTB+ groups", filtered: false},
		{name: "'minusvalida' should be 'persona discapacitada'", input: "Mi vecina es minusvalida", filtered: true},
		{name: "'persona con discapacidad' is right ", input: "Mi vecino es una persona con discapacidad", filtered: false},
		{name: "bcneng sucks is rude", input: "bcneng sucks", filtered: true},
		{name: "bcneng is awesome is nice", input: "bcneng is awesome", filtered: false},
		{name: "'buena localización' is right even though contains the word 'loca'", input: "buena localización", filtered: false},
		{name: "'ladies' is not usually used in the right context", input: "hi ladies!", filtered: true},
	}

	extraFilters := []InclusiveFilter{
		{
			Filter: "bcneng sucks",
			Reply:  "We do not expect you to love us unconditionally, however we do really love you all!",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			output := Filter(test.input, extraFilters...)
			if test.filtered {
				require.NotEmpty(t, output, output)
				require.NotEmpty(t, output.Filter)
				require.NotEmpty(t, output.Reply)
			} else {
				require.Nil(t, output, output)
			}
		})
	}
}
