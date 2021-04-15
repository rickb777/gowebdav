package netrc

import (
	. "github.com/onsi/gomega"
	"net/url"
	"strings"
	"testing"
)

func TestParseConfig(t *testing.T) {
	g := NewGomegaWithT(t)

	u, _ := url.Parse("https://my.server.com:444/foo")
	cases := map[string]string{
		// blank case
		"|": ``,

		// simple one-liner
		"alpha|secret": `machine my.server.com:444 login alpha password secret`,

		// ignore default and match machine
		"beta|secret": `default
			  login a1
			  password aaa111
			
			machine my.server.com
			  login xyz
			  password xyz123
			
			machine my.server.com:444
			  login beta 
			  password secret
			  account acct`,

		// ignore machine and match default
		"gamma|secret": `default login gamma
			  password secret
			
			machine other.server.com
			  login xyz
			  password xyz123`,
	}
	for e, input := range cases {
		l, p := parseConfig(strings.NewReader(input), u)
		exp := strings.Split(e, "|")
		g.Expect(l).To(Equal(exp[0]))
		g.Expect(p).To(Equal(exp[1]))
	}
}
