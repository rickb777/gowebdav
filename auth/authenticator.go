package auth

import "net/http"

// Authenticator stub
type Authenticator interface {
	Type() string
	User() string
	Password() string
	Authorize(*http.Request)
}

var Anonymous Authenticator = &noAuth{}

func Deferred(user string, pw string) Authenticator {
	return &noAuth{
		user: user,
		pw:   pw,
	}
}

// noAuth structure holds our credentials but doesn't use them.
type noAuth struct {
	user string
	pw   string
}

// Type identifies the authenticator.
func (n *noAuth) Type() string {
	return "NoAuth"
}

// User returns the current user.
func (n *noAuth) User() string {
	return n.user
}

// Password returns the current password.
func (n *noAuth) Password() string {
	return n.pw
}

// Authorize the current request
func (n *noAuth) Authorize(_ *http.Request) {}
