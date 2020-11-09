package bot

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsValidJobOffer(t *testing.T) {
	tests := []struct {
		name  string
		text  string
		valid bool
	}{
		{
			name:  "Offer following the format is valid",
			text:  ":computer: Full Stack Engineer @ BcnEng - :moneybag: 70 - 90k  - :round_pushpin: Barcelona - :link: www.bcneng.com/netiquette - :raised_hands: More info DM @smoya",
			valid: true,
		},
		{
			name:  "Offer without min range is valid",
			text:  ":computer: Full Stack Engineer @ BcnEng - :moneybag:  - 90k  - :round_pushpin: Barcelona - :link: https://www.bcneng.com/netiquette - :raised_hands: More info DM @smoya",
			valid: true,
		},
		{
			name:  "Offer with a too long min range is invalid",
			text:  ":computer: Full Stack Engineer @ BcnEng - :moneybag: 70 but depending on blabla - 90k  - :round_pushpin: Barcelona - :link: https://www.bcneng.com/netiquette - :raised_hands: More info DM @smoya",
			valid: false,
		},
		{
			name:  "Offer with a too long max range is invalid",
			text:  ":computer: Full Stack Engineer @ BcnEng - :moneybag: 70 - 90k but depending on blabla  - :round_pushpin: Barcelona - :link: https://www.bcneng.com/netiquette - :raised_hands: More info DM @smoya",
			valid: false,
		},
		{
			name:  "Offer without max range is invalid",
			text:  ":computer: Full Stack Engineer @ BcnEng - :moneybag:  -  - :round_pushpin: Barcelona - :link: https://www.bcneng.com/netiquette - :raised_hands: More info DM @smoya",
			valid: false,
		},
		{
			name:  "Offer with an invalid link",
			text:  ":computer: Full Stack Engineer @ BcnEng - :moneybag: 70 - 90k  - :round_pushpin: Barcelona - :link: wrong-link - :raised_hands: More info DM @smoya",
			valid: false,
		},
		{
			name:  "Offer with a missing link is invalid",
			text:  ":computer: Full Stack Engineer @ BcnEng - :moneybag: 70 - 90k  - :round_pushpin: Barcelona - :link:  - :raised_hands: More info DM @smoya",
			valid: false,
		},
		{
			name:  "Offer with missing role is invalid",
			text:  ":computer:  @ BcnEng - :moneybag: 70 - 90k  - :round_pushpin: Barcelona - :link: www.bcneng.com/netiquette - :raised_hands: More info DM @smoya",
			valid: false,
		},
		{
			name:  "Offer with missing company is invalid",
			text:  ":computer: Full Stack Engineer @  - :moneybag: 70 - 90k  - :round_pushpin: Barcelona - :link: www.bcneng.com/netiquette - :raised_hands: More info DM @smoya",
			valid: false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.valid, isValidJobOffer(test.text), test.text)
		})
	}
}
