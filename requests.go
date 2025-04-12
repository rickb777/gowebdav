package gowebdav

import (
	"bytes"
	"fmt"
	authpkg "github.com/rickb777/gowebdav/auth"
	"io"
	"net/http"
	pathpkg "path"
	"strings"
)

func (c *client) request(method, path string, body io.Reader, header http.Header) (req *http.Response, err error) {
	// Tee the body, because if authorization fails we will need to read from it again.
	var r *http.Request
	var ba *bytes.Buffer
	var bb io.Reader
	if body != nil {
		switch v := body.(type) {
		case *bytes.Buffer:
			// two buffers wrapping the same byte slice
			ba = bytes.NewBuffer(v.Bytes())
			bb = bytes.NewReader(v.Bytes())
		default:
			// an extra buffer and tee copying of the bytes
			ba = &bytes.Buffer{}
			bb = io.TeeReader(body, ba)
		}
	}

	u := c.root + pathEscape(path)
	if body == nil {
		r, err = http.NewRequest(method, u, nil)
	} else {
		r, err = http.NewRequest(method, u, bb)
	}

	if err != nil {
		return nil, err
	}

	for k, vals := range c.headers {
		for _, v := range vals {
			r.Header.Add(k, v)
		}
	}

	// Make sure we read 'c.auth' only once because it may be substituted below,
	// which is unsafe to do when multiple goroutines are running at the same time.
	c.authMutex.Lock()
	auth := c.auth
	c.authMutex.Unlock()

	auth.Authorize(r)

	for k, vs := range header {
		r.Header[k] = vs
	}

	res, err := c.hc.Do(r)
	if err != nil {
		return nil, err
	}

	if res.StatusCode == http.StatusUnauthorized && auth.Type() == "noAuth" {
		wwwAuthenticateHeader := res.Header.Get("Www-Authenticate")
		wwwAuthenticateHeaderLC := strings.ToLower(wwwAuthenticateHeader)

		if strings.Contains(wwwAuthenticateHeaderLC, "digest") {
			c.authMutex.Lock()
			c.auth = authpkg.Digest(auth.User(), auth.Password()).DigestParts(wwwAuthenticateHeader)
			c.authMutex.Unlock()
		} else if strings.Contains(wwwAuthenticateHeaderLC, "basic") {
			c.authMutex.Lock()
			c.auth = authpkg.Basic(auth.User(), auth.Password())
			c.authMutex.Unlock()
		} else {
			return res, newPathError("Authorize", c.root, res.StatusCode)
		}

		_ = res.Body.Close()

		if body == nil {
			return c.request(method, path, nil, header)
		} else {
			return c.request(method, path, ba, header)
		}

	} else if res.StatusCode == http.StatusUnauthorized {
		return res, newPathError("Authorize", c.root, res.StatusCode)
	}

	return res, err
}

func (c *client) mkcol(path string) int {
	res, err := c.request(MethodMkcol, withLeadingSlash(path), nil, nil)
	if err != nil {
		return http.StatusBadRequest
	}
	defer res.Body.Close()

	// TODO explain why???
	if res.StatusCode == http.StatusMethodNotAllowed {
		return http.StatusCreated
	}

	return res.StatusCode
}

func (c *client) options(path string) (*http.Response, error) {
	header := make(http.Header)
	header.Add("Depth", "0")
	return c.request(http.MethodOptions, withLeadingSlash(path), nil, header)
}

func (c *client) propfind(path string, self bool, body string, resp interface{}, parse func(resp interface{}) error) error {
	path = withLeadingSlash(path)

	header := make(http.Header)
	if self {
		header.Add("Depth", "0")
	} else {
		header.Add("Depth", "1")
	}
	header.Add("Content-Type", "application/xml;charset=UTF-8")
	header.Add("Accept", "application/xml,text/xml")
	header.Add("Accept-Charset", "utf-8")
	// TODO add support for 'gzip,deflate;q=0.8,q=0.7'
	//header.Add("Accept-Encoding", "")

	res, err := c.request(MethodPropfind, path, strings.NewReader(body), header)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusMultiStatus {
		return fmt.Errorf("%s - %s %s", res.Status, MethodPropfind, path)
	}

	return parseXML(res.Body, resp, parse)
}

func (c *client) copymove(method string, oldpath string, newpath string, overwrite bool) error {
	oldpath = withLeadingSlash(oldpath)
	newpath = withLeadingSlash(newpath)

	header := make(http.Header)
	header.Add("Destination", c.root+newpath)
	if overwrite {
		header.Add("Overwrite", "T")
	} else {
		header.Add("Overwrite", "F")
	}

	res, err := c.request(method, oldpath, nil, header)
	if err != nil {
		return newPathErrorErr(method, oldpath, err)
	}

	defer res.Body.Close()

	switch res.StatusCode {
	case http.StatusCreated, http.StatusNoContent:
		return nil

	case http.StatusMultiStatus:
		// TODO handle multistat errors, worst case ...
		log(fmt.Sprintf(" TODO handle %s - %s multistatus result %s", method, oldpath, readString(res.Body)))

	case http.StatusConflict:
		err := c.createParentCollection(newpath)
		if err != nil {
			return err
		}

		return c.copymove(method, oldpath, newpath, overwrite)
	}

	return newPathError(method, oldpath, res.StatusCode)
}

func (c *client) put(path string, stream io.Reader, header http.Header) (int, error) {
	res, err := c.request(http.MethodPut, withLeadingSlash(path), stream, header)
	if err != nil {
		return 0, err
	}
	_ = res.Body.Close()

	return res.StatusCode, nil
}

func (c *client) createParentCollection(itemPath string) (err error) {
	parentPath := pathpkg.Dir(withLeadingSlash(itemPath))
	if parentPath == "." || parentPath == "/" {
		return nil
	}

	return c.MkdirAll(parentPath, 0755)
}
