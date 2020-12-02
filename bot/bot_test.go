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
			text:  ":computer: Senior Go Engineer @ BcnEng - :moneybag: 55k - 70k - :round_pushpin: Barcelona - :link: `<https://bcneng.org/jobs/senior-go-developer-123|ttps://bcneng.org/jobs/senior-go-developer-123>` - :raised_hands: More info DM <@U2WPLA0KA>",
			valid: true,
		},
		{
			name:  "Offer without min range is valid",
			text:  ":computer: Senior Go Engineer @ BcnEng - :moneybag: - 70k - :round_pushpin: Barcelona - :link: `<https://bcneng.org/jobs/senior-go-developer-123|ttps://bcneng.org/jobs/senior-go-developer-123>` - :raised_hands: More info DM <@U2WPLA0KA>",
			valid: true,
		},
		{
			name:  "Offer with a too long min range is invalid",
			text:  ":computer: Senior Go Engineer @ BcnEng - :moneybag: 55k but depending on blabla - 70k - :round_pushpin: Barcelona - :link: `<https://bcneng.org/jobs/senior-go-developer-123|ttps://bcneng.org/jobs/senior-go-developer-123>` - :raised_hands: More info DM <@U2WPLA0KA>",
			valid: false,
		},
		{
			name:  "Offer with a too long max range is invalid",
			text:  ":computer: Senior Go Engineer @ BcnEng - :moneybag: - 70k but depending on blabla - :round_pushpin: Barcelona - :link: `<https://bcneng.org/jobs/senior-go-developer-123|ttps://bcneng.org/jobs/senior-go-developer-123>` - :raised_hands: More info DM <@U2WPLA0KA>",
			valid: false,
		},
		{
			name:  "Offer without max range is invalid",
			text:  ":computer: Senior Go Engineer @ BcnEng - :moneybag: 55k -  - :round_pushpin: Barcelona - :link: `<https://bcneng.org/jobs/senior-go-developer-123|ttps://bcneng.org/jobs/senior-go-developer-123>` - :raised_hands: More info DM <@U2WPLA0KA>",
			valid: false,
		},
		{
			name:  "Offer with an invalid link",
			text:  ":computer: Senior Go Engineer @ BcnEng - :moneybag: 55k - 70k - :round_pushpin: Barcelona - :link: `<wrong-link>` - :raised_hands: More info DM <@U2WPLA0KA>",
			valid: false,
		},
		{
			name:  "Offer with a missing link is invalid",
			text:  ":computer: Senior Go Engineer @ BcnEng - :moneybag: 55k - 70k - :round_pushpin: Barcelona - :link: `` - :raised_hands: More info DM <@U2WPLA0KA>",
			valid: false,
		},
		{
			name:  "Offer with missing role is invalid",
			text:  ":computer:  @ BcnEng - :moneybag: 55k - 70k - :round_pushpin: Barcelona - :link: `<https://bcneng.org/jobs/senior-go-developer-123|ttps://bcneng.org/jobs/senior-go-developer-123>` - :raised_hands: More info DM <@U2WPLA0KA>",
			valid: false,
		},
		{
			name:  "Offer with missing company is invalid",
			text:  ":computer: Senior Go Engineer @  - :moneybag: 55k - 70k - :round_pushpin: Barcelona - :link: `<https://bcneng.org/jobs/senior-go-developer-123|ttps://bcneng.org/jobs/senior-go-developer-123>` - :raised_hands: More info DM <@U2WPLA0KA>",
			valid: false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.valid, isValidJobOffer(test.text), test.text)
		})
	}
}
