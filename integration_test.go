package gowebdav_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"

	"github.com/rickb777/expect"
	"github.com/rickb777/gowebdav"
	"github.com/rickb777/gowebdav/auth"
	"github.com/rickb777/httpclient/logging"
	"github.com/rickb777/httpclient/loggingclient"
	"golang.org/x/net/webdav"
)

var (
	expectedError  string
	expectedErrorµ sync.Mutex
)

func TestIntegration_no_auth(t *testing.T) {
	testIntegration(t, auth.Anonymous)
}

func TestIntegration_basic_auth(t *testing.T) {
	testIntegration(t, auth.Basic("user1", "secret"))
}

func TestIntegration_digest_auth(t *testing.T) {
	testIntegration(t, auth.Digest("user1", "secret"))
}

func TestIntegration_saml_auth(t *testing.T) {
	testIntegration(t, auth.SAML("user1", "secret", "https://tenant123.sharepoint.com/sites/testsitealpha", nil))
}

func testIntegration(t *testing.T, authenticator auth.Authenticator) {
	handler := &webdav.Handler{
		Prefix:     "/a/",
		FileSystem: webdav.NewMemFS(),
		LockSystem: webdav.NewMemLS(),
		Logger: func(req *http.Request, err error) {
			tLogf(t, "%s %s (%v)\n", req.Method, req.URL, err)
			expectedErrorµ.Lock()
			if expectedError == "" {
				expect.Error(err).ToBeNil(t)
			} else {
				expect.Error(err).ToContain(t, expectedError)
			}
			expectedError = ""
			expectedErrorµ.Unlock()
		},
	}

	server := httptest.NewServer(handler)
	server.Client()

	logger := logging.LogWriter(os.Stdout)
	level := logging.Summary
	if testing.Verbose() {
		//level = logging.WithHeaders
		level = logging.WithHeadersAndBodies
	}
	httpClient := loggingclient.New(server.Client(), logger, level)

	client := gowebdav.NewClient(server.URL+"/a",
		gowebdav.SetAuthentication(authenticator),
		gowebdav.SetHttpClient(httpClient))

	tLogf(t, "Ping\n")
	expect.Error(client.Ping()).ToBeNil(t)

	f, err := os.Open("LICENSE")
	must(t, err)

	buf := &bytes.Buffer{}
	_, err = io.Copy(buf, f)
	must(t, err)
	must(t, f.Close())
	content := buf.Bytes()

	tLogf(t, "Mkdir foo\n")
	must(t, client.Mkdir("foo", 0755))

	tLogf(t, "WriteFile foo/LICENSE\n")
	must(t, client.WriteFile("foo/LICENSE", content, "text/plain"))
	buf.Reset()

	tLogf(t, "Stat foo/\n")
	fi1, err := client.Stat("foo/")
	expect.Error(err).ToBeNil(t)
	expect.Bool(fi1.IsDir()).ToBeTrue(t)

	tLogf(t, "Stat foo/LICENSE\n")
	fi2, err := client.Stat("foo/LICENSE")
	expect.Any(err).ToBeNil(t)
	expect.Bool(fi2.IsDir()).ToBeFalse(t)

	tLogf(t, "Mkdir tmp\n")
	must(t, client.Mkdir("tmp", 0755))

	tLogf(t, "Copy foo/LICENSE tmp/copy-of-license\n")
	err = client.Copy("foo/LICENSE", "tmp/copy-of-license")
	expect.Error(err).ToBeNil(t)

	tLogf(t, "Copy foo/LICENSE tmp/copy-of-license2\n")
	err = client.Copy("foo/LICENSE", "tmp/copy-of-license2")
	expect.Error(err).ToBeNil(t)

	tLogf(t, "CopyWithoutOverwriting foo/LICENSE tmp/copy-of-license2\n")
	expectError("file already exists")
	err = client.CopyWithoutOverwriting("foo/LICENSE", "tmp/copy-of-license2")
	expect.Error(err).ToHaveOccurred(t)

	tLogf(t, "ReadFile tmp/copy-of-license\n")
	bs, err := client.ReadFile("tmp/copy-of-license")
	expect.Slice(bs, err).ToHaveLength(t, len(content))

	tLogf(t, "Rename tmp/copy-of-license tmp/other\n")
	err = client.Rename("tmp/copy-of-license", "tmp/other")
	expect.Error(err).ToBeNil(t)

	tLogf(t, "Rename tmp/copy-of-license2 tmp/other\n")
	err = client.Rename("tmp/copy-of-license2", "tmp/other")
	expect.Error(err).ToBeNil(t)

	tLogf(t, "RenameWithoutOverwriting tmp/other foo/LICENSE\n")
	expectError("file already exists")
	err = client.RenameWithoutOverwriting("tmp/other", "foo/LICENSE")
	expect.Error(err).ToHaveOccurred(t)

	tLogf(t, "ReadDir foo\n")
	fis, err := client.ReadDir("foo")
	expect.Slice(fis, err).ToHaveLength(t, 1)

	tLogf(t, "ReadDir tmp\n")
	fis, err = client.ReadDir("tmp")
	expect.Slice(fis, err).ToHaveLength(t, 1)

	tLogf(t, "ReadDir /\n")
	fis, err = client.ReadDir("/")
	expect.Slice(fis, err).ToHaveLength(t, 2)
	expect.String("foo,tmp").ToContain(t, fis[0].Name())
	expect.String("foo,tmp").ToContain(t, fis[1].Name())
	expect.Any(fis[0].Name()).Not().ToBe(t, fis[1].Name())

	tLogf(t, "Remove tmp/other\n")
	err = client.Remove("tmp/other")
	expect.Error(err).ToBeNil(t)

	tLogf(t, "closing...\n")
	//FIXME
	//tLogf(t,"ReadDir /\n")
	//fis, err = client.ReadDir("/")
	//expect.Any(fis, err).To(HaveLen(1))

	server.Close()
}

func must(t *testing.T, err error) {
	t.Helper()
	expect.Error(err).ToBeNil(t)
}

func expectError(msg string) {
	expectedErrorµ.Lock()
	expectedError = msg
	expectedErrorµ.Unlock()
}

func tLogf(t *testing.T, format string, args ...any) {
	if testing.Verbose() {
		t.Helper()
		t.Logf(format, args...)
	}
}
