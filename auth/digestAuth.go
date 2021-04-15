package auth

import (
	md5pkg "crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
)

var _ Authenticator = &DigestAuth{}

// Digest implements HTTP digest authentication.
// see https://tools.ietf.org/html/rfc7616
func Digest(user string, pw string) *DigestAuth {
	return &DigestAuth{
		user:        user,
		pw:          pw,
		digestParts: map[string]string{},
	}
}

// DigestAuth structure holds our credentials.
type DigestAuth struct {
	user        string
	pw          string
	digestParts map[string]string
}

// Type identifies the Digest authenticator.
func (d *DigestAuth) Type() string {
	return "Digest"
}

// User holds the DigestAuth username.
func (d *DigestAuth) User() string {
	return d.user
}

// Password holds the DigestAuth password.
func (d *DigestAuth) Password() string {
	return d.pw
}

// Authorize the current request.
func (d *DigestAuth) Authorize(req *http.Request) {
	d.digestParts["uri"] = req.URL.Path
	d.digestParts["method"] = req.Method
	d.digestParts["username"] = d.user
	d.digestParts["password"] = d.pw
	req.Header.Set("Authorization", getDigestAuthorization(d.digestParts))
}

func (d *DigestAuth) DigestParts(wwwAuthenticateHeader string) Authenticator {
	d.digestParts = map[string]string{}
	if len(wwwAuthenticateHeader) > 0 {
		// unwanted headers: domain, stale, charset, userhash
		wantedHeaders := []string{"nonce", "realm", "qop", "opaque", "algorithm", "entityBody"}
		responseHeaders := strings.Split(wwwAuthenticateHeader, ",")
		for _, r := range responseHeaders {
			for _, w := range wantedHeaders {
				if strings.Contains(r, w) {
					value := strings.SplitN(r, `=`, 2)[1]
					d.digestParts[w] = strings.Trim(value, `"`)
				}
			}
		}
	}
	return d
}

func md5(text string) string {
	hasher := md5pkg.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}

func getCnonce() string {
	b := make([]byte, 8)
	io.ReadFull(rand.Reader, b)
	return fmt.Sprintf("%x", b)[:16]
}

func getDigestAuthorization(d map[string]string) string {
	// These are the correct ha1 and ha2 for qop=auth. We should probably check for other types of qop.

	var (
		ha1        string
		ha2        string
		nonceCount = 00000001
		cnonce     = getCnonce()
		response   string
	)

	// 'ha1' value depends on value of "algorithm" field
	switch d["algorithm"] {
	case "MD5", "":
		ha1 = md5(d["username"] + ":" + d["realm"] + ":" + d["password"])
	case "MD5-sess":
		ha1 = md5(fmt.Sprintf("%s:%v:%s",
			md5(d["username"]+":"+d["realm"]+":"+d["password"]),
			nonceCount,
			cnonce))
	}

	// 'ha2' value depends on value of "qop" field
	switch d["qop"] {
	case "auth", "":
		ha2 = md5(d["method"] + ":" + d["uri"])
	case "auth-int":
		if d["entityBody"] != "" {
			ha2 = md5(d["method"] + ":" + d["uri"] + ":" + md5(d["entityBody"]))
		}
	}

	// 'response' value depends on value of "qop" field
	switch d["qop"] {
	case "":
		response = md5(
			fmt.Sprintf("%s:%s:%s",
				ha1,
				d["nonce"],
				ha2,
			),
		)
	case "auth", "auth-int":
		response = md5(
			fmt.Sprintf("%s:%s:%v:%s:%s:%s",
				ha1,
				d["nonce"],
				nonceCount,
				cnonce,
				d["qop"],
				ha2,
			),
		)
	}

	authorization := fmt.Sprintf(`Digest username="%s", realm="%s", nonce="%s", uri="%s", nc=%v, cnonce="%s", response="%s"`,
		d["username"], d["realm"], d["nonce"], d["uri"], nonceCount, cnonce, response)

	if d["qop"] != "" {
		authorization += fmt.Sprintf(`, qop=%s`, d["qop"])
	}

	if d["opaque"] != "" {
		authorization += fmt.Sprintf(`, opaque="%s"`, d["opaque"])
	}

	return authorization
}
