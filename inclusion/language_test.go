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
		{name: "'retrassada' should be 'Trastorn del desenvolupament intel·lectual'", input: "L'entrega ha sigut retrassada", filtered: true},
		{name: "'noi' should be 'persona'", input: "És un noi simpàtic", filtered: true},
		{name: "'noies' should be 'gent'", input: "Hola noies", filtered: true},
		{name: "'nois' should be 'gent'", input: "Hola nois", filtered: true},
		{name: "'noia' should be 'persona'", input: "És una noia simpàtica", filtered: true},
		{name: "'noible' should be ok", input: "És una persona noible", filtered: false},
		{name: "URL containing 'noises' should be ok", input: "https://abcnews.com/US/meow-meow-pilots-scolded-after-animal-noises-heard/story?id=132076661", filtered: false},
		{name: "'noises' inside a sentence should be ok", input: "the animal noises were loud", filtered: false},
		{name: "'paranoid' should be ok", input: "don't be paranoid", filtered: false},
		{name: "'noir' should be ok", input: "I love film noir", filtered: false},
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
