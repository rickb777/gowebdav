package netrc

import (
	"github.com/rickb777/expect"
	"net/url"
	"strings"
	"testing"
)

func TestParseConfig(t *testing.T) {

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
		expect.String(l).ToBe(t, exp[0])
		expect.String(p).ToBe(t, exp[1])
	}
}
