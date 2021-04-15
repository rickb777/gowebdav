package netrc

import (
	"bufio"
	"io"
	"net/url"
	"os"
)

// ReadConfig reads login and password configuration from ~/.netrc
// machine foo.com login username password 123456
func ReadConfig(uri, netrc string) (string, string) {
	u, err := url.Parse(uri)
	if err != nil {
		return "", ""
	}

	file, err := os.Open(netrc)
	if err != nil {
		return "", ""
	}
	defer file.Close()

	return parseConfig(file, u)
}

func parseConfig(file io.Reader, u *url.URL) (string, string) {
	// netrc syntax consists of pairs of words
	//
	// "machine" name
	// "login" name
	// "password" name
	// (others are ignored here)
	//
	// The separating whitespace can optionally include newlines.
	// The order of the nouns is normally "machine" then "login" then
	// "password", but we allow "login" and "password" to be swapped.

	var machine, login, password string
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanWords)
	inDefault := false

	for scanner.Scan() {
		id := ""
		noun := scanner.Text()
		if noun == "default" {
			inDefault = true
		} else if scanner.Scan() {
			id = scanner.Text()
		}

		switch noun {
		case "machine":
			inDefault = false
			if id == u.Host {
				machine = id
				login = ""
				password = ""
			}
		case "login":
			if inDefault || machine == u.Host {
				login = id
			}
		case "password":
			if inDefault || machine == u.Host {
				password = id
			}
		}

		if machine == u.Host && login != "" && password != "" {
			return login, password
		}
	}

	return login, password
}
