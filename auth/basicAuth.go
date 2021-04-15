package auth

import (
	"encoding/base64"
	"net/http"
)

func Basic(user string, pw string) Authenticator {
	return &basicAuth{
		user: user,
		pw:   pw,
	}
}

type basicAuth struct {
	user string
	pw   string
}

// Type identifies the Basic authenticator.
func (b *basicAuth) Type() string {
	return "Basic"
}

// User holds the BasicAuth username.
func (b *basicAuth) User() string {
	return b.user
}

// Password holds the BasicAuth password.
func (b *basicAuth) Password() string {
	return b.pw
}

// Authorize the current request.
func (b *basicAuth) Authorize(req *http.Request) {
	a := b.user + ":" + b.pw
	auth := "Basic " + base64.StdEncoding.EncodeToString([]byte(a))
	req.Header.Set("Authorization", auth)
}
