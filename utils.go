package gowebdav

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

func log(msg interface{}) {
	fmt.Println(msg)
}

func newPathError(op string, path string, statusCode int) error {
	return newPathErrorErr(op, path, fmt.Errorf("%d", statusCode))
}

func newPathErrorErr(op string, path string, err error) error {
	return &os.PathError{
		Op:   op,
		Path: path,
		Err:  err,
	}
}

// pathEscape escapes all segments of a given path
func pathEscape(path string) string {
	s := strings.Split(path, "/")
	for i, e := range s {
		s[i] = url.PathEscape(e)
	}
	return strings.Join(s, "/")
}

// withoutTrailingSlash removes any trailing / from a string
func withoutTrailingSlash(s string) string {
	if strings.HasSuffix(s, "/") {
		return s[:len(s)-1]
	}
	return s
}

// withTrailingSlash appends a trailing / to a string
func withTrailingSlash(s string) string {
	if strings.HasSuffix(s, "/") {
		return s
	}
	return s + "/"
}

// withLeadingSlash prepends a leading / to a string
func withLeadingSlash(s string) string {
	if strings.HasPrefix(s, "/") {
		return s
	}
	return "/" + s
}

// withSurroundingSlashes appends and prepends a / if they are missing
func withSurroundingSlashes(s string) string {
	if !strings.HasPrefix(s, "/") {
		s = "/" + s
	}
	return withTrailingSlash(s)
}

// readString pulls a string out of our io.Reader
func readString(r io.Reader) string {
	buf := new(bytes.Buffer)
	// TODO - make String return an error as well
	_, _ = buf.ReadFrom(r)
	return buf.String()
}

func parseUint(s *string) uint {
	if n, e := strconv.ParseUint(*s, 10, 32); e == nil {
		return uint(n)
	}
	return 0
}

func parseInt64(s *string) int64 {
	if n, e := strconv.ParseInt(*s, 10, 64); e == nil {
		return n
	}
	return 0
}

func parseModified(s *string) time.Time {
	if t, e := time.Parse(time.RFC1123, *s); e == nil {
		return t
	}
	return time.Unix(0, 0)
}

func parseXML(data io.Reader, resp interface{}, parse func(resp interface{}) error) error {
	decoder := xml.NewDecoder(data)
	for t, _ := decoder.Token(); t != nil; t, _ = decoder.Token() {
		switch se := t.(type) {
		case xml.StartElement:
			if se.Name.Local == "response" {
				if e := decoder.DecodeElement(resp, &se); e == nil {
					if err := parse(resp); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}
