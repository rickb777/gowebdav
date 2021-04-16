package gowebdav_test

import (
	"bytes"
	"github.com/rickb777/gowebdav/auth"
	"github.com/rickb777/httpclient/logging"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/rickb777/gowebdav"
	"github.com/rickb777/httpclient/loggingclient"
	"golang.org/x/net/webdav"
)

var (
	expectedError   string
	expectedErrorMu sync.Mutex
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
	g := NewGomegaWithT(t)

	handler := &webdav.Handler{
		Prefix:     "/a/",
		FileSystem: webdav.NewMemFS(),
		LockSystem: webdav.NewMemLS(),
		Logger: func(req *http.Request, err error) {
			t.Logf("%s %s (%v)\n", req.Method, req.URL, err)
			expectedErrorMu.Lock()
			if expectedError == "" {
				g.Expect(err).NotTo(HaveOccurred())
			} else {
				g.Expect(err).To(HaveOccurred())
				g.Expect(err.Error()).To(Equal(expectedError))
			}
			expectedError = ""
			expectedErrorMu.Unlock()
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

	t.Logf("Ping\n")
	g.Expect(client.Ping()).NotTo(HaveOccurred())

	f, err := os.Open("LICENSE")
	must(t, err)

	buf := &bytes.Buffer{}
	_, err = io.Copy(buf, f)
	must(t, err)
	must(t, f.Close())
	content := buf.Bytes()

	t.Logf("Mkdir foo\n")
	must(t, client.Mkdir("foo", 0755))

	t.Logf("WriteStream foo/LICENSE\n")
	expectError("file already exists")
	must(t, client.WriteStream("foo/LICENSE", bytes.NewBuffer(content), 0644))
	buf.Reset()

	t.Logf("Stat foo/\n")
	fi1, err := client.Stat("foo/")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(fi1.IsDir()).To(BeTrue())

	t.Logf("Stat foo/LICENSE\n")
	fi2, err := client.Stat("foo/LICENSE")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(fi2.IsDir()).To(BeFalse())

	t.Logf("Mkdir tmp\n")
	must(t, client.Mkdir("tmp", 0755))

	t.Logf("Copy foo/LICENSE tmp/copy-of-license\n")
	err = client.Copy("foo/LICENSE", "tmp/copy-of-license")
	g.Expect(err).NotTo(HaveOccurred())

	t.Logf("Copy foo/LICENSE tmp/copy-of-license2\n")
	err = client.Copy("foo/LICENSE", "tmp/copy-of-license2")
	g.Expect(err).NotTo(HaveOccurred())

	t.Logf("CopyWithoutOverwriting foo/LICENSE tmp/copy-of-license2\n")
	expectError("file already exists")
	err = client.CopyWithoutOverwriting("foo/LICENSE", "tmp/copy-of-license2")
	g.Expect(err).To(HaveOccurred())

	t.Logf("ReadFile tmp/copy-of-license\n")
	bs, err := client.ReadFile("tmp/copy-of-license")
	g.Expect(bs, err).To(HaveLen(len(content)))

	t.Logf("Rename tmp/copy-of-license tmp/other\n")
	err = client.Rename("tmp/copy-of-license", "tmp/other")
	g.Expect(err).NotTo(HaveOccurred())

	t.Logf("Rename tmp/copy-of-license2 tmp/other\n")
	err = client.Rename("tmp/copy-of-license2", "tmp/other")
	g.Expect(err).NotTo(HaveOccurred())

	t.Logf("RenameWithoutOverwriting tmp/other foo/LICENSE\n")
	expectError("file already exists")
	err = client.RenameWithoutOverwriting("tmp/other", "foo/LICENSE")
	g.Expect(err).To(HaveOccurred())

	t.Logf("ReadDir foo\n")
	fis, err := client.ReadDir("foo")
	g.Expect(fis, err).To(HaveLen(1))

	t.Logf("ReadDir tmp\n")
	fis, err = client.ReadDir("tmp")
	g.Expect(fis, err).To(HaveLen(1))

	t.Logf("ReadDir /\n")
	fis, err = client.ReadDir("/")
	g.Expect(fis, err).To(HaveLen(2))
	g.Expect("foo,tmp").To(ContainSubstring(fis[0].Name()))
	g.Expect("foo,tmp").To(ContainSubstring(fis[1].Name()))
	g.Expect(fis[0].Name()).NotTo(Equal(fis[1].Name()))

	t.Logf("Remove tmp/other\n")
	err = client.Remove("tmp/other")
	g.Expect(err).NotTo(HaveOccurred())

	//FIXME
	//t.Logf("ReadDir /\n")
	//fis, err = client.ReadDir("/")
	//g.Expect(fis, err).To(HaveLen(1))

	server.Close()
}

func must(t *testing.T, err error) {
	t.Helper()
	NewGomegaWithT(t).Expect(err).NotTo(HaveOccurred())
}

func expectError(msg string) {
	expectedErrorMu.Lock()
	expectedError = msg
	expectedErrorMu.Unlock()
}
